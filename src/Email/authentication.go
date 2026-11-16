package email

import (
	"log"
	"net/http"
	"strings"

	"email-send-API/src/Database"

	"github.com/gin-gonic/gin"
)

// applicationContextKey relie le middleware d'authentification au handler email.
const applicationContextKey = "authenticated_application"

// authenticateApplication valide X-API-Key et stocke l'application dans le contexte HTTP.
func authenticateApplication(repository *database.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := strings.TrimSpace(c.GetHeader("X-API-Key"))
		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "L'en-tête X-API-Key est obligatoire.",
			})
			return
		}

		application, err := repository.FindApplicationByAPIKey(c.Request.Context(), apiKey)
		if database.IsNotFound(err) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Clé API invalide.",
			})
			return
		}
		if err != nil {
			log.Printf("authentification de l'application impossible: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Impossible de vérifier la clé API.",
			})
			return
		}

		c.Set(applicationContextKey, application)
		c.Next()
	}
}
