package email

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Model
// =========================================
type Application struct {
	IDApplication int64     `gorm:"column:primaryKey" json:"id_application"`
	Nom           string    `gorm:"column:nom;not null" json:"nom"`
	APIKeyHash    string    `gorm:"-"`
	Statut        bool      `gorm:"column statut" json:"statut"`
	DateCreation  time.Time `gorm:"column:date_creation" json:"date_creation"`
}

func (Application) TableName() string {
	return "applications"
}

//==============================================

// DTO
// =============================================
type CreateApplicationDTO struct {
	Nom string `json:"nom" binding:"required"`
}

type ApplicationResponseDTO struct {
	IDApplication int64  `json:"id_application"`
	Nom           string `json:"nom"`
	Statut        bool   `json:"statut"`
}

// ============================================

// Handlers

// ajout d'une application

func AddApplicationhandlers(c *gin.Context) {
	var dto CreateApplicationDTO

	// Vérifier les données de l'application
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
