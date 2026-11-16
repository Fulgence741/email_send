package setting

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// StockageLimiteDebit expose le compteur atomique partagé utilisé par les routes.
type StockageLimiteDebit interface {
	Autoriser(context.Context, string, int, time.Duration) (bool, int64, time.Duration, error)
}

// LimiteurDeDebit applique une fenêtre de limitation propre à l'identité fournie.
func LimiteurDeDebit(stockage StockageLimiteDebit, limite int, fenetre time.Duration, identite func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Refuse une configuration incomplète plutôt que de laisser passer les requêtes.
		if stockage == nil || identite == nil || limite < 1 || fenetre <= 0 {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "ERREUR_CONFIGURATION_LIMITE",
				"message": "La limitation de débit est mal configurée.",
			})
			return
		}

		// Une clé stable permet à Redis de partager le quota entre toutes les instances.
		cle := identite(c)
		if cle == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "ERREUR_IDENTITE_LIMITE",
				"message": "Impossible d'identifier la requête pour la limitation de débit.",
			})
			return
		}

		// Le compteur est incrémenté atomiquement dans Redis.
		autorisee, restante, delai, err := stockage.Autoriser(c.Request.Context(), cle, limite, fenetre)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"error":   "LIMITE_DEBIT_INDISPONIBLE",
				"message": "Le service de limitation de débit est indisponible.",
			})
			return
		}

		// Les en-têtes indiquent le quota consommé et le délai avant réinitialisation.
		c.Header("X-RateLimit-Limit", strconv.Itoa(limite))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(restante, 10))
		if delai > 0 {
			c.Header("X-RateLimit-Reset", strconv.FormatInt(int64(math.Ceil(delai.Seconds())), 10))
		}
		// Si la limite est dépassée, renvoie un code 429 avec le délai avant réessai.
		if !autorisee {
			secondes := int64(math.Ceil(delai.Seconds()))
			if secondes < 1 {
				secondes = 1
			}
			// Indique au client combien de temps attendre avant de réessayer.
			c.Header("Retry-After", strconv.FormatInt(secondes, 10))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   "LIMITE_DEBIT_DEPASSEE",
				"message": "Trop de requêtes. Réessayez plus tard.",
			})
			return
		}
		// La requête est autorisée, passe au handler suivant.
		c.Next()
	}
}
