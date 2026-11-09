package email

import "time"

type Envoie struct {
	IDTentative     int64     `gorm:"primaryKey;column:id_tentative"`
	IDEmail         int64     `gorm:"column:id_email;not null" json:"id_email"`
	NumeroTentative int       `gorm:"columun:numero_tentative" json:"numero_tentative"`
	DateTentative   time.Time `gorm:"column:date_tentative;not null" json:"date_tentative"`
	Statut          string    `gorm:"column:statut;not null" json:"statut"`
	MessageErreur   *string   `gorm:"column:message_erreur" json:"message_erreur"`
	ReponseServeur  *string   `gorm:"column:reponse_serveur" json:"reponse_serveur"`
}

func (Envoie) TableName() string {
	return "tentatives_envoi"
}

type ResponseEnvoieDTO struct {
	IDTentative     int64     `json:"id_tentative"`
	NumeroTentative int       `json:"numero_tentative"`
	DateTentative   time.Time `json:"date_tentative"`
	Statut          string    `json:"statut"`
	MessageErreur   *string   `json:"message_erreur"`
	ReponseServeur  *string   `json:"reponse_serveur"`
}
