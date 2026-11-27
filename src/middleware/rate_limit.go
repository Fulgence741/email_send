package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	setting "email-send-API/src/Setting"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client *redis.Client
	limit  int
	window time.Duration
}

var allowRequestScript = redis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
	redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return count
`)

func NewRateLimiter(client *redis.Client, limit int, window time.Duration) (*RateLimiter, error) {
	if client == nil {
		return nil, fmt.Errorf("client Redis absent")
	}
	if limit < 1 {
		return nil, fmt.Errorf("la limite de requêtes doit être positive")
	}
	if window.Milliseconds() < 1 {
		return nil, fmt.Errorf("la fenêtre du rate limit doit durer au moins une milliseconde")
	}

	return &RateLimiter{client: client, limit: limit, window: window}, nil
}

func (limiter *RateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	if limiter == nil || limiter.client == nil {
		return false, fmt.Errorf("rate limiter Redis non initialisé")
	}
	if key == "" {
		return false, fmt.Errorf("clé de rate limit absente")
	}

	count, err := allowRequestScript.Run(ctx, limiter.client, []string{key}, limiter.window.Milliseconds()).Int64()
	if err != nil {
		return false, fmt.Errorf("impossible de vérifier le rate limit: %w", err)
	}
	return count <= int64(limiter.limit), nil
}

func (limiter *RateLimiter) RetryAfterSeconds() int {
	if limiter == nil || limiter.window <= 0 {
		return 60
	}
	return int((limiter.window + time.Second - 1) / time.Second)
}

func RateLimitByIP(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, err := limiter.Allow(c.Request.Context(), "rate_limit:ip:"+c.ClientIP())
		if err != nil {
			log.Printf("rate limit par IP indisponible: %v", err)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "Service temporairement indisponible",
			})
			return
		}
		if !ok {
			c.Header("Retry-After", strconv.Itoa(limiter.RetryAfterSeconds()))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Trop de requêtes, réessayez plus tard",
			})
			return
		}
		c.Next()
	}
}

func RateLimitByApplication(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		applicationID, ok := setting.AuthenticatedApplicationID(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Application non authentifiée",
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
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Trop de requêtes, réessayez plus tard",
			})
			return
		}
		c.Next()
	}
}
