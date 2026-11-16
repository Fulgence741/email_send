package email

import (
	"errors"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	database "email-send-API/src/Database"
	middleware "email-send-API/src/middleware"

	"github.com/gin-gonic/gin"
)

// CreateEmailDTO valide le contenu minimal d'un email à mettre en file d'attente.
type CreateEmailDTO struct {
	Destinataire string `json:"destinataire" binding:"required,email,max=255"`
	Objet        string `json:"objet" binding:"required,max=255"`
	Contenu      string `json:"contenu" binding:"required"`
	TypeContenu  string `json:"type_contenu" binding:"required,oneof=TEXT HTML"`
}

// formulaireCreationEmailDTO représente les champs texte du formulaire d'envoi avec fichiers.
type formulaireCreationEmailDTO struct {
	Destinataire string `form:"destinataire" binding:"required,email,max=255"`
	Objet        string `form:"objet" binding:"required,max=255"`
	Contenu      string `form:"contenu" binding:"required"`
	TypeContenu  string `form:"type_contenu" binding:"required,oneof=TEXT HTML"`
}

// AddEmailHandlers persiste un email appartenant à l'application authentifiée.
func AddEmailHandlers(repository *database.Repository, stockage *StockagePiecesJointes) gin.HandlerFunc {
	return func(c *gin.Context) {
		var dto CreateEmailDTO
		var fichiers []*multipart.FileHeader
		var err error

		// Accepte le JSON existant ou un formulaire multipart contenant les champs et les fichiers.
		if strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
			var formulaire formulaireCreationEmailDTO
			err = c.Request.ParseMultipartForm(1 << 20)
			if err == nil {
				defer c.Request.MultipartForm.RemoveAll()
				err = c.ShouldBind(&formulaire)
			}
			dto = CreateEmailDTO{
				Destinataire: formulaire.Destinataire,
				Objet:        formulaire.Objet,
				Contenu:      formulaire.Contenu,
				TypeContenu:  formulaire.TypeContenu,
			}
			if err == nil {
				if c.Request.MultipartForm == nil {
					err = errors.New("formulaire multipart indisponible")
				} else {
					fichiers = c.Request.MultipartForm.File["pieces_jointes"]
				}
			}
		} else {
			err = c.ShouldBindJSON(&dto)
		}
		if err != nil {
			var limiteCorps *http.MaxBytesError
			if errors.As(err, &limiteCorps) {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{"success": false, "message": "La requête dépasse la taille maximale autorisée."})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Données invalides.",
				"error":   err.Error(),
			})
			return
		}

		// L'application est fournie exclusivement par le middleware d'authentification.
		application := middleware.ApplicationAuthentifiee(c)
		if application == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Contexte d'application authentifiée indisponible.",
			})
			return

		}

		// Les fichiers sont enregistrés avant la transaction, puis supprimés si celle-ci échoue.
		var pieces []database.PieceJointe
		if len(fichiers) > 0 {
			if stockage == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "Le stockage des pièces jointes est indisponible."})
				return
			}
			pieces, err = stockage.Sauvegarder(fichiers)
			if err != nil {
				switch {
				case errors.Is(err, ErrLimitePiecesJointes):
					c.JSON(http.StatusRequestEntityTooLarge, gin.H{"success": false, "message": err.Error()})
				case errors.Is(err, ErrPieceJointeInvalide):
					c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
				default:
					log.Printf("enregistrement des fichiers impossible: %v", err)
					c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Impossible d'enregistrer les pièces jointes."})
				}
				return
			}
		}

		// Les valeurs de reprise sont explicites pour rester compatibles avec le schéma SQL.
		emailRecord := database.Email{
			IDApplication:       application.IDApplication,
			Destinataire:        dto.Destinataire,
			Objet:               dto.Objet,
			Message:             dto.Contenu,
			TypeContenu:         dto.TypeContenu,
			Statut:              "PENDING",
			NombreMaxTentatives: 3,
			DelaiRetrySecondes:  60,
			DateCreation:        time.Now().UTC(),
		}
		// L'email et ses pièces jointes sont validés ensemble afin que le worker ne voie jamais un envoi incomplet.
		var nouvelEmail bool
		if len(pieces) > 0 {
			nouvelEmail, err = repository.CreerEmailAvecPiecesJointes(c.Request.Context(), &emailRecord, pieces)
		} else {
			nouvelEmail, err = repository.CreateEmail(c.Request.Context(), &emailRecord)
		}
		if err != nil {
			if stockage != nil {
				if nettoyageErr := stockage.SupprimerFichiers(pieces); nettoyageErr != nil {
					log.Printf("nettoyage des fichiers après échec impossible: %v", nettoyageErr)
				}
			}
			if errors.Is(err, database.ErrConflict) {
				c.JSON(http.StatusConflict, gin.H{
					"success": false,
					"message": "Cette clé d'idempotence est associée à un autre email.",
				})
				return
			}
			// Si une autre erreur survient, elle est loguée et une réponse 500 Internal Server Error est renvoyée.
			log.Printf("enregistrement de l'email impossible: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Impossible d'enregistrer l'email.",
			})
			return
		}
		if !nouvelEmail && len(pieces) > 0 {
			if nettoyageErr := stockage.SupprimerFichiers(pieces); nettoyageErr != nil {
				log.Printf("nettoyage des fichiers idempotents impossible: %v", nettoyageErr)
			}
		}

		// La réponse indique que le worker prendra en charge l'envoi en arrière-plan.
		c.JSON(http.StatusAccepted, gin.H{
			"success": true,
			"message": "Email enregistré pour traitement.",
			"data": gin.H{
				"id_email": emailRecord.IDEmail,
				"statut":   emailRecord.Statut,
			},
		})
	}
}

