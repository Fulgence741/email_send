package email

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
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

// ListApplicationsHandler retourne la liste des applications sans leurs clés API.
func ListApplicationsHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		applicationID, ok := setting.AuthenticatedApplicationID(c)
		if !ok {
			log.Printf("identifiant d'application absent après authentification")
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Erreur lors de l'authentification."})
			return
		}

		applications := make([]database.Application, 0)
		if err := db.Where("id_application = ?", applicationID).
			Find(&applications).Error; err != nil {
			log.Printf("impossible de consulter les applications: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Erreur lors de la consultation des applications."})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "data": applications})
	}
}

// DeleteApplicationHandler supprime une application et ses emails dans une transaction.
func DeleteApplicationHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authenticatedID, ok := setting.AuthenticatedApplicationID(c)
		if !ok {
			log.Printf("identifiant d'application absent après authentification")
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Erreur lors de l'authentification."})
			return
		}

		applicationID, err := strconv.ParseInt(c.Param("id_application"), 10, 64)
		if err != nil || applicationID < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Identifiant d'application invalide."})
			return
		}
		if applicationID != authenticatedID {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Application introuvable."})
			return
		}

		var attachments []database.PieceJointe
		err = db.Transaction(func(tx *gorm.DB) error {
			var application database.Application
			if err := tx.First(&application, "id_application = ?", applicationID).Error; err != nil {
				return err
			}

			if err := tx.Model(&database.PieceJointe{}).
				Joins("JOIN emails ON emails.id_email = pieces_jointes.id_email").
				Where("emails.id_application = ?", applicationID).
				Find(&attachments).Error; err != nil {
				return err
			}

			// Les tentatives et pièces jointes sont supprimées par les cascades SQL liées aux emails.
			if err := tx.Where("id_application = ?", applicationID).Delete(&database.Email{}).Error; err != nil {
				return err
			}
			return tx.Delete(&application).Error
		})
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Application introuvable."})
			return
		}
		if err != nil {
			log.Printf("impossible de supprimer l'application %d: %v", applicationID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Erreur lors de la suppression de l'application."})
			return
		}

		for _, attachment := range attachments {
			if err := os.Remove(attachment.CheminFichier); err != nil && !os.IsNotExist(err) {
				log.Printf("impossible de supprimer la pièce jointe %q: %v", attachment.CheminFichier, err)
			}
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Application supprimée avec succès."})
	}
}
