package controller

import (
	"github.com/gin-gonic/gin"
	applicationControllers "github.com/rachel-lawrie/verus_app_backend/internal/applicant/controllers"
	applicantServices "github.com/rachel-lawrie/verus_app_backend/internal/applicant/services"
	"github.com/rachel-lawrie/verus_app_backend/internal/auth/middleware"
	documentControllers "github.com/rachel-lawrie/verus_app_backend/internal/document/controllers"
	documentServices "github.com/rachel-lawrie/verus_app_backend/internal/document/services"
	"github.com/rachel-lawrie/verus_backend_core/auth"
	"github.com/rachel-lawrie/verus_backend_core/common"
	"github.com/rachel-lawrie/verus_backend_core/models"
	"github.com/rachel-lawrie/verus_backend_core/utils"
	"github.com/rachel-lawrie/verus_backend_core/zaplogger"
	"go.uber.org/zap"
)

func (c *controller) InitializeRoutes() {
	r := c.router
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "The server is up and working :-)",
		})
	})

	r.GET("/success", c.Success)
	r.GET("/error", c.Error)

	r.Static("/docs", "./docs")

	ApiRouting(r, c.cfg)
}

func ApiRouting(r *gin.Engine, cfg *models.Config) {

	logger := zaplogger.GetLogger()

	kmsUploader, err := utils.NewKMSUploader(cfg.AWS.Region, cfg.AWS.AccessKeyID, cfg.AWS.SecretAccessKey, cfg.AWS.KeyID)
	if err != nil {
		logger.Fatal("Failed to initialize KMS uploader",
			zap.Error(err),
		)
	}

	vehicles := r.Group("/api")
	v1 := vehicles.Group("/v1")

	// Group for routes that require API key authentication
	protected := v1.Group("/protected")
	protected.Use(middleware.APIKeyAuthMiddleware(common.GetCollection("client_secrets_table")))
	{

		// Initialize applicant service
		applicantService := applicantServices.GetApplicantServiceImpl()
		protected.POST("/applicants", func(c *gin.Context) {
			applicationControllers.CreateApplicant(c, &applicantService, kmsUploader)
		})

		protected.PUT("/applicants/:id", func(c *gin.Context) {
			applicationControllers.UpdateApplicant(c, &applicantService)
		})

		// Initialize S3 uploader
		uploader, err := utils.NewS3Uploader(cfg.AWS.BucketName, cfg.AWS.Region, cfg.AWS.AccessKeyID, cfg.AWS.SecretAccessKey)
		if err != nil {
			logger.Fatal("Failed to initialize S3 uploader",
				zap.Error(err),
			)
		}

		documentService := documentServices.GetDocumentServiceImpl()
		documentService.Uploader = uploader
		documentService.KMSUploader = kmsUploader

		protected.POST("/documents", func(c *gin.Context) {
			documentControllers.CreateDocument(c, &documentService)
		})

		protected.GET("/documents/:id", func(c *gin.Context) {
			documentControllers.GetDocument(c, &documentService)
		})

		protected.POST("/downloads/:id", func(c *gin.Context) {
			documentControllers.SaveDocument(c, &documentService)
		})

		protected.PUT("/documents/:id", func(c *gin.Context) {
			documentControllers.UpdateDocument(c, &documentService)
		})
	}

	// Group for routes that require JWT or API key authentication
	protected2 := v1.Group("/protected2")
	protected2.Use(auth.CombinedAuthMiddleware(common.GetCollection("client_secrets_table")))
	{
		// Initialize applicant service
		applicantService := applicantServices.GetApplicantServiceImpl()

		protected2.GET("/applicants", func(c *gin.Context) {
			applicationControllers.GetAllApplicants(c, &applicantService)
		})

		protected2.GET("/applicants/:id", func(c *gin.Context) {
			applicationControllers.GetApplicant(c, &applicantService)
		})
	}
}
