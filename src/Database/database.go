// Package database centralise les modèles et les accès PostgreSQL de l'application.
package database

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	setting "email-send-API/src/Setting"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Application représente une application autorisée à utiliser l'API.
// La clé API brute n'est jamais conservée en base : seul son condensat est stocké.
type Application struct {
	IDApplication int64     `gorm:"primaryKey;column:id_application" json:"id_application"`
	Nom           string    `gorm:"column:nom;not null" json:"nom"`
	APIKeyHash    string    `gorm:"column:api_key_hash;not null" json:"-"`
	Statut        string    `gorm:"column:statut;not null" json:"statut"`
	DateCreation  time.Time `gorm:"column:date_creation;not null" json:"date_creation"`
}

// TableName fixe le nom SQL au lieu de laisser GORM le déduire.
func (Application) TableName() string {
	return "applications"
}

// Email représente un message persistant en attente de traitement.
type Email struct {
	IDEmail       int64      `gorm:"primaryKey;column:id_email" json:"id_email"`
	IDApplication int64      `gorm:"column:id_application;not null" json:"id_application"`
	Destinataire  string     `gorm:"column:destinataire;not null" json:"destinataire"`
	Objet         string     `gorm:"column:objet;not null" json:"objet"`
	Contenu       string     `gorm:"column:contenu;not null" json:"contenu"`
	TypeContenu   string     `gorm:"column:type_contenu;not null" json:"type_contenu"`
	Statut        string     `gorm:"column:statut;not null" json:"statut"`
	DateCreation  time.Time  `gorm:"column:date_creation;not null" json:"date_creation"`
	DateEnvoi     *time.Time `gorm:"column:date_envoi" json:"date_envoi,omitempty"`

	DateDebutTraitement    *time.Time `gorm:"column:date_debut_traitement" json:"-"`
	DateProchaineTentative *time.Time `gorm:"column:date_prochaine_tentative" json:"-"`

	MessageErreur *string `gorm:"column:message_erreur" json:"message_erreur,omitempty"`

	// Colonne message existante dans PostgreSQL
	Message *string `gorm:"column:message" json:"message,omitempty"`
}

func (Email) TableName() string {
	return "emails"
}

// PieceJointe décrit un fichier associé à un email.
type PieceJointe struct {
	IDPieceJointe int64     `gorm:"primaryKey;column:id_piece_jointe" json:"id_piece_jointe"`
	IDEmail       int64     `gorm:"column:id_email;not null" json:"id_email"`
	NomFichier    string    `gorm:"column:nom_fichier;not null" json:"nom_fichier"`
	TypeMIME      string    `gorm:"column:type_mime;not null" json:"type_mime"`
	CheminFichier string    `gorm:"column:chemin_fichier;not null" json:"chemin_fichier"`
	TailleFichier int64     `gorm:"column:taille_fichier;not null" json:"taille_fichier"`
	DateCreation  time.Time `gorm:"column:date_creation;not null" json:"date_creation"`
}

// TableName retourne le nom SQL défini par les migrations.
func (PieceJointe) TableName() string {
	return "pieces_jointes"
}

// Envoie représente une tentative de livraison d'un email.
type Envoie struct {
	IDTentative     int64     `gorm:"primaryKey;column:id_tentative"`
	IDEmail         int64     `gorm:"column:id_email;not null" json:"id_email"`
	NumeroTentative int       `gorm:"column:numero_tentative;not null" json:"numero_tentative"`
	DateTentative   time.Time `gorm:"column:date_tentative;not null" json:"date_tentative"`
	Statut          string    `gorm:"column:statut;not null" json:"statut"`
	MessageErreur   *string   `gorm:"column:message_erreur" json:"message_erreur,omitempty"`
	ReponseServeur  *string   `gorm:"column:reponse_serveur" json:"reponse_serveur,omitempty"`
}

