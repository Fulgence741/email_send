package email

import "time"

type Application struct {
	IDApplication int64     `gorm:"column:primaryKey" json:"id_application"`
	Nom           string    `gorm:"column:nom;not null" json:"nom"`
	APIKeyHash    string    `gorm:"-"`
	Statut        bool      `gorm:"column statut" json:"statut"`
	DateCreation  time.Time `gorm:"column:date_creation" json:"date_creation"`
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
