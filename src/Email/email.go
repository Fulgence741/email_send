package email

import "time"

type Email struct {
	IDEmail       int64      `gorm:"primaryKey;column:id_email" json:"id_email"`
	IDApplication int64      `gorm:"column:id_application;not null" json:"id_application"`
	Destinataire  string     `gorm:"column:destinataire;not nulll" json:"destinataire"`
	Objet         string     `gorm:"column:objet;not null" json:"objet"`
	Contenu       string     `gorm:"column:contenu;not null" josn:"contenu"`
	TypeContenu   string     `gorm:"column:type_contenu;not null" json:"type_contennu"`
	Statut        string     `gorm:"column:statut;not nulle" josn:"statut"`
	DateCreation  time.Time  `gorm:"column:date_creation;not null" json:"date_creation"`
	DateEnvoi     *time.Time `gorm:"column:date_envoi;not null" json:"date_envoi"`
	MessageErreur *string    `gorm:"column:message_erreur; not null"  json:"message_serveur"`
}

func (Email) TableName() string {
	return "emails"
}

type CreateEmailDTO struct {
	Destinataire string `json:"destinataire" binding:"required,email"`
	Objet        string `json:"objet" binding:"required"`
	Contenu      string `json:"contenu" binding:"required"`
	TypeContenu  string `json:"type_contenu" binding:"required,oneof=TEXT HTML"`
}
