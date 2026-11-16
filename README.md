# Email Send API

## Organisation

- `main.go` est le point d'entrée de l'application.
- `src/Setting`
  centralise la configuration et le limiteur de débit Redis.
- `src/Database`
  contient la couche de persistance GORM, les modèles PostgreSQL et les opérations sur les applications et les emails.
- `src/Email`
  regroupe les routes, les DTO, le stockage des pièces jointes, le worker et l'envoi SMTP.
- `src/middleware`
  contient l'authentification par clé API et les protections HTTP communes.
- `migrations`
  contient le schéma de la base de données.
- `.env` et `.env.example`
  contiennent la configuration locale et son modèle.

## Configuration et démarrage

Renseignez `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD` et `DB_NAME` dans l'environnement ou dans `.env`, puis appliquez les migrations SQL du dossier `migrations`. Redis est requis au démarrage : configurez `REDIS_ADDR`, `REDIS_PASSWORD` et `REDIS_DB`.
Configurez également `SMTP_HOST`, `SMTP_FROM` et les identifiants SMTP si le relais les exige. Les fichiers sont stockés localement dans `ATTACHMENTS_DIR`, qui vaut `storage/attachments` par défaut.
En déploiement multi-instance, montez le même répertoire de pièces jointes sur chaque instance, car toutes peuvent traiter la file PostgreSQL.
Les valeurs par défaut de `APP_PORT`, `DB_PORT`, `DB_SSLMODE`, `SMTP_PORT`, `REDIS_ADDR`, `REDIS_DB`, `RATE_LIMIT_APPLICATIONS_PER_MINUTE` et `RATE_LIMIT_API_PER_MINUTE` sont respectivement `8080`, `5432`, `disable`, `587`, `127.0.0.1:6379`, `0`, `10` et `60`.
Lancez l'API avec `go run .`.

## API

- `POST /v1/applications` crée une application et renvoie une clé API affichée une seule fois.
- `POST /v1/emails` accepte du JSON ou un formulaire `multipart/form-data`. Le formulaire contient `destinataire`, `objet`, `contenu`, `type_contenu` et zéro à cinq fichiers `pieces_jointes`. La taille totale des fichiers est limitée à 20 Mio, avec 10 Mio maximum par fichier.
- Toute route privée requiert `X-API-Key`. `POST /v1/emails` requiert aussi `Idempotency-Key`; réutiliser la clé avec le même contenu n'ajoute pas de doublon.
- `GET /v1/emails?limit=20&offset=0&status=PENDING` liste les emails de l'application authentifiée. `limit` est compris entre 1 et 100.
- `GET /v1/emails/:id` consulte le statut d'un email appartenant à l'application authentifiée.
- `GET /v1/emails/:id/attempts` consulte l'historique des tentatives de livraison.
- `GET /v1/emails/:id/attachments` liste les métadonnées des fichiers liés à l'email.
- `GET /v1/emails/:id/attachments/:piece_id` télécharge une pièce jointe appartenant à l'application.
- `GET /healthz` vérifie que le processus HTTP est actif. `GET /readyz` vérifie PostgreSQL et Redis.

Les emails sont envoyés par un worker en arrière-plan. Chaque email dispose de trois tentatives au maximum, séparées de 60 secondes; les statuts et erreurs sont consultables via les routes de suivi.

Les middlewares communs ajoutent un identifiant de requête, des en-têtes de sécurité et une limite de corps HTTP de 25 Mio. Les routes privées vérifient la clé API avant d'accéder aux données. La création d'applications est limitée par adresse IP; les routes privées le sont par IP puis par application, avec des compteurs Redis partagés. En cas d'indisponibilité de Redis, les requêtes limitées échouent avec `503`; les dépassements renvoient `429` avec `Retry-After`.