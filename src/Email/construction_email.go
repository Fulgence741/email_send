package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"os"
	"path"
	"strings"

	"email-send-API/src/database"
)

// Structure pour gérer l'écriture de données
type mimeLineWriter struct {
	writer io.Writer // Destination
	column int       // Compte le nombre de caractère ecris sur la ligne
}

// Fonction pour controler la  longueur des lignes pendant l'écriture
func (request *mimeLineWriter) Write(data []byte) (int, error) {
	total := 0 // Permet de compter le nombre d'octets écrits
	for len(data) > 0 {

		//Retourner à la ligne si l'on atteint 76 caractères
		if request.column == 76 {

			//gère le retour à la ligne
			n, err := io.WriteString(request.writer, "\r\n")
			if err != nil {
				return total, err
			}
			if n != 2 {
				return total, io.ErrShortWrite
			}

			// Reinitiallisation du compteur sur une nouvelle ligne
			request.column = 0
		}

		//determiner le nombre de caractère restant avant d'atteindre 76
		size := 76 - request.column
		if size > len(data) {
			size = len(data)
		}
		n, err := request.writer.Write(data[:size])
		total += n
		request.column += n
		data = data[n:]
		if err != nil {
			return total, err
		}
		if n != size {
			return total, io.ErrShortWrite
		}
	}
	return total, nil
}

// Fonction pour contruire un email
func buildMIMEMessage(from, recipient *mail.Address, subject, body, contentType string, attachments []database.PieceJointe) ([]byte, error) {
	//Determiner le format
	bodyType := "text/plain"
	if contentType == "HTML" {
		bodyType = "text/html"
	}

	//Stockage progrssif des données
	var message bytes.Buffer
	message.WriteString("From: " + from.String() + "\r\n")                              // ajout d'expéditeur
	message.WriteString("To: " + recipient.String() + "\r\n")                           // ajout du destinataire
	message.WriteString("Subject: " + mime.QEncoding.Encode("UTF-8", subject) + "\r\n") // ajout de l'objet
	message.WriteString("MIME-Version: 1.0\r\n")                                        // la version utilisée

	// vérifier si le message contient des pieces jointes
	if len(attachments) == 0 {
		message.WriteString("Content-Type: " + bodyType + "; charset=UTF-8\r\n")
		message.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		writer := quotedprintable.NewWriter(&message)

		// ecrire le corps  du message
		if _, err := io.WriteString(writer, body); err != nil {
			return nil, err
		}
		// fermer l'écrivain du message
		if err := writer.Close(); err != nil {
			return nil, err
		}
		// recupérer le contenu sous forme de tableau d'octets
		return message.Bytes(), nil
	}

	// construire un email contenant une ou plusieurs pieces jointes

	multipartWriter := multipart.NewWriter(&message)
	message.WriteString("Content-Type: multipart/mixed; boundary=" + multipartWriter.Boundary() + "\r\n\r\n")

	bodyHeader := make(textproto.MIMEHeader)
	bodyHeader.Set("Content-Type", bodyType+"; charset=UTF-8")
	bodyHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	bodyPart, err := multipartWriter.CreatePart(bodyHeader)
	if err != nil {
		return nil, err
	}
	bodyWriter := quotedprintable.NewWriter(bodyPart)
	if _, err := io.WriteString(bodyWriter, body); err != nil {
		return nil, err
	}
	if err := bodyWriter.Close(); err != nil {
		return nil, err
	}

	// Parcourir toutes les pieces jointes en ignorant l'index
	for _, attachment := range attachments {
		//
		if err := writeAttachment(multipartWriter, attachment); err != nil {
			return nil, err
		}
	}
	//fermerture de multipart
	if err := multipartWriter.Close(); err != nil {
		return nil, err
	}
	return message.Bytes(), nil
}

// intégration d'une piece jointe dans l'email
func writeAttachment(writer *multipart.Writer, attachment database.PieceJointe) error {
	filename := path.Base(strings.ReplaceAll(attachment.NomFichier, "\\", "/"))
	if filename == "." || filename == "/" || strings.ContainsAny(filename, "\r\n") {
		return fmt.Errorf("nom de pièce jointe invalide")
	}

	contentType, params, err := mime.ParseMediaType(attachment.TypeMIME)
	if err != nil || strings.ContainsAny(attachment.TypeMIME, "\r\n") {
		contentType = "application/octet-stream"
		params = nil
	}
	contentType = mime.FormatMediaType(contentType, params)
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": filename})
	if disposition == "" {
		return fmt.Errorf("nom de pièce jointe invalide")
	}

	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", contentType)
	header.Set("Content-Disposition", disposition)
	header.Set("Content-Transfer-Encoding", "base64")
	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}

	file, err := os.Open(attachment.CheminFichier)
	if err != nil {
		return fmt.Errorf("ouverture de la pièce jointe %q impossible: %w", filename, err)
	}

	encoder := base64.NewEncoder(base64.StdEncoding, &mimeLineWriter{writer: part})
	_, copyErr := io.Copy(encoder, file)
	encoderErr := encoder.Close()
	fileErr := file.Close()
	if copyErr != nil {
		return fmt.Errorf("lecture de la pièce jointe %q impossible: %w", filename, copyErr)
	}
	if encoderErr != nil {
		return fmt.Errorf("encodage de la pièce jointe %q impossible: %w", filename, encoderErr)
	}
	if fileErr != nil {
		return fmt.Errorf("fermeture de la pièce jointe %q impossible: %w", filename, fileErr)
	}
	return nil
}
