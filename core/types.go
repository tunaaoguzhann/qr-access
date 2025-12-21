package core

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Token struct {
	ID        uuid.UUID `json:"id"`
	UserID    string    `json:"user_id"`
	Action    string    `json:"action"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
}

var (
	ErrNotFound         = errors.New("token not found")
	ErrExpired          = errors.New("token expired")
	ErrUsed             = errors.New("token already used")
	ErrBadSignature     = errors.New("signature mismatch")
	ErrBadPayload       = errors.New("invalid payload")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

