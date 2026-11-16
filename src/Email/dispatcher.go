package email

import (
	"context"
	"errors"
	"log"
	"time"

	database "email-send-API/src/Database"
)

// WorkerEnvoi traite la file PostgreSQL et enregistre chaque tentative SMTP.
type WorkerEnvoi struct {
	depot   *database.Repository
	smtp    *SMTPClient
	attente time.Duration
}

// NouveauWorkerEnvoi crée le traitement de file partagé entre les instances du service.
func NouveauWorkerEnvoi(depot *database.Repository, smtp *SMTPClient) *WorkerEnvoi {
	return &WorkerEnvoi{
		depot:   depot,
		smtp:    smtp,
		attente: time.Second,
	}
}

// Executer traite les emails jusqu'à l'arrêt du processus.
func (d *WorkerEnvoi) Executer(contexte context.Context) {
	ticker := time.NewTicker(d.attente)
	defer ticker.Stop()

	for {
		traite, err := d.traiterEmail(contexte)
		if err != nil {
			log.Printf("traitement de la file d'emails impossible: %v", err)
		}
		if traite {
			continue
		}
		select {
		case <-contexte.Done():
			return
		case <-ticker.C:
		}
	}
}

// traiterEmail réserve un email, le remet au relais SMTP et persiste le résultat.
func (d *WorkerEnvoi) traiterEmail(contexte context.Context) (bool, error) {
	email, pieces, tentative, err := d.depot.ReclamerProchainEmail(contexte)
	if errors.Is(err, database.ErrAucunEmailEnAttente) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	piecesSMTP := make([]PieceJointePourSMTP, 0, len(pieces))
	for _, piece := range pieces {
		piecesSMTP = append(piecesSMTP, PieceJointePourSMTP{
			NomFichier: piece.NomFichier,
			TypeMIME:   piece.TypeMIME,
			Chemin:     piece.CheminFichier,
		})
	}
	errEnvoi := d.smtp.SendEmailAvecPiecesJointes(
		email.Destinataire,
		email.Objet,
		email.Message,
		email.TypeContenu,
		piecesSMTP,
	)

	// La finalisation utilise un contexte indépendant pour persister le résultat même à l'arrêt.
	contexteFinalisation, annuler := context.WithTimeout(context.Background(), 5*time.Second)
	defer annuler()
	if err := d.depot.FinaliserTentative(contexteFinalisation, email.IDEmail, tentative.IDTentative, errEnvoi); err != nil {
		return true, err
	}
	if errEnvoi != nil {
		log.Printf("envoi de l'email %d, tentative %d: %v", email.IDEmail, tentative.NumeroTentative, errEnvoi)
	}
	return true, nil
}
