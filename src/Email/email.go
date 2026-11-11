package email

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Model
//
//	==================================================================================
type Email struct {
	IDEmail       int64      `gorm:"primaryKey;column:id_email" json:"id_email"`
	IDApplication int64      `gorm:"column:id_application;not null" json:"id_application"`
	Destinataire  string     `gorm:"column:destinataire;not null" json:"destinataire"`
	Objet         string     `gorm:"column:objet;not null" json:"objet"`
	Contenu       string     `gorm:"column:contenu;not null" json:"contenu"`
	TypeContenu   string     `gorm:"column:type_contenu;not null" json:"type_contenu"`
	Statut        string     `gorm:"column:statut;not nulle" json:"statut"`
	DateCreation  time.Time  `gorm:"column:date_creation;not null" json:"date_creation"`
	DateEnvoi     *time.Time `gorm:"column:date_envoi;not null" json:"date_envoi"`
	MessageErreur *string    `gorm:"column:message_erreur; not null"  json:"message_serveur"`
}

func (Email) TableName() string {
	return "emails"
}

// ============================================================================

// DTO
// ===============================================================================
type CreateEmailDTO struct {
	Destinataire string `json:"destinataire" binding:"required,email"`
	Objet        string `json:"objet" binding:"required"`
	Contenu      string `json:"contenu" binding:"required"`
	TypeContenu  string `json:"type_contenu" binding:"required,oneof=TEXT HTML"`
}

// ================================================================================

// Handlers =================================

// Ajout d'un email
func AddEmailHandlers(c *gin.Context) {

	var dto CreateEmailDTO

	// Validation des données reçues
	if err := c.ShouldBindJSON(&dto); err != nil {

		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Données invalides.",
			"error":   err.Error(),
		})

		return
	}

	// Réponse en cas de succès
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Données envoyées avec succès.",
		"data":    dto,
	})
}
