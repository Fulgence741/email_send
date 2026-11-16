package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// IdentifiantRequete attribue un identifiant à chaque requête pour faciliter le suivi des journaux.
func IdentifiantRequete() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Réutilise un identifiant fourni uniquement s'il est court et sûr pour un en-tête HTTP.
		identifiant := c.GetHeader("X-Request-ID")
		if !identifiantValide(identifiant) {
			octets := make([]byte, 16)
			if _, err := rand.Read(octets); err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": "Impossible de créer l'identifiant de requête.",
				})
				return
			}
			identifiant = hex.EncodeToString(octets)
		}
		c.Set("identifiant_requete", identifiant)
		c.Header("X-Request-ID", identifiant)
		c.Next()
	}
}

// EntetesSecurite ajoute des protections HTTP sans modifier le contenu des réponses.
func EntetesSecurite() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Next()
	}
}

// LimiterTailleCorps empêche une requête de dépasser la taille maximale autorisée.
func LimiterTailleCorps(tailleMaximale int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if tailleMaximale <= 0 {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "La limite de taille des requêtes est mal configurée.",
			})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, tailleMaximale)
		c.Next()
	}
}

// identifiantValide restreint la valeur réutilisée dans l'en-tête de réponse.
func identifiantValide(identifiant string) bool {
	if identifiant == "" || len(identifiant) > 128 {
		return false
	}
	for _, caractere := range identifiant {
		if !(caractere >= 'a' && caractere <= 'z') &&
			!(caractere >= 'A' && caractere <= 'Z') &&
			!(caractere >= '0' && caractere <= '9') &&
			!strings.ContainsRune("-_", caractere) {
			return false
		}
	}
	return true
}
