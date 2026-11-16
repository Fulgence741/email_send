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
	IDEmail           int64      `gorm:"primaryKey;column:id_email" json:"id_email"`
	IDApplication     int64      `gorm:"column:id_application;not null" json:"id_application"`
	Destinataire      string     `gorm:"column:destinataire;not null" json:"destinataire"`
	Objet             string     `gorm:"column:objet;not null" json:"objet"`
	Message           string     `gorm:"column:message;not null" json:"message"`
	TypeContenu       string     `gorm:"column:type_contenu;not null" json:"type_contenu"`
	Statut            string     `gorm:"column:statut;not null" json:"statut"`
	CleIdempotence    string     `gorm:"column:cle_idempotence;not null" json:"-"`
	DateCreation      time.Time  `gorm:"column:date_creation;not null" json:"date_creation"`
	DateFinTraitement *time.Time `gorm:"column:date_fin_traitement" json:"date_fin_traitement,omitempty"`
	MessageErreur     *string    `gorm:"column:message_erreur" json:"message_erreur,omitempty"`
}

// TableName associe Email à la table existante de la migration PostgreSQL.
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

// IsNotFound indique si la recherche GORM n'a trouvé aucun enregistrement.
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// isUniqueViolation identifie une erreur de contrainte d'unicité PostgreSQL.
func isUniqueViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "23505"
}
