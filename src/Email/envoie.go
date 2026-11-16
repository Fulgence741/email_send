package email

import "time"

// ResponseEnvoieDTO expose le résultat d'une tentative de livraison.
type ResponseEnvoieDTO struct {
	IDTentative     int64     `json:"id_tentative"`
	NumeroTentative int       `json:"numero_tentative"`
	DateTentative   time.Time `json:"date_tentative"`
	Statut          string    `json:"statut"`
	MessageErreur   *string   `json:"message_erreur"`
	ReponseServeur  *string   `json:"reponse_serveur"`
}
