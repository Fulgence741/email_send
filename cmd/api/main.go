package main

import (
	"email-send-API/config"
	"email-send-API/database"
	"email-send-API/routes"
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
)

func main() {
	// Charger la configuration
	cfg := config.LoadConfig()

	db, err := database.ConnectDB(cfg)
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
	routes.RegisterRoutes(router, cfg)

	// Demarrage du serveur
	port := strconv.Itoa(cfg.AppPort)

	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
