package core

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Manager struct {
	store       Store
	now         func() time.Time
	minTTL      time.Duration
	maxTTL      time.Duration
	rateLimiter RateLimiter
	rateLimit   int
	rateWindow  time.Duration
}

type Config struct {
	Store       Store
	Now         func() time.Time
	MinTTL      time.Duration
	MaxTTL      time.Duration
	RateLimiter RateLimiter
	RateLimit   int
	RateWindow  time.Duration
}

func newManager(cfg Config) (*Manager, error) {
	if cfg.Store == nil {
		return nil, fmt.Errorf("store is required")
	}
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	return &Manager{
		store:       cfg.Store,
		now:         nowFn,
		minTTL:      cfg.MinTTL,
		maxTTL:      cfg.MaxTTL,
		rateLimiter: cfg.RateLimiter,
		rateLimit:   cfg.RateLimit,
		rateWindow:  cfg.RateWindow,
	}, nil
}

func (m *Manager) Generate(ctx context.Context, secretKey, userID, action string, ttl time.Duration) (Token, string, error) {
	if secretKey == "" {
		return Token{}, "", fmt.Errorf("secret key is required")
	}
	if userID == "" || action == "" {
		return Token{}, "", fmt.Errorf("userID and action are required")
	}
	if ttl <= 0 {
		return Token{}, "", fmt.Errorf("ttl must be positive")
	}

	if m.rateLimiter != nil && m.rateLimit > 0 {
		if err := m.rateLimiter.CheckAndIncrement(ctx, userID, m.rateLimit, m.rateWindow); err != nil {
			return Token{}, "", err
		}
	}

	if m.maxTTL > 0 && ttl > m.maxTTL {
		ttl = m.maxTTL
	}
	if m.minTTL > 0 && ttl < m.minTTL {
		ttl = m.minTTL
	}

	id := uuid.New()
	now := m.now()
	token := Token{
		ID:        id,
		UserID:    userID,
		Action:    action,
		ExpiresAt: now.Add(ttl),
		Used:      false,
	}

	if err := m.store.Save(ctx, token, ttl); err != nil {
		return Token{}, "", err
	}

	signer := NewSigner(secretKey)
	signature := signer.Sign(id[:])
	payload, err := EncodePayload(id.String(), signature)
	if err != nil {
		return Token{}, "", err
	}
	return token, payload, nil
}

func (m *Manager) Verify(ctx context.Context, secretKey, encoded string) (*Token, error) {
	if secretKey == "" {
		return nil, fmt.Errorf("secret key is required")
	}

	data, err := DecodePayload(encoded)
	if err != nil {
		return nil, err
	}

	tokenID, err := uuid.Parse(data.ID)
	if err != nil {
		return nil, ErrBadPayload
	}

	signer := NewSigner(secretKey)
	if ok := signer.Verify(tokenID[:], data.Sig); !ok {
		return nil, ErrBadSignature
	}

	token, err := m.store.Get(ctx, tokenID)
	if err != nil {
		return nil, err
	}

	now := m.now()
	if token.ExpiresAt.Before(now) {
		return nil, ErrExpired
	}
	if token.Used {
		return nil, ErrUsed
	}

	if err := m.store.MarkUsed(ctx, tokenID); err != nil {
		return nil, err
	}
	token.Used = true
	return token, nil
}

