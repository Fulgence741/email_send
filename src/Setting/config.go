package setting

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

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
	SMTPTLS      bool

	RedisHost     string
	RedisPort     int
	RedisPassword string
}

// LoadConfig charge et valide les variables d'environnement.
func LoadConfig() (*Config, error) {

	// Charger le fichier .env
	if err := godotenv.Load(); err != nil {
		fmt.Println("Fichier .env introuvable, utilisation des variables système")
	}

	// Récupérer les ports
	appPort, err := getIntEnv("APP_PORT", "8080")
	if err != nil {
		return nil, err
	}

	dbPort, err := getIntEnv("DB_PORT", "5432")
	if err != nil {
		return nil, err
	}

	smtpPort, err := getIntEnv("SMTP_PORT", "587")
	if err != nil {
		return nil, err
	}

	redisPort, err := getIntEnv("REDIS_PORT", "6379")
	if err != nil {
		return nil, err
	}

	// Récupérer la configuration TLS SMTP
	smtpTLS, err := strconv.ParseBool(getEnv("SMTP_TLS", "true"))
	if err != nil {
		return nil, fmt.Errorf("SMTP_TLS doit être un booléen valide : %w", err)
	}

	// Construire la configuration
	cfg := &Config{
		AppName: getEnv("APP_NAME", "email-send-API"),
		AppEnv:  getEnv("APP_ENV", "development"),
		AppPort: appPort,

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     dbPort,
		DBUser:     getEnv("DB_USER", ""),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", ""),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     smtpPort,
		SMTPUser:     getEnv("SMTP_USER", ""),	
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", ""),
		SMTPTLS:      smtpTLS,

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     redisPort,
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
	}

	// Valider la configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate vérifie la validité de la configuration.
func (cfg *Config) Validate() error {

	if cfg == nil {
		return fmt.Errorf("configuration absente")
	}

	// Application
	if cfg.AppPort < 1 || cfg.AppPort > 65535 {
		return fmt.Errorf("APP_PORT invalide")
	}

	// Base de données
	if cfg.DBPort < 1 || cfg.DBPort > 65535 {
		return fmt.Errorf("DB_PORT invalide")
	}

	// SMTP
	if cfg.SMTPHost == "" {
		return fmt.Errorf("SMTP_HOST est requis")
	}

	if cfg.SMTPPort < 1 || cfg.SMTPPort > 65535 {
		return fmt.Errorf("SMTP_PORT invalide")
	}

	if cfg.SMTPUser == "" {
		return fmt.Errorf("SMTP_USER est requis")
	}

	if cfg.SMTPPassword == "" {
		return fmt.Errorf("SMTP_PASSWORD est requis")
	}

	if cfg.SMTPFrom == "" {
		return fmt.Errorf("SMTP_FROM est requis")
	}

	// Redis
	if cfg.RedisPort < 1 || cfg.RedisPort > 65535 {
		return fmt.Errorf("REDIS_PORT invalide")
	}

	return nil
}

// getEnv récupère une variable d'environnement.
// Si elle est absente, retourne la valeur par défaut.
func getEnv(key, fallback string) string {

	value := strings.TrimSpace(os.Getenv(key))

	if value == "" {
		return fallback
	}

	return value
}

// getIntEnv récupère une variable d'environnement et la convertit en entier.
func getIntEnv(key, fallback string) (int, error) {

	value := getEnv(key, fallback)

	result, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s doit être un entier valide : %w", key, err)
	}

	if result < 1 || result > 65535 {
		return 0, fmt.Errorf("%s doit être compris entre 1 et 65535", key)
	}

	return result, nil
}
