// Package email expose les routes HTTP et les fonctions liées à l'envoi d'emails.
package email

import (
	database "email-send-API/src/Database"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CreateApplicationDTO représente les données acceptées à la création d'une application.
type CreateApplicationDTO struct {
	Nom string `json:"nom" binding:"required,max=100"`
}

// ApplicationResponseDTO expose les informations de création et la clé montrée une fois.
type ApplicationResponseDTO struct {
	IDApplication int64  `json:"id_application"`
	Nom           string `json:"nom"`
	Statut        string `json:"statut"`
	APIKey        string `json:"api_key"`
}

// AddApplicationHandlers valide la demande, crée l'application et renvoie sa clé API.
func AddApplicationHandlers(repository *database.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Valide les données JSON entrantes et renvoie une erreur si elles sont invalides.
		var dto CreateApplicationDTO
		if err := c.ShouldBindJSON(&dto); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Données invalides.",
				"error":   err.Error(),
			})
			return
		}

		// Valide le nom de l'application pour s'assurer qu'il n'est pas vide après suppression des espaces.
		name := strings.TrimSpace(dto.Nom)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Le nom de l'application est obligatoire.",
			})
			return
		}

		// Crée l'application dans la base de données et récupère l'ID et la clé API.
		application, apiKey, err := repository.CreateApplication(c.Request.Context(), name)
		if err != nil {
			if errors.Is(err, database.ErrConflict) {
				c.JSON(http.StatusConflict, gin.H{
					"success": false,
					"message": "Une application portant ce nom existe déjà.",
				})
				return
			}
			// Log l'erreur côté serveur pour le débogage et renvoie une réponse générique.
			log.Printf("création de l'application impossible: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Impossible de créer l'application.",
			})
			return
		}
		// Renvoie les détails de l'application créée et la clé API au client.
		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"data": ApplicationResponseDTO{
				IDApplication: application.IDApplication,
				Nom:           application.Nom,
				Statut:        application.Statut,
				APIKey:        apiKey,
			},
		})
	}
}
