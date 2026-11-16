package email

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Dispatch fournit une réponse uniforme pour les anciennes routes de ressources inconnues.
func Dispatch(c *gin.Context) {
	resource := strings.TrimSpace(strings.ToLower(c.Param("ressource")))
	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error":   "RESOURCE_NOT_FOUND",
		"message": "Ressource non disponible: " + resource,
	})
}
