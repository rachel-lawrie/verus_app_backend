package main

import (
	"log"

	"github.com/rachel-lawrie/verus_app_backend/internal/app/controller"
	"github.com/rachel-lawrie/verus_backend_core/common"

	"github.com/gin-gonic/gin"
	"github.com/rachel-lawrie/verus_app_backend/app"

	"github.com/rachel-lawrie/verus_app_backend/internal/config"
)

func main() {

	// Load the configuration for the dev environment
	cfg := config.LoadConfig("sandbox")

	// Log the start of the dev server
	log.Printf("Starting sandbox server on port %s...\n", cfg.Server.Port)

	// Connect to the database
	if err := common.ConnectDatabase(cfg.Database); err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}

	// Set Gin to release mode
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	// Use recovery middleware
	r.Use(gin.Recovery())

	// Start the server using the configured port
	log.Printf("Running application with configuration: %+v\n", cfg)

	appController := controller.New(controller.Params{
		Router: r,
		Config: &cfg,
	})
	appController.InitializeRoutes()

	serviceApp := app.Build(app.Params{
		Router:     r,
		Controller: appController,
	})

	serviceApp.Run(cfg, r)
}
