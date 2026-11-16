package email

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"os"
	"strings"
	"time"

	setting "email-send-API/src/Setting"
)

// PieceJointePourSMTP décrit un fichier à joindre au message transmis par SMTP.
type PieceJointePourSMTP struct {
	NomFichier string
	TypeMIME   string
	Chemin     string
}

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

// VerifierConfiguration valide les paramètres nécessaires au relais avant le démarrage.
func (client *SMTPClient) VerifierConfiguration() error {
	if client == nil || client.Configuration == nil {
		return fmt.Errorf("configuration SMTP absente")
	}
	if strings.TrimSpace(client.Configuration.SMTPHost) == "" {
		return fmt.Errorf("SMTP_HOST est obligatoire")
	}
	if client.Configuration.SMTPPort < 1 || client.Configuration.SMTPPort > 65535 {
		return fmt.Errorf("SMTP_PORT doit être compris entre 1 et 65535")
	}
	if _, err := mail.ParseAddress(client.Configuration.SMTPFrom); err != nil {
		return fmt.Errorf("adresse expéditeur invalide : %w", err)
	}
	if (client.Configuration.SMTPUser == "") != (client.Configuration.SMTPPassword == "") {
		return fmt.Errorf("SMTP_USER et SMTP_PASSWORD doivent être renseignés ensemble")
	}
	return nil
}

// SendEmail valide les champs d'en-tête puis remet un message au serveur SMTP.
func (client *SMTPClient) SendEmail(recipientAddress, subject, body, contentType string) error {
	return client.SendEmailAvecPiecesJointes(recipientAddress, subject, body, contentType, nil)
}

// SendEmailAvecPiecesJointes assemble le message MIME et l'envoie avec un délai réseau borné.
func (client *SMTPClient) SendEmailAvecPiecesJointes(adresseDestinataire, objet, contenu string, typeContenu string, pieces []PieceJointePourSMTP) error {
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

	recipient, err := mail.ParseAddress(adresseDestinataire)
	if err != nil {
		return fmt.Errorf("adresse destinataire invalide : %w", err)
	}

	if strings.ContainsAny(objet, "\r\n") {
		return fmt.Errorf("objet email invalide")
	}

	if typeContenu != "TEXT" && typeContenu != "HTML" {
		return fmt.Errorf("type de contenu invalide")
	}

	// Le type MIME dépend du format accepté par le contrat HTTP.
	mimeType := "text/plain"
	if typeContenu == "HTML" {
		mimeType = "text/html"
	}

	message, err := construireMessageMIME(from, recipient, objet, contenu, mimeType, pieces)
	if err != nil {
		return err
	}

	// La connexion est limitée en durée afin qu'un serveur SMTP bloqué ne bloque pas le worker.
	var connexion net.Conn
	if client.Configuration.SMTPPort == 465 {
		connexion, err = tls.DialWithDialer(
			&net.Dialer{Timeout: 20 * time.Second},
			"tcp",
			address,
			&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12},
		)
	} else {
		connexion, err = net.DialTimeout("tcp", address, 20*time.Second)
	}
	if err != nil {
		return fmt.Errorf("connexion au serveur SMTP impossible: %w", err)
	}
	defer connexion.Close()
	if err := connexion.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return fmt.Errorf("définir le délai SMTP: %w", err)
	}

	clientSMTP, err := smtp.NewClient(connexion, host)
	if err != nil {
		return fmt.Errorf("initialiser le client SMTP: %w", err)
	}
	defer clientSMTP.Close()

	// Active TLS lorsque le serveur le propose avant d'envoyer les identifiants.
	if client.Configuration.SMTPPort != 465 {
		if disponible, _ := clientSMTP.Extension("STARTTLS"); disponible {
			if err := clientSMTP.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
				return fmt.Errorf("activer TLS pour SMTP: %w", err)
			}
		}
	}
	if client.Configuration.SMTPUser != "" {
		auth := smtp.PlainAuth("", client.Configuration.SMTPUser, client.Configuration.SMTPPassword, host)
		if err := clientSMTP.Auth(auth); err != nil {
			return fmt.Errorf("authentification SMTP impossible: %w", err)
		}
	}
	if err := clientSMTP.Mail(from.Address); err != nil {
		return fmt.Errorf("déclarer l'expéditeur SMTP: %w", err)
	}
	if err := clientSMTP.Rcpt(recipient.Address); err != nil {
		return fmt.Errorf("déclarer le destinataire SMTP: %w", err)
	}
	corps, err := clientSMTP.Data()
	if err != nil {
		return fmt.Errorf("ouvrir le message SMTP: %w", err)
	}
	if _, err := corps.Write(message); err != nil {
		_ = corps.Close()
		return fmt.Errorf("transmettre le message SMTP: %w", err)
	}
	if err := corps.Close(); err != nil {
		return fmt.Errorf("terminer le message SMTP: %w", err)
	}
	if err := clientSMTP.Quit(); err != nil {
		// DATA a déjà été accepté; une erreur de fermeture ne doit pas provoquer un renvoi en doublon.
		log.Printf("fermeture de session SMTP après acceptation du message impossible: %v", err)
	}
	return nil
}

