package email

import (
	"email-send-API/src/database"
	setting "email-send-API/src/setting"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"io"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

const (
	uploadDir   = "storage/uploads"
	maxFileSize = 10 << 20
)

func AddPieceJointeHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		// Récupérer l'application authentifiée
		applicationID, ok := setting.AuthentificatedApplictionID(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"succes":  false,
				"message": "Erreur lors de l'authentification.",
			})
			return
		}

		// Récupérer l'identifiant de l'email
		emailID := c.Param("id_email")

		//Vérifier que l'email appartient à l'application

		var emailRecord database.Email

		err := db.Where(
			"id_email=? AND id_application=?",
			emailID,
			applicationID,
		).First(&emailRecord).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{
					"succes":   false,
					"messaage": "Email introuvable.",
				})
				return
			}
c.JSON(http.StatusInternalServerError, gin.H{
	"succes": false,
	"message": "Erreur lors de la récupération de l'émail"
})
		}

	}
}
