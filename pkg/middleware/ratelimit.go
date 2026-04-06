package middleware

import (
	"net/http"
	"time"

	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/ulule/limiter/v3"
	mw "github.com/ulule/limiter/v3/drivers/middleware/stdlib"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

// RateLimit creates a rate limiting middleware from config.
func RateLimit(cfg config.RateLimitConfig) Middleware {
	rate, err := limiter.NewRateFromFormatted(cfg.Rate)
	if err != nil {
		// Fallback: 100 requests per second
		rate = limiter.Rate{Limit: 100, Period: time.Second}
	}
	store := memory.NewStore()
	instance := limiter.New(store, rate)
	stdlib := mw.NewMiddleware(instance)

	return func(next http.Handler) http.Handler {
		return stdlib.Handler(next)
	}
}
