package email

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	setting "email-send-API/src/Setting"
	"email-send-API/src/database"
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
func (client *SMTPClient) SendEmail(destinataire, subject, body, contentType string, attachments []database.PieceJointe) error {

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

	message, err := buildMIMEMessage(from, recipient, subject, body, contentType, attachments)
	if err != nil {
		return fmt.Errorf("construction du message impossible : %w", err)
	}
	// Construire l'adresse du serveur SMTP
	address := net.JoinHostPort(host, fmt.Sprint(cfg.SMTPPort))

	// Établir une connexion avec timeout
	dialer := net.Dialer{
		Timeout: 10 * time.Second,
	}

	connected, err := dialer.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("connexion SMTP impossible : %w", err)
	}

	defer connected.Close()

	if err := connected.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return fmt.Errorf("impossible de définir le délai réseau : %w", err)
	}
	// Créer le client SMTP
	smtpConnection, err := smtp.NewClient(connected, host)
	if err != nil {
		return fmt.Errorf("initialisation SMTP impossible : %w", err)
	}

	defer smtpConnection.Close()

	// Configurer TLS
	tlsConfig := &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	}
	// Vérifier si le serveur supporte STARTTLS
	if !cfg.SMTPTLS {
		return fmt.Errorf("une connexion TLS est obligatoire")
	}
	ok, _ := smtpConnection.Extension("STARTTLS")

	if !ok {
		return fmt.Errorf("STARTTLS non supporté par le serveur SMTP")
	}

	if err := smtpConnection.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("échec de l'activation de TLS : %w", err)
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
	n, err := writer.Write(message)

	if err != nil {
		_ = writer.Close()
		return fmt.Errorf("erreur écriture du message : %w", err)
	}

	if n != len(message) {
		_ = writer.Close()
		return fmt.Errorf("écriture incomplète du message")
	}

	// Terminer l'envoi
	if err := writer.Close(); err != nil {
		return fmt.Errorf("erreur finalisation de l'envoi : %w", err)
	}

	// Fermer proprement la session SMTP
	if err := smtpConnection.Quit(); err != nil {
		log.Printf("email accepté par le serveur SMTP, mais fermeture de session impossible: %v", err)
	}

	return nil
}
