package email

import (
	"email-send-API/src/Database"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes déclare les routes HTTP et protège les opérations d'envoi par clé API.
func RegisterRoutes(router *gin.Engine, repository *database.Repository) {
	api := router.Group("/v1")
	api.POST("/applications", AddApplicationHandlers(repository))
	api.POST("/emails", authenticateApplication(repository), AddEmailHandlers(repository))
	api.POST("/:ressource", Dispatch)
}