// TableName retourne le nom SQL défini par les migrations.
func (Envoie) TableName() string {
	return "tentatives_envoi"
}

var ErrConflict = errors.New("conflit de données") // ErrConflict est utilisé pour signaler une violation de contrainte d'unicité.

// ErrAucunEmailEnAttente indique que la file ne contient aucun email prêt à être envoyé.
var ErrAucunEmailEnAttente = errors.New("aucun email en attente")

// Repository regroupe les opérations de persistance utilisées par l'API.
type Repository struct {
	db *gorm.DB
}

// Open établit la connexion GORM à PostgreSQL et vérifie qu'elle répond.
func Open(cfg *setting.Config) (*Repository, error) {
	if cfg == nil {
		return nil, errors.New("configuration de la base de données absente")
	}
	// Vérifie que les paramètres essentiels sont présents et valides.
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "DB_HOST", value: cfg.DBHost},
		{name: "DB_USER", value: cfg.DBUser},
		{name: "DB_NAME", value: cfg.DBName},
	} {
		// Si un champ essentiel est vide, renvoie une erreur descriptive.
		if field.value == "" {
			return nil, fmt.Errorf("%s est obligatoire", field.name)
		}
	}
	// Vérifie que le port est dans la plage valide pour TCP.
	if cfg.DBPort < 1 || cfg.DBPort > 65535 { // Si le port est hors de la plage valide, renvoie une erreur.
		return nil, errors.New("DB_PORT doit être compris entre 1 et 65535")
	}

	// Construit l'URL de connexion PostgreSQL à partir des paramètres de configuration.
	dsnURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.DBUser, cfg.DBPassword),
		Host:   net.JoinHostPort(cfg.DBHost, strconv.Itoa(cfg.DBPort)),
		Path:   "/" + cfg.DBName,
	}
	// Ajoute le paramètre sslmode à l'URL de connexion pour contrôler la sécurité SSL.
	query := dsnURL.Query()
	query.Set("sslmode", cfg.DBSSLMode)
	dsnURL.RawQuery = query.Encode()

	// Ouvre la connexion GORM à PostgreSQL en utilisant l'URL construite.
	db, err := gorm.Open(postgres.Open(dsnURL.String()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("ouvrir la connexion PostgreSQL: %w", err)
	}
	// Récupère l'objet sql.DB sous-jacent pour effectuer des opérations de bas niveau.
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("récupérer la connexion PostgreSQL: %w", err)
	}
	// Définit un délai d'attente pour le ping de la base de données afin de vérifier la connectivité.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Effectue un ping à la base de données pour s'assurer qu'elle est accessible et répond correctement.
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("vérifier la connexion PostgreSQL: %w", err)
	}

	return &Repository{db: db}, nil
}

// Close libère la connexion SQL détenue par GORM.
func (r *Repository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return fmt.Errorf("récupérer la connexion PostgreSQL: %w", err)
	}
	// Ferme la connexion SQL et renvoie toute erreur rencontrée.
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("fermer la connexion PostgreSQL: %w", err)
	}
	return nil
}

// VerifierConnexion confirme que PostgreSQL répond.
func (r *Repository) VerifierConnexion(ctx context.Context) error {
	baseSQL, err := r.db.DB()
	if err != nil {
		return fmt.Errorf("récupérer la connexion PostgreSQL: %w", err)
	}
	if err := baseSQL.PingContext(ctx); err != nil {
		return fmt.Errorf("vérifier la connexion PostgreSQL: %w", err)
	}
	return nil
}

