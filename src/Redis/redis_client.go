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

func NewRedisClient(cfg *setting.Config) (*redis.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration Redis absente")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     net.JoinHostPort(cfg.RedisHost, strconv.Itoa(cfg.RedisPort)),
		Password: cfg.RedisPassword,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		if closeErr := rdb.Close(); closeErr != nil {
			return nil, fmt.Errorf("vérification Redis impossible: %w (fermeture: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("vérification Redis impossible: %w", err)
	}

	return rdb, nil
}
