# Email Send API

## Organisation

- `main.go` est le point d'entrée de l'application.
- `src/Setting` centralise la configuration.
- `src/database/database.go` regroupe la connexion GORM, les modèles et les DTO.
- `src/Email` contient les routes, les handlers et l'envoi SMTP.
- `migrations` contient le schéma de la base de données.
- `.env`  contien la configuration locale et son modèle.

## Routes

- `POST /v1/applications` crée une application. Corps JSON : `{"nom":"nom-de-l'application"}`.
  La clé `api_key` de la réponse n'est affichée qu'à la création ; on la notz pour l'authentification plus tard.
- `POST /v1/emails` envoie un email et enregistre son résultat dans `emails` et
  `tentatives_envoi`. La clé API identifie l'application ; à mettre dans
  l'en-tête `X-API-Key`. En cas d'échec, l'envoi est réessayé jusqu'à trois fois,
  avec un délai de 60 secondes entre les tentatives. Corps JSON :

```json
{
  "destinataire": "destinataire@example.com",
  "objet": "Bonjour",
  "message": "Contenu du message",
  "type_contenu": "TEXT"
}
```

`type_contenu` accepte `TEXT` ou `HTML`.

- `GET /v1/tentatives` retourne directement la liste détaillée des tentatives
  d'envoi de l'application identifiée par la clé `X-API-Key`.