// CreateApplication crée une application et ne renvoie sa clé API qu'une seule fois.
func (r *Repository) CreateApplication(ctx context.Context, name string) (*Application, string, error) {
	keyBytes := make([]byte, 32)
	// Génère 32 octets aléatoires pour la clé API. Si la génération échoue, renvoie une erreur.
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, "", fmt.Errorf("générer la clé API: %w", err)
	}
	// Encode la clé API en base64 URL-safe et calcule son condensat SHA-256 pour le stockage sécurisé.
	apiKey := base64.RawURLEncoding.EncodeToString(keyBytes)
	keyHash := sha256.Sum256([]byte(apiKey))
	// Crée un nouvel enregistrement Application avec le nom fourni, le condensat de la clé API, le statut actif et la date de création actuelle.
	application := Application{
		Nom:          name,
		APIKeyHash:   hex.EncodeToString(keyHash[:]),
		Statut:       "ACTIF",
		DateCreation: time.Now().UTC(),
	}

	// Tente d'insérer l'application dans la base de données. Si une violation de contrainte d'unicité se produit, renvoie ErrConflict.
	if err := r.db.WithContext(ctx).Create(&application).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, "", fmt.Errorf("%w: l'application existe déjà", ErrConflict)
		}
		// Si une autre erreur survient lors de l'insertion, renvoie une erreur descriptive.
		return nil, "", fmt.Errorf("créer l'application: %w", err)
	}
	//
	return &application, apiKey, nil
}

// FindApplicationByAPIKey compare le condensat de la clé et ignore les applications inactives.
func (r *Repository) FindApplicationByAPIKey(ctx context.Context, apiKey string) (*Application, error) {
	keyHash := sha256.Sum256([]byte(apiKey))
	// Rechercher l'application dans la base de données en comparant le condensat de la clé API et en filtrant par statut actif.
	var application Application
	err := r.db.WithContext(ctx).
		// Utilise la clause WHERE pour filtrer les applications par le condensat de la clé API et le statut "ACTIF".
		Where("api_key_hash = ? AND statut = ?", hex.EncodeToString(keyHash[:]), "ACTIF").
		First(&application).Error
	if err != nil {
		return nil, err
	}
	// Si l'application est trouvée, renvoie un pointeur vers l'objet Application et nil pour l'erreur.
	return &application, nil
}

// CreateEmail enregistre un email ou reconnaît une nouvelle tentative idempotente.
// Le booléen indique si une nouvelle ligne a été créée.
func (r *Repository) CreateEmail(ctx context.Context, email *Email) (bool, error) {
	// Tente d'insérer l'email dans la base de données. Si une violation de contrainte d'unicité se produit,
	//  vérifie si les détails correspondent à un email existant.
	if err := r.db.WithContext(ctx).Create(email).Error; err != nil {
		if !isUniqueViolation(err) {
			return false, fmt.Errorf("enregistrer l'email: %w", err)
		}
		// Si une violation de contrainte d'unicité se produit, recherche l'email existant avec la même clé d'idempotence.
		var existing Email
		lookupErr := r.db.WithContext(ctx).
			Where("id_application = ? AND cle_idempotence = ?", email.IDApplication, email.CleIdempotence).
			First(&existing).Error
		// Si une erreur survient lors de la recherche de l'email existant, renvoie une erreur descriptive.
		if lookupErr != nil {
			return false, fmt.Errorf("retrouver l'email déjà enregistré: %w", lookupErr)
		}
		// Si les détails de l'email existant ne correspondent pas à ceux de l'email en cours d'enregistrement, renvoie ErrConflict.
		if existing.Destinataire != email.Destinataire ||
			existing.Objet != email.Objet ||
			existing.Message != email.Message ||
			existing.TypeContenu != email.TypeContenu {
			return false, fmt.Errorf("%w: cette clé d'idempotence est associée à un autre email", ErrConflict)
		}
		// Si les détails correspondent, met à jour l'objet email avec l'ID et le statut de l'email existant.
		*email = existing
		return false, nil
	}
	return true, nil
}

