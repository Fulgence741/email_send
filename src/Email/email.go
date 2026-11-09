package email

import "time"

type Email struct {
	IDEmail       int64      `gorm:"primaryKey" json:"id_email"`
	IDApplication int64      `json:"id_application"`
	Destinataire  string     `json:"destinataire"`
	Objet         string     `json:"objet"`
	Contenu       string     `json:"contenu"`
	TypeContenu   string     `json:"type_contenu"`
	Statut        string     `json:"statut"`
	DateCreation  time.Time  `json:"date_creation"`
	DateEnvoi     *time.Time `json:"date_envoi"`
	MessageErreur *string    `json:"message_erreur"`
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
