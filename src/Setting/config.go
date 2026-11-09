package setting

import (
	"log"
	"os"
	"strconv"

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
}

func LoadConfig() *Config {
	// Charger le fichier .env
	if err := godotenv.Load(); err != nil {
		log.Println("Fichier .env introuvable, utilisation des variables du système")
	}
	// Récupérer et convertir le port de l'application en entier
	AppPort, err := strconv.Atoi(os.Getenv("APP_PORT"))
	if err != nil {
		log.Fatal("APP_PORT doit être un entier valide")
	}

	// Récupérer et convertir le port de la base de données en entier
	DBPort, err := strconv.Atoi(os.Getenv("DB_PORT"))
	if err != nil {
		log.Fatal("DB_PORT doit être un entier valide")
	}

	// Récupérer et convertir le port SMTP en entier
	SMTPPort, err := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil {
		log.Fatal("SMTP_PORT doit être un entier valide")
	}

	return &Config{
		AppName: os.Getenv("APP_NAME"),
		AppEnv:  os.Getenv("APP_ENV"),
		AppPort: AppPort,

		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     DBPort,
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
		DBSSLMode:  os.Getenv("DB_SSLMODE"),

		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     SMTPPort,
		SMTPUser:     os.Getenv("SMTP_USER"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:     os.Getenv("SMTP_FROM"),
	}

}

// getEnv récupère une variable d'environnement et renvoie sa valeur
// par défaut si elle n'est pas définie

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
