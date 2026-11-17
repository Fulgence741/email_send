package setting

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Clé pour stocker l'ID de l'application authentifiée dans le contexte Gin.
const authenticatedApplicationIDKey = "authenticated_application_id"

// apiKeyApplication représente une application avec son ID et son statut.
type apiKeyApplication struct {
	IDApplication int64  `gorm:"column:id_application"`
	Statut        string `gorm:"column:statut"`
}

// GenerateAPIKey crée une clé aléatoire et son hash à conserver en base.
func GenerateAPIKey() (apiKey, apiKeyHash string, err error) {
	// Génère une clé API aléatoire de 32 octets.
	rawKey := make([]byte, 32)
	// Remplit le tableau avec des octets aléatoires.
	if _, err := rand.Read(rawKey); err != nil {
		return "", "", err
	}

	// Convertit la clé brute en une chaîne hexadécimale.
	apiKey = hex.EncodeToString(rawKey)
	return apiKey, HashAPIKey(apiKey), nil
}

// HashAPIKey calcule le hash à rechercher dans la table applications.
func HashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	// Retourne le hash sous forme de chaîne hexadécimale.
	return hex.EncodeToString(hash[:])
}

// ValidateAPIKey compare une clé reçue au hash enregistré en base.
func ValidateAPIKey(apiKey, apiKeyHash string) bool {
	if apiKey == "" {
		return false
	}

	// Décodage du hash attendu depuis la chaîne hexadécimale.
	expectedHash, err := hex.DecodeString(apiKeyHash)
	// Vérifie si le décodage a échoué ou si la longueur du hash attendu est incorrecte.
	if err != nil || len(expectedHash) != sha256.Size {
		return false
	}

	// Calcule le hash de la clé API reçue.
	hash := sha256.Sum256([]byte(apiKey))
	// Compare le hash calculé avec le hash attendu .
	return subtle.ConstantTimeCompare(hash[:], expectedHash) == 1
}

// APIKeyAuth authentifie une application à partir de l'en-tête X-API-Key.
func APIKeyAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key") // Récupère la clé API depuis l'en-tête de la requête.
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Clé API manquante ou invalide."})
			c.Abort()
			return
		}

		// Vérifie la clé API dans la base de données.
		var application apiKeyApplication
		err := db.Table("applications").
			Select("id_application, statut").
			Where("api_key_hash = ?", HashAPIKey(apiKey)).
			Take(&application).Error
		// Si aucune application n'est trouvée, retourne une erreur 401 Unauthorized.
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Clé API manquante ou invalide."})
			c.Abort()
			return
		}
		// Si une autre erreur survient lors de la vérification de la clé API, retourne une erreur du serveur.
		if err != nil {
			log.Printf("erreur lors de la vérification de la clé API: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Erreur lors de l'authentification."})
			c.Abort()
			return
		}
		// Vérifie si l'application est active.
		if application.Statut != "ACTIF" {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Application inactive."})
			c.Abort()
			return
		}
		// Stocke l'ID de l'application authentifiée dans le contexte Gin.
		c.Set(authenticatedApplicationIDKey, application.IDApplication)
		c.Next()
	}
}

// AuthenticatedApplicationID retourne l'application validée par le middleware.
func AuthenticatedApplicationID(c *gin.Context) (int64, bool) {
	value, exists := c.Get(authenticatedApplicationIDKey)
	if !exists {
		return 0, false
	}
	// Vérifie si la valeur est bien un entier.
	id, ok := value.(int64)
	return id, ok
}
