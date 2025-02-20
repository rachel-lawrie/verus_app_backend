package services

import (
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rachel-lawrie/verus_backend_core/common"
	"github.com/rachel-lawrie/verus_backend_core/constants"
	"github.com/rachel-lawrie/verus_backend_core/interfaces"
	models "github.com/rachel-lawrie/verus_backend_core/models"
	"go.mongodb.org/mongo-driver/bson"
)

// DocumentServiceImpl is the concrete implementation of the DocumentService interface
type DocumentServiceImpl struct {
	Uploader       interfaces.Uploader
	KMSUploader    interfaces.KMSUploader
	CollectionName string
}

var (
	instance DocumentServiceImpl
	once     sync.Once
)

func GetDocumentServiceImpl() DocumentServiceImpl {
	once.Do(func() {
		instance = DocumentServiceImpl{
			CollectionName: constants.CollectionApplicants,
		}
	})
	return instance
}

// A simple in-memory store for demo purposes (use a database in production)
var documents = make(map[string]models.Document)
var mu sync.Mutex // Mutex to ensure thread-safety for map access

var mimeTypeToExtension = map[string]string{
	"application/pdf": ".pdf",
	"image/jpeg":      ".jpeg",
	"image/png":       ".png",
}

// GetFileExtension returns the file extension based on MIME type
func GetFileExtension(mimeType string) (string, error) {
	if ext, ok := mimeTypeToExtension[mimeType]; ok {
		return ext, nil
	}

	// Use mime.ExtensionsByType for unknown MIME types
	exts, err := mime.ExtensionsByType(mimeType)
	if err != nil || len(exts) == 0 {
		return "", fmt.Errorf("unsupported MIME type: %s", mimeType)
	}
	return exts[0], nil
}

// UploadDocument handles the file upload and saves the document
func (s *DocumentServiceImpl) UploadDocument(c *gin.Context, collection common.CollectionInterface) (models.Document, error) {
	r := c.Request
	// Parse the form data (including file)
	err := r.ParseMultipartForm(10 << 20) // 10MB max file size
	if err != nil {
		return models.Document{}, fmt.Errorf("unable to parse form data: %v", err)
	}

	// Get the file from the request
	file, fileHeader, err := r.FormFile("document")
	if err != nil {
		return models.Document{}, fmt.Errorf("unable to retrieve the file: %v", err)
	}
	defer file.Close()

	// Get MIME type of the uploaded file
	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		return models.Document{}, fmt.Errorf("unable to determine MIME type")
	}

	// Check for known MIME types and return an error if unsupported
	if _, ok := mimeTypeToExtension[mimeType]; !ok {
		return models.Document{}, fmt.Errorf("unsupported MIME type: %s", mimeType)
	}

	// Determine the file extension based on MIME type
	ext, err := GetFileExtension(mimeType)
	if err != nil {
		return models.Document{}, fmt.Errorf("unsupported file extension type: %v", mimeType)
	}

	applicantID := r.FormValue("applicant_id")
	if applicantID == "" {
		return models.Document{}, fmt.Errorf("applicant_id is required")
	}
	documentType := r.FormValue("document_type")
	if documentType == "" {
		return models.Document{}, fmt.Errorf("document_type is required")
	}
	country := r.FormValue("country")
	if country == "" {
		return models.Document{}, fmt.Errorf("country is required")
	}

	// Create document metadata
	doc := createDocumentObject(applicantID, documentType, country)
	fileName := doc.DocumentID + ext

	// Upload file to S3
	fileURL, err := s.Uploader.UploadFile(c, file, fileName, mimeType, s.KMSUploader)
	if err != nil {
		return models.Document{}, fmt.Errorf("error uploading file to S3: %v", err)
	}
	doc.FileURL = fileURL

	mu.Lock()
	CreateDocument(c, applicantID, doc, collection)
	mu.Unlock()

	// Return document metadata along with success
	return doc, nil
}

// createApplicantObject creates a new applicant object with provided name, dob, address, email, phone and auto-generates fields like applicant id and timestamps.
func createDocumentObject(applicantID, documentType, country string) models.Document {
	now := time.Now()
	document_type, _ := models.ParseDocumentType(documentType)
	return models.Document{
		DocumentID:   uuid.New().String(), // Generate a unique ID for the document
		ApplicantID:  applicantID,         // Set the Applicant ID
		DocumentType: document_type,       // Set the document type
		Country:      country,             // Set the country
		FileURL:      "placeholder",       // Set placeholder URL then update after saving the file
		FileSize:     0,
		Status:       models.DocumentUploaded,
		CreatedAt:    now,
		UpdatedAt:    now,
		Deleted:      false,
		DeletedAt:    nil,
		DeletedBy:    nil,
	}
}

