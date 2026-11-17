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
	emailSending = "EN_ATTENTE"
	emailSent    = "ENVOYE"
	emailFailed  = "ECHEC"

	// représente les différents statuts possibles pour une tentative d'envoi.
	attemptRunning = "EN_COURS"
	attemptSuccess = "REUSSIE"
	attemptFailure = "ECHEC"
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

		// Enregistrer l'email dans la base de données avec le statut "EN_ATTENTE" et créer une tentative d'envoi initiale.
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
		// Créer une tentative d'envoi initiale avec le statut "EN_COURS".
		attempt := database.Envoie{
			NumeroTentative: 1,
			DateTentative:   now,
			Statut:          attemptRunning, // Statut initial de la tentative
		}
		// Enregistrer l'email et la tentative dans une transaction
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&emailRecord).Error; err != nil {
				return err
			}
			// Associer l'ID de l'email à la tentative d'envoi
			attempt.IDEmail = emailRecord.IDEmail
			return tx.Create(&attempt).Error
		}); err != nil {
			log.Printf("impossible d'enregistrer l'email: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Erreur lors de l'enregistrement de l'email."})
			return
		}
		// Envoyer l'email en utilisant le client SMTP.
		sendErr := smtpClient.SendEmail(dto.Destinataire, dto.Objet, dto.Message, dto.TypeContenu)
		finishedAt := time.Now().UTC()
		updates := map[string]interface{}{"date_fin_traitement": finishedAt} // Mettre à jour la date de fin de traitement de l'email
		attemptUpdates := map[string]interface{}{}
		if sendErr != nil {
			message := sendErr.Error()
			updates["statut"] = emailFailed
			updates["message_erreur"] = message
			attemptUpdates["statut"] = attemptFailure
			attemptUpdates["message_erreur"] = message
		} else {
			updates["statut"] = emailSent
			attemptUpdates["statut"] = attemptSuccess
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&emailRecord).Updates(updates).Error; err != nil {
				return err
			}
			// Mettre à jour la tentative d'envoi avec le statut et le message d'erreur si nécessaire
			return tx.Model(&attempt).Updates(attemptUpdates).Error
		}); err != nil {
			log.Printf("impossible d'enregistrer le résultat d'envoi: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "L'envoi a été tenté, mais son résultat n'a pas pu être enregistré."})
			return
		}

		// Mettre à jour l'objet emailRecord avec les nouvelles valeurs pour la réponse
		emailRecord.DateFinTraitement = &finishedAt
		if sendErr != nil {
			message := sendErr.Error()
			emailRecord.Statut = emailFailed
			emailRecord.MessageErreur = &message
			c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "L'envoi de l'email a échoué.", "data": emailRecord})
			return
		}
		// Si l'envoi a réussi, renvoyer une réponse de succès avec les détails de l'email.

		emailRecord.Statut = emailSent
		c.JSON(http.StatusCreated, gin.H{"success": true, "message": "Email envoyé avec succès.", "data": emailRecord})
	}
}
