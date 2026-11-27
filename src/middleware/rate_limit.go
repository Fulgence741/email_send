package middleware

import (
	"context"
	setting "email-send-API/src/Setting"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// les éléments neccessaire au fonctionnement du rate limit
type RateLimiter struct {
	client *redis.Client
	limit  int
	window time.Duration
}

// construction de script lua pour gérer le nombre de requetes

var allowRequestScript = redis.NewScript(`
	local count = redis.call("INCR", KEYS[1])
	if count == 1 then
	redis.call("PEXPIRE", KEYS[1], ARGV[1])
	end
	return count
`)

// Creation du rate limit et configuraation

func NewRateLimiter(client *redis.Client, limit int, window time.Duration) (*RateLimiter, error) {
	if client == nil {
		return nil, fmt.Errorf("Client redis absent")
	}

	if limit < 1 {
		return nil, fmt.Errorf("La limite de requête doit être positive")
	}
	if window.Microseconds() < 1 {
		return nil, fmt.Errorf("La fenêtre du rate limit doit durer au moins une miilliseconde")
	}

	return &RateLimiter{
			client: client,
			limit:  limit,
			window: window,
		},

		nil
}

//Determiner si un requêtes respectes les normes et si elle peut être autorisée

func (limiter *RateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	if limiter == nil || limiter.client == nil {
		return false, fmt.Errorf("Rate limit redis non initialisée")
	}
	if key == "" {
		return false, fmt.Errorf("Clé rate limit absente")

	}
	count, err := allowRequestScript.Run(
		ctx,
		limiter.client,
		[]string{key},
		limiter.window.Milliseconds()).Int64()
	if err != nil {
		return false, fmt.Errorf("impossible de vérifier le rate limit: %w", err)
	}
	return count <= int64(limiter.limit), nil
}

// Determiner le delais d'attente avant ressaie

func (limiter *RateLimiter) RetryAfterSeconds() int {
	if limiter == nil || limiter.window <= 0 {
		return 60
	}

	return int((limiter.window + time.Second - 1) / time.Second)
}

//Configurer le rate limit par IP

func RateLimitByIP(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, err := limiter.Allow(c.Request.Context(), "rate_limit:ip"+c.ClientIP())
		if err != nil {
			log.Printf("rate limit par IP indisponible: %v", err)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "service temporairement indisponible",
			})
			return
		}
		if !ok {
			c.Header("Retry-After", strconv.Itoa(limiter.RetryAfterSeconds()))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "Trop de tentatives, ressayez plus tard",
			})
			return
		}
		c.Next()
	}
}

// configuraation de rate limit par application authentifiée
func RateLimitByApplication(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		applicationID, ok := setting.AuthenticatedApplicationID(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "application non authentifiée",
			})
			return
		}
		key := "rate_limit:application:" + strconv.FormatInt(applicationID, 10)
		ok, err := limiter.Allow(c.Request.Context(), key)
		if err != nil {
			log.Printf("rate limit par application indisponible: %v", err)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "Service temporairement indisponible",
			})
			return
		}

		if !ok {
			c.Header("Retry-After", strconv.Itoa(limiter.RetryAfterSeconds()))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "Trop de tentatives, ressayez plus tard",
			})
			return
		}

		c.Next()
	}
}
