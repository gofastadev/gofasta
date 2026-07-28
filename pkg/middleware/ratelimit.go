package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gofastadev/gofasta/pkg/config"
	libredis "github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	mw "github.com/ulule/limiter/v3/drivers/middleware/stdlib"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	redisstore "github.com/ulule/limiter/v3/drivers/store/redis"
)

// redisPingTimeout bounds the startup connectivity check. Short on purpose:
// this runs during server construction, and a wedged Redis should surface as a
// loud error rather than a hung boot.
const redisPingTimeout = 3 * time.Second

// RateLimit creates a rate limiting middleware from config.
//
// The store honors cfg.Store: "redis" shares one counter across every replica,
// anything else keeps counters in process memory. That distinction is the whole
// point of the setting — with a memory store, N replicas enforce N times the
// configured limit and a restart resets every counter. (An earlier version
// always used the memory store and ignored cfg.Store entirely, so deployments
// that configured redis silently got per-process limits.)
//
// When cfg.Store is "redis" but the store cannot be built, this logs at ERROR
// and falls back to memory so the service still starts with some limiting in
// place. Callers that would rather fail fast should use NewRateLimit, which
// returns the error instead.
func RateLimit(cfg config.RateLimitConfig) Middleware {
	m, err := NewRateLimit(cfg)
	if err != nil {
		slog.Error("rate limiter falling back to in-process memory store — limits are now PER REPLICA, not global",
			"configured_store", cfg.Store, "error", err)
		return rateLimitWithStore(memory.NewStore(), cfg.Rate)
	}
	return m
}

// NewRateLimit builds the rate limiting middleware, returning an error when the
// configured store is unavailable. Prefer this at startup when a misconfigured
// limiter should stop the process rather than degrade quietly.
func NewRateLimit(cfg config.RateLimitConfig) (Middleware, error) {
	if cfg.Store != "redis" {
		return rateLimitWithStore(memory.NewStore(), cfg.Rate), nil
	}

	client := libredis.NewClient(&libredis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), redisPingTimeout)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("rate limit redis store unreachable at %s:%s: %w", cfg.Redis.Host, cfg.Redis.Port, err)
	}

	store, err := redisstore.NewStore(client)
	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("building redis rate limit store: %w", err)
	}
	return rateLimitWithStore(store, cfg.Rate), nil
}

// rateLimitWithStore wires a limiter store into the stdlib middleware. The rate
// string is parsed here so both constructors share the same fallback.
func rateLimitWithStore(store limiter.Store, rateStr string) Middleware {
	rate, err := limiter.NewRateFromFormatted(rateStr)
	if err != nil {
		// Fallback: 100 requests per second
		rate = limiter.Rate{Limit: 100, Period: time.Second}
	}
	stdlib := mw.NewMiddleware(limiter.New(store, rate))
	return func(next http.Handler) http.Handler {
		return stdlib.Handler(next)
	}
}
