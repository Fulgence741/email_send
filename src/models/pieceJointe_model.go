package models

import "time"

type PieceJointe struct {
	IDPieceJointe int64     `gorm:"primaryKey" json:"id_piece_jointe"`
	IDEmail       int64     `json:"id_email"`
	NomFichier    string    `json:"nom_fichier"`
	TypeMIME      string    `json:"type_mime"`
	CheminFichier string    `json:"chemin_fichier"`
	TailleFichier int64     `json:"taille_fichier"`
	DateCreation  time.Time `json:"date_creation"`
}

func (PieceJointe) TableName() string {
	return "pieces_jointes"
}
