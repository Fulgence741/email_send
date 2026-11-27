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
  l'en-tête `X-API-Key`. Utilisez un formulaire `multipart/form-data` avec les
  champs `destinataire`, `objet`, `message`, `type_contenu` (`TEXT` ou `HTML`)
  et un ou plusieurs champs `fichiers`. Jusqu'à 5 fichiers sont acceptés, de
  10 Mio chacun et 25 Mio au total. En cas d'échec SMTP, l'envoi est réessayé
  jusqu'à trois fois, avec un délai de 60 secondes entre les tentatives.
- `POST /v1/emails/liste` consulte les emails de l'application identifiée par la clé `X-API-Key`.
- `POST /v1/emails/supprimer/{id}` supprime un email de l'application identifiée par la clé `X-API-Key`.
- `POST /v1/tentatives` retourne directement la liste détaillée des tentatives
  d'envoi de l'application identifiée par la clé `X-API-Key`.

Les routes `/v1` sont limitées à 60 requêtes par minute et par adresse IP.
Les routes protégées ont également une limite de 60 requêtes par minute et par
application. Les compteurs sont partagés via Redis.

Exemple avec `curl` :

```sh
curl -X POST http://localhost:8080/v1/emails \
  -H "X-API-Key: votre-cle-api" \
  -F "destinataire=destinataire@mail.com" \
  -F "objet=Bonjour" \
  -F "message=Contenu du message" \
  -F "type_contenu=TEXT" \
  -F "fichiers=@./document.pdf" \
  -F "fichiers=@./image.png"
```




