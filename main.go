package main

import (
	"log"
	"strconv"

	email "email-send-API/src/Email"
	setting "email-send-API/src/Setting"

	"github.com/gin-gonic/gin"
)

func main() {
	// Charger la configuration
	cfg := setting.LoadConfig()

	db, err := setting.ConnectDB(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// Récupérer l'instance *sql.DB pour gérer la connexion
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}

	// Fermer la connexion à la base de données lorsque l'application se termine
	defer sqlDB.Close()
	router := gin.Default()

	// initialiser les routes
	email.RegisterRoutes(router)

	// Demarrage du serveur
	port := strconv.Itoa(cfg.AppPort)

	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
