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

// ErrConflict signale une collision de nom ou une clé d'idempotence réutilisée
// avec un contenu différent.
var ErrConflict = errors.New("conflit de données")

// Repository regroupe les opérations de persistance utilisées par l'API.
type Repository struct {
	db *gorm.DB
}

// Open établit la connexion GORM à PostgreSQL et vérifie qu'elle répond.
func Open(cfg *setting.Config) (*Repository, error) {
	if cfg == nil {
		return nil, errors.New("configuration de la base de données absente")
	}
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "DB_HOST", value: cfg.DBHost},
		{name: "DB_USER", value: cfg.DBUser},
		{name: "DB_NAME", value: cfg.DBName},
	} {
		if field.value == "" {
			return nil, fmt.Errorf("%s est obligatoire", field.name)
		}
	}
	if cfg.DBPort < 1 || cfg.DBPort > 65535 {
		return nil, errors.New("DB_PORT doit être compris entre 1 et 65535")
	}

	dsnURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.DBUser, cfg.DBPassword),
		Host:   net.JoinHostPort(cfg.DBHost, strconv.Itoa(cfg.DBPort)),
		Path:   "/" + cfg.DBName,
	}
	query := dsnURL.Query()
	query.Set("sslmode", cfg.DBSSLMode)
	dsnURL.RawQuery = query.Encode()

	db, err := gorm.Open(postgres.Open(dsnURL.String()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("ouvrir la connexion PostgreSQL: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("récupérer la connexion PostgreSQL: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("fermer la connexion PostgreSQL: %w", err)
	}
	return nil
}

// CreateApplication crée une application et ne renvoie sa clé API qu'une seule fois.
func (r *Repository) CreateApplication(ctx context.Context, name string) (*Application, string, error) {
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, "", fmt.Errorf("générer la clé API: %w", err)
	}

	apiKey := base64.RawURLEncoding.EncodeToString(keyBytes)
	keyHash := sha256.Sum256([]byte(apiKey))
	application := Application{
		Nom:          name,
		APIKeyHash:   hex.EncodeToString(keyHash[:]),
		Statut:       "ACTIF",
		DateCreation: time.Now().UTC(),
	}

	if err := r.db.WithContext(ctx).Create(&application).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, "", fmt.Errorf("%w: l'application existe déjà", ErrConflict)
		}
		return nil, "", fmt.Errorf("créer l'application: %w", err)
	}
	return &application, apiKey, nil
}

// FindApplicationByAPIKey compare le condensat de la clé et ignore les applications inactives.
func (r *Repository) FindApplicationByAPIKey(ctx context.Context, apiKey string) (*Application, error) {
	keyHash := sha256.Sum256([]byte(apiKey))
	var application Application
	err := r.db.WithContext(ctx).
		Where("api_key_hash = ? AND statut = ?", hex.EncodeToString(keyHash[:]), "ACTIF").
		First(&application).Error
	if err != nil {
		return nil, err
	}
	return &application, nil
}

// CreateEmail enregistre un email ou reconnaît une nouvelle tentative idempotente.
// Le booléen indique si une nouvelle ligne a été créée.
func (r *Repository) CreateEmail(ctx context.Context, email *Email) (bool, error) {
	if err := r.db.WithContext(ctx).Create(email).Error; err != nil {
		if !isUniqueViolation(err) {
			return false, fmt.Errorf("enregistrer l'email: %w", err)
		}

		var existing Email
		lookupErr := r.db.WithContext(ctx).
			Where("id_application = ? AND cle_idempotence = ?", email.IDApplication, email.CleIdempotence).
			First(&existing).Error
		if lookupErr != nil {
			return false, fmt.Errorf("retrouver l'email déjà enregistré: %w", lookupErr)
		}
		if existing.Destinataire != email.Destinataire ||
			existing.Objet != email.Objet ||
			existing.Message != email.Message ||
			existing.TypeContenu != email.TypeContenu {
			return false, fmt.Errorf("%w: cette clé d'idempotence est associée à un autre email", ErrConflict)
		}
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
