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
		// Récupérer les données de l'application depuis la requête JSON
		var dto database.CreateApplicationDTO
		if err := c.ShouldBindJSON(&dto); err != nil {
			// Si la liaison échoue, renvoyer une réponse d'erreur
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Données invalides.",
				"error":   err.Error(),
			})
			return
		}
		// Générer une nouvelle clé API et son hash
		apiKey, apiKeyHash, err := setting.GenerateAPIKey()
		if err != nil {
			log.Printf("impossible de générer une clé API: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Erreur interne."})
			return
		}

		// Créer une nouvelle application avec le nom fourni et le hash de la clé API
		application := database.Application{
			Nom:          dto.Nom,
			APIKeyHash:   apiKeyHash,
			Statut:       "ACTIF",
			DateCreation: time.Now().UTC(),
		}
		//   créer l'application dans la base de données
		if err := db.Create(&application).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				c.JSON(http.StatusConflict, gin.H{"success": false, "message": "Une application porte déjà ce nom."})
				return
			}
			log.Printf("impossible de créer l'application: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Erreur lors de la création de l'application."})
			return
		}
		//   Retourner la réponse avec la clé API générée
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
