package email

import (
	database "email-send-API/src/Database"
	middleware "email-send-API/src/middleware"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// PieceJointeResponseDTO expose les métadonnées publiques d'une pièce jointe.
type PieceJointeResponseDTO struct {
	IDPieceJointe int64  `json:"id_piece_jointe"`
	NomFichier    string `json:"nom_fichier"`
	TypeMIME      string `json:"type_mime"`
	TailleFichier int64  `json:"taille_fichier"`
}

const (
	nombrePiecesJointesMax = 5
	taillePieceJointeMax   = 10 << 20
	tailleTotalePiecesMax  = 20 << 20
)

var (
	// ErrPieceJointeInvalide signale un nom de fichier refusé par la validation.
	ErrPieceJointeInvalide = errors.New("pièce jointe invalide")
	// ErrLimitePiecesJointes signale une taille ou un nombre de fichiers excessif.
	ErrLimitePiecesJointes = errors.New("limite des pièces jointes dépassée")
)

// StockagePiecesJointes écrit les fichiers sur disque avant de les associer en base.
type StockagePiecesJointes struct {
	repertoire string
}

// NouveauStockagePiecesJointes crée le répertoire où les fichiers seront conservés.
func NouveauStockagePiecesJointes(repertoire string) (*StockagePiecesJointes, error) {
	if strings.TrimSpace(repertoire) == "" {
		return nil, errors.New("le répertoire des pièces jointes est obligatoire")
	}
	repertoireAbsolu, err := filepath.Abs(repertoire)
	if err != nil {
		return nil, fmt.Errorf("résoudre le répertoire des pièces jointes: %w", err)
	}
	if err := os.MkdirAll(repertoireAbsolu, 0700); err != nil {
		return nil, fmt.Errorf("créer le répertoire des pièces jointes: %w", err)
	}
	return &StockagePiecesJointes{repertoire: repertoireAbsolu}, nil
}

// Sauvegarder écrit les fichiers avec des noms générés et vérifie les limites avant la persistance.
func (s *StockagePiecesJointes) Sauvegarder(fichiers []*multipart.FileHeader) ([]database.PieceJointe, error) {
	if len(fichiers) > nombrePiecesJointesMax {
		return nil, fmt.Errorf("%w: le nombre de fichiers ne peut pas dépasser %d", ErrLimitePiecesJointes, nombrePiecesJointesMax)
	}
	pieces := make([]database.PieceJointe, 0, len(fichiers))
	var tailleTotale int64
	for _, fichierFormulaire := range fichiers {
		if fichierFormulaire.Size > taillePieceJointeMax {
			s.supprimerFichiers(pieces)
			return nil, fmt.Errorf("%w: chaque fichier doit faire au maximum %d Mio", ErrLimitePiecesJointes, taillePieceJointeMax>>20)
		}
		tailleTotale += fichierFormulaire.Size
		if tailleTotale > tailleTotalePiecesMax {
			s.supprimerFichiers(pieces)
			return nil, fmt.Errorf("%w: la taille totale ne peut pas dépasser %d Mio", ErrLimitePiecesJointes, tailleTotalePiecesMax>>20)
		}

		nomFichier, err := nomFichierSecurise(fichierFormulaire.Filename)
		if err != nil {
			s.supprimerFichiers(pieces)
			return nil, err
		}
		source, err := fichierFormulaire.Open()
		if err != nil {
			s.supprimerFichiers(pieces)
			return nil, fmt.Errorf("ouvrir le fichier %q: %w", nomFichier, err)
		}
		destination, err := os.CreateTemp(s.repertoire, "piece-*")
		if err != nil {
			_ = source.Close()
			s.supprimerFichiers(pieces)
			return nil, fmt.Errorf("créer le fichier temporaire pour %q: %w", nomFichier, err)
		}

		taille, copieErr := io.Copy(destination, io.LimitReader(source, taillePieceJointeMax+1))
		fermetureSourceErr := source.Close()
		fermetureDestinationErr := destination.Close()
		if copieErr != nil || fermetureSourceErr != nil || fermetureDestinationErr != nil {
			_ = os.Remove(destination.Name())
			s.supprimerFichiers(pieces)
			return nil, errors.Join(copieErr, fermetureSourceErr, fermetureDestinationErr)
		}
		if taille > taillePieceJointeMax {
			_ = os.Remove(destination.Name())
			s.supprimerFichiers(pieces)
			return nil, fmt.Errorf("%w: chaque fichier doit faire au maximum %d Mio", ErrLimitePiecesJointes, taillePieceJointeMax>>20)
		}
		tailleTotale += taille - fichierFormulaire.Size
		if tailleTotale > tailleTotalePiecesMax {
			_ = os.Remove(destination.Name())
			s.supprimerFichiers(pieces)
			return nil, fmt.Errorf("%w: la taille totale ne peut pas dépasser %d Mio", ErrLimitePiecesJointes, tailleTotalePiecesMax>>20)
		}

		typeMIME, err := detecterTypeMIME(destination.Name())
		if err != nil {
			_ = os.Remove(destination.Name())
			s.supprimerFichiers(pieces)
			return nil, err
		}
		pieces = append(pieces, database.PieceJointe{
			NomFichier:    nomFichier,
			TypeMIME:      typeMIME,
			CheminFichier: destination.Name(),
			TailleFichier: taille,
		})
	}
	return pieces, nil
}

// SupprimerFichiers retire les fichiers temporaires si l'enregistrement en base échoue.
func (s *StockagePiecesJointes) SupprimerFichiers(pieces []database.PieceJointe) error {
	var erreurs []error
	for _, piece := range pieces {
		if err := os.Remove(piece.CheminFichier); err != nil && !errors.Is(err, os.ErrNotExist) {
			erreurs = append(erreurs, fmt.Errorf("supprimer %q: %w", piece.NomFichier, err))
		}
	}
	return errors.Join(erreurs...)
}

func (s *StockagePiecesJointes) supprimerFichiers(pieces []database.PieceJointe) {
	if err := s.SupprimerFichiers(pieces); err != nil {
		log.Printf("nettoyage des pièces jointes impossible: %v", err)
	}
}

// nomFichierSecurise conserve le nom d'affichage sans accepter de chemin ou de contrôle.
func nomFichierSecurise(nom string) (string, error) {
	nom = path.Base(strings.ReplaceAll(nom, "\\", "/"))
	nom = strings.TrimSpace(nom)
	if nom == "" || nom == "." || len(nom) > 255 {
		return "", fmt.Errorf("%w: nom de fichier invalide", ErrPieceJointeInvalide)
	}
	for _, caractere := range nom {
		if caractere < 32 || caractere == 127 {
			return "", fmt.Errorf("%w: le nom contient un caractère interdit", ErrPieceJointeInvalide)
		}
	}
	return nom, nil
}

// detecterTypeMIME déduit le type du contenu réel au lieu de faire confiance au navigateur.
func detecterTypeMIME(chemin string) (string, error) {
	fichier, err := os.Open(chemin)
	if err != nil {
		return "", fmt.Errorf("ouvrir le fichier pour détecter son type: %w", err)
	}
	defer fichier.Close()
	buffer := make([]byte, 512)
	nombre, err := fichier.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("lire le fichier pour détecter son type: %w", err)
	}
	return http.DetectContentType(buffer[:nombre]), nil
}

