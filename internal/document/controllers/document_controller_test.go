package controllers // import "github.com/rachel-lawrie/verus_app_backend/internal/document/controllers"

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	localMocks "github.com/rachel-lawrie/verus_app_backend/internal/mocks"
	"github.com/rachel-lawrie/verus_backend_core/mocks"
	"github.com/rachel-lawrie/verus_backend_core/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestCreateDocument tests the CreateDocument function
func TestCreateDocument(t *testing.T) {
	now := time.Now()
	gin.SetMode(gin.TestMode) // Set Gin to test mode

	tests := []struct {
		name               string
		inputDocument      models.Document
		mockError          error
		expectedStatusCode int
		expectedResponse   map[string]interface{}
	}{
		{
			name: "Success Case",
			inputDocument: models.Document{
				FileURL:      "../../../testdata/testDocument.pdf",
				DocumentID:   "doc12345",
				ApplicantID:  "123",
				DocumentType: models.DocumentPassport,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
			mockError:          nil,
			expectedStatusCode: http.StatusOK,
			expectedResponse: map[string]interface{}{
				"applicant_id":  "123",
				"document_type": float64(models.DocumentPassport),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockService := new(localMocks.MockDocumentService)
			mockService.Uploader = new(mocks.MockS3Uploader)
			mockCollection := new(mocks.MockCollection) // Properly mock the CollectionInterface
			mockUploader := new(mocks.MockS3Uploader)

			// Prepare a real file for testing
			filePath := tt.inputDocument.FileURL // Place a valid PDF file in the "testdata" directory.

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
			partHeader.Set("Content-Disposition", `form-data; name="document"; filename=`+filePath)
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
			// Close the multipart writer
			writer.Close()

			// Create an HTTP request
			req := httptest.NewRequest(http.MethodPost, "/documents", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			w := httptest.NewRecorder()
			// Create a Gin context with the request
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Mock the behavior of UploadDocument with correct arguments
			mockService.On("UploadDocument", c, mockCollection, mockUploader).Return(tt.inputDocument, tt.mockError)
			// Set up mock behavior
			mockCollection.On("InsertOne", c, tt.inputDocument, mock.AnythingOfType("[]*options.InsertOneOptions")).Return(nil, tt.mockError)

			CreateDocument(c, mockService) // Pass the mock service

			// Assert status code
			assert.Equal(t, tt.expectedStatusCode, w.Code)

			// Parse the response
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)

			assert.NoError(t, err)

			// Assert response content
			for key, expectedValue := range tt.expectedResponse {
				assert.Equal(t, expectedValue, response[key])
			}
		})
	}
}

// TestGetDocument tests the GetDocument function
func TestGetDocument(t *testing.T) {
	// Set up Gin in test mode
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Initialize mock service
	mockService := new(localMocks.MockDocumentService)
	mockService.Uploader = new(mocks.MockS3Uploader)

	// Set up the route
	router.GET("/documents/:id", func(c *gin.Context) {
		GetDocument(c, mockService)
	})

	now := time.Now()
	tests := []struct {
		name               string
		docID              string
		applicantID        string
		requestBody        string
		mockReturn         models.Document
		mockError          error
		expectedStatusCode int
		expectedResponse   map[string]interface{}
	}{
		{
			name:               "Success",
			docID:              "123",
			applicantID:        "applicant123",
			requestBody:        `{"applicant_id": "applicant123"}`,
			mockReturn:         models.Document{DocumentID: "123", ApplicantID: "applicant123", DocumentType: models.DocumentPassport, FileURL: "http://example.com/file.pdf", FileSize: 1024, Status: models.DocumentUploaded, CreatedAt: now, UpdatedAt: now},
			mockError:          nil,
			expectedStatusCode: http.StatusOK,
			expectedResponse: map[string]interface{}{
				"document_id":  "123",
				"applicant_id": "applicant123",
			},
		},
		{
			name:               "NotFound",
			docID:              "nonexistent",
			applicantID:        "applicant123",
			requestBody:        `{"applicant_id": "applicant123"}`,
			mockReturn:         models.Document{},
			mockError:          errors.New("document not found"),
			expectedStatusCode: http.StatusNotFound,
			expectedResponse: map[string]interface{}{
				"error": "document not found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the GetDocument behavior based on the test case
			mockService.On("GetDocument", mock.Anything, tt.applicantID, tt.docID, mock.Anything).Return(tt.mockReturn, tt.mockError)

			// Perform the test request
			req, _ := http.NewRequest(http.MethodGet, "/documents/"+tt.docID, nil)
			req.Body = ioutil.NopCloser(strings.NewReader(tt.requestBody))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert the response
			assert.Equal(t, tt.expectedStatusCode, w.Code)

			// Parse the response
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)

			assert.NoError(t, err)

			// Assert response content
			for key, expectedValue := range tt.expectedResponse {
				assert.Equal(t, expectedValue, response[key])
			}
			mockService.AssertExpectations(t)
		})
	}
}

