package main

import (
	"github.com/rachel-lawrie/verus_app_backend/internal/app/controller"
	"github.com/rachel-lawrie/verus_backend_core/common"
	"github.com/rachel-lawrie/verus_backend_core/errors"
	"github.com/rachel-lawrie/verus_backend_core/zaplogger"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/rachel-lawrie/verus_app_backend/app"
	"github.com/rachel-lawrie/verus_app_backend/internal/config"
)

const ENV = "dev"

func main() {

	// Initialize Zap logger
	logger := zaplogger.GetLogger()
	defer logger.Sync()

	// Add Loki-compatible labels/fields
	logger = logger.With(
		zap.String("app", "verus"),
		zap.String("env", ENV),
	)

	// Load the configuration for the dev environment
	cfg := config.LoadConfig(ENV)

	logger.Debug("Running application with configuration",
		zap.Any("config", cfg),
	)

	// Log the start of the dev server
	logger.Info("Starting dev server on port ",
		zap.String("port", cfg.Server.Port),
	)

	// Connect to the database
	if err := common.ConnectDatabase(cfg.Database); err != nil {
		// Log the fatal error with context and exit the application
		logger.Fatal("Critical error occurred",
			zap.Error(err), // Log the error
			zap.String("action", "connecting to database"), // Log the action
		)
	}

	// Set Gin to release mode
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	// Use recovery middleware
	r.Use(gin.Recovery())

	// Add global error handler middleware
	r.Use(errors.ErrorHandler())

	// Add Zap logger middleware
	r.Use(zaplogger.ZapLogger(logger))

	appController := controller.New(controller.Params{
		Router: r,
		Config: &cfg,
	})
	appController.InitializeRoutes()

	serviceApp := app.Build(app.Params{
		Router:     r,
		Controller: appController,
	})

	if err := serviceApp.Run(cfg, r); err != nil {
		logger.Fatal("Failed to start the server: %v", zap.Error(err))
	}
}