// CreerEmailAvecPiecesJointes enregistre l'email et ses fichiers dans une transaction.
func (r *Repository) CreerEmailAvecPiecesJointes(ctx context.Context, email *Email, pieces []PieceJointe) (bool, error) {
	nouvelEmail := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		resultat := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id_application"}, {Name: "cle_idempotence"}},
			DoNothing: true,
		}).Create(email)
		if resultat.Error != nil {
			return fmt.Errorf("enregistrer l'email: %w", resultat.Error)
		}
		if resultat.RowsAffected == 0 {
			// Une répétition idempotente renvoie l'email existant sans interrompre la transaction.
			var existant Email
			if lookupErr := tx.Where("id_application = ? AND cle_idempotence = ?", email.IDApplication, email.CleIdempotence).
				First(&existant).Error; lookupErr != nil {
				return fmt.Errorf("retrouver l'email existant: %w", lookupErr)
			}
			if existant.Destinataire != email.Destinataire ||
				existant.Objet != email.Objet ||
				existant.Message != email.Message ||
				existant.TypeContenu != email.TypeContenu {
				return fmt.Errorf("%w: cette clé d'idempotence est associée à un autre email", ErrConflict)
			}
			var piecesExistantes []PieceJointe
			if err := tx.Where("id_email = ?", existant.IDEmail).
				Order("id_piece_jointe ASC").
				Find(&piecesExistantes).Error; err != nil {
				return fmt.Errorf("comparer les pièces jointes existantes: %w", err)
			}
			if len(piecesExistantes) != len(pieces) {
				return fmt.Errorf("%w: les pièces jointes ne correspondent pas à la demande initiale", ErrConflict)
			}
			for index, piece := range pieces {
				pieceExistante := piecesExistantes[index]
				if pieceExistante.NomFichier != piece.NomFichier ||
					pieceExistante.TypeMIME != piece.TypeMIME ||
					pieceExistante.TailleFichier != piece.TailleFichier {
					return fmt.Errorf("%w: les pièces jointes ne correspondent pas à la demande initiale", ErrConflict)
				}
			}
			*email = existant
			return nil
		}

		// Les pièces jointes ne deviennent visibles qu'après l'insertion de leur email parent.
		for index := range pieces {
			pieces[index].IDEmail = email.IDEmail
			if err := tx.Create(&pieces[index]).Error; err != nil {
				return fmt.Errorf("enregistrer la pièce jointe: %w", err)
			}
		}
		nouvelEmail = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return nouvelEmail, nil
}

// ListerPiecesJointes retourne les métadonnées des fichiers liés à un email.
func (r *Repository) ListerPiecesJointes(ctx context.Context, idEmail int64) ([]PieceJointe, error) {
	var pieces []PieceJointe
	if err := r.db.WithContext(ctx).
		Where("id_email = ?", idEmail).
		Order("id_piece_jointe ASC").
		Find(&pieces).Error; err != nil {
		return nil, fmt.Errorf("lister les pièces jointes: %w", err)
	}
	return pieces, nil
}

// RechercherPieceJointe retrouve un fichier sans sortir du périmètre de l'email.
func (r *Repository) RechercherPieceJointe(ctx context.Context, idEmail, idPiece int64) (*PieceJointe, error) {
	var piece PieceJointe
	if err := r.db.WithContext(ctx).
		Where("id_email = ? AND id_piece_jointe = ?", idEmail, idPiece).
		First(&piece).Error; err != nil {
		return nil, err
	}
	return &piece, nil
}

