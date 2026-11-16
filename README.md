# Email Send API

## Organisation

- `main.go` est le point d'entrée de l'application.
- `src/Setting`
  centralise la configuration de l'application.
- `src/Database`
  contient la couche de persistance GORM, les modèles PostgreSQL et les opérations sur les applications et les emails.
- `src/Email`
  regroupe les routes HTTP, les DTO, l'authentification par clé API et l'envoi SMTP.
- `migrations`
  contient le schéma de la base de données.
- `.env` et `.env.example`
  contiennent la configuration locale et son modèle.

## Configuration et démarrage

Renseignez `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD` et `DB_NAME` dans l'environnement ou dans `.env`, puis appliquez les migrations SQL du dossier `migrations`.
Les valeurs par défaut de `APP_PORT`, `DB_PORT`, `DB_SSLMODE` et `SMTP_PORT` sont respectivement `8080`, `5432`, `disable` et `587`.
Lancez l'API avec `go run .`.

## API

- `POST /v1/applications` crée une application et renvoie une clé API affichée une seule fois.
- `POST /v1/emails` enregistre un email à traiter. Cette route requiert les en-têtes `X-API-Key` et `Idempotency-Key`. Réutiliser la même clé avec le même contenu ne crée pas de doublon.