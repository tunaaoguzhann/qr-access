package core

import (
	"time"

	"github.com/redis/go-redis/v9"
)

type ManagerOptions struct {
	RedisAddr      string
	RedisKeyPrefix string
	MinTTL         time.Duration
	MaxTTL         time.Duration
	RateLimit      int
	RateWindow     time.Duration
}

func NewManager() (*Manager, error) {
	return NewManagerWithOptions(ManagerOptions{})
}

func NewManagerWithOptions(opts ManagerOptions) (*Manager, error) {
	var store Store
	var rateLimiter RateLimiter

	if opts.RedisAddr != "" {
		client := redis.NewClient(&redis.Options{
			Addr: opts.RedisAddr,
		})
		keyPrefix := opts.RedisKeyPrefix
		if keyPrefix == "" {
			keyPrefix = "qr-token:"
		}
		store = NewRedisStore(client, keyPrefix)
		if opts.RateLimit > 0 {
			rateLimiter = NewRedisRateLimiter(client, "qr-rate:")
		}
	} else {
		store = NewMemoryStore()
		if opts.RateLimit > 0 {
			rateLimiter = NewMemoryRateLimiter()
		}
	}

	rateWindow := opts.RateWindow
	if rateWindow == 0 && opts.RateLimit > 0 {
		rateWindow = 1 * time.Hour
	}

	cfg := Config{
		Store:       store,
		MinTTL:      opts.MinTTL,
		MaxTTL:      opts.MaxTTL,
		RateLimiter: rateLimiter,
		RateLimit:   opts.RateLimit,
		RateWindow:  rateWindow,
	}
	return newManager(cfg)
}
