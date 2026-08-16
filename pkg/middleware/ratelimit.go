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
	return RateLimitWith(cfg, RateLimitOptions{})
}

// RateLimitOptions configures what the counter is keyed by and where it is
// stored.
//
// Both matter on a Redis shared with other services. Without a Prefix every
// service counts into the same keys, so three services behind one Redis
// enforce a third of the intended limit each — and the symptom is users being
// throttled by traffic to a service they never called.
type RateLimitOptions struct {
	// Prefix namespaces the counter keys. Set it to something naming this
	// service and endpoint, e.g. "ratelimit:orders:graphql". Defaults to the
	// limiter library's shared prefix.
	Prefix string

	// KeyFunc decides what a request is counted against. Defaults to the
	// client IP. Use it to count per authenticated user instead, so one office
	// behind a NAT does not exhaust a shared limit.
	KeyFunc func(r *http.Request) string
}

// RateLimitWith is RateLimit with the key policy under the caller's control.
// It degrades to a memory store on error, the same way RateLimit does.
func RateLimitWith(cfg config.RateLimitConfig, opts RateLimitOptions) Middleware {
	m, err := NewRateLimitWith(cfg, opts)
	if err != nil {
		slog.Error("rate limiter falling back to in-process memory store — limits are now PER REPLICA, not global",
			"configured_store", cfg.Store, "error", err)
		return rateLimitWithStore(memory.NewStore(), cfg.Rate, opts)
	}
	return m
}

// NewRateLimit builds the rate limiting middleware, returning an error when the
// configured store is unavailable. Prefer this at startup when a misconfigured
// limiter should stop the process rather than degrade quietly.
func NewRateLimit(cfg config.RateLimitConfig) (Middleware, error) {
	return NewRateLimitWith(cfg, RateLimitOptions{})
}

// NewRateLimitWith is NewRateLimit with the key policy under the caller's
// control.
func NewRateLimitWith(cfg config.RateLimitConfig, opts RateLimitOptions) (Middleware, error) {
	if cfg.Store != "redis" {
		return rateLimitWithStore(memory.NewStore(), cfg.Rate, opts), nil
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

	store, err := redisstore.NewStoreWithOptions(client, redisStoreOptions(opts))
	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("building redis rate limit store: %w", err)
	}
	return rateLimitWithStore(store, cfg.Rate, opts), nil
}

// redisStoreOptions builds the limiter store options, starting from the
// library's own defaults.
//
// Seeding them is load-bearing rather than tidy. redisstore.NewStoreWithOptions
// copies these fields verbatim — only limiter's NewStore constructor fills them
// in — so a zero value would:
//
//   - write counters at the root of the keyspace as ":<client>" rather than
//     under "limiter:", contradicting what RateLimitOptions.Prefix documents and
//     disagreeing with the memory store, so that switching cfg.Store silently
//     resets every counter; and
//   - set MaxRetry to 0, dropping the retry the store performs when two
//     requests race on the same key, which shows up only as occasional lost
//     increments under load.
func redisStoreOptions(opts RateLimitOptions) limiter.StoreOptions {
	storeOpts := limiter.StoreOptions{
		Prefix:          limiter.DefaultPrefix,
		CleanUpInterval: limiter.DefaultCleanUpInterval,
		MaxRetry:        limiter.DefaultMaxRetry,
	}
	if opts.Prefix != "" {
		storeOpts.Prefix = opts.Prefix
	}
	return storeOpts
}

// rateLimitWithStore wires a limiter store into the stdlib middleware. The rate
// string is parsed here so both constructors share the same fallback.
func rateLimitWithStore(store limiter.Store, rateStr string, opts RateLimitOptions) Middleware {
	rate, err := limiter.NewRateFromFormatted(rateStr)
	if err != nil {
		// Fallback: 100 requests per second
		rate = limiter.Rate{Limit: 100, Period: time.Second}
	}

	instance := limiter.New(store, rate)
	mwOpts := []mw.Option{}
	if opts.KeyFunc != nil {
		mwOpts = append(mwOpts, mw.WithKeyGetter(mw.KeyGetter(opts.KeyFunc)))
	}
	stdlib := mw.NewMiddleware(instance, mwOpts...)
	return func(next http.Handler) http.Handler {
		return stdlib.Handler(next)
	}
}
