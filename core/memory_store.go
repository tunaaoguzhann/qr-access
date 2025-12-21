package core

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MemoryStore struct {
	mu   sync.RWMutex
	data map[uuid.UUID]Token
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[uuid.UUID]Token),
	}
}

func (s *MemoryStore) Save(_ context.Context, t Token, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[t.ID] = t
	return nil
}

func (s *MemoryStore) Get(_ context.Context, id uuid.UUID) (*Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.data[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &t, nil
}

func (s *MemoryStore) MarkUsed(_ context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.data[id]
	if !ok {
		return ErrNotFound
	}
	t.Used = true
	s.data[id] = t
	return nil
}

