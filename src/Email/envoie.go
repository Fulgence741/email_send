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
func sendWithAttempts(db *gorm.DB,
	emailRecord *database.Email,
	smtpClient *SMTPClient,
	attachments []database.PieceJointe) (
	sendErr error,
	attemptErr error) {
	if emailRecord.NombreMaxTentatives < 1 {
		return nil, fmt.Errorf("le nombre de tentatives doit être supérieur à zéro")
	}
	// Vérifier que le délai entre les tentatives n'est pas négatif
	if emailRecord.DelaiRetrySecondes < 0 {
		return nil, fmt.Errorf("le délai entre les tentatives ne peut pas être négatif")
	}

	// Boucle pour envoyer l'email jusqu'à ce qu'il réussisse ou que le nombre maximum de tentatives soit atteint
	var lastSendErr error
	for numero := 1; numero <= emailRecord.NombreMaxTentatives; numero++ {
		// Si ce n'est pas la première tentative, attendre le délai spécifié avant de réessayer
		if numero > 1 {
			nextAttempt := time.Now().UTC().Add(time.Duration(emailRecord.DelaiRetrySecondes) * time.Second)
			if err := db.Model(emailRecord).Update("date_prochaine_tentative", nextAttempt).Error; err != nil {
				return nil, fmt.Errorf("impossible d'enregistrer la prochaine tentative: %w", err)
			}
			time.Sleep(time.Until(nextAttempt)) // Attendre jusqu'à la prochaine tentative
		}

		// Créer un enregistrement de tentative d'envoi dans la base de données
		attempt := database.Envoie{
			IDEmail:         emailRecord.IDEmail,
			NumeroTentative: numero,
			DateTentative:   time.Now().UTC(),
			Statut:          attemptRunning,
		}
		// Enregistrer la tentative dans la base de données
		if err := db.Create(&attempt).Error; err != nil {
			return nil, fmt.Errorf("impossible de créer la tentative %d: %w", numero, err)
		}
		// Envoyer l'email en utilisant le client SMTP et enregistrer le résultat de la tentative
		sendErr := smtpClient.SendEmail(
			emailRecord.Destinataire,
			emailRecord.Objet,
			emailRecord.Message,
			emailRecord.TypeContenu,
			attachments,
		)
		// Mettre à jour le statut de la tentative en fonction du résultat de l'envoi
		attemptUpdates := map[string]interface{}{"statut": attemptSuccess}
		if sendErr != nil {
			attemptUpdates["statut"] = attemptFailure
			attemptUpdates["message_erreur"] = sendErr.Error()
			lastSendErr = sendErr
		}
		// Enregistrer le résultat de la tentative dans la base de données
		if err := db.Model(&attempt).Updates(attemptUpdates).Error; err != nil {
			return nil, fmt.Errorf("impossible d'enregistrer le résultat de la tentative %d: %w", numero, err)
		}
		if sendErr == nil {
			return nil, nil
		}
	}
	// Retourner la dernière erreur d'envoi si toutes les tentatives ont échoué
	return lastSendErr, nil
}

// ListAttemptsHandler retourne les tentatives de l'application authentifiée.
func ListAttemptsHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		applicationID, ok := setting.AuthenticatedApplicationID(c) // Récupérer l'identifiant de l'application authentifiée depuis le contexte
		if !ok {
			log.Printf("identifiant d'application absent après authentification")
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Erreur lors de l'authentification."})
			return
		}

		// Récupérer toutes les tentatives d'envoi pour l'application authentifiée
		attempts := make([]database.Envoie, 0)
		if err := db.Model(&database.Envoie{}).
			Select("tentatives_envoi.*").                                        // Sélectionner toutes les colonnes de la table tentatives_envoi
			Joins("JOIN emails ON emails.id_email = tentatives_envoi.id_email"). // Joindre la table emails pour filtrer par application
			Where("emails.id_application = ?", applicationID).                   // Filtrer par l'identifiant de l'application
			Order("tentatives_envoi.id_email ASC").                              // Trier par identifiant d'email pour regrouper les tentatives par email
			Order("tentatives_envoi.numero_tentative ASC").                      // Trier par numéro de tentative pour avoir l'ordre chronologique des tentatives
			Find(&attempts).Error; err != nil {
			// Gérer les erreurs lors de la récupération des tentatives d'envoi
			log.Printf("impossible de consulter les tentatives: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Erreur lors de la consultation des tentatives."})
			return
		}

		// Retourner la liste des tentatives d'envoi dans la réponse JSON
		c.JSON(http.StatusOK, attempts)
	}
}
