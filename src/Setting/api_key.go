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

const authenticatedApplicationIDKey = "authenticated_application_id"

type apiKeyApplication struct {
	IDApplication int64  `gorm:"column:id_application"`
	Statut        string `gorm:"column:statut"`
}

// GenerateAPIKey crée une clé aléatoire et son hash à conserver en base.
func GenerateAPIKey() (apiKey, apiKeyHash string, err error) {
	rawKey := make([]byte, 32)
	if _, err := rand.Read(rawKey); err != nil {
		return "", "", err
	}

	apiKey = hex.EncodeToString(rawKey)
	return apiKey, HashAPIKey(apiKey), nil
}

// HashAPIKey calcule le hash à rechercher dans la table applications.
func HashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}

// ValidateAPIKey compare une clé reçue au hash enregistré en base.
func ValidateAPIKey(apiKey, apiKeyHash string) bool {
	if apiKey == "" {
		return false
	}

	expectedHash, err := hex.DecodeString(apiKeyHash)
	if err != nil || len(expectedHash) != sha256.Size {
		return false
	}

	hash := sha256.Sum256([]byte(apiKey))
	return subtle.ConstantTimeCompare(hash[:], expectedHash) == 1
}

// APIKeyAuth authentifie une application à partir de l'en-tête X-API-Key.
func APIKeyAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Clé API manquante ou invalide."})
			c.Abort()
			return
		}

		var application apiKeyApplication
		err := db.Table("applications").
			Select("id_application, statut").
			Where("api_key_hash = ?", HashAPIKey(apiKey)).
			Take(&application).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Clé API manquante ou invalide."})
			c.Abort()
			return
		}
		if err != nil {
			log.Printf("erreur lors de la vérification de la clé API: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Erreur lors de l'authentification."})
			c.Abort()
			return
		}
		if application.Statut != "ACTIF" {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Application inactive."})
			c.Abort()
			return
		}

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
	id, ok := value.(int64)
	return id, ok
}
