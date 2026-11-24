package email

import (
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

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

	// Vérifier la configuration
	if client == nil || client.Configuration == nil {
		return fmt.Errorf("configuration SMTP absente")
	}

	cfg := client.Configuration

	host := strings.TrimSpace(cfg.SMTPHost)

	// Vérifier le serveur SMTP
	if host == "" {
		return fmt.Errorf("SMTP_HOST est requis")
	}

	// Vérifier le port SMTP
	if cfg.SMTPPort < 1 || cfg.SMTPPort > 65535 {
		return fmt.Errorf("SMTP_PORT invalide")
	}

	// Vérifier les identifiants SMTP
	if strings.TrimSpace(cfg.SMTPUser) == "" || cfg.SMTPPassword == "" {
		return fmt.Errorf("identifiants SMTP manquants")
	}

	// Vérifier l'adresse expéditeur
	from, err := mail.ParseAddress(cfg.SMTPFrom)
	if err != nil {
		return fmt.Errorf("adresse expéditeur invalide : %w", err)
	}

	// Vérifier l'adresse destinataire
	recipient, err := mail.ParseAddress(destinataire)
	if err != nil {
		return fmt.Errorf("adresse destinataire invalide : %w", err)
	}

	// Vérifier l'objet de l'email contre les injections d'en-têtes
	if strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("objet email invalide")
	}

	// Vérifier le type de contenu
	contentType = strings.ToUpper(strings.TrimSpace(contentType))

	if contentType != "TEXT" && contentType != "HTML" {
		return fmt.Errorf("type de contenu invalide : TEXT ou HTML attendu")
	}

	// Déterminer le type MIME
	mimeType := "text/plain"

	if contentType == "HTML" {
		mimeType = "text/html"
	}

	// Encoder le sujet pour supporter les caractères UTF-8
	encodedSubject := mime.QEncoding.Encode("UTF-8", subject)

	// Construire le message email
	message := []byte(
		"From: " + from.String() + "\r\n" +
			"To: " + recipient.String() + "\r\n" +
			"Subject: " + encodedSubject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: " + mimeType + "; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: 8bit\r\n" +
			"\r\n" +
			body,
	)

	// Construire l'adresse du serveur SMTP
	address := net.JoinHostPort(host, fmt.Sprint(cfg.SMTPPort))

	// Établir une connexion avec timeout
	dialer := net.Dialer{
		Timeout: 10 * time.Second,
	}

	conn, err := dialer.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("connexion SMTP impossible : %w", err)
	}

	// Créer le client SMTP
	smtpConnection, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("initialisation SMTP impossible : %w", err)
	}

	defer smtpConnection.Close()

	// Configurer TLS
	tlsConfig := &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	}

	// Vérifier si le serveur supporte STARTTLS
	if cfg.SMTPTLS {

		ok, _ := smtpConnection.Extension("STARTTLS")

		if !ok {
			return fmt.Errorf("STARTTLS non supporté par le serveur SMTP")
		}

		// Activer STARTTLS
		if err := smtpConnection.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("erreur STARTTLS : %w", err)
		}
	}

	// Authentification SMTP
	auth := smtp.PlainAuth(
		"",
		cfg.SMTPUser,
		cfg.SMTPPassword,
		host,
	)

	if err := smtpConnection.Auth(auth); err != nil {
		return fmt.Errorf("erreur authentification SMTP : %w", err)
	}

	// Définir l'expéditeur
	if err := smtpConnection.Mail(from.Address); err != nil {
		return fmt.Errorf("erreur définition expéditeur : %w", err)
	}

	// Définir le destinataire
	if err := smtpConnection.Rcpt(recipient.Address); err != nil {
		return fmt.Errorf("erreur définition destinataire : %w", err)
	}

	// Préparer l'envoi du message
	writer, err := smtpConnection.Data()
	if err != nil {
		return fmt.Errorf("impossible de préparer le message : %w", err)
	}

	// Écrire le contenu du message
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return fmt.Errorf("erreur écriture du message : %w", err)
	}

	// Terminer l'envoi
	if err := writer.Close(); err != nil {
		return fmt.Errorf("erreur finalisation de l'envoi : %w", err)
	}

	// Fermer proprement la session SMTP
	if err := smtpConnection.Quit(); err != nil {
		return fmt.Errorf("erreur fermeture session SMTP : %w", err)
	}

	return nil
}
