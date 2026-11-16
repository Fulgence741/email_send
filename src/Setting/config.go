// Package setting charge et valide la configuration de l'application.
package setting

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config contient les paramètres du serveur, de PostgreSQL et du relais SMTP.
type Config struct {
	AppName string
	AppEnv  string
	AppPort int

	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
}

// LoadConfig lit les variables d'environnement et applique les valeurs par défaut.
// La validation des paramètres propres à PostgreSQL est faite à l'ouverture de la base,
// afin que les outils qui n'utilisent que SMTP puissent charger cette configuration.
func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("charger le fichier .env: %w", err)
	}

	appPort, err := parsePort("APP_PORT", "8080")
	if err != nil {
		return nil, err
	}
	dbPort, err := parsePort("DB_PORT", "5432")
	if err != nil {
		return nil, err
	}
	smtpPort, err := parsePort("SMTP_PORT", "587")
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		AppName: getEnv("APP_NAME", "email-send-API"),
		AppEnv:  getEnv("APP_ENV", "development"),
		AppPort: appPort,

		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     dbPort,
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     smtpPort,
		SMTPUser:     os.Getenv("SMTP_USER"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:     os.Getenv("SMTP_FROM"),
	}

	return cfg, nil
}

// getEnv retourne la valeur d'une variable ou son remplacement si elle est absente.
func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

// parsePort convertit et valide un port TCP, en utilisant fallback si nécessaire.
func parsePort(key, fallback string) (int, error) {
	value := getEnv(key, fallback)
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("%s doit être un port valide compris entre 1 et 65535", key)
	}
	return port, nil
}
