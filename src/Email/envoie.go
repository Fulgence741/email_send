package email

import (
	"fmt"
	"log"
	"net/http"
	"time"

	setting "email-send-API/src/Setting"
	"email-send-API/src/database"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	attemptRunning = "EN_COURS"
	attemptSuccess = "REUSSIE"
	attemptFailure = "ECHEC"
)

// sendWithAttempts envoie l'email et conserve le résultat de chaque tentative.
func sendWithAttempts(db *gorm.DB, emailRecord *database.Email, smtpClient *SMTPClient) (sendErr error, attemptErr error) {
	if emailRecord.NombreMaxTentatives < 1 {
		return nil, fmt.Errorf("le nombre de tentatives doit être supérieur à zéro")
	}
	if emailRecord.DelaiRetrySecondes < 0 {
		return nil, fmt.Errorf("le délai entre les tentatives ne peut pas être négatif")
	}

	var lastSendErr error
	for numero := 1; numero <= emailRecord.NombreMaxTentatives; numero++ {
		if numero > 1 {
			nextAttempt := time.Now().UTC().Add(time.Duration(emailRecord.DelaiRetrySecondes) * time.Second)
			if err := db.Model(emailRecord).Update("date_prochaine_tentative", nextAttempt).Error; err != nil {
				return nil, fmt.Errorf("impossible d'enregistrer la prochaine tentative: %w", err)
			}
			time.Sleep(time.Until(nextAttempt))
		}

		attempt := database.Envoie{
			IDEmail:         emailRecord.IDEmail,
			NumeroTentative: numero,
			DateTentative:   time.Now().UTC(),
			Statut:          attemptRunning,
		}
		if err := db.Create(&attempt).Error; err != nil {
			return nil, fmt.Errorf("impossible de créer la tentative %d: %w", numero, err)
		}

		sendErr := smtpClient.SendEmail(
			emailRecord.Destinataire,
			emailRecord.Objet,
			emailRecord.Message,
			emailRecord.TypeContenu,
		)
		attemptUpdates := map[string]interface{}{"statut": attemptSuccess}
		if sendErr != nil {
			attemptUpdates["statut"] = attemptFailure
			attemptUpdates["message_erreur"] = sendErr.Error()
			lastSendErr = sendErr
		}
		if err := db.Model(&attempt).Updates(attemptUpdates).Error; err != nil {
			return nil, fmt.Errorf("impossible d'enregistrer le résultat de la tentative %d: %w", numero, err)
		}
		if sendErr == nil {
			return nil, nil
		}
	}

	return lastSendErr, nil
}

// ListAttemptsHandler retourne les tentatives de l'application authentifiée.
func ListAttemptsHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		applicationID, ok := setting.AuthenticatedApplicationID(c)
		if !ok {
			log.Printf("identifiant d'application absent après authentification")
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Erreur lors de l'authentification."})
			return
		}

		attempts := make([]database.Envoie, 0)
		if err := db.Model(&database.Envoie{}).
			Select("tentatives_envoi.*").
			Joins("JOIN emails ON emails.id_email = tentatives_envoi.id_email").
			Where("emails.id_application = ?", applicationID).
			Order("tentatives_envoi.id_email ASC").
			Order("tentatives_envoi.numero_tentative ASC").
			Find(&attempts).Error; err != nil {
			log.Printf("impossible de consulter les tentatives: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Erreur lors de la consultation des tentatives."})
			return
		}

		c.JSON(http.StatusOK, attempts)
	}
}
