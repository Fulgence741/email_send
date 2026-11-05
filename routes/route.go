package routes

import (
	"email-send-API/config"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes enregistre les routes de l'API
func RegisterRoutes(router *gin.Engine, cfg *config.Config) {

	// Route test pour vérifier que l'API fonctionne
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "l'API fonctionne correctement",
		})
	})

}
