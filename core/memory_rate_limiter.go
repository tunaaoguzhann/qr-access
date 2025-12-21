package core

import (
	"context"
	"sync"
	"time"
)

type MemoryRateLimiter struct {
	mu    sync.RWMutex
	users map[string]*userLimit
}

type userLimit struct {
	count     int
	windowEnd time.Time
}

func NewMemoryRateLimiter() *MemoryRateLimiter {
	return &MemoryRateLimiter{
		users: make(map[string]*userLimit),
	}
}

func (r *MemoryRateLimiter) CheckAndIncrement(ctx context.Context, userID string, limit int, window time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	ul, exists := r.users[userID]

	if !exists || now.After(ul.windowEnd) {
		r.users[userID] = &userLimit{
			count:     1,
			windowEnd: now.Add(window),
		}
		return nil
	}

	if ul.count >= limit {
		return ErrRateLimitExceeded
	}

	ul.count++
	return nil
}

