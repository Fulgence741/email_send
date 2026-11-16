package main

import (
	email "email-send-API/src/Email"
	setting "email-send-API/src/Setting"
	"os"
	"testing"
)

// TestSendEmailRejectsMissingConfiguration vérifie qu'un client nil échoue proprement.
func TestSendEmailRejectsMissingConfiguration(t *testing.T) {
	var client *email.SMTPClient
	if err := client.SendEmail("to@example.com", "Objet", "Contenu", "TEXT"); err == nil {
		t.Fatal("un client SMTP sans configuration devrait être refusé")
	}
}

// TestSendEmailRejectsInvalidRecipient s'assure qu'une adresse incorrecte n'est pas envoyée.
func TestSendEmailRejectsInvalidRecipient(t *testing.T) {
	client := email.NewSMTPClient(&setting.Config{
		SMTPHost: "smtp.example.com",
		SMTPFrom: "from@example.com",
	})
	if err := client.SendEmail("adresse-invalide", "Objet", "Contenu", "TEXT"); err == nil {
		t.Fatal("une adresse destinataire invalide devrait être refusée")
	}
}

// TestSendEmail est un test d'intégration réel, désactivé sauf activation explicite.
func TestSendEmail(t *testing.T) {
	if os.Getenv("SMTP_INTEGRATION_TEST") != "1" {
		t.Skip("définir SMTP_INTEGRATION_TEST=1 pour envoyer un vrai email")
	}

	cfg, err := setting.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.SMTPUser == "" {
		t.Fatal("SMTP_USER est vide")
	}

	if cfg.SMTPPassword == "" {
		t.Fatal("SMTP_PASSWORD est vide")
	}

	if cfg.SMTPFrom == "" {
		t.Fatal("SMTP_FROM est vide")
	}

	t.Logf("SMTP Host : %s", cfg.SMTPHost)
	t.Logf("SMTP Port : %d", cfg.SMTPPort)
	t.Logf("SMTP User : %s", cfg.SMTPUser)
	t.Logf("SMTP From : %s", cfg.SMTPFrom)

	client := email.NewSMTPClient(cfg)

	err = client.SendEmail(
		"reply@sensei.ci",
		"Test SMTP Sensei",
		"Bonjour, ceci est un test de mon serveur SMTP.",
		"TEXT",
	)

	if err != nil {
		t.Fatalf("Échec de l'envoi SMTP : %v", err)
	}

	t.Log("Email accepté par le serveur SMTP !")
}