const (
	taillePageParDefaut = 20
	taillePageMaximale  = 100
)

// StatutEmailDTO expose le suivi d'un email sans révéler son contenu.
type StatutEmailDTO struct {
	IDEmail           int64      `json:"id_email"`
	Destinataire      string     `json:"destinataire"`
	Objet             string     `json:"objet"`
	TypeContenu       string     `json:"type_contenu"`
	Statut            string     `json:"statut"`
	DateCreation      time.Time  `json:"date_creation"`
	DateFinTraitement *time.Time `json:"date_fin_traitement,omitempty"`
	MessageErreur     *string    `json:"message_erreur,omitempty"`
}

// ListerEmailsHandler liste uniquement les emails de l'application authentifiée.
func ListerEmailsHandler(repository *database.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// L'identité de l'application est ajoutée par le middleware d'authentification.
		application := middleware.ApplicationAuthentifiee(c)
		if application == nil {
			ecrireErreurContexteApplication(c)
			return
		}

		// Valide la pagination avant d'interroger la base de données.
		limite, decalage, err := lirePagination(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
			return
		}

		// Le filtre de statut est limité aux valeurs définies par le schéma.
		statut := strings.ToUpper(strings.TrimSpace(c.Query("status")))
		switch statut {
		case "", "PENDING", "SENDING", "SENT", "FAILED":
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Le statut doit être PENDING, SENDING, SENT ou FAILED.",
			})
			return
		}

		// La requête est toujours limitée à l'application authentifiée.
		emails, total, err := repository.ListerEmails(c.Request.Context(), application.IDApplication, limite, decalage, statut)
		if err != nil {
			log.Printf("liste des emails impossible: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Impossible de lister les emails."})
			return
		}

		// Convertit les modèles en réponses publiques sans inclure le corps du message.
		donnees := make([]StatutEmailDTO, 0, len(emails))
		for _, emailRecord := range emails {
			donnees = append(donnees, convertirStatutEmail(emailRecord))
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    donnees,
			"pagination": gin.H{
				"limit":  limite,
				"offset": decalage,
				"total":  total,
			},
		})
	}
}

