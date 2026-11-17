package email

import (
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strings"

	setting "email-send-API/src/Setting"
)

type SMTPClient struct {
	Configuration *setting.Config
}

func NewSMTPClient(cfg *setting.Config) *SMTPClient {
	return &SMTPClient{
		Configuration: cfg,
	}
}

func (client *SMTPClient) SendEmail(destinataire, subject, body, contentType string) error {
	if client == nil || client.Configuration == nil {
		return fmt.Errorf("configuration SMTP absente")
	}
	// Validation des paramètres de configuration SMTP
	if strings.TrimSpace(client.Configuration.SMTPHost) == "" {
		return fmt.Errorf("SMTP_HOST est requis")
	}
	// Validation du port SMTP
	if client.Configuration.SMTPPort < 1 || client.Configuration.SMTPPort > 65535 {
		return fmt.Errorf("SMTP_PORT doit être compris entre 1 et 65535")
	}
	// Validation de l'utilisateur SMTP
	if strings.TrimSpace(client.Configuration.SMTPUser) == "" {
		return fmt.Errorf("SMTP_USER est requis")
	}
	// Validation du mot de passe SMTP
	if client.Configuration.SMTPPassword == "" {
		return fmt.Errorf("SMTP_PASSWORD est requis")
	}

	// Récupération de la configuration
	host := strings.TrimSpace(client.Configuration.SMTPHost)

	// Construction de l'adresse réseau complète
	address := net.JoinHostPort(host, fmt.Sprint(client.Configuration.SMTPPort))

	// Vérifier les adresses email de l'expéditeur
	from, err := mail.ParseAddress(client.Configuration.SMTPFrom)
	if err != nil {
		return fmt.Errorf("adresse expéditeur invalide : %w", err)
	}

	// Vérifier l'adresse du destinataire.
	recipient, err := mail.ParseAddress(destinataire)
	if err != nil {
		return fmt.Errorf("adresse destinataire invalide : %w", err)
	}

	// Éviter l'injection d'en-têtes
	if strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("objet email invalide")
	}

	if contentType != "TEXT" && contentType != "HTML" {
		return fmt.Errorf("type de contenu invalide")
	}

	mimeType := "text/plain"
	if contentType == "HTML" {
		mimeType = "text/html"
	}

	// Construire le message
	message := []byte(
		"From: " + from.String() + "\r\n" +
			"To: " + recipient.String() + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: " + mimeType + "; charset=UTF-8\r\n" +
			"\r\n" +
			body,
	)

	// Authentification SMTP
	auth := smtp.PlainAuth(
		"",
		client.Configuration.SMTPUser,
		client.Configuration.SMTPPassword,
		host,
	)

	// Envoi du mail
	err = smtp.SendMail(
		address,
		auth,
		from.Address,
		[]string{recipient.Address},
		message,
	)

	if err != nil {
		return fmt.Errorf("erreur envoi SMTP : %w", err)
	}

	return nil
}
