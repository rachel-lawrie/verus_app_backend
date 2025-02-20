package controllers

import (
	"net/http"

	"github.com/rachel-lawrie/verus_app_backend/internal/interfaces"
	"github.com/rachel-lawrie/verus_backend_core/common"
	models "github.com/rachel-lawrie/verus_backend_core/models"

	"github.com/gin-gonic/gin"
)

// CreateDocument handles the document upload and responds with metadata
func CreateDocument(c *gin.Context, service interfaces.DocumentService) {

	// Set content type to application/json
	c.Header("Content-Type", "application/json")
	collection := common.GetCollection("applicants")

	// Call the upload service to handle the file upload
	doc, err := service.UploadDocument(c, collection)
	if err != nil {
		// Return a JSON response with an error message
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Respond with document metadata as JSON
	c.JSON(http.StatusOK, doc)
}

// GetDocument is the handler function for retrieving document metadata by ID
func GetDocument(c *gin.Context, service interfaces.DocumentService) {
	// Get the document ID from the URL parameter
	docID := c.Param("id")

	// Get the status from the JSON request body
	var requestBody struct {
		ApplicantID string `json:"applicant_id"`
	}

	// Bind the request body to the struct
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	collection := common.GetCollection("applicants")

	// Call the service to retrieve the document metadata
	doc, err := service.GetDocument(c, requestBody.ApplicantID, docID, collection)
	if err != nil {
		// Return a JSON response with an error message if document not found
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Respond with the document metadata
	c.JSON(http.StatusOK, doc)
}

// UpdateDocument is the handler function for updating the status of a document
func UpdateDocument(c *gin.Context, service interfaces.DocumentService) {
	// Get the document ID from the URL parameter
	docID := c.Param("id")

	// Get the status from the JSON request body
	var requestBody struct {
		Status      string `json:"status"`
		ApplicantID string `json:"applicant_id"`
	}

	// Bind the request body to the struct
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	// Call the service to update the document status
	status, err := models.ParseDocumentStatus(requestBody.Status)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}
	doc, err := service.UpdateDocument(c, requestBody.ApplicantID, docID, status)
	if err != nil {
		// Return a JSON response with an error message if document not found
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Respond with the updated document metadata
	c.JSON(http.StatusOK, doc)
}

// SaveDocument is the handler function for saving a document locally for testing from S3 bucket
func SaveDocument(c *gin.Context, service interfaces.DocumentService) {
	// Step 1: Get document ID from the header
	docID := c.Param("id")
	if docID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document id parameter is required"})
		return
	}

	// Step 2: Get applicant ID from the JSON request body
	var requestBody struct {
		ApplicantID string `json:"applicant_id"`
	}
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	if requestBody.ApplicantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Applicant ID is required"})
		return
	}

	// Step 3: Get the MongoDB collection
	collection := common.GetCollection("applicants")

	// Step 4: Call the service to save the file locally
	filePath, err := service.DownloadDocument(c, docID, requestBody.ApplicantID, collection)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Step 5: Respond with the local file path
	c.JSON(http.StatusOK, gin.H{"message": "File saved successfully", "file_path": filePath})
}