// LireEmailHandler consulte le statut d'un email appartenant à l'appelant.
func LireEmailHandler(repository *database.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Vérifie l'identité et l'identifiant avant l'accès aux données.
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

		// La recherche utilise l'identifiant de l'application pour éviter toute fuite inter-clients.
		emailRecord, err := repository.RechercherEmailParID(c.Request.Context(), application.IDApplication, idEmail)
		if database.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Email introuvable."})
			return
		}
		if err != nil {
			log.Printf("récupération de l'email impossible: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Impossible de récupérer l'email."})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": convertirStatutEmail(*emailRecord)})
	}
}

// ListerTentativesEmailHandler renvoie l'historique d'un email appartenant à l'appelant.
func ListerTentativesEmailHandler(repository *database.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Valide l'application et l'identifiant avant de lire l'historique.
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

		// Confirme la propriété de l'email avant de charger ses tentatives.
		if _, err := repository.RechercherEmailParID(c.Request.Context(), application.IDApplication, idEmail); err != nil {
			if database.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Email introuvable."})
				return
			}
			log.Printf("vérification de l'email impossible: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Impossible de récupérer l'email."})
			return
		}

		// Récupère les tentatives et construit le DTO public correspondant.
		tentatives, err := repository.ListerTentativesEmail(c.Request.Context(), idEmail)
		if err != nil {
			log.Printf("historique des tentatives impossible: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Impossible de récupérer les tentatives d'envoi."})
			return
		}
		donnees := make([]ResponseEnvoieDTO, 0, len(tentatives))
		for _, tentative := range tentatives {
			donnees = append(donnees, ResponseEnvoieDTO{
				IDTentative:     tentative.IDTentative,
				NumeroTentative: tentative.NumeroTentative,
				DateTentative:   tentative.DateTentative,
				Statut:          tentative.Statut,
				MessageErreur:   tentative.MessageErreur,
				ReponseServeur:  tentative.ReponseServeur,
			})
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": donnees})
	}
}

// lirePagination applique les bornes acceptées par les paramètres limit et offset.
func lirePagination(c *gin.Context) (int, int, error) {
	limite := taillePageParDefaut
	if valeur := c.Query("limit"); valeur != "" {
		parse, err := strconv.Atoi(valeur)
		if err != nil || parse < 1 || parse > taillePageMaximale {
			return 0, 0, errors.New("limit doit être un entier compris entre 1 et 100")
		}
		limite = parse
	}

	decalage := 0
	if valeur := c.Query("offset"); valeur != "" {
		parse, err := strconv.Atoi(valeur)
		if err != nil || parse < 0 {
			return 0, 0, errors.New("offset doit être un entier positif ou nul")
		}
		decalage = parse
	}
	return limite, decalage, nil
}

// lireIDEmail rejette les identifiants absents, non numériques ou non positifs.
func lireIDEmail(c *gin.Context) (int64, error) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("identifiant invalide")
	}
	return id, nil
}

// ecrireErreurContexteApplication signale un défaut de configuration de la chaîne HTTP.
func ecrireErreurContexteApplication(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"success": false,
		"message": "Contexte d'application authentifiée indisponible.",
	})
}

// convertirStatutEmail limite les champs exposés au client.
func convertirStatutEmail(emailRecord database.Email) StatutEmailDTO {
	return StatutEmailDTO{
		IDEmail:           emailRecord.IDEmail,
		Destinataire:      emailRecord.Destinataire,
		Objet:             emailRecord.Objet,
		TypeContenu:       emailRecord.TypeContenu,
		Statut:            emailRecord.Statut,
		DateCreation:      emailRecord.DateCreation,
		DateFinTraitement: emailRecord.DateFinTraitement,
		MessageErreur:     emailRecord.MessageErreur,
	}
}
