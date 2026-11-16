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

	AdresseRedis    string
	MotDePasseRedis string
	BaseRedis       int

	LimiteCreationApplication int
	LimiteAPI                 int
	RepertoirePiecesJointes   string

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

	// Lit et valide les ports et les limites de débit.
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
	baseRedis, err := parseEntierPositifOuNul("REDIS_DB", "0")
	if err != nil {
		return nil, err
	}
	limiteCreationApplication, err := parseEntierPositif("RATE_LIMIT_APPLICATIONS_PER_MINUTE", "10")
	if err != nil {
		return nil, err
	}
	limiteAPI, err := parseEntierPositif("RATE_LIMIT_API_PER_MINUTE", "60")
	if err != nil {
		return nil, err
	}

	// Crée la configuration finale avec les valeurs lues et validées.
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

		AdresseRedis:    getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		MotDePasseRedis: os.Getenv("REDIS_PASSWORD"),
		BaseRedis:       baseRedis,

		LimiteCreationApplication: limiteCreationApplication,
		LimiteAPI:                 limiteAPI,
		RepertoirePiecesJointes:   getEnv("ATTACHMENTS_DIR", "storage/attachments"),

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

// parseEntierPositifOuNul lit un entier supérieur ou égal à zéro.
func parseEntierPositifOuNul(key, fallback string) (int, error) {
	value := getEnv(key, fallback)
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("%s doit être un entier positif ou nul", key)
	}
	return parsed, nil
}

// parseEntierPositif lit un entier strictement supérieur à zéro.
func parseEntierPositif(key, fallback string) (int, error) {
	value := getEnv(key, fallback)
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, fmt.Errorf("%s doit être un entier strictement positif", key)
	}
	return parsed, nil
}
