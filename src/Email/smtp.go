package email

import (
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strings"

	setting "email-send-API/src/Setting"
)

// SMTPClient envoie les emails à l'aide de la configuration SMTP fournie.
type SMTPClient struct {
	Configuration *setting.Config
}

// NewSMTPClient crée un client SMTP utilisant cfg pour chaque envoi.
func NewSMTPClient(cfg *setting.Config) *SMTPClient {
	return &SMTPClient{
		Configuration: cfg,
	}
}

// SendEmail valide les champs d'en-tête puis remet un message au serveur SMTP.
func (client *SMTPClient) SendEmail(recipientAddress, subject, body, contentType string) error {
	if client == nil || client.Configuration == nil {
		return fmt.Errorf("configuration SMTP absente")
	}

	host := client.Configuration.SMTPHost
	if host == "" {
		return fmt.Errorf("hôte SMTP absent")
	}

	address := net.JoinHostPort(host, fmt.Sprint(client.Configuration.SMTPPort))

	from, err := mail.ParseAddress(client.Configuration.SMTPFrom)
	if err != nil {
		return fmt.Errorf("adresse expéditeur invalide : %w", err)
	}

	recipient, err := mail.ParseAddress(recipientAddress)
	if err != nil {
		return fmt.Errorf("adresse destinataire invalide : %w", err)
	}

	if strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("objet email invalide")
	}

	if contentType != "TEXT" && contentType != "HTML" {
		return fmt.Errorf("type de contenu invalide")
	}

	// Le type MIME dépend du format accepté par le contrat HTTP.
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

	// L'authentification SMTP est nécessaire pour la plupart des serveurs.
	auth := smtp.PlainAuth(
		"",
		client.Configuration.SMTPUser,
		client.Configuration.SMTPPassword,
		host,
	)

	// Envoie l'email via le serveur SMTP configuré.
	err = smtp.SendMail(
		address,
		auth,
		from.Address,
		[]string{recipient.Address},
		message,
	)

	// Retourne une erreur si l'envoi échoue, sinon nil.
	if err != nil {
		return fmt.Errorf("erreur envoi SMTP : %w", err)
	}

	return nil
}
