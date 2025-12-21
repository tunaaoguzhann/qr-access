package core

import (
	"context"
	"time"
)

type RateLimiter interface {
	CheckAndIncrement(ctx context.Context, userID string, limit int, window time.Duration) error
}

