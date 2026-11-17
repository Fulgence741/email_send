package email

import (
	"log"
	"net/http"
	"time"

	setting "email-send-API/src/Setting"
	"email-send-API/src/database"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	// représente les différents statuts possibles pour un email.
	emailSending = "SENDING"
	emailSent    = "SENT"
	emailFailed  = "FAILED"
)

// AddEmailHandler enregistre l'email, l'envoie et conserve le résultat.
func AddEmailHandler(db *gorm.DB, smtpClient *SMTPClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var dto database.CreateEmailDTO
		if err := c.ShouldBindJSON(&dto); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Données invalides.",
				"error":   err.Error(),
			})
			return
		}

		// Récupérer l'ID de l'application authentifiée depuis le contexte Gin.
		applicationID, ok := setting.AuthenticatedApplicationID(c)
		if !ok {
			log.Printf("identifiant d'application absent après authentification")
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Erreur lors de l'authentification."})
			return
		}

		// Enregistrer l'email avant de commencer son envoi.
		now := time.Now().UTC()
		emailRecord := database.Email{
			IDApplication:       applicationID,
			Destinataire:        dto.Destinataire,
			Objet:               dto.Objet,
			Message:             dto.Message,
			TypeContenu:         dto.TypeContenu,
			Statut:              emailSending,
			NombreMaxTentatives: 3,  // Nombre maximum de tentatives d'envoi
			DelaiRetrySecondes:  60, // Délai entre les tentatives d'envoi en secondes
			DateCreation:        now,
			DateDebutTraitement: &now, // Date de début du traitement de l'email
		}
		if err := db.Create(&emailRecord).Error; err != nil {
			log.Printf("impossible d'enregistrer l'email: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Erreur lors de l'enregistrement de l'email."})
			return
		}

		// Envoyer l'email et enregistrer ses tentatives.
		sendErr, attemptErr := sendWithAttempts(db, &emailRecord, smtpClient)
		if attemptErr != nil {
			sendErr = attemptErr
		}
		finishedAt := time.Now().UTC()
		updates := map[string]interface{}{
			"date_fin_traitement":      finishedAt,
			"date_prochaine_tentative": nil,
		}
		if sendErr != nil {
			message := sendErr.Error()
			updates["statut"] = emailFailed
			updates["message_erreur"] = message
		} else {
			updates["statut"] = emailSent
		}

		if err := db.Model(&emailRecord).Updates(updates).Error; err != nil {
			log.Printf("impossible d'enregistrer le résultat d'envoi: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "L'envoi a été tenté, mais son résultat n'a pas pu être enregistré."})
			return
		}

		// Mettre à jour l'objet emailRecord avec les nouvelles valeurs pour la réponse.
		emailRecord.DateFinTraitement = &finishedAt
		emailRecord.DateProchaineTentative = nil
		if sendErr != nil {
			message := sendErr.Error()
			emailRecord.Statut = emailFailed
			emailRecord.MessageErreur = &message
			if attemptErr != nil {
				log.Printf("impossible d'enregistrer une tentative d'envoi: %v", attemptErr)
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Erreur lors de l'enregistrement d'une tentative d'envoi.", "data": emailRecord})
				return
			}
			c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "L'envoi de l'email a échoué.", "data": emailRecord})
			return
		}

		emailRecord.Statut = emailSent
		c.JSON(http.StatusCreated, gin.H{"success": true, "message": "Email envoyé avec succès.", "data": emailRecord})
	}
}
