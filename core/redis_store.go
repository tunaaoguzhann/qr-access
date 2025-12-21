package core

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client    *redis.Client
	keyPrefix string
}

func NewRedisStore(client *redis.Client, keyPrefix string) *RedisStore {
	if keyPrefix == "" {
		keyPrefix = "qr-token:"
	}
	return &RedisStore{
		client:    client,
		keyPrefix: keyPrefix,
	}
}

func (s *RedisStore) key(id uuid.UUID) string {
	return fmt.Sprintf("%s%s", s.keyPrefix, id.String())
}

func (s *RedisStore) Save(ctx context.Context, t Token, ttl time.Duration) error {
	raw, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, s.key(t.ID), raw, ttl).Err()
}

func (s *RedisStore) Get(ctx context.Context, id uuid.UUID) (*Token, error) {
	val, err := s.client.Get(ctx, s.key(id)).Result()
	if err == redis.Nil {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var t Token
	if err := json.Unmarshal([]byte(val), &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *RedisStore) MarkUsed(ctx context.Context, id uuid.UUID) error {
	script := redis.NewScript(`
local val = redis.call("GET", KEYS[1])
if not val then return 0 end
local obj = cjson.decode(val)
if obj.used then return -1 end
obj.used = true
local ttl = redis.call("PTTL", KEYS[1])
if ttl and ttl > 0 then
  redis.call("SET", KEYS[1], cjson.encode(obj), "PX", ttl)
else
  redis.call("SET", KEYS[1], cjson.encode(obj))
end
return 1
`)

	res, err := script.Run(ctx, s.client, []string{s.key(id)}).Int()
	if err != nil {
		return err
	}
	switch res {
	case 0:
		return ErrNotFound
	case -1:
		return ErrUsed
	default:
		return nil
	}
}

