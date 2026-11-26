package email

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	setting "email-send-API/src/Setting"
	"email-send-API/src/database"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	emailSending = "SENDING"
	emailSent    = "SENT"
	emailFailed  = "FAILED"

	uploadDir      = "storage/uploads"
	maxFileSize    = 10 << 20
	maxTotalSize   = 25 << 20
	maxFiles       = 5
	maxRequestSize = 27 << 20
)

// AddEmailHandler crée un email avec ses pièces jointes et lance son envoi.
func AddEmailHandler(db *gorm.DB, smtpClient *SMTPClient) gin.HandlerFunc {
	return func(c *gin.Context) {

		// Limiter la taille de la requête.
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestSize)

		// Récupérer les informations de l'email.
		var dto database.CreateEmailDTO

		if err := c.ShouldBind(&dto); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Données invalides.",
				"error":   err.Error(),
			})
			return
		}

		// Récupérer le formulaire multipart.
		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Impossible de récupérer les fichiers.",
			})
			return
		}
		defer form.RemoveAll()

		// Récupérer les fichiers.
		files := form.File["fichiers"]

		// Vérifier le nombre de fichiers.
		if len(files) > maxFiles {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Maximum 5 fichiers autorisés.",
			})
			return
		}

		// Vérifier les tailles individuelles et cumulées.
		var totalSize int64

		for _, file := range files {
			filename := filepath.Base(strings.ReplaceAll(file.Filename, "\\", "/"))
			if filename == "." || filename == "" || utf8.RuneCountInString(filename) > 255 {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"message": "Nom de fichier invalide (255 caractères maximum).",
				})
				return
			}

			if file.Size <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"message": "Un fichier est vide ou invalide.",
				})
				return
			}

			if file.Size > maxFileSize {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"message": "Un fichier dépasse la limite de 10 Mio.",
				})
				return
			}

			totalSize += file.Size
		}

		if totalSize > maxTotalSize {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "La taille totale dépasse 25 Mio.",
			})
			return
		}

		// Récupérer l'application authentifiée.
		applicationID, ok := setting.AuthenticatedApplicationID(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Erreur lors de l'authentification.",
			})
			return
		}

		// Préparer l'email.
		now := time.Now().UTC()

		emailRecord := database.Email{
			IDApplication:       applicationID,
			Destinataire:        dto.Destinataire,
			Objet:               dto.Objet,
			Message:             dto.Message,
			TypeContenu:         dto.TypeContenu,
			Statut:              emailSending,
			NombreMaxTentatives: 3,
			DelaiRetrySecondes:  60,
			DateCreation:        now,
			DateDebutTraitement: &now,
		}

		var attachments []database.PieceJointe
		var savedPaths []string

		// Nettoyer les fichiers en cas d'échec.
		cleanup := func() {
			for _, path := range savedPaths {
				if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
					log.Printf("Erreur suppression fichier %s: %v", path, err)
				}
			}
		}

		// Enregistrer l'email et les pièces jointes.
		err = db.Transaction(func(tx *gorm.DB) error {

			if err := tx.Create(&emailRecord).Error; err != nil {
				return err
			}

			for _, file := range files {
				attachment, err := saveAttachment(file, emailRecord.IDEmail)
				if err != nil {
					return err
				}

				savedPaths = append(savedPaths, attachment.CheminFichier)
				attachments = append(attachments, attachment)
			}

			if len(attachments) > 0 {
				if err := tx.Create(&attachments).Error; err != nil {
					return err
				}
			}

			return nil
		})

		if err != nil {
			cleanup()

			log.Printf("Erreur enregistrement email: %v", err)

			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Erreur lors de l'enregistrement de l'email et des pièces jointes.",
			})
			return
		}

		// Envoyer l'email.
		sendErr, attemptErr := sendWithAttempts(db, &emailRecord, smtpClient, attachments)

		if attemptErr != nil {
			sendErr = attemptErr
		}

		// Enregistrer le résultat de l'envoi.
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
			updates["message_erreur"] = nil
		}

		if err := db.Model(&emailRecord).Updates(updates).Error; err != nil {
			log.Printf("Erreur mise à jour email: %v", err)

			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "L'envoi a été tenté, mais le résultat n'a pas pu être enregistré.",
			})
			return
		}

		emailRecord.DateFinTraitement = &finishedAt
		emailRecord.DateProchaineTentative = nil

		if sendErr != nil {
			message := sendErr.Error()
			emailRecord.Statut = emailFailed
			emailRecord.MessageErreur = &message

			if attemptErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": "Erreur lors de l'enregistrement d'une tentative.",
					"data":    emailRecord,
				})
				return
			}

			c.JSON(http.StatusBadGateway, gin.H{
				"success": false,
				"message": "L'envoi de l'email a échoué.",
				"data": gin.H{
					"email":          emailRecord,
					"pieces_jointes": pieceJointeResponses(attachments),
				},
			})
			return
		}

		emailRecord.Statut = emailSent
		emailRecord.MessageErreur = nil

		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"message": "Email envoyé avec succès.",
			"data": gin.H{
				"email":          emailRecord,
				"pieces_jointes": pieceJointeResponses(attachments),
			},
		})
	}
}

