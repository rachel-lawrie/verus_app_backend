package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rachel-lawrie/verus_backend_core/models"
)

var (
	errTest = errors.New("test error")
)

type Params struct {
	Router *gin.Engine
	Config *models.Config
}

type controller struct {
	router *gin.Engine
	cfg    *models.Config
}

func New(p Params) Controller {
	ctrl := &controller{
		router: p.Router,
		cfg:    p.Config,
	}
	return ctrl
}

func (c *controller) Success(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{})
}

func (c *controller) Error(ctx *gin.Context) {
	ctx.Error(gin.Error{
		Err:  errTest,
		Type: gin.ErrorTypePublic,
	})
}
