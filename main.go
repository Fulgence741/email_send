package main

import (
	"log"
	"strconv"

	email "email-send-API/src/Email"
	setting "email-send-API/src/Setting"
	"email-send-API/src/database"

	redis "email-send-API/src/Redis"

	"github.com/gin-gonic/gin"
)

func main() {

	// Charger la configuration
	cfg, err := setting.LoadConfig()
	if err != nil {
		log.Fatalf("Erreur chargement configuration : %v", err)
	}

	// Connexion PostgreSQL
	db, err := database.ConnectDB(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// Connexion Redis
	rdb, err := redis.NewRedisClient(cfg)
	if err != nil {
		log.Fatal("Erreur Redis :", err)
	}
	defer rdb.Close()

	// Fermer proprement la connexion PostgreSQL
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	// Initialiser Gin
	router := gin.Default()

	// Enregistrer les routes
	email.RegisterRoutes(router, db, email.NewSMTPClient(cfg))

	// Démarrer le serveur
	port := strconv.Itoa(cfg.AppPort)

	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}

}
