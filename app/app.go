package app

import (
	"github.com/gin-gonic/gin"
	"github.com/rachel-lawrie/verus_app_backend/internal/app/controller"
	"github.com/rachel-lawrie/verus_backend_core/models"
)

type Params struct {
	Router     *gin.Engine
	Controller controller.Controller
}

type App interface {
	Run(cfg models.Config, r *gin.Engine) error
}

type app struct {
	router     *gin.Engine
	controller controller.Controller
}

func Build(p Params) App {
	return &app{
		router:     p.Router,
		controller: p.Controller,
	}
}

func (a *app) Run(cfg models.Config, r *gin.Engine) error {
	return r.Run(":" + cfg.Server.Port)
}
