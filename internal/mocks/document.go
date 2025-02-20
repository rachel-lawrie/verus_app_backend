package mocks

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/rachel-lawrie/verus_backend_core/common"
	"github.com/rachel-lawrie/verus_backend_core/interfaces"
	"github.com/rachel-lawrie/verus_backend_core/models"
	"github.com/stretchr/testify/mock"
)

// Mocking the UploadDocument service
type MockDocumentService struct {
	mock.Mock
	Uploader interfaces.Uploader
}

func (m *MockDocumentService) UploadDocument(c *gin.Context, collection common.CollectionInterface) (models.Document, error) {
	// Mock the behavior here
	r := c.Request
	// Parse the form data (including file)
	err := r.ParseMultipartForm(10 << 20) // 10MB max file size
	if err != nil {
		return models.Document{}, fmt.Errorf("unable to parse form data: %v", err)
	}
	// Get the file from the request
	applicant_id := r.FormValue("applicant_id")
	document_type := r.FormValue("document_type")
	documentType, _ := models.ParseDocumentType(document_type)
	return models.Document{
		ApplicantID:  applicant_id,
		DocumentType: documentType,
		Status:       models.DocumentUploaded,
	}, nil
}

func (m *MockDocumentService) GetDocument(c *gin.Context, applicantID, docID string, collection common.CollectionInterface) (models.Document, error) {
	args := m.Called(c, applicantID, docID, collection)
	return args.Get(0).(models.Document), args.Error(1)
}

func (m *MockDocumentService) UpdateDocument(c *gin.Context, applicantID, docID string, status models.DocumentStatus) (models.Document, error) {
	args := m.Called(c, applicantID, docID, status)
	return args.Get(0).(models.Document), args.Error(1)
}

// Mock implementation of DownloadDocument
func (m *MockDocumentService) DownloadDocument(c *gin.Context, docID string, applicantID string, collection common.CollectionInterface) (string, error) {
	args := m.Called(c, m.Uploader, collection)
	return args.String(0), args.Error(1)
}
