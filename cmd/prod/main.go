package main

import (
	"log"

	"github.com/rachel-lawrie/verus_app_backend/app"
	"github.com/rachel-lawrie/verus_app_backend/internal/app/controller"

	"github.com/gin-gonic/gin"
	"github.com/rachel-lawrie/verus_app_backend/internal/config"
)

func main() {
	cfg := config.LoadConfig("prod") // or "dev"
	log.Printf("Starting server on port %s...\n", cfg.Server.Port)

	r := gin.Default()

	// Start the server using the configured port
	log.Printf("Running application with configuration: %+v\n", cfg)

	appController := controller.New(controller.Params{
		Router: r,
		Config: &cfg,
	})

	serviceApp := app.Build(app.Params{
		Router:     r,
		Controller: appController,
	})

	serviceApp.Run(cfg, r)
}
