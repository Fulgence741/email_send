package email

import (
	database "email-send-API/src/Database"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes déclare les routes HTTP et protège les opérations d'envoi par clé API.
func RegisterRoutes(router *gin.Engine, repository *database.Repository) {
	// Groupe de routes versionnées pour l'API.
	api := router.Group("/v1") // Préfixe toutes les routes avec /v1 pour la version de l'API.
	api.POST("/applications", AddApplicationHandlers(repository))
	api.POST("/emails", authenticateApplication(repository), AddEmailHandlers(repository))
	api.POST("/:ressource", Dispatch)
}