// ReclamerProchainEmail réserve atomiquement le prochain email éligible à l'envoi.
func (r *Repository) ReclamerProchainEmail(ctx context.Context) (*Email, []PieceJointe, *Envoie, error) {
	var email Email
	var pieces []PieceJointe
	var tentative Envoie
	emailEpuise := false

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		maintenant := time.Now().UTC()
		resultat := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("(statut = ? AND (date_prochaine_tentative IS NULL OR date_prochaine_tentative <= ?)) OR (statut = ? AND date_debut_traitement < ?)",
				"PENDING", maintenant, "SENDING", maintenant.Add(-time.Minute)).
			Order("date_creation ASC, id_email ASC").
			Limit(1).
			Find(&email)
		if resultat.Error != nil {
			return fmt.Errorf("rechercher un email en attente: %w", resultat.Error)
		}
		if resultat.RowsAffected == 0 {
			return ErrAucunEmailEnAttente
		}
		if email.Statut == "SENDING" {
			erreurInterruption := "Traitement interrompu avant la fin de l'envoi."
			if err := tx.Model(&Envoie{}).
				Where("id_email = ? AND statut = ?", email.IDEmail, "EN_COURS").
				Updates(map[string]interface{}{
					"statut":         "ECHEC",
					"message_erreur": erreurInterruption,
				}).Error; err != nil {
				return fmt.Errorf("clôturer la tentative interrompue: %w", err)
			}
		}

		var derniereTentative int
		if err := tx.Model(&Envoie{}).
			Where("id_email = ?", email.IDEmail).
			Select("COALESCE(MAX(numero_tentative), 0)").
			Scan(&derniereTentative).Error; err != nil {
			return fmt.Errorf("compter les tentatives d'envoi: %w", err)
		}
		if derniereTentative >= email.NombreMaxTentatives {
			maintenant = time.Now().UTC()
			if err := tx.Model(&Email{}).Where("id_email = ?", email.IDEmail).Updates(map[string]interface{}{
				"statut":                   "FAILED",
				"date_fin_traitement":      maintenant,
				"date_prochaine_tentative": nil,
				"message_erreur":           "Nombre maximal de tentatives atteint.",
			}).Error; err != nil {
				return fmt.Errorf("clôturer l'email sans tentative restante: %w", err)
			}
			emailEpuise = true
			return nil
		}

		maintenant = time.Now().UTC()
		if err := tx.Model(&Email{}).Where("id_email = ?", email.IDEmail).Updates(map[string]interface{}{
			"statut":                   "SENDING",
			"date_debut_traitement":    maintenant,
			"date_fin_traitement":      nil,
			"date_prochaine_tentative": nil,
		}).Error; err != nil {
			return fmt.Errorf("réserver l'email pour traitement: %w", err)
		}
		email.Statut = "SENDING"
		email.DateDebutTraitement = &maintenant
		tentative = Envoie{
			IDEmail:         email.IDEmail,
			NumeroTentative: derniereTentative + 1,
			DateTentative:   maintenant,
			Statut:          "EN_COURS",
		}
		if err := tx.Create(&tentative).Error; err != nil {
			return fmt.Errorf("créer la tentative d'envoi: %w", err)
		}
		if err := tx.Where("id_email = ?", email.IDEmail).Order("id_piece_jointe ASC").Find(&pieces).Error; err != nil {
			return fmt.Errorf("charger les pièces jointes: %w", err)
		}
		return nil
	})
	if errors.Is(err, ErrAucunEmailEnAttente) || (err == nil && emailEpuise) {
		return nil, nil, nil, ErrAucunEmailEnAttente
	}
	if err != nil {
		return nil, nil, nil, err
	}
	return &email, pieces, &tentative, nil
}

