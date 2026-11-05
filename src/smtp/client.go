package smtp

import (
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strings"

	"email-send-API/config"
)

type SMTPClient struct {
	Config *config.Config
}

func NewSMTPClient(cfg *config.Config) *SMTPClient {
	return &SMTPClient{
		Config: cfg,
	}
}

func (s *SMTPClient) SendEmail(to, subject, body, contentType string) error {

	host := s.Config.SMTPHost
	address := net.JoinHostPort(host, fmt.Sprint(s.Config.SMTPPort))

	// Vérifier les adresses email
	from, err := mail.ParseAddress(s.Config.SMTPFrom)
	if err != nil {
		return fmt.Errorf("adresse expéditeur invalide : %w", err)
	}

	recipient, err := mail.ParseAddress(to)
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
		s.Config.SMTPUser,
		s.Config.SMTPPassword,
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
