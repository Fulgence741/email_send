package redis

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	setting "email-send-API/src/Setting"

	"github.com/redis/go-redis/v9"
)

// NewRateLimitRedisClient crée un client Redis dédié au Rate Limiting.

func NewRateLimitRedisClient(cfg *setting.Config) (*redis.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration Redis absente")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     net.JoinHostPort(cfg.RedisHost, strconv.Itoa(cfg.RedisPort)),
		Username: cfg.RedisRateLimitUsername,
		Password: cfg.RedisRateLimitPassword,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		if closeErr := rdb.Close(); closeErr != nil {
			return nil, fmt.Errorf("connexion Redis Rate Limit impossible: %w (fermeture: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("connexion Redis Rate Limit impossible: %w", err)
	}

	return rdb, nil

}

// NewCacheRedisClient crée un client Redis dédié au Cache.
func NewCacheRedisClient(cfg *setting.Config) (*redis.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration Redis absente")

	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     net.JoinHostPort(cfg.RedisHost, strconv.Itoa(cfg.RedisPort)),
		Username: cfg.RedisCacheUsername,
		Password: cfg.RedisCachePassword,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		if closeErr := rdb.Close(); closeErr != nil {
			return nil, fmt.Errorf("connexion Redis Cache impossible: %w (fermeture: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("connexion Redis Cache impossible: %w", err)
	}

	return rdb, nil

}
