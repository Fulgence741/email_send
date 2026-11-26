package email

import (
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"email-send-API/src/database"
)

func TestBuildMIMEMessageIncludesAttachment(t *testing.T) {
	attachmentContent := []byte(strings.Repeat("x", 160))
	attachmentPath := filepath.Join(t.TempDir(), "document.txt")
	if err := os.WriteFile(attachmentPath, attachmentContent, 0600); err != nil {
		t.Fatal(err)
	}

	message, err := buildMIMEMessage(
		&mail.Address{Address: "sender@example.com"},
		&mail.Address{Address: "recipient@example.com"},
		"Bonjour",
		"Contenu du message",
		"TEXT",
		[]database.PieceJointe{{
			NomFichier:    "document.txt",
			TypeMIME:      "text/plain",
			CheminFichier: attachmentPath,
		}},
	)
	if err != nil {
		t.Fatalf("buildMIMEMessage() error = %v", err)
	}

	headerEnd := strings.Index(string(message), "\r\n\r\n")
	if headerEnd < 0 {
		t.Fatal("message is missing its header separator")
	}
	header, err := mail.ReadMessage(strings.NewReader(string(message)))
	if err != nil {
		t.Fatalf("mail.ReadMessage() error = %v", err)
	}
	mediaType, params, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("invalid message content type: %v", err)
	}
	if mediaType != "multipart/mixed" {
		t.Fatalf("Content-Type = %q, want multipart/mixed", mediaType)
	}

	reader := multipart.NewReader(header.Body, params["boundary"])
	bodyPart, err := reader.NextPart()
	if err != nil {
		t.Fatalf("read body part: %v", err)
	}
	body, err := io.ReadAll(bodyPart)
	if err != nil {
		t.Fatalf("read message body: %v", err)
	}
	if !strings.Contains(string(body), "Contenu du message") {
		t.Fatalf("message body = %q, missing original content", body)
	}

	attachmentPart, err := reader.NextPart()
	if err != nil {
		t.Fatalf("read attachment part: %v", err)
	}
	if got := attachmentPart.FileName(); got != "document.txt" {
		t.Fatalf("attachment filename = %q, want document.txt", got)
	}
	encodedAttachment, err := io.ReadAll(attachmentPart)
	if err != nil {
		t.Fatalf("read attachment: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSuffix(string(encodedAttachment), "\r\n"), "\r\n") {
		if len(line) > 76 {
			t.Fatalf("base64 line length = %d, want at most 76", len(line))
		}
	}
	decodedAttachment, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, strings.NewReader(string(encodedAttachment))))
	if err != nil {
		t.Fatalf("decode attachment: %v", err)
	}
	if string(decodedAttachment) != string(attachmentContent) {
		t.Fatalf("attachment content = %q, want %q", decodedAttachment, attachmentContent)
	}

	if _, err := reader.NextPart(); err != io.EOF {
		t.Fatalf("unexpected extra MIME part, err = %v", err)
	}
}
