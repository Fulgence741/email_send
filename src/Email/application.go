package email

import "time"

type Application struct {
	IDApplication int64     `gorm:"primaryKey" json:"id_application"`
	Nom           string    `json:"nom"`
	APIKeyHash    string    `json:"-"`
	Statut        bool      `json:"statut"`
	DateCreation  time.Time `json:"date_creation"`
}

func (Application) TableName() string {
	return "applications"
}

type CreateApplicationDTO struct {
	Nom string `json:"nom" binding:"required"`
}

type ApplicationResponseDTO struct {
	IDApplication int64  `json:"id_application"`
	Nom           string `json:"nom"`
	Statut        bool   `json:"statut"`
}
