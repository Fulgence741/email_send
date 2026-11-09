# Email Send API

## Organisation

- `main.go` 
est le point d'entrée de l'application.
- `src/Setting` 
centralise la configuration, la connexion à PostgreSQL, les routes et les middlewares.
- `src/Email` 
regroupe les modèles et DTO par ressource : application, email et pièce jointe, ainsi que l'envoi SMTP.
- `migrations`
 contient le schéma de la base de données.
- `.env` et `.env.example` 
contiennent la configuration locale et son modèle.