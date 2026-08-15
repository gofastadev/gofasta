package middleware

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimit_AllowsRequests(t *testing.T) {
	cfg := config.RateLimitConfig{
		Enabled: true,
		Rate:    "100-S",
		Store:   "memory",
	}
	handler := RateLimit(cfg)(noopHandler)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.NotEqual(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimit_InvalidRateFallback(t *testing.T) {
	cfg := config.RateLimitConfig{
		Rate:  "invalid-rate",
		Store: "memory",
	}
	handler := RateLimit(cfg)(noopHandler)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	// Should fall back to 100/s and still allow the request
	assert.NotEqual(t, http.StatusTooManyRequests, rec.Code)
}

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

// TestNewRateLimit_RedisStoreWhenReachable is the success half of the store
// selection that TestNewRateLimit_RedisStoreUnreachableIsAnError only proves in
// failure: when redis is configured and reachable, the limiter must actually be
// built on it, because that shared counter is the whole reason for the setting.
func TestNewRateLimit_RedisStoreWhenReachable(t *testing.T) {
	mr := miniredis.RunT(t)
	host, port, err := net.SplitHostPort(mr.Addr())
	require.NoError(t, err)

	m, err := NewRateLimit(config.RateLimitConfig{
		Rate:  "100-S",
		Store: "redis",
		Redis: config.RedisConfig{Host: host, Port: port},
	})
	require.NoError(t, err)
	require.NotNil(t, m)

	// Serving a request proves the store is wired, not merely constructed.
	called := false
	h := m(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	h.ServeHTTP(httptest.NewRecorder(), req)
	assert.True(t, called)

	// The counter must live in redis rather than in process memory — that is
	// the difference between a global limit and one limit per replica.
	assert.NotEmpty(t, mr.Keys(), "rate limit state should be held in redis")
}

// TestNewRateLimit_StoreConstructionFailureIsAnError covers the branch between
// "redis is unreachable" and "redis works": a server that answers the
// connectivity probe but cannot preload the limiter's Lua scripts. Neither a
// real redis nor a dead port produces that, so it needs a purpose-built server.
func TestNewRateLimit_StoreConstructionFailureIsAnError(t *testing.T) {
	host, port := startScriptlessRedis(t)

	m, err := NewRateLimit(config.RateLimitConfig{
		Rate:  "100-S",
		Store: "redis",
		Redis: config.RedisConfig{Host: host, Port: port},
	})
	require.Error(t, err)
	assert.Nil(t, m)
	assert.Contains(t, err.Error(), "building redis rate limit store")
}

// startScriptlessRedis serves just enough of the Redis protocol to pass the
// startup PING while rejecting SCRIPT, and returns its host and port.
func startScriptlessRedis(t *testing.T) (host, port string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, acceptErr := ln.Accept()
			if acceptErr != nil {
				return
			}
			go serveScriptlessRedis(conn)
		}
	}()

	host, port, err = net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	return host, port
}

func serveScriptlessRedis(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	reader := bufio.NewReader(conn)
	for {
		name, err := readRESPCommandName(reader)
		if err != nil {
			return
		}

		var reply string
		switch strings.ToUpper(name) {
		case "PING":
			reply = "+PONG\r\n"
		case "HELLO":
			// Refusing HELLO makes the client fall back to RESP2, which it
			// tolerates; any other error here would fail connection setup.
			reply = "-ERR unknown command 'HELLO'\r\n"
		case "SCRIPT":
			reply = "-ERR this server does not support scripting\r\n"
		default:
			reply = "+OK\r\n"
		}
		if _, err := io.WriteString(conn, reply); err != nil {
			return
		}
	}
}

// readRESPCommandName reads one array-encoded command and returns its verb,
// discarding the arguments.
func readRESPCommandName(r *bufio.Reader) (string, error) {
	header, err := readRESPLine(r)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(header, "*") {
		return "", fmt.Errorf("expected an array header, got %q", header)
	}
	argc, err := strconv.Atoi(header[1:])
	if err != nil {
		return "", err
	}

	var name string
	for i := range argc {
		bulkHeader, err := readRESPLine(r)
		if err != nil {
			return "", err
		}
		if !strings.HasPrefix(bulkHeader, "$") {
			return "", fmt.Errorf("expected a bulk header, got %q", bulkHeader)
		}
		size, err := strconv.Atoi(bulkHeader[1:])
		if err != nil {
			return "", err
		}
		// +2 consumes the trailing CRLF.
		buf := make([]byte, size+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return "", err
		}
		if i == 0 {
			name = string(buf[:size])
		}
	}
	return name, nil
}

func readRESPLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
