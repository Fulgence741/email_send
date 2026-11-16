// Package main assemble la configuration, les routes et les dépendances de l'API.
package main

import (
	"errors"
	"log"
	"strconv"

	database "email-send-API/src/Database"
	email "email-send-API/src/Email"
	setting "email-send-API/src/Setting"

	"github.com/gin-gonic/gin"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (runErr error) {
	cfg, err := setting.LoadConfig()
	if err != nil {
		return err
	}

	// Ouvre la base de données et s'assure qu'elle est fermée à la fin de l'exécution.
	repository, err := database.Open(cfg)
	if err != nil {
		return err
	}
	// Ferme la base de données à la fin de l'exécution, en combinant les erreurs si nécessaire.
	defer func() {
		if closeErr := repository.Close(); closeErr != nil {
			runErr = errors.Join(runErr, closeErr)
		}
	}()

	// Configure le routeur Gin et enregistre les routes de l'API.

	router := gin.Default()
	email.RegisterRoutes(router, repository)

	// Démarre le serveur HTTP sur le port configuré.
	port := strconv.Itoa(cfg.AppPort)
	return router.Run(":" + port)
}
