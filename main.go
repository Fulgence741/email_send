// Package main assemble la configuration, les routes et les dépendances de l'API.
package main

import (
	"errors"
	"log"
	"strconv"

	"email-send-API/src/Database"
	email "email-send-API/src/Email"
	setting "email-send-API/src/Setting"

	"github.com/gin-gonic/gin"
)

// main démarre l'API et arrête le processus si son initialisation échoue.
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run charge la configuration, ouvre la base de données et démarre le serveur HTTP.
// La connexion est fermée même si le serveur s'arrête avec une erreur.
func run() (runErr error) {
	cfg, err := setting.LoadConfig()
	if err != nil {
		return err
	}

	repository, err := database.Open(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := repository.Close(); closeErr != nil {
			runErr = errors.Join(runErr, closeErr)
		}
	}()

	router := gin.Default()
	email.RegisterRoutes(router, repository)

	port := strconv.Itoa(cfg.AppPort)
	return router.Run(":" + port)
}
