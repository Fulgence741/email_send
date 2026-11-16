package middleware

import (
	database "email-send-API/src/Database"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const cleApplicationAuthentifiee = "application_authentifiee"

// AuthentifierApplication vérifie la clé API et associe l'application à la requête.
func AuthentifierApplication(depot *database.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Une clé absente ou invalide ne doit jamais atteindre les handlers privés.
		cleAPI := strings.TrimSpace(c.GetHeader("X-API-Key"))
		if cleAPI == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "L'en-tête X-API-Key est obligatoire.",
			})
			return
		}

		// Le dépôt ne renvoie que les applications actives et compare le condensat de la clé.
		application, err := depot.FindApplicationByAPIKey(c.Request.Context(), cleAPI)
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

		// Les handlers obtiennent l'identifiant uniquement depuis ce contexte vérifié.
		c.Set(cleApplicationAuthentifiee, application)
		c.Next()
	}
}

// ApplicationAuthentifiee récupère l'application ajoutée par le middleware d'authentification.
func ApplicationAuthentifiee(c *gin.Context) *database.Application {
	valeur, existe := c.Get(cleApplicationAuthentifiee)
	if !existe {
		return nil
	}
	// L'application est stockée dans le contexte, mais doit être convertie en type correct.
	application, valide := valeur.(*database.Application)
	if !valide {
		return nil
	}
	return application
}
