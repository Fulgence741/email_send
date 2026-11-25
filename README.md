# Email Send API

## Organisation

- `main.go` est le point d'entrée de l'application.
- `src/Setting` centralise la configuration.
- `src/database/database.go` regroupe la connexion GORM, les modèles et les DTO.
- `src/Email` contient les routes, les handlers et l'envoi SMTP.
- `migrations` contient le schéma de la base de données.
- `.env` contient la configuration locale et son modèle.

## Routes

- `POST /v1/applications` crée une application. Corps JSON : `{"nom":"nom-de-l'application"}`.
  La clé `api_key` de la réponse n'est affichée qu'à la création ; conservez-la pour l'authentification.
- `POST /v1/applications/liste` consulte l'application associée à la clé `X-API-Key`, sans exposer sa clé API.
- `POST /v1/applications/supprimer/{id}` supprime l'application associée à la clé `X-API-Key` et ses emails, tentatives et pièces jointes.
- `POST /v1/emails` envoie un email et enregistre son résultat dans `emails` et
  `tentatives_envoi`. La clé API identifie l'application ; à mettre dans
  l'en-tête `X-API-Key`. En cas d'échec, l'envoi est réessayé jusqu'à trois fois,
  avec un délai de 60 secondes entre les tentatives. Corps JSON :
- `POST /v1/emails/liste` consulte les emails de l'application identifiée par la clé `X-API-Key`.
- `POST /v1/emails/supprimer/{id}` supprime un email de l'application identifiée par la clé `X-API-Key`.
- `POST /v1/tentatives` retourne directement la liste détaillée des tentatives
  d'envoi de l'application identifiée par la clé `X-API-Key`.

```json
{
  "destinataire": "destinataire@mail.com",
  "objet": "Bonjour",
  "message": "Contenu du message",
  "type_contenu": "TEXT"
}
```



`type_contenu` accepte `TEXT` ou `HTML`.




