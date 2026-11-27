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

	// Fermer proprement la connexion PostgreSQL
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	// Client Redis dédié au Rate Limiting
	rateLimitRedis, err := redis.NewRateLimitRedisClient(cfg)
	if err != nil {
		log.Fatal("Erreur Redis Rate Limit :", err)
	}
	defer rateLimitRedis.Close()

	// Client Redis dédié au Cache
	cacheRedis, err := redis.NewCacheRedisClient(cfg)
	if err != nil {
		log.Fatal("Erreur Redis Cache :", err)
	}
	defer cacheRedis.Close()

	// Initialiser le Rate Limiter par IP
	ipLimiter, err := middleware.NewRateLimiter(rateLimitRedis, 5, time.Minute)
	if err != nil {
		log.Fatal("Erreur rate limit par IP :", err)
	}

	// Initialiser le Rate Limiter par application
	appLimiter, err := middleware.NewRateLimiter(rateLimitRedis, 10, time.Minute)
	if err != nil {
		log.Fatal("Erreur rate limit par application :", err)
	}

	// Initialiser Gin
	router := gin.Default()

	// Enregistrer les routes
	email.RegisterRoutes(
		router,
		db,
		email.NewSMTPClient(cfg),
		ipLimiter,
		appLimiter,
	)

	// Démarrer le serveur
	port := strconv.Itoa(cfg.AppPort)

	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}

	// cacheRedis sera injecté dans les services qui utiliseront le cache.
	_ = cacheRedis

}
