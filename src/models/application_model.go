package models

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
