package main

import (
	email "email-send-API/src/Email"
	setting "email-send-API/src/Setting"
	"testing"
)

func TestSendEmail(t *testing.T) {
	//Récupérer la configuration existente

	cfg := setting.LoadConfig()

	t.Log("SMTP Host :", cfg.SMTPHost)
	t.Log("SMTP Port :", cfg.SMTPPort)
	t.Log("SMTP User :", cfg.SMTPUser)
	t.Log("SMTP From :", cfg.SMTPFrom)
	t.Log("Password renseigné :", cfg.SMTPPassword != "")

	// Créer un client smtp

	client := email.NewSMTPClient(cfg)

	// Envoyer un email test

	err := client.SendEmail(
		"reply@sensei.ci",
		"test SMTP",
		"Bonjour ceci est un test de mon serveur smtp",
		"TEXT",
	)

	if err != nil {
		t.Fatalf("Erreur lors de l'enovoie : %v", err)
	}

	t.Log("Email envoyé avec succès !")
}
