package email

import (
	setting "email-send-API/src/Setting"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes enregistre les routes de l'API.
func RegisterRoutes(router *gin.Engine, db *gorm.DB, smtpClient *SMTPClient) {

	api := router.Group("/v1")

	// Routes publiques
	public := api.Group("")
	{
		public.POST("/applications", AddApplicationHandler(db))
	}

	// Routes protégées par clé API
	protected := api.Group("")
	protected.Use(setting.APIKeyAuth(db))

	{
		protected.POST("/applications/liste", ListApplicationsHandler(db))
		protected.POST("/applications/supprimer/:id_application", DeleteApplicationHandler(db))

		protected.POST("/emails", AddEmailHandler(db, smtpClient))
		protected.POST("/emails/liste", ListEmailsHandler(db))
		protected.POST("/emails/supprimer/:id_email", DeleteEmailHandler(db))

		protected.POST("/tentatives", ListAttemptsHandler(db))
	}
}