// removeFields removes specified fields from a map.
func removeFields(m map[string]interface{}, fieldsToIgnore []string) {
	for _, field := range fieldsToIgnore {
		delete(m, field)
	}
}

// TestUpdateDocument tests the UpdateDocument function
func TestUpdateDocument(t *testing.T) {
	now := time.Now()
	// Set up Gin in test mode
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Initialize mock service
	mockService := new(localMocks.MockDocumentService)
	mockService.Uploader = new(mocks.MockS3Uploader)

	// Set up the route
	router.PUT("/documents/:id", func(c *gin.Context) {
		UpdateDocument(c, mockService)
	})

	tests := []struct {
		name               string
		docID              string
		applicantID        string
		requestBody        string
		mockReturn         models.Document
		mockError          error
		expectedStatusCode int
		expectedResponse   map[string]interface{}
	}{
		{
			name:               "Success",
			docID:              "123",
			applicantID:        "applicant123",
			requestBody:        `{"applicant_id": "applicant123", "status": "` + models.DocumentVerified.String() + `"}`,
			mockReturn:         models.Document{DocumentID: "123", ApplicantID: "applicant123", DocumentType: models.DocumentPassport, FileURL: "http://example.com/file.pdf", FileSize: 1024, Status: models.DocumentVerified, CreatedAt: now, UpdatedAt: now},
			mockError:          nil,
			expectedStatusCode: http.StatusOK,
			expectedResponse: map[string]interface{}{
				"document_id": "123",
				"status":      float64(models.DocumentVerified),
			},
		},
		{
			name:               "InvalidJSON",
			docID:              "123",
			applicantID:        "applicant123",
			requestBody:        `{"applicant_id": "applicant123", "status": }`, // Invalid JSON
			mockReturn:         models.Document{},
			mockError:          nil,
			expectedStatusCode: http.StatusBadRequest,
			expectedResponse: map[string]interface{}{
				"error": "Invalid JSON",
			},
		},
		{
			name:               "DocumentNotFound",
			docID:              "nonexistent",
			applicantID:        "applicant123",
			requestBody:        `{"applicant_id": "applicant123", "status": "` + models.DocumentVerified.String() + `"}`,
			mockReturn:         models.Document{},
			mockError:          errors.New("document not found"),
			expectedStatusCode: http.StatusNotFound,
			expectedResponse: map[string]interface{}{
				"error": "document not found",
			},
		},
		{
			name:               "InvalidStatus",
			docID:              "123",
			applicantID:        "applicant123",
			requestBody:        `{"applicant_id": "applicant123", "status": "invalid_status"}`, // Invalid status
			mockReturn:         models.Document{},
			mockError:          nil,
			expectedStatusCode: http.StatusBadRequest,
			expectedResponse: map[string]interface{}{
				"error": "Invalid status",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the UpdateDocument behavior based on the test case
			mockService.On("UpdateDocument", mock.Anything, tt.applicantID, tt.docID, mock.Anything).Return(tt.mockReturn, tt.mockError)

			// Perform the test request
			req, _ := http.NewRequest(http.MethodPut, "/documents/"+tt.docID, nil)
			req.Body = ioutil.NopCloser(strings.NewReader(tt.requestBody))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert the response
			assert.Equal(t, tt.expectedStatusCode, w.Code)
			// Parse the response
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			t.Logf("Response: %v", response)

			// Assert response content
			for key, expectedValue := range tt.expectedResponse {
				assert.Equal(t, expectedValue, response[key])
			}
		})
	}
}

func assertDocumentResponse(w *httptest.ResponseRecorder, t *testing.T) {
	var expectedMap, actualMap map[string]interface{}

	err := json.Unmarshal([]byte(w.Body.String()), &actualMap)
	assert.NoError(t, err, "Failed to unmarshal actual JSON")

	fieldsToIgnore := []string{"created_at", "deleted", "deleted_at", "deleted_by", "updated_at", "file_url", "file_size", "client_id", "applicant_id"}
	removeFields(actualMap, fieldsToIgnore)

	expectedResponse := `{"document_id":"123", "document_type":0, "status":0}`
	err = json.Unmarshal([]byte(expectedResponse), &expectedMap)
	assert.NoError(t, err, "Failed to unmarshal expected JSON")

	assert.Equal(t, expectedMap, actualMap)
}
