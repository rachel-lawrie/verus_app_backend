package app

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rachel-lawrie/verus_app_backend/internal/app/controller"
	"github.com/rachel-lawrie/verus_backend_core/models"
	"github.com/stretchr/testify/assert"
)

func TestBuild(t *testing.T) {
	router := gin.Default()

	cfg := models.Config{
		Server: models.ServerConfig{
			Port: "8080",
		},
	}

	ctrl := controller.New(controller.Params{
		Router: router,
		Config: &cfg,
	})

	myApp := Build(Params{
		Router:     router,
		Controller: ctrl,
	})

	assert.NotNil(t, myApp)
	assert.IsType(t, &app{}, myApp)
}

func TestRun(t *testing.T) {
	router := gin.Default()

	cfg := models.Config{
		Server: models.ServerConfig{
			Port: "8080",
		},
	}

	ctrl := controller.New(controller.Params{
		Router: router,
		Config: &cfg,
	})

	app := Build(Params{
		Router:     router,
		Controller: ctrl,
	})

	go func() {
		err := app.Run(cfg, router)
		assert.NoError(t, err)
	}()
}
