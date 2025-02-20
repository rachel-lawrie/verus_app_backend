package controller

import (
	"github.com/gin-gonic/gin"
)

type Controller interface {
	Success(ctx *gin.Context)
	Error(ctx *gin.Context)
	InitializeRoutes()
}
