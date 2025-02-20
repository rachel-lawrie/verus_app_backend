package services

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	mocks "github.com/rachel-lawrie/verus_backend_core/mocks"
	models "github.com/rachel-lawrie/verus_backend_core/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetFileExtension(t *testing.T) {
	tests := []struct {
		name      string
		mimeType  string
		want      string
		expectErr bool
	}{
		{"Valid MIME type - image/jpeg", "image/jpeg", ".jpeg", false},
		{"Valid MIME type - image/png", "image/png", ".png", false},
		{"Valid MIME type - application/pdf", "application/pdf", ".pdf", false},
		{"Invalid MIME type", "invalid/mime", "", true},
		{"Empty MIME type", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetFileExtension(tt.mimeType)
			if (err != nil) != tt.expectErr {
				t.Errorf("GetFileExtension() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetFileExtension() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateDocument(t *testing.T) {
	tests := []struct {
		name          string
		inputDocument models.Document
		mockError     error
		expectErr     bool
		statusCode    int
		responseBody  string
	}{
		{
			name: "Valid document creation",
			inputDocument: models.Document{
				DocumentID:   "doc12345",
				ApplicantID:  "applicant123",
				DocumentType: models.DocumentPassport,
				FileURL:      "https://example.com/doc12345.pdf",
				FileSize:     2048,
				Status:       models.DocumentUploaded,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			mockError:  nil,
			expectErr:  false,
			statusCode: http.StatusOK,
			responseBody: `{
                "message": "Document created successfully",
                "document_id": "doc12345"
            }`,
		},
		{
			name: "Error during document creation",
			inputDocument: models.Document{
				DocumentID:   "doc67890",
				ApplicantID:  "applicant456",
				DocumentType: models.DocumentUtilityBill,
				FileURL:      "https://example.com/doc67890.pdf",
				FileSize:     4096,
				Status:       models.DocumentUploaded,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			mockError:  assert.AnError,
			expectErr:  true,
			statusCode: http.StatusInternalServerError,
			responseBody: `{
                "error": "Could not create document"
            }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock collection
			mockCollection := new(mocks.MockCollection)

			// Set up mock behavior
			mockCollection.On("UpdateOne", mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("[]*options.UpdateOptions")).Return(nil, tt.mockError)

			// Create JSON body from input document
			jsonBody, _ := json.Marshal(tt.inputDocument)
			body := bytes.NewReader(jsonBody)

			// Create a request and response recorder
			req := httptest.NewRequest(http.MethodPost, "/documents", body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Set up the Gin router and handler
			router := gin.Default()
			router.POST("/documents", func(c *gin.Context) {
				CreateDocument(c, tt.inputDocument.ApplicantID, tt.inputDocument, mockCollection)
			})

			// Perform the test
			router.ServeHTTP(w, req)

			// Assert the response
			assert.Equal(t, tt.statusCode, w.Code)
			assert.JSONEq(t, tt.responseBody, w.Body.String())
		})
	}
}

// TestUploadDocument tests the UploadDocument function
func TestDocumentServiceImpl_UploadDocument_WithMockData(t *testing.T) {
	tests := []struct {
		name          string
		fileName      string
		fileData      []byte
		mimeType      string
		expectErr     bool
		mockError     error
		inputDocument models.Document
	}{
		{
			name:      "Invalid file - Unsupported MIME type",
			fileName:  "document.unknown",
			fileData:  []byte("This is a test file with an unsupported MIME type."),
			mimeType:  "invalid/mime",
			expectErr: true,
		},
		{
			name:      "File with default MIME type",
			fileName:  "defaultfile.bin",
			fileData:  []byte("This file has no valid MIME type."),
			mimeType:  "application/octet-stream",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a multipart form writer
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			// Add the file part with test data
			part, err := writer.CreateFormFile("document", tt.fileName)
			if err != nil {
				t.Fatalf("Error creating form file: %v", err)
			}

			_, err = part.Write(tt.fileData)
			if err != nil {
				t.Fatalf("Error writing data to form file: %v", err)
			}
			writer.Close()

			// Create an HTTP request
			req := httptest.NewRequest(http.MethodPost, "/documents", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			w := httptest.NewRecorder()

			// Initialize the DocumentService
			service := GetDocumentServiceImpl()

			// Create a Gin context with the request
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			// Create a mock collection
			mockCollection := new(mocks.MockCollection)
			mockUploader := new(mocks.MockS3Uploader)
			service.Uploader = mockUploader

			// Set up mock behavior
			mockCollection.On("InsertOne", mock.Anything, tt.inputDocument, mock.AnythingOfType("[]*options.InsertOneOptions")).Return(nil, tt.mockError)

			// Call the function under test
			_, err = service.UploadDocument(c, mockCollection)
			if (err != nil) != tt.expectErr {
				t.Errorf("UploadDocument() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

// TestUploadDocument tests the UploadDocument function
func TestDocumentServiceImpl_UploadDocument_WithActualFile(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name          string
		fileName      string
		fileData      []byte
		mimeType      string
		expectErr     bool
		mockError     error
		country       string
		inputDocument models.Document
	}{
		{
			name:      "Valid file - PDF",
			fileName:  "document.unknown",
			fileData:  []byte("This is a test file with an unsupported MIME type."),
			mimeType:  "invalid/mime",
			expectErr: true,
			country:   "US",
			inputDocument: models.Document{
				DocumentID:   "doc12345",
				ApplicantID:  "applicant123",
				FileURL:      "placeholder",
				DocumentType: models.DocumentPassport,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Prepare a real file for testing
			filePath := "../../../testdata/testDocument.pdf" // Place a valid PDF file in the "testdata" directory.

			// Open the file
			file, err := os.Open(filePath)
			if err != nil {
				t.Fatalf("Failed to open test file: %v", err)
			}
			defer file.Close()

			// Create a multipart form with the file
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			// Manually create the part with custom headers (including Content-Type)
			partHeader := textproto.MIMEHeader{}
			partHeader.Set("Content-Disposition", `form-data; name="document"; filename="testDocument.pdf"`)
			partHeader.Set("Content-Type", "application/pdf")

			// Create the part with custom headers
			part, err := writer.CreatePart(partHeader)
			if err != nil {
				t.Fatalf("Error creating part: %v", err)
			}

			_, err = io.Copy(part, file)
			if err != nil {
				t.Fatalf("Error copying file content to form: %v", err)
			}

			err = writer.WriteField("applicant_id", tt.inputDocument.ApplicantID)
			if err != nil {
				t.Fatalf("Failed to write form field: %v", err)
			}

			err = writer.WriteField("document_type", tt.inputDocument.DocumentType.String())
			if err != nil {
				t.Fatalf("Failed to write form field: %v", err)
			}

			err = writer.WriteField("country", tt.country)
			if err != nil {
				t.Fatalf("Failed to write form field: %v", err)
			}
			// Close the multipart writer
			writer.Close()

			// Create an HTTP request
			req := httptest.NewRequest(http.MethodPost, "/documents", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			w := httptest.NewRecorder()

			// Initialize the DocumentService
			service := GetDocumentServiceImpl()

			// Create a Gin context with the request
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			// Create a mock collection
			mockCollection := new(mocks.MockCollection)
			mockUploader := new(mocks.MockS3Uploader)
			service.Uploader = mockUploader
			// Set up mock behavior
			mockUploader.On("UploadFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("https://example.com/"+tt.fileName, tt.mockError)
			mockCollection.On("UpdateOne", mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("[]*options.UpdateOptions")).Return(nil, tt.mockError)

			// Set up mock behavior
			mockCollection.On("InsertOne", c.Request.Context(), mock.MatchedBy(func(doc models.Document) bool {
				return doc.ApplicantID == "applicant123" &&
					doc.DocumentType == models.DocumentPassport &&
					doc.FileURL == "placeholder" &&
					doc.Status == models.DocumentUploaded &&
					doc.Deleted == false
			}), mock.AnythingOfType("[]*options.InsertOneOptions")).Return(nil, tt.mockError)

			// Call the function under test
			_, err = service.UploadDocument(c, mockCollection)
			if err != nil {
				t.Errorf("UploadDocument() error = %v, expectErr false", err)
			}
		})
	}

}