// FinaliserTentative met à jour la tentative et planifie une reprise si nécessaire.
func (r *Repository) FinaliserTentative(ctx context.Context, idEmail, idTentative int64, erreurEnvoi error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var email Email
		if err := tx.First(&email, "id_email = ?", idEmail).Error; err != nil {
			return fmt.Errorf("retrouver l'email à finaliser: %w", err)
		}
		var tentative Envoie
		if err := tx.First(&tentative, "id_tentative = ? AND id_email = ?", idTentative, idEmail).Error; err != nil {
			return fmt.Errorf("retrouver la tentative à finaliser: %w", err)
		}

		maintenant := time.Now().UTC()
		if erreurEnvoi == nil {
			if err := tx.Model(&Envoie{}).Where("id_tentative = ?", idTentative).Updates(map[string]interface{}{
				"statut": "REUSSIE",
			}).Error; err != nil {
				return fmt.Errorf("clôturer la tentative réussie: %w", err)
			}
			if err := tx.Model(&Email{}).Where("id_email = ?", idEmail).Updates(map[string]interface{}{
				"statut":                   "SENT",
				"date_fin_traitement":      maintenant,
				"date_prochaine_tentative": nil,
				"message_erreur":           nil,
			}).Error; err != nil {
				return fmt.Errorf("marquer l'email comme envoyé: %w", err)
			}
			return nil
		}

		messageErreur := erreurEnvoi.Error()
		if err := tx.Model(&Envoie{}).Where("id_tentative = ?", idTentative).Updates(map[string]interface{}{
			"statut":         "ECHEC",
			"message_erreur": messageErreur,
		}).Error; err != nil {
			return fmt.Errorf("enregistrer l'échec de la tentative: %w", err)
		}
		if tentative.NumeroTentative >= email.NombreMaxTentatives {
			if err := tx.Model(&Email{}).Where("id_email = ?", idEmail).Updates(map[string]interface{}{
				"statut":                   "FAILED",
				"date_fin_traitement":      maintenant,
				"date_prochaine_tentative": nil,
				"message_erreur":           messageErreur,
			}).Error; err != nil {
				return fmt.Errorf("marquer l'email comme échoué: %w", err)
			}
			return nil
		}

		prochaineTentative := maintenant.Add(time.Duration(email.DelaiRetrySecondes) * time.Second)
		if err := tx.Model(&Email{}).Where("id_email = ?", idEmail).Updates(map[string]interface{}{
			"statut":                   "PENDING",
			"date_fin_traitement":      nil,
			"date_prochaine_tentative": prochaineTentative,
			"message_erreur":           messageErreur,
		}).Error; err != nil {
			return fmt.Errorf("planifier la prochaine tentative: %w", err)
		}
		return nil
	})
}

// ListerEmails retourne les emails d'une application du plus récent au plus ancien.
func (r *Repository) ListerEmails(ctx context.Context, idApplication int64, limite, decalage int, statut string) ([]Email, int64, error) {
	requete := r.db.WithContext(ctx).Model(&Email{}).Where("id_application = ?", idApplication)
	if statut != "" {
		requete = requete.Where("statut = ?", statut)
	}

	var total int64
	if err := requete.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("compter les emails: %w", err)
	}

	var emails []Email
	if err := requete.Order("date_creation DESC, id_email DESC").
		Limit(limite).
		Offset(decalage).
		Find(&emails).Error; err != nil {
		return nil, 0, fmt.Errorf("lister les emails: %w", err)
	}
	return emails, total, nil
}

// RechercherEmailParID respecte la propriété de l'application lors de la recherche.
func (r *Repository) RechercherEmailParID(ctx context.Context, idApplication, idEmail int64) (*Email, error) {
	var email Email
	if err := r.db.WithContext(ctx).
		Where("id_application = ? AND id_email = ?", idApplication, idEmail).
		First(&email).Error; err != nil {
		return nil, err
	}
	return &email, nil
}

// ListerTentativesEmail retourne les tentatives dans l'ordre chronologique.
func (r *Repository) ListerTentativesEmail(ctx context.Context, idEmail int64) ([]Envoie, error) {
	var tentatives []Envoie
	if err := r.db.WithContext(ctx).
		Where("id_email = ?", idEmail).
		Order("numero_tentative ASC").
		Find(&tentatives).Error; err != nil {
		return nil, fmt.Errorf("lister les tentatives d'envoi: %w", err)
	}
	return tentatives, nil
}

// IsNotFound indique si la recherche GORM n'a trouvé aucun enregistrement.
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// isUniqueViolation identifie une erreur de contrainte d'unicité PostgreSQL.
func isUniqueViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "23505"
}
