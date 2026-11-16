package email

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Dispatch fournit une réponse uniforme pour les anciennes routes de ressources inconnues.
func Dispatch(c *gin.Context) {
	// Récupère le nom de la ressource demandée depuis l'URL et le normalise.
	resource := strings.TrimSpace(strings.ToLower(c.Param("ressource")))
	c.JSON(http.StatusNotFound, gin.H{
		// Renvoie un message d'erreur standardisé pour les ressources non trouvées.
		"success": false,                                   // Indique que la requête a échoué.
		"error":   "RESOURCE_NOT_FOUND",                    // Code d'erreur standard pour les ressources non trouvées.
		"message": "Ressource non disponible: " + resource, // Message détaillant la ressource demandée qui n'est pas disponible.
	})
}
