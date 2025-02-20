package interfaces

import (
	"context"

	"mime/multipart"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/rachel-lawrie/verus_backend_core/common"
	models "github.com/rachel-lawrie/verus_backend_core/models"
)

// DocumentService defines the methods available for document operations
type DocumentService interface {
	// UploadDocument handles the upload of a document and returns metadata
	UploadDocument(c *gin.Context, collection common.CollectionInterface) (models.Document, error)

	// GetDocumentByID retrieves a document by its ID
	GetDocument(c *gin.Context, applicantID string, docID string, collection common.CollectionInterface) (models.Document, error)

	// UpdateDocument updates a document by its ID with new data
	UpdateDocument(c *gin.Context, applicantID string, docID string, status models.DocumentStatus) (models.Document, error)

	DownloadDocument(c *gin.Context, docID string, applicantID string, collection common.CollectionInterface) (string, error)
}

// ApplicantService defines the methods available for applicant operations
type ApplicantService interface {
	// UploadApplicant handles the upload of a applicant and returns metadata
	CreateApplicant(c *gin.Context, applicant *models.Applicant) (models.Applicant, error)

	// GetAllApplicants retrieves all applicants
	GetAllApplicants(c *gin.Context) ([]models.Applicant, error)

	// GetApplicantByID retrieves a applicant by its ID
	GetApplicant(c *gin.Context, applicantID string) (models.Applicant, error)

	// UpdateApplicant updates a applicant by its ID with new data
	UpdateApplicant(c *gin.Context, applicantID string, updates map[string]interface{}) (models.Applicant, error)
}

// Uploader defines the method that an uploader must implement
type Uploader interface {
	UploadFile(ctx context.Context, file multipart.File, fileName string, mimeType string, kmsUploader KMSUploader) (string, error)

	// DownloadFile downloads a file from S3 and returns the GetObjectOutput or an error
	DownloadFile(ctx context.Context, objectKey string) (*s3.GetObjectOutput, error)
}

// KMSUploader defines the methods available for KMS operations
type KMSUploader interface {
	GenerateDataKey(ctx context.Context) ([]byte, []byte, error)       // Returns plaintext and encrypted keys
	EncryptData(ctx context.Context, plaintext []byte) ([]byte, error) // Encrypts plaintext data
	DecryptData(ctx context.Context, encrypted []byte) ([]byte, error) // Decrypts encrypted data
}

type KMSClient interface {
	GenerateDataKey(ctx context.Context, input *kms.GenerateDataKeyInput, opts ...func(*kms.Options)) (*kms.GenerateDataKeyOutput, error)
	Encrypt(ctx context.Context, input *kms.EncryptInput, opts ...func(*kms.Options)) (*kms.EncryptOutput, error)
	Decrypt(ctx context.Context, input *kms.DecryptInput, opts ...func(*kms.Options)) (*kms.DecryptOutput, error)
}
