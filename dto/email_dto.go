package dto

type CreateEmailDTO struct {
	Destinataire string `json:"destinataire" binding:"required,email"`
	Objet        string `json:"objet" binding:"required"`
	Contenu      string `json:"contenu" binding:"required"`
	TypeContenu  string `json:"type_contenu" binding:"required,oneof=TEXT HTML"`
}