func CreateDocument(c *gin.Context, applicantID string, document models.Document, collection common.CollectionInterface) {

	// Log the full document object before insertion
	log.Printf("CreateDocument: Document object to be inserted: %+v", document)

	filter := bson.M{"applicant_id": applicantID, "deleted": false}
	update := bson.M{
		"$push": bson.M{
			"documents": document, // Add the new document
		},
	}

	_, err := collection.UpdateOne(c.Request.Context(), filter, update)
	if err != nil {
		log.Printf("Error inserting document into MongoDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create document"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Document created successfully", "document_id": document.DocumentID})
}

func (s *DocumentServiceImpl) GetDocument(c *gin.Context, applicantID string, docID string, collection common.CollectionInterface) (models.Document, error) {

	collectionName := constants.CollectionApplicants
	log.Println("Using MongoDB collection:", collectionName)

	filter, cacheKey, err := GenerateFilterAndCacheKey(applicantID, docID, collectionName)
	if err != nil {
		log.Printf("Error generating filter and cache key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating filter and cache key"})
		return models.Document{}, err
	}

	projection := bson.M{
		"documents": bson.M{
			"$elemMatch": bson.M{"document_id": docID},
		},
	}

	var result struct {
		Documents []models.Document `bson:"documents"`
	}

	if err != nil {
		return models.Document{}, err
	}
	err = common.CacheWrapper(c, collectionName, cacheKey, filter, projection, &result)
	if err != nil {
		return models.Document{}, err
	}

	// Extract the matched document
	if len(result.Documents) > 0 {
		return result.Documents[0], nil
	}

	return models.Document{}, nil
}

func (s *DocumentServiceImpl) UpdateDocument(c *gin.Context, applicantID string, docID string, status models.DocumentStatus) (models.Document, error) {
	collectionName := constants.CollectionApplicants
	collection := common.GetCollection(collectionName)
	fmt.Println("Using MongoDB collection:", collectionName)

	filter, cacheKey, err := GenerateFilterAndCacheKey(applicantID, docID, collectionName)
	if err != nil {
		log.Printf("Error generating filter and cache key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating filter and cache key"})
		return models.Document{}, err
	}
	// Use $set with a positional operator to update the specific document
	update := bson.M{
		"$set": bson.M{
			"documents.$.status":     status,
			"documents.$.updated_at": time.Now(),
		},
	}
	// Invalidate the cache after a successful update
	err = common.InvalidateCache(c, collectionName, cacheKey, filter, update, nil)
	if err != nil {
		log.Printf("Error invalidating cache: %v : %s", err, cacheKey)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error invalidating cache"})
		return models.Document{}, err
	}

	// Retrieve the updated document
	result, err := s.GetDocument(c, applicantID, docID, collection)
	if err != nil {
		log.Printf("Error retrieving updated document: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve updated document"})
		return models.Document{}, err
	}
	return result, err
}

// Test download from S3
// Save the file to the local filesystem
func (s *DocumentServiceImpl) DownloadDocument(c *gin.Context, docID string, applicantID string, collection common.CollectionInterface) (string, error) {

	// Step 3: Get the file URL from MongoDB using documentID and clientID
	var applicant struct {
		Documents []models.Document `bson:"documents"`
	}

	filter := bson.M{
		"applicant_id": applicantID,
	}

	err := collection.FindOne(c.Request.Context(), filter).Decode(&applicant)
	if err != nil {
		return "", fmt.Errorf("failed to find document in database: %v", err)
	}

	var fileURL string
	for _, doc := range applicant.Documents {
		if doc.DocumentID == docID {
			fileURL = doc.FileURL
			break
		}
	}

	if fileURL == "" {
		return "", fmt.Errorf("document with ID %s not found for client %s", docID, applicantID)
	}

	// Step 4: Create the local file path + url
	dirPath := "../../../saveddoc/"
	filePath := dirPath + docID

	// Ensure the directory exists
	err = os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to create directory: %v", err)
	}

	// Step 5: create object key from url
	objectKey, err := getObjectKeyFromURL(fileURL)
	if err != nil {
		return "", fmt.Errorf("failed to extract object key from URL: %v", err)
	}

	// Step 6: Get the file from the S3 bucket
	output, err := s.Uploader.DownloadFile(c.Request.Context(), objectKey)
	if err != nil {
		return "", fmt.Errorf("failed to download file from S3: %v", err)
	}
	defer output.Body.Close()

	// Step 7: Create a new file in the local filesystem
	outFile, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create local file: %v", err)
	}
	defer outFile.Close()

	// Step 8: Copy the file content from S3 to the local file
	_, err = io.Copy(outFile, output.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save file to local filesystem: %v", err)
	}

	return filePath, nil
}

// Helper function to get objectkey for S3 request
func getObjectKeyFromURL(fileURL string) (string, error) {
	// Parse the file URL
	parsedURL, err := url.Parse(fileURL)
	if err != nil {
		return "", fmt.Errorf("invalid file URL: %v", err)
	}

	// Extract the path from the URL (everything after amazonaws.com/)
	objectKey := strings.TrimPrefix(parsedURL.Path, "/")

	if objectKey == "" {
		return "", fmt.Errorf("failed to extract object key from URL")
	}

	return objectKey, nil
}

// GenerateFilterAndCacheKey generates the filter and cache key for a document
func GenerateFilterAndCacheKey(applicantID, docID, collectionName string) (bson.M, string, error) {
	filter := bson.M{
		"applicant_id":          applicantID,
		"deleted":               false,
		"documents.document_id": docID, // Ensure the specific document exists in the array
	}
	cacheKey, err := common.GenerateCacheKey(collectionName, filter)
	if err != nil {
		return nil, "", err
	}
	return filter, cacheKey, nil
}
