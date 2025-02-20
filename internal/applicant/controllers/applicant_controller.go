package controllers

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rachel-lawrie/verus_app_backend/internal/interfaces"
	"github.com/rachel-lawrie/verus_backend_core/models"
	"github.com/rachel-lawrie/verus_backend_core/models_sumsub"
	"github.com/rachel-lawrie/verus_backend_core/utils"
	"github.com/rachel-lawrie/verus_backend_core/zaplogger"
	"go.uber.org/zap"
)

// createApplicantObject creates a new applicant object with provided name, dob, address, email, phone and auto-generates fields like applicant id and timestamps.

func createApplicantObject(firstName, middleName, lastName, email, phone, level string, encryptedData models.EncryptedData) models.Applicant {
	return models.Applicant{
		ApplicantID:       uuid.New().String(),       // Generate a unique ID for the applicant
		FirstName:         firstName,                 // Set the provided name
		MiddleName:        middleName,                // Set the provided middle name
		LastName:          lastName,                  // Set the provided last name
		Email:             email,                     // Email can be set later if required
		Phone:             phone,                     // Phone can be set later if required
		ClientID:          "placeholder",             // Associate with a client ID if available
		EncryptedData:     encryptedData,             // Encrypted DOB and address
		CreatedAt:         time.Now(),                // Set the current time as creation time
		UpdatedAt:         time.Now(),                // Set the current time as the last update time
		Deleted:           false,                     // Set the applicant as active (not deleted)
		DeletedAt:         nil,                       // No deletion timestamp initially
		DeletedBy:         nil,                       // No deletion information initially
		Documents:         []models.Document{},       // No documents initially
		SumsubApplicant:   models_sumsub.Applicant{}, // No Sumsub applicant initially
		VerificationLevel: level,                     // Verification level
	}
}

func CreateApplicant(c *gin.Context, service interfaces.ApplicantService, kmsUploader interfaces.KMSUploader) {
	logger := zaplogger.GetLogger()
	// Define the input struct for the applicant
	var input struct {
		FirstName  string            `json:"first_name" binding:"required"`
		MiddleName string            `json:"middle_name" binding:"required"`
		LastName   string            `json:"last_name" binding:"required"`
		Email      string            `json:"email" binding:"required"` // Applicant's email address
		Phone      string            `json:"phone" binding:"required"` // Applicant's phone number
		Address    models.RawAddress `json:"address" binding:"required"`
		DOB        string            `json:"dob" binding:"required"`   // Applicant's date of birth
		Level      string            `json:"level" binding:"required"` // Verification level
	}

	// Set content type to application/json
	c.Header("Content-Type", "application/json")

	// Read and log the raw body
	bodyBytes, err := c.GetRawData()
	if err != nil {
		logger.Error("Error reading raw body: ", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Could not read request body"})
		return
	}
	logger.Debug("Raw request body: ", zap.String("body", string(bodyBytes)))

	// Set the body back
	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("CreateApplicant: Error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate a DEK using KMSUploader
	plaintextKey, encryptedKey, err := kmsUploader.GenerateDataKey(c.Request.Context())
	if err != nil {
		log.Printf("Error generating data key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create applicant"})
		return
	}

	// Encrypt DOB
	encryptedDOB, err := utils.EncryptField(input.DOB, plaintextKey)
	if err != nil {
		log.Printf("Error encrypting DOB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create applicant"})
		return
	}

	// Encrypt Address
	address := models.RawAddress{
		Line1:      input.Address.Line1,
		Line2:      input.Address.Line2,
		City:       input.Address.City,
		Region:     input.Address.Region,
		PostalCode: input.Address.PostalCode,
		Country:    input.Address.Country,
	}

	encryptedAddress, err := utils.EncryptAddress(address, plaintextKey)
	if err != nil {
		log.Printf("Error encrypting address: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create applicant"})
		return
	}

	encryptedData := models.EncryptedData{
		DOB:          encryptedDOB,
		Address:      encryptedAddress,
		EncryptedKey: encryptedKey,
	}

	applicant := createApplicantObject(input.FirstName, input.MiddleName, input.LastName, input.Email, input.Phone, input.Level, encryptedData)

	// Log the full applicant object before insertion
	log.Printf("CreateApplicant: Applicant object to be inserted: %+v", applicant)

	// Call the upload service to handle the file upload
	applicant, err = service.CreateApplicant(c, &applicant)
	if err != nil {
		log.Printf("CreateApplicant: Error creating applicant: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create applicant"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Applicant created successfully", "applicant_id": applicant.ApplicantID})
}

// GetAllApplicants is the handler function for retrieving all applicants
func GetAllApplicants(c *gin.Context, service interfaces.ApplicantService) {
	applicants, err := service.GetAllApplicants(c)
	if err != nil {
		log.Printf("GetAllApplicants: Error retrieving applicants: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve applicants"})
		return
	}

	// Respond with the list of applicants
	c.JSON(http.StatusOK, applicants)
}

// GetDocument is the handler function for retrieving document metadata by ID
func GetApplicant(c *gin.Context, service interfaces.ApplicantService) {
	// Get the document ID from the URL parameter
	appliantID := c.Param("id")
	log.Printf("GetApplicant: Applicant ID: %v", appliantID)

	applicant, err := service.GetApplicant(c, appliantID)

	if err != nil {
		// Return a JSON response with an error message if document not found
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	// Respond with the document metadata
	c.JSON(http.StatusOK, applicant)
}

// UpdateDocument is the handler function for updating the status of a document
func UpdateApplicant(c *gin.Context, service interfaces.ApplicantService) {
	// Get the document ID from the URL parameter
	appliantID := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		log.Printf("UpdateApplicant: Error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	doc, err := service.UpdateApplicant(c, appliantID, updates)
	if err != nil {
		// Return a JSON response with an error message if document not found
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Respond with the updated document metadata
	c.JSON(http.StatusOK, doc)
}
