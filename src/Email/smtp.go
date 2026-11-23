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

// NewSMTPClient crée un nouveau client SMTP avec la configuration fournie.
func NewSMTPClient(cfg *setting.Config) *SMTPClient {
	return &SMTPClient{
		Configuration: cfg,
	}
}

// SendEmail envoie un email en utilisant la configuration SMTP du client.
func (client *SMTPClient) SendEmail(destinataire, subject, body, contentType string) error {
	if client == nil || client.Configuration == nil {
		return fmt.Errorf("configuration SMTP absente")
	}

	cfg := client.Configuration

	host := strings.TrimSpace(cfg.SMTPHost)

	// Vérifier la validité des paramètres SMTP
	if host == "" {
		return fmt.Errorf("SMTP_HOST est requis")
	}
	// Vérifier la validité du port SMTP
	if cfg.SMTPPort < 1 || cfg.SMTPPort > 65535 {
		return fmt.Errorf("SMTP_PORT invalide")
	}
	// Vérifier la validité des identifiants SMTP
	if cfg.SMTPUser == "" || cfg.SMTPPassword == "" {
		return fmt.Errorf("identifiants SMTP manquants")
	}
	// Vérifier la validité de l'adresse expéditeur
	from, err := mail.ParseAddress(cfg.SMTPFrom)
	if err != nil {
		return fmt.Errorf("adresse expéditeur invalide : %w", err)
	}
	// Vérifier la validité de l'adresse destinataire
	recipient, err := mail.ParseAddress(destinataire)
	if err != nil {
		return fmt.Errorf("adresse destinataire invalide : %w", err)
	}
	// Vérifier la validité de l'objet de l'email
	if strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("objet email invalide")
	}
	// Vérifier la validité du type de contenu
	if contentType != "TEXT" && contentType != "HTML" {
		return fmt.Errorf("type de contenu invalide")
	}
	// Construire le message email avec les en-têtes appropriés
	mimeType := "text/plain"
	if contentType == "HTML" {
		mimeType = "text/html"
	}

	message := []byte(
		"From: " + from.String() + "\r\n" +
			"To: " + recipient.String() + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: " + mimeType + "; charset=UTF-8\r\n" +
			"\r\n" +
			body,
	)
	// Configurer l'authentification SMTP avec les identifiants fournis
	auth := smtp.PlainAuth(
		"",
		cfg.SMTPUser,
		cfg.SMTPPassword,
		host,
	)
	// Construire l'adresse du serveur SMTP avec le port approprié
	address := net.JoinHostPort(host, fmt.Sprint(cfg.SMTPPort))

	// Envoyer l'email en utilisant la fonction SendMail du package smtp
	err = smtp.SendMail(
		address,
		auth,
		from.Address,
		[]string{recipient.Address},
		message,
	)
	// Vérifier si l'envoi de l'email a échoué et retourner une erreur descriptive
	if err != nil {
		return fmt.Errorf(
			"échec envoi SMTP vers %s:%d : %w",
			host,
			cfg.SMTPPort,
			err,
		)
	}

	return nil
}
