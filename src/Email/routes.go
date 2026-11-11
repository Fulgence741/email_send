package email

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes enregistre les routes de l'API

func RegisterRoutes(router *gin.Engine) {
	api := router.Group("/v1")
	api.POST("/:ressource", Dispatch)
}
