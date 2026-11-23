package email

import (
	setting "email-send-API/src/Setting"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes enregistre les routes de l'API.
func RegisterRoutes(router *gin.Engine, db *gorm.DB, smtpClient *SMTPClient) {
	api := router.Group("/v1")
	api.POST("/applications", AddApplicationHandler(db))
	api.POST("/emails", setting.APIKeyAuth(db), AddEmailHandler(db, smtpClient))
	api.POST("/tentatives", setting.APIKeyAuth(db), ListAttemptsHandler(db))
}
