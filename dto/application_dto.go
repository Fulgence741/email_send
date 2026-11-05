package dto

type CreateApplicationDTO struct {
	Nom string `json:"nom" binding:"required"`
}

type ApplicationResponseDTO struct {
	IDApplication int64  `json:"id_application"`
	Nom           string `json:"nom"`
	Statut        bool   `json:"statut"`
}
