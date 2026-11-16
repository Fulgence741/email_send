// Package main assemble la configuration, les routes et les dépendances de l'API.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	database "email-send-API/src/Database"
	email "email-send-API/src/Email"
	redisclient "email-send-API/src/Redis"
	setting "email-send-API/src/Setting"
	middleware "email-send-API/src/middleware"

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

	redis, err := redisclient.Ouvrir(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := redis.Fermer(); closeErr != nil {
			runErr = errors.Join(runErr, closeErr)
		}
	}()

	// Prépare le stockage local et vérifie la configuration SMTP avant d'accepter des emails.
	stockagePieces, err := email.NouveauStockagePiecesJointes(cfg.RepertoirePiecesJointes)
	if err != nil {
		return err
	}
	clientSMTP := email.NewSMTPClient(cfg)
	if err := clientSMTP.VerifierConfiguration(); err != nil {
		return err
	}

	// Configure le routeur et ajoute les protections HTTP communes à toutes les routes.
	router := gin.Default()
	router.Use(middleware.IdentifiantRequete())
	router.Use(middleware.EntetesSecurite())
	router.Use(middleware.LimiterTailleCorps(25 << 20))
	email.RegisterRoutes(router, repository, redis, redis, cfg.LimiteCreationApplication, cfg.LimiteAPI, stockagePieces)

	// Le worker traite les emails et reprend les tentatives après un délai d'attente.
	contexte, arreter := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer arreter()
	contexteWorker, arreterWorker := context.WithCancel(contexte)
	defer arreterWorker()
	workerTermine := make(chan struct{})
	go func() {
		defer close(workerTermine)
		email.NouveauWorkerEnvoi(repository, clientSMTP).Executer(contexteWorker)
	}()

	// Les délais HTTP bornent les connexions et permettent un arrêt propre du serveur.
	serveur := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.AppPort),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      45 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	erreursServeur := make(chan error, 1)
	go func() {
		err := serveur.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			erreursServeur <- nil
			return
		}
		erreursServeur <- err
	}()

	// En cas de signal, le serveur termine les requêtes actives avant de fermer les dépendances.
	select {
	case <-contexte.Done():
		contexteArret, annuler := context.WithTimeout(context.Background(), 10*time.Second)
		erreurArret := serveur.Shutdown(contexteArret)
		annuler()
		arreterWorker()
		select {
		case <-workerTermine:
		case <-time.After(40 * time.Second):
			erreurArret = errors.Join(erreurArret, errors.New("le worker d'envoi ne s'est pas arrêté à temps"))
		}
		return erreurArret
	case err := <-erreursServeur:
		arreterWorker()
		select {
		case <-workerTermine:
		case <-time.After(40 * time.Second):
			err = errors.Join(err, errors.New("le worker d'envoi ne s'est pas arrêté à temps"))
		}
		return err
	}
}
