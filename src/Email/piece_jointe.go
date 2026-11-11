package email

import (
	"time"
)

// Model
// ====================================================================
type PieceJointe struct {
	IDPieceJointe int64     `gorm:"column:primaryKey" json:"id_piece_jointe"`
	IDEmail       int64     `gorm:"column:id_email;not null" json:"id_email"`
	NomFichier    string    `gorm:"column:nom_fichier;not null" json:"nom_fichier"`
	TypeMIME      string    `gorm:"column:type_mime;not null" json:"type_mime"`
	CheminFichier string    `gorm:"column:chemin_fichier" json:"chemin_fichier"`
	TailleFichier int64     `gorm:"column:taille_fichier" json:"taille_fichier"`
	DateCreation  time.Time `gorm:"column:date_creation" json:"date_creation"`
}

func (PieceJointe) TableName() string {
	return "pieces_jointes"
}

// ==========================================

// DTO
// ========================================================
type PieceJointeResponseDTO struct {
	IDPieceJointe int64  `json:"id_piece_jointe"`
	NomFichier    string `json:"nom_fichier"`
	TypeMIME      string `json:"type_mime"`
	TailleFichier int64  `json:"taille_fichier"`
}

// =========================================================

// Handler

//func PieceJointehandlers(c *gin.Context)
