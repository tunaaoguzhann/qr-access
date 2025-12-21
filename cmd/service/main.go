package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"

	"github.com/tunaaoguzhann/qr-access/core"
)

type contextKey string

const userIDKey contextKey = "user_id"

func main() {
	cfg := loadConfig()

	store := buildStore(cfg)
	manager, err := core.NewManager(core.Config{
		Store:  store,
		Signer: core.NewSigner(cfg.HMACSecret),
		MinTTL: 10 * time.Second,
		MaxTTL: 10 * time.Minute,
	})
	if err != nil {
		log.Fatalf("init manager: %v", err)
	}

	r := chi.NewRouter()
	r.Use(rateLimit(10, time.Minute)) // basic rate limit demo
	r.Use(loggingMiddleware)
	r.Group(func(api chi.Router) {
		api.With(jwtAuth(cfg.JWTSecret)).Post("/qr/generate", handleGenerate(manager, cfg.DefaultTTL))
	})
	r.Post("/qr/verify", handleVerify(manager))

	addr := ":" + strconv.Itoa(cfg.Port)
	log.Printf("listening on %s (redis=%v)", addr, cfg.RedisAddr != "")
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

type generateRequest struct {
	Action string `json:"action"`
	TTL    int64  `json:"ttl_seconds"`
}

type generateResponse struct {
	TokenID   string    `json:"token_id"`
	Payload   string    `json:"payload"`
	ExpiresAt time.Time `json:"expires_at"`
}

func handleGenerate(manager *core.Manager, defaultTTL time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := r.Context().Value(userIDKey).(string)
		if !ok || uid == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req generateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		ttl := defaultTTL
		if req.TTL > 0 {
			ttl = time.Duration(req.TTL) * time.Second
		}

		token, payload, err := manager.Generate(r.Context(), uid, req.Action, ttl)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp := generateResponse{
			TokenID:   token.ID.String(),
			Payload:   payload,
			ExpiresAt: token.ExpiresAt,
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

type verifyRequest struct {
	Payload string `json:"payload"`
}

type verifyResponse struct {
	Valid     bool      `json:"valid"`
	TokenID   string    `json:"token_id"`
	UserID    string    `json:"user_id"`
	Action    string    `json:"action"`
	ExpiresAt time.Time `json:"expires_at"`
}

func handleVerify(manager *core.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req verifyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Payload == "" {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		token, err := manager.Verify(r.Context(), req.Payload)
		if err != nil {
			status := http.StatusBadRequest
			switch err {
			case core.ErrNotFound:
				status = http.StatusNotFound
			case core.ErrExpired:
				status = http.StatusGone
			case core.ErrUsed:
				status = http.StatusConflict
			case core.ErrBadSignature, core.ErrBadPayload:
				status = http.StatusUnauthorized
			}
			http.Error(w, err.Error(), status)
			return
		}

		resp := verifyResponse{
			Valid:     true,
			TokenID:   token.ID.String(),
			UserID:    token.UserID,
			Action:    token.Action,
			ExpiresAt: token.ExpiresAt,
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

// --- middleware & helpers ---

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// rateLimit is a simple in-process sliding window limiter per client address.
func rateLimit(max int, window time.Duration) func(http.Handler) http.Handler {
	type entry struct {
		count int
		ts    time.Time
	}
	var (
		mu      sync.Mutex
		buckets = make(map[string]entry)
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.RemoteAddr
			now := time.Now()

			mu.Lock()
			e := buckets[key]
			if now.Sub(e.ts) > window {
				e = entry{count: 0, ts: now}
			}
			e.count++
			e.ts = now
			buckets[key] = e
			mu.Unlock()

			if e.count > max {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func jwtAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}
			raw := strings.TrimSpace(auth[7:])
			userID, err := parseAndValidateJWT(raw, secret)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func parseAndValidateJWT(tokenStr, secret string) (string, error) {
	tok, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !tok.Valid {
		return "", errors.New("invalid jwt")
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return "", errors.New("missing sub")
	}
	return sub, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// --- config ---

type config struct {
	HMACSecret string
	JWTSecret  string
	Port       int
	DefaultTTL time.Duration
	RedisAddr  string
}

func loadConfig() config {
	port := 8080
	if p := os.Getenv("PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}
	ttl := 5 * time.Minute
	if t := os.Getenv("DEFAULT_TTL_SECONDS"); t != "" {
		if v, err := strconv.Atoi(t); err == nil && v > 0 {
			ttl = time.Duration(v) * time.Second
		}
	}
	return config{
		HMACSecret: envOr("QR_HMAC_SECRET", "dev-hmac-secret-change-me"),
		JWTSecret:  envOr("JWT_SECRET", "dev-jwt-secret-change-me"),
		Port:       port,
		DefaultTTL: ttl,
		RedisAddr:  os.Getenv("REDIS_ADDR"),
	}
}

func envOr(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func buildStore(cfg config) core.Store {
	if cfg.RedisAddr == "" {
		log.Printf("using in-memory store")
		return core.NewMemoryStore()
	}
	opts := &redis.Options{Addr: cfg.RedisAddr}
	client := redis.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}
	log.Printf("using redis store at %s", cfg.RedisAddr)
	return core.NewRedisStore(client, "qr-token:")
}

