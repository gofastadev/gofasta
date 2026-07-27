package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/stretchr/testify/assert"
)

// TestNewRateLimit_MemoryStoreByDefault pins that an unset or non-redis store
// keeps working without any Redis dependency.
func TestNewRateLimit_MemoryStoreByDefault(t *testing.T) {
	for _, store := range []string{"", "memory"} {
		m, err := NewRateLimit(config.RateLimitConfig{Rate: "100-S", Store: store})
		assert.NoError(t, err, "store=%q", store)
		assert.NotNil(t, m, "store=%q", store)
	}
}

// TestNewRateLimit_RedisStoreUnreachableIsAnError is the regression test for
// the defect: RateLimit previously hardcoded the memory store and ignored
// cfg.Store entirely, so a deployment that configured redis silently got
// per-process limits — N replicas enforcing N times the intended rate. The
// error path must now be reachable and explicit.
func TestNewRateLimit_RedisStoreUnreachableIsAnError(t *testing.T) {
	// Port 1 is reserved and never listening, so this exercises the failure
	// branch without depending on a running Redis.
	_, err := NewRateLimit(config.RateLimitConfig{
		Rate:  "100-S",
		Store: "redis",
		Redis: config.RedisConfig{Host: "127.0.0.1", Port: "1"},
	})
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "redis"),
		"error should name the store that failed, got: %v", err)
}

// TestRateLimit_DegradesLoudlyRatherThanFailing keeps the non-error-returning
// constructor usable: it must still return a working middleware when redis is
// unreachable (having logged at ERROR), so a limiter misconfiguration cannot
// take the whole service down.
func TestRateLimit_DegradesLoudlyRatherThanFailing(t *testing.T) {
	m := RateLimit(config.RateLimitConfig{
		Rate:  "100-S",
		Store: "redis",
		Redis: config.RedisConfig{Host: "127.0.0.1", Port: "1"},
	})
	assert.NotNil(t, m)

	called := false
	h := m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	assert.True(t, called, "fallback limiter should still serve requests")
}