// construireMessageMIME encode le corps et les fichiers sans placer leurs noms dans les en-têtes bruts.
func construireMessageMIME(expediteur, destinataire *mail.Address, objet, contenu, typeMIME string, pieces []PieceJointePourSMTP) ([]byte, error) {
	var tamponMIME bytes.Buffer
	ecrivain := multipart.NewWriter(&tamponMIME)
	entetes := "From: " + expediteur.String() + "\r\n" +
		"To: " + destinataire.String() + "\r\n" +
		"Subject: " + mime.QEncoding.Encode("UTF-8", objet) + "\r\n" +
		"MIME-Version: 1.0\r\n"

	if len(pieces) == 0 {
		entetes += "Content-Type: " + typeMIME + "; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: quoted-printable\r\n\r\n"
		var corpsEncode bytes.Buffer
		encodeur := quotedprintable.NewWriter(&corpsEncode)
		if _, err := io.WriteString(encodeur, contenu); err != nil {
			return nil, fmt.Errorf("encoder le corps du message: %w", err)
		}
		if err := encodeur.Close(); err != nil {
			return nil, fmt.Errorf("terminer l'encodage du corps: %w", err)
		}
		return append([]byte(entetes), corpsEncode.Bytes()...), nil
	}

	entetes += "Content-Type: multipart/mixed; boundary=" + ecrivain.Boundary() + "\r\n\r\n"
	corpsEntete := make(textproto.MIMEHeader)
	corpsEntete.Set("Content-Type", typeMIME+"; charset=UTF-8")
	corpsEntete.Set("Content-Transfer-Encoding", "quoted-printable")
	partieCorps, err := ecrivain.CreatePart(corpsEntete)
	if err != nil {
		return nil, fmt.Errorf("créer la partie texte du message: %w", err)
	}
	encodeurTexte := quotedprintable.NewWriter(partieCorps)
	if _, err := io.WriteString(encodeurTexte, contenu); err != nil {
		return nil, fmt.Errorf("encoder le corps du message: %w", err)
	}
	if err := encodeurTexte.Close(); err != nil {
		return nil, fmt.Errorf("terminer l'encodage du corps: %w", err)
	}

	for _, piece := range pieces {
		if err := ajouterPieceJointeMIME(ecrivain, piece); err != nil {
			return nil, err
		}
	}
	if err := ecrivain.Close(); err != nil {
		return nil, fmt.Errorf("terminer le message MIME: %w", err)
	}
	return append([]byte(entetes), tamponMIME.Bytes()...), nil
}

// ajouterPieceJointeMIME encode un fichier en base64 dans une partie MIME.
func ajouterPieceJointeMIME(ecrivain *multipart.Writer, piece PieceJointePourSMTP) error {
	typeMIME := piece.TypeMIME
	if _, _, err := mime.ParseMediaType(typeMIME); err != nil {
		typeMIME = "application/octet-stream"
	}
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": piece.NomFichier})
	entetes := make(textproto.MIMEHeader)
	entetes.Set("Content-Type", typeMIME)
	entetes.Set("Content-Disposition", disposition)
	entetes.Set("Content-Transfer-Encoding", "base64")
	partie, err := ecrivain.CreatePart(entetes)
	if err != nil {
		return fmt.Errorf("créer la partie jointe: %w", err)
	}
	fichier, err := os.Open(piece.Chemin)
	if err != nil {
		return fmt.Errorf("ouvrir la pièce jointe %q: %w", piece.NomFichier, err)
	}
	defer fichier.Close()
	ecrivainLignes := &ecrivainBase64SurLignes{destination: partie}
	encodeur := base64.NewEncoder(base64.StdEncoding, ecrivainLignes)
	if _, err := io.Copy(encodeur, fichier); err != nil {
		_ = encodeur.Close()
		return fmt.Errorf("encoder la pièce jointe %q: %w", piece.NomFichier, err)
	}
	if err := encodeur.Close(); err != nil {
		return fmt.Errorf("terminer l'encodage de la pièce jointe %q: %w", piece.NomFichier, err)
	}
	if err := ecrivainLignes.Terminer(); err != nil {
		return fmt.Errorf("terminer les lignes de la pièce jointe %q: %w", piece.NomFichier, err)
	}
	return nil
}

// ecrivainBase64SurLignes respecte la taille maximale des lignes MIME base64.
type ecrivainBase64SurLignes struct {
	destination io.Writer
	colonne     int
}

func (e *ecrivainBase64SurLignes) Write(donnees []byte) (int, error) {
	longueurInitiale := len(donnees)
	for len(donnees) > 0 {
		place := 76 - e.colonne
		taille := len(donnees)
		if taille > place {
			taille = place
		}
		nombre, err := e.destination.Write(donnees[:taille])
		e.colonne += nombre
		donnees = donnees[nombre:]
		if err != nil {
			return longueurInitiale - len(donnees), err
		}
		if nombre != taille {
			return longueurInitiale - len(donnees), io.ErrShortWrite
		}
		if e.colonne == 76 {
			if _, err := io.WriteString(e.destination, "\r\n"); err != nil {
				return longueurInitiale - len(donnees), err
			}
			e.colonne = 0
		}
	}
	return longueurInitiale, nil
}

// Terminer ferme la dernière ligne lorsque l'encodage n'a pas atteint 76 caractères.
func (e *ecrivainBase64SurLignes) Terminer() error {
	if e.colonne == 0 {
		return nil
	}
	_, err := io.WriteString(e.destination, "\r\n")
	e.colonne = 0
	return err
}
