package email

import (
	"log"
	"net/http"
	"strings"

	database "email-send-API/src/Database"

	"github.com/gin-gonic/gin"
)

// applicationContextKey relie le middleware d'authentification au handler email.
const applicationContextKey = "authenticated_application"

// authenticateApplication valide X-API-Key et stocke l'application dans le contexte HTTP.
func authenticateApplication(repository *database.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Récupère la clé API depuis l'en-tête de la requête.
		apiKey := strings.TrimSpace(c.GetHeader("X-API-Key"))
		if apiKey == "" {
			// Si la clé API est absente, renvoie une réponse 401 Unauthorized.
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "L'en-tête X-API-Key est obligatoire.",
			})
			return
		}

		// Vérifie la validité de la clé API en recherchant l'application correspondante dans la base de données.
		application, err := repository.FindApplicationByAPIKey(c.Request.Context(), apiKey)
		if database.IsNotFound(err) {
			// Si l'application n'est pas trouvée, renvoie une réponse 401 Unauthorized.
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Clé API invalide.",
			})
			return
		}
		// Si une autre erreur survient lors de la recherche, renvoie une réponse 500 Internal Server Error.
		if err != nil {
			log.Printf("authentification de l'application impossible: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Impossible de vérifier la clé API.",
			})
			return
		}

		// Si l'application est trouvée, stocke-la dans le contexte de la requête pour les handlers suivants.
		c.Set(applicationContextKey, application) // Stocke l'application authentifiée dans le contexte de la requête.
		c.Next()                                  // Passe au handler suivant dans la chaîne de middleware.
	}
}