func pieceJointeResponses(attachments []database.PieceJointe) []database.PieceJointeResponseDTO {
	responses := make([]database.PieceJointeResponseDTO, 0, len(attachments))
	for _, attachment := range attachments {
		responses = append(responses, database.PieceJointeResponseDTO{
			IDPieceJointe: attachment.IDPieceJointe,
			NomFichier:    attachment.NomFichier,
			TypeMIME:      attachment.TypeMIME,
			TailleFichier: attachment.TailleFichier,
		})
	}
	return responses
}

// saveAttachment enregistre un fichier et prépare ses informations.
func saveAttachment(file *multipart.FileHeader, emailID int64) (database.PieceJointe, error) {

	var attachment database.PieceJointe

	src, err := file.Open()
	if err != nil {
		return attachment, err
	}
	defer src.Close()

	// Détecter le type MIME.
	buffer := make([]byte, 512)
	n, err := io.ReadFull(src, buffer)

	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return attachment, err
	}

	mimeType := http.DetectContentType(buffer[:n])

	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return attachment, err
	}

	// Générer un nom unique.
	randomBytes := make([]byte, 16)

	if _, err := rand.Read(randomBytes); err != nil {
		return attachment, err
	}

	originalName := filepath.Base(strings.ReplaceAll(file.Filename, "\\", "/"))
	if originalName == "." || originalName == "" || utf8.RuneCountInString(originalName) > 255 {
		return attachment, fmt.Errorf("nom de fichier invalide (255 caractères maximum)")
	}
	filename := hex.EncodeToString(randomBytes) +
		strings.ToLower(filepath.Ext(originalName))

	destination := filepath.Join(uploadDir, filename)

	if err := os.MkdirAll(uploadDir, 0700); err != nil {
		return attachment, err
	}

	// Enregistrer physiquement le fichier.
	dst, err := os.OpenFile(
		destination,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0600,
	)
	if err != nil {
		return attachment, err
	}

	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()

	if copyErr != nil {
		_ = os.Remove(destination)
		return attachment, copyErr
	}

	if closeErr != nil {
		_ = os.Remove(destination)
		return attachment, closeErr
	}

	attachment = database.PieceJointe{
		IDEmail:       emailID,
		NomFichier:    originalName,
		TypeMIME:      mimeType,
		CheminFichier: destination,
		TailleFichier: file.Size,
		DateCreation:  time.Now().UTC(),
	}

	return attachment, nil
}

// ListEmailsHandler retourne les emails de l'application authentifiée.
func ListEmailsHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		applicationID, ok := setting.AuthenticatedApplicationID(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Erreur lors de l'authentification.",
			})
			return
		}

		emails := make([]database.Email, 0)

		if err := db.Where(
			"id_application = ?",
			applicationID,
		).Order("id_email DESC").Find(&emails).Error; err != nil {

			log.Printf("Erreur consultation emails: %v", err)

			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Erreur lors de la consultation des emails.",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    emails,
		})
	}
}

// DeleteEmailHandler supprime un email et ses pièces jointes.
func DeleteEmailHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		applicationID, ok := setting.AuthenticatedApplicationID(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Erreur lors de l'authentification.",
			})
			return
		}

		emailID, err := strconv.ParseInt(c.Param("id_email"), 10, 64)

		if err != nil || emailID < 1 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Identifiant d'email invalide.",
			})
			return
		}

		var emailRecord database.Email

		if err := db.Where(
			"id_email = ? AND id_application = ?",
			emailID,
			applicationID,
		).First(&emailRecord).Error; err != nil {

			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{
					"success": false,
					"message": "Email introuvable.",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Erreur lors de la récupération de l'email.",
			})
			return
		}

		var attachments []database.PieceJointe

		if err := db.Where(
			"id_email = ?",
			emailID,
		).Find(&attachments).Error; err != nil {

			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Erreur lors de la récupération des pièces jointes.",
			})
			return
		}

		// Supprimer les données en base.
		err = db.Transaction(func(tx *gorm.DB) error {

			if err := tx.Where(
				"id_email = ?",
				emailID,
			).Delete(&database.PieceJointe{}).Error; err != nil {
				return err
			}

			return tx.Delete(&emailRecord).Error
		})

		if err != nil {
			log.Printf("Erreur suppression email: %v", err)

			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Erreur lors de la suppression de l'email.",
			})
			return
		}

		// Supprimer les fichiers physiques.
		for _, attachment := range attachments {
			if err := os.Remove(attachment.CheminFichier); err != nil &&
				!os.IsNotExist(err) {
				c.JSON(http.StatusInternalServerError, gin.H{
					"succes":  false,
					"message": "Erreur lors de la suppression de l'email",
				})
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Email et pièces jointes supprimés avec succès.",
		})
	}
}
