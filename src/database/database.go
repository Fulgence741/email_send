package database

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	setting "email-send-API/src/Setting"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ConnectDB ouvre une connexion GORM à PostgreSQL et vérifie qu'elle répond.
func ConnectDB(cfg *setting.Config) (*gorm.DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration de base de données absente")
	}

	dsnURL := &url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(cfg.DBHost, strconv.Itoa(cfg.DBPort)),
		Path:   "/" + cfg.DBName,
		User:   url.UserPassword(cfg.DBUser, cfg.DBPassword),
	}

	query := dsnURL.Query()
	query.Set("sslmode", cfg.DBSSLMode)
	dsnURL.RawQuery = query.Encode()

	db, err := gorm.Open(postgres.Open(dsnURL.String()), &gorm.Config{
		TranslateError: true,
		Logger:         logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		return nil, fmt.Errorf("erreur de connexion à PostgreSQL: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("impossible de récupérer la connexion SQL: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("impossible de joindre la base de données: %w", err)
	}

	return db, nil
}

// Application correspond à la table applications.
type Application struct {
	IDApplication int64     `gorm:"primaryKey;column:id_application" json:"id_application"`
	Nom           string    `gorm:"column:nom;not null" json:"nom"`
	APIKeyHash    string    `gorm:"column:api_key_hash;not null" json:"-"`
	Statut        string    `gorm:"column:statut;not null" json:"statut"`
	DateCreation  time.Time `gorm:"column:date_creation;not null" json:"date_creation"`
}

func (Application) TableName() string {
	return "applications"
}

// CreateApplicationDTO valide les données nécessaires à la création d'une application.
type CreateApplicationDTO struct {
	Nom string `json:"nom" binding:"required,max=100"`
}

// Email correspond à la table emails.
type Email struct {
	IDEmail                int64      `gorm:"primaryKey;column:id_email" json:"id_email"`
	IDApplication          int64      `gorm:"column:id_application;not null" json:"id_application"`
	Destinataire           string     `gorm:"column:destinataire;not null" json:"destinataire"`
	Objet                  string     `gorm:"column:objet;not null" json:"objet"`
	Message                string     `gorm:"column:message;not null" json:"message"`
	TypeContenu            string     `gorm:"column:type_contenu;not null" json:"type_contenu"`
	Statut                 string     `gorm:"column:statut;not null" json:"statut"`
	NombreMaxTentatives    int        `gorm:"column:nombre_max_tentatives;not null" json:"nombre_max_tentatives"`
	DelaiRetrySecondes     int        `gorm:"column:delai_retry_secondes;not null" json:"delai_retry_secondes"`
	DateCreation           time.Time  `gorm:"column:date_creation;not null" json:"date_creation"`
	DateDebutTraitement    *time.Time `gorm:"column:date_debut_traitement" json:"date_debut_traitement"`
	DateFinTraitement      *time.Time `gorm:"column:date_fin_traitement" json:"date_fin_traitement"`
	DateProchaineTentative *time.Time `gorm:"column:date_prochaine_tentative" json:"date_prochaine_tentative"`
	MessageErreur          *string    `gorm:"column:message_erreur" json:"message_erreur"`
}

func (Email) TableName() string {
	return "emails"
}

// CreateEmailDTO valide les données nécessaires à l'envoi d'un email.
type CreateEmailDTO struct {
	Destinataire string `form:"destinataire" binding:"required,email"`
	Objet        string `form:"objet" binding:"required"`
	Message      string `form:"message" binding:"required"`
	TypeContenu  string `form:"type_contenu" binding:"required,oneof=TEXT HTML"`
}

// Envoie correspond à une tentative d'envoi d'email.
type Envoie struct {
	IDTentative     int64     `gorm:"primaryKey;column:id_tentative" json:"id_tentative"`
	IDEmail         int64     `gorm:"column:id_email;not null" json:"id_email"`
	NumeroTentative int       `gorm:"column:numero_tentative;not null" json:"numero_tentative"`
	DateTentative   time.Time `gorm:"column:date_tentative;not null" json:"date_tentative"`
	Statut          string    `gorm:"column:statut;not null" json:"statut"`
	MessageErreur   *string   `gorm:"column:message_erreur" json:"message_erreur"`
	ReponseServeur  *string   `gorm:"column:reponse_serveur" json:"reponse_serveur"`
}

func (Envoie) TableName() string {
	return "tentatives_envoi"
}

// ResponseEnvoieDTO représente une tentative renvoyée par l'API.
type ResponseEnvoieDTO struct {
	IDTentative     int64     `json:"id_tentative"`
	NumeroTentative int       `json:"numero_tentative"`
	DateTentative   time.Time `json:"date_tentative"`
	Statut          string    `json:"statut"`
	MessageErreur   *string   `json:"message_erreur"`
	ReponseServeur  *string   `json:"reponse_serveur"`
}

// PieceJointe correspond à la table pieces_jointes.
type PieceJointe struct {
	IDPieceJointe int64     `gorm:"primaryKey;column:id_piece_jointe" json:"id_piece_jointe"`
	IDEmail       int64     `gorm:"column:id_email;not null" json:"id_email"`
	NomFichier    string    `gorm:"column:nom_fichier;not null" json:"nom_fichier"`
	TypeMIME      string    `gorm:"column:type_mime;not null" json:"type_mime"`
	CheminFichier string    `gorm:"column:chemin_fichier;not null" json:"chemin_fichier"`
	TailleFichier int64     `gorm:"column:taille_fichier;not null" json:"taille_fichier"`
	DateCreation  time.Time `gorm:"column:date_creation;not null" json:"date_creation"`
}

func (PieceJointe) TableName() string {
	return "pieces_jointes"
}

// PieceJointeResponseDTO représente une pièce jointe renvoyée par l'API.
type PieceJointeResponseDTO struct {
	IDPieceJointe int64  `json:"id_piece_jointe"`
	NomFichier    string `json:"nom_fichier"`
	TypeMIME      string `json:"type_mime"`
	TailleFichier int64  `json:"taille_fichier"`
}
