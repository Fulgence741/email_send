package setting

import (
	"fmt"

	"context"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ConnectDB établit une connexion à la base de données PostgreSQL
// en utilisant les paramètres de configuration fournis.
func ConnectDB(cfg *Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",

		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBSSLMode)

	// Connexion à la base de données PostgreSQL
	db, err := gorm.Open(postgres.Open(dsn),
		&gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("Erreur de connexion à postgreSQL: %w", err)
	}

	// Recupérer l'instance *sql.DB pour gérer la connexion
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("Erreur lors de la récupération de l'instance *sql.DB: %w", err)
	}

	// Construire le context avec timeout de 5 secondes
	context, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ping pour vérifier la connexion à la base de données
	if err := sqlDB.PingContext(context); err != nil {
		return nil, fmt.Errorf("Erreur lors du ping à la base de données: %w", err)
	}

	return db, nil

}
