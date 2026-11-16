package email

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"email-send-API/src/Database"

	"github.com/gin-gonic/gin"
)

// CreateEmailDTO valide le contenu minimal d'un email à mettre en file d'attente.
type CreateEmailDTO struct {
	Destinataire string `json:"destinataire" binding:"required,email,max=255"`
	Objet        string `json:"objet" binding:"required,max=255"`
	Contenu      string `json:"contenu" binding:"required"`
	TypeContenu  string `json:"type_contenu" binding:"required,oneof=TEXT HTML"`
}

// AddEmailHandlers persiste un email appartenant à l'application authentifiée.
func AddEmailHandlers(repository *database.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var dto CreateEmailDTO
		if err := c.ShouldBindJSON(&dto); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Données invalides.",
				"error":   err.Error(),
			})
			return
		}

		idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
		if idempotencyKey == "" || len(idempotencyKey) > 255 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "L'en-tête Idempotency-Key est obligatoire et limité à 255 caractères.",
			})
			return
		}

		// L'identité de l'application est injectée par le middleware d'authentification.
		value, exists := c.Get(applicationContextKey)
		application, ok := value.(*database.Application)
		if !exists || !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Contexte d'application authentifiée indisponible.",
			})
			return
		}

		emailRecord := database.Email{
			IDApplication:  application.IDApplication,
			Destinataire:   dto.Destinataire,
			Objet:          dto.Objet,
			Message:        dto.Contenu,
			TypeContenu:    dto.TypeContenu,
			Statut:         "PENDING",
			CleIdempotence: idempotencyKey,
			DateCreation:   time.Now().UTC(),
		}
		if _, err := repository.CreateEmail(c.Request.Context(), &emailRecord); err != nil {
			if errors.Is(err, database.ErrConflict) {
				c.JSON(http.StatusConflict, gin.H{
					"success": false,
					"message": "Cette clé d'idempotence est associée à un autre email.",
				})
				return
			}
			log.Printf("enregistrement de l'email impossible: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Impossible d'enregistrer l'email.",
			})
			return
		}

		c.JSON(http.StatusAccepted, gin.H{
			"success": true,
			"message": "Email enregistré pour traitement.",
			"data": gin.H{
				"id_email": emailRecord.IDEmail,
				"statut":   emailRecord.Statut,
			},
		})
	}
}
