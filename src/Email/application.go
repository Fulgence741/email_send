package email

import (
	"errors"
	"log"
	"net/http"
	"time"

	setting "email-send-API/src/Setting"
	"email-send-API/src/database"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AddApplicationHandler crée une application et ne révèle sa clé API qu'une fois.
func AddApplicationHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var dto database.CreateApplicationDTO
		if err := c.ShouldBindJSON(&dto); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Données invalides.",
				"error":   err.Error(),
			})
			return
		}

		apiKey, apiKeyHash, err := setting.GenerateAPIKey()
		if err != nil {
			log.Printf("impossible de générer une clé API: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Erreur interne."})
			return
		}

		application := database.Application{
			Nom:          dto.Nom,
			APIKeyHash:   apiKeyHash,
			Statut:       "ACTIF",
			DateCreation: time.Now().UTC(),
		}
		if err := db.Create(&application).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				c.JSON(http.StatusConflict, gin.H{"success": false, "message": "Une application porte déjà ce nom."})
				return
			}
			log.Printf("impossible de créer l'application: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Erreur lors de la création de l'application."})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"message": "Application créée avec succès.",
			"data": gin.H{
				"id_application": application.IDApplication,
				"nom":            application.Nom,
				"statut":         application.Statut,
				"date_creation":  application.DateCreation,
				"api_key":        apiKey,
			},
		})
	}
}
