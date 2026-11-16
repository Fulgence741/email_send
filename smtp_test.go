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

	// Charge la configuration depuis les variables d'environnement.
	cfg, err := setting.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}

	// Vérifie que les variables d'environnement SMTP sont présentes.
	if cfg.SMTPUser == "" {
		t.Fatal("SMTP_USER est vide")
	}
	// Vérifie que les variables d'environnement SMTP sont présentes.
	if cfg.SMTPPassword == "" {
		t.Fatal("SMTP_PASSWORD est vide")
	}

	// Vérifie que les variables d'environnement SMTP sont présentes.
	if cfg.SMTPFrom == "" {
		t.Fatal("SMTP_FROM est vide")
	}

	// Vérifie que les variables d'environnement SMTP sont présentes.
	if cfg.SMTPHost == "" {
		t.Fatal("SMTP_HOST est vide")
	}

	// Vérifie que les variables d'environnement SMTP sont présentes.
	if cfg.SMTPPort == 0 {
		t.Fatal("SMTP_PORT est vide")
	}
	// Affiche les informations de configuration pour le débogage.
	t.Logf("SMTP Host : %s", cfg.SMTPHost)
	t.Logf("SMTP Port : %d", cfg.SMTPPort)
	t.Logf("SMTP User : %s", cfg.SMTPUser)
	t.Logf("SMTP From : %s", cfg.SMTPFrom)

	// Crée un client SMTP avec la configuration chargée.
	client := email.NewSMTPClient(cfg)

	// Envoie un email de test à une adresse de test. Remplacez "reply@sensei.ci" par l'adresse de test souhaitée.
	err = client.SendEmail(
		"reply@sensei.ci",
		"Test SMTP Sensei",
		"Bonjour, ceci est un test de mon serveur SMTP.",
		"TEXT",
	)

	// Vérifie si l'envoi a échoué et échoue le test si c'est le cas.
	if err != nil {
		t.Fatalf("Échec de l'envoi SMTP : %v", err)
	}

	t.Log("Email accepté par le serveur SMTP !")
}
