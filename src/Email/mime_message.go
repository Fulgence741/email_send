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

type mimeLineWriter struct {
	writer io.Writer
	column int
}

func (w *mimeLineWriter) Write(data []byte) (int, error) {
	total := 0
	for len(data) > 0 {
		if w.column == 76 {
			n, err := io.WriteString(w.writer, "\r\n")
			if err != nil {
				return total, err
			}
			if n != 2 {
				return total, io.ErrShortWrite
			}
			w.column = 0
		}

		size := 76 - w.column
		if size > len(data) {
			size = len(data)
		}
		n, err := w.writer.Write(data[:size])
		total += n
		w.column += n
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

func buildMIMEMessage(from, recipient *mail.Address, subject, body, contentType string, attachments []database.PieceJointe) ([]byte, error) {
	bodyType := "text/plain"
	if contentType == "HTML" {
		bodyType = "text/html"
	}

	var message bytes.Buffer
	message.WriteString("From: " + from.String() + "\r\n")
	message.WriteString("To: " + recipient.String() + "\r\n")
	message.WriteString("Subject: " + mime.QEncoding.Encode("UTF-8", subject) + "\r\n")
	message.WriteString("MIME-Version: 1.0\r\n")

	if len(attachments) == 0 {
		message.WriteString("Content-Type: " + bodyType + "; charset=UTF-8\r\n")
		message.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		writer := quotedprintable.NewWriter(&message)
		if _, err := io.WriteString(writer, body); err != nil {
			return nil, err
		}
		if err := writer.Close(); err != nil {
			return nil, err
		}
		return message.Bytes(), nil
	}

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

	for _, attachment := range attachments {
		if err := writeAttachment(multipartWriter, attachment); err != nil {
			return nil, err
		}
	}

	if err := multipartWriter.Close(); err != nil {
		return nil, err
	}
	return message.Bytes(), nil
}

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
