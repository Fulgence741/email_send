package email

import (
	"context"
	database "email-send-API/src/Database"
	setting "email-send-API/src/Setting"
	middleware "email-send-API/src/middleware"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes déclare les routes et applique les limites de débit partagées par Redis.
func RegisterRoutes(
	router *gin.Engine,
	repository *database.Repository,
	stockageLimite setting.StockageLimiteDebit,
	verificationRedis interface{ VerifierDisponibilite(context.Context) error },
	limiteCreationApplication, limiteAPI int,
	stockagePieces *StockagePiecesJointes,
) {
	// La sonde de vie vérifie uniquement que le serveur HTTP répond.
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "status": "ok"})
	})
	router.GET("/readyz", readinessHandler(repository, verificationRedis))

	// La création d'applications est limitée par adresse IP.
	api := router.Group("/v1")
	api.POST("/applications",
		setting.LimiteurDeDebit(stockageLimite, limiteCreationApplication, time.Minute, func(c *gin.Context) string {
			return "applications:ip:" + c.ClientIP()
		}),
		AddApplicationHandlers(repository),
	)

	// Les routes privées sont limitées par IP, authentifiées, puis limitées par application.
	protected := api.Group("")
	protected.Use(setting.LimiteurDeDebit(stockageLimite, limiteAPI, time.Minute, func(c *gin.Context) string {
		return "api:ip:" + c.ClientIP()
	}))
	protected.Use(middleware.AuthentifierApplication(repository))
	protected.Use(setting.LimiteurDeDebit(stockageLimite, limiteAPI, time.Minute, func(c *gin.Context) string {
		application := middleware.ApplicationAuthentifiee(c)
		if application == nil {
			return ""
		}
		return "api:application:" + strconv.FormatInt(application.IDApplication, 10)
	}))

	// Chaque consultation reste circonscrite à l'application portée par la clé API.
	protected.POST("/emails", AddEmailHandlers(repository, stockagePieces))
	protected.POST("/emails/liste", ListerEmailsHandler(repository))
	protected.POST("/emails/:id", LireEmailHandler(repository))
	protected.POST("/emails/:id/attempts", ListerTentativesEmailHandler(repository))
	protected.POST("/emails/:id/attachments", ListerPiecesJointesHandler(repository))
	protected.POST("/emails/:id/attachments/:piece_id", TelechargerPieceJointeHandler(repository))
}

// readinessHandler vérifie que les dépendances nécessaires au service répondent.
func readinessHandler(repository *database.Repository, verificationRedis interface {
	VerifierDisponibilite(context.Context) error
}) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Un délai maximal empêche la sonde de rester bloquée sur une dépendance.
		contexte, annuler := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer annuler()
		if err := repository.VerifierConnexion(contexte); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"error":   "BASE_DONNEES_INDISPONIBLE",
				"message": "La base de données est indisponible.",
			})
			return
		}
		if verificationRedis == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"error":   "REDIS_INDISPONIBLE",
				"message": "Redis est indisponible.",
			})
			return
		}
		if err := verificationRedis.VerifierDisponibilite(contexte); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"error":   "REDIS_INDISPONIBLE",
				"message": "Redis est indisponible.",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "status": "ready"})
	}
}