// ListerPiecesJointesHandler renvoie les métadonnées des fichiers d'un email appartenant à l'appelant.
func ListerPiecesJointesHandler(repository *database.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		application := middleware.ApplicationAuthentifiee(c)
		idEmail, err := lireIDEmail(c)
		if application == nil {
			ecrireErreurContexteApplication(c)
			return
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Identifiant d'email invalide."})
			return
		}
		if _, err := repository.RechercherEmailParID(c.Request.Context(), application.IDApplication, idEmail); err != nil {
			if database.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Email introuvable."})
				return
			}
			log.Printf("vérification de l'email impossible: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Impossible de récupérer l'email."})
			return
		}

		pieces, err := repository.ListerPiecesJointes(c.Request.Context(), idEmail)
		if err != nil {
			log.Printf("liste des pièces jointes impossible: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Impossible de lister les pièces jointes."})
			return
		}
		donnees := make([]PieceJointeResponseDTO, 0, len(pieces))
		for _, piece := range pieces {
			donnees = append(donnees, PieceJointeResponseDTO{
				IDPieceJointe: piece.IDPieceJointe,
				NomFichier:    piece.NomFichier,
				TypeMIME:      piece.TypeMIME,
				TailleFichier: piece.TailleFichier,
			})
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": donnees})
	}
}

// TelechargerPieceJointeHandler sert uniquement un fichier appartenant à l'application authentifiée.
func TelechargerPieceJointeHandler(repository *database.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		application := middleware.ApplicationAuthentifiee(c)
		idEmail, errEmail := lireIDEmail(c)
		idPiece, errPiece := strconv.ParseInt(c.Param("piece_id"), 10, 64)
		if application == nil {
			ecrireErreurContexteApplication(c)
			return
		}
		if errEmail != nil || errPiece != nil || idPiece < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Identifiant invalide."})
			return
		}
		if _, err := repository.RechercherEmailParID(c.Request.Context(), application.IDApplication, idEmail); err != nil {
			if database.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Email introuvable."})
				return
			}
			log.Printf("vérification de l'email impossible: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Impossible de récupérer l'email."})
			return
		}
		piece, err := repository.RechercherPieceJointe(c.Request.Context(), idEmail, idPiece)
		if database.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Pièce jointe introuvable."})
			return
		}
		if err != nil {
			log.Printf("récupération de la pièce jointe impossible: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Impossible de récupérer la pièce jointe."})
			return
		}
		c.Header("Content-Type", piece.TypeMIME)
		c.FileAttachment(piece.CheminFichier, piece.NomFichier)
	}
}
