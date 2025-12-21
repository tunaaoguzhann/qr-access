package core

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Store interface {
	Save(ctx context.Context, t Token, ttl time.Duration) error
	Get(ctx context.Context, id uuid.UUID) (*Token, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
}

