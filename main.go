package main

import (
	"log"
	"strconv"
	"time"

	email "email-send-API/src/Email"
	redis "email-send-API/src/Redis"
	setting "email-send-API/src/Setting"
	"email-send-API/src/database"
	"email-send-API/src/middleware"

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

	// Initialiser le rate limit 
	limiter, err := middleware.NewRateLimiter(rdb, 5, time.Minute)
	if err != nil {
		log.Fatal("Erreur rate limit :", err)
	}

	// Fermer proprement la connexion PostgreSQL
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	// Initialiser Gin
	router := gin.Default()

	// Enregistrer les routes
	email.RegisterRoutes(router, db, email.NewSMTPClient(cfg), limiter)

	// Démarrer le serveur
	port := strconv.Itoa(cfg.AppPort)

	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}

}
