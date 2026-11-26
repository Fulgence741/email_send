package email

import (
	"encoding/json"
	"strings"
	"testing"

	"email-send-API/src/database"
)

func TestPieceJointeResponsesDoesNotExposeStoragePath(t *testing.T) {
	responses := pieceJointeResponses([]database.PieceJointe{{
		IDPieceJointe: 12,
		NomFichier:    "rapport.pdf",
		TypeMIME:      "application/pdf",
		CheminFichier: "storage/uploads/private-file.pdf",
		TailleFichier: 256,
	}})

	encoded, err := json.Marshal(responses)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "private-file.pdf") || strings.Contains(string(encoded), "chemin_fichier") {
		t.Fatalf("attachment response exposes its storage path: %s", encoded)
	}
}
