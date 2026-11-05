package dto

type PieceJointeResponseDTO struct {
	IDPieceJointe int64  `json:"id_piece_jointe"`
	NomFichier    string `json:"nom_fichier"`
	TypeMIME      string `json:"type_mime"`
	TailleFichier int64  `json:"taille_fichier"`
}
