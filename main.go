package main

import (
	"log"
	"strconv"

	email "email-send-API/src/Email"
	setting "email-send-API/src/Setting"
	"email-send-API/src/database"

	"github.com/gin-gonic/gin"
)

func main() {
	// Charger la configuration
	cfg := setting.LoadConfig()

	db, err := database.ConnectDB(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// Récupérer l'instance *sql.DB pour fermer proprement la connexion.
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}

	defer sqlDB.Close()
	router := gin.Default()

	// Enregistrer les routes et réutiliser le client SMTP.
	email.RegisterRoutes(router, db, email.NewSMTPClient(cfg))

	// Demarrage du serveur
	port := strconv.Itoa(cfg.AppPort)

	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
