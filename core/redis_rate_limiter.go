package core

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisRateLimiter struct {
	client    *redis.Client
	keyPrefix string
}

func NewRedisRateLimiter(client *redis.Client, keyPrefix string) *RedisRateLimiter {
	if keyPrefix == "" {
		keyPrefix = "qr-rate:"
	}
	return &RedisRateLimiter{
		client:    client,
		keyPrefix: keyPrefix,
	}
}

func (r *RedisRateLimiter) key(userID string) string {
	return fmt.Sprintf("%s%s", r.keyPrefix, userID)
}

func (r *RedisRateLimiter) CheckAndIncrement(ctx context.Context, userID string, limit int, window time.Duration) error {
	key := r.key(userID)

	script := redis.NewScript(`
		local current = redis.call("GET", KEYS[1])
		if current == false then
			redis.call("SET", KEYS[1], 1, "EX", ARGV[2])
			return 1
		end
		local count = tonumber(current)
		if count >= tonumber(ARGV[1]) then
			return 0
		end
		redis.call("INCR", KEYS[1])
		return 1
	`)

	result, err := script.Run(ctx, r.client, []string{key}, limit, int(window.Seconds())).Int()
	if err != nil {
		return err
	}

	if result == 0 {
		return ErrRateLimitExceeded
	}

	return nil
}

