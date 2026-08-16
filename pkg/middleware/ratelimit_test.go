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
	"github.com/ulule/limiter/v3"
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

func TestRateLimitWith_KeyFuncDecidesWhatIsCounted(t *testing.T) {
	// The default counts per client IP. A service that wants per-user limits
	// had no way to say so, which meant one office behind a NAT could exhaust
	// a limit meant for an individual.
	mwFn := RateLimitWith(config.RateLimitConfig{Rate: "2-S"}, RateLimitOptions{
		KeyFunc: func(r *http.Request) string { return r.Header.Get("X-User") },
	})
	h := mwFn(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	call := func(user string) int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-User", user)
		req.RemoteAddr = "10.0.0.1:1234" // identical IP for both users
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	// Exhaust ada's allowance.
	call("ada")
	call("ada")
	if got := call("ada"); got != http.StatusTooManyRequests {
		t.Errorf("third request for ada = %d, want 429", got)
	}

	// grace shares the IP but not the key, so she is unaffected.
	if got := call("grace"); got != http.StatusOK {
		t.Errorf("first request for grace = %d, want 200 — the key is the IP, not the user", got)
	}
}

func TestRateLimitWith_NoKeyFuncKeepsTheIPDefault(t *testing.T) {
	mwFn := RateLimitWith(config.RateLimitConfig{Rate: "1-S"}, RateLimitOptions{})
	h := mwFn(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	call := func() int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.9:1234"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	call()
	if got := call(); got != http.StatusTooManyRequests {
		t.Errorf("second request from one IP = %d, want 429", got)
	}
}

// testClientAddr is the one client every prefix test counts against; keeping it
// fixed is what makes "two services, two counters" mean something.
const testClientAddr = "203.0.113.10:1234"

// serveOnce drives one request through the middleware, which is what makes the
// limiter touch its store.
func serveOnce(t *testing.T, m Middleware) {
	t.Helper()
	called := false
	h := m(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = testClientAddr
	h.ServeHTTP(httptest.NewRecorder(), req)
	require.True(t, called, "the request never reached the handler")
}

// This is the failure the Prefix option exists to prevent, and it is invisible
// from inside any one service: three services sharing a Redis and no prefix
// count into the same keys, so each enforces roughly a third of its configured
// limit — and the users being throttled are being throttled by traffic to a
// service they never called.
func TestNewRateLimitWith_PrefixNamespacesTheCounterKeys(t *testing.T) {
	mr := miniredis.RunT(t)
	host, port, err := net.SplitHostPort(mr.Addr())
	require.NoError(t, err)

	m, err := NewRateLimitWith(config.RateLimitConfig{
		Rate:  "100-S",
		Store: "redis",
		Redis: config.RedisConfig{Host: host, Port: port},
	}, RateLimitOptions{Prefix: "ratelimit:orders:graphql"})
	require.NoError(t, err)
	require.NotNil(t, m)

	serveOnce(t, m)

	keys := mr.Keys()
	require.NotEmpty(t, keys, "rate limit state should be held in redis")
	for _, key := range keys {
		assert.True(t, strings.HasPrefix(key, "ratelimit:orders:graphql"),
			"key %q is outside this service's namespace", key)
	}
}

// The other half of the pair, and the regression test for a defect this file
// found: NewRateLimitWith builds the redis store through limiter's
// NewStoreWithOptions, which takes the StoreOptions fields verbatim — only
// NewStore fills in the library defaults. Passing a zero value therefore wrote
// counters at the root of the keyspace as ":<client>" rather than under
// "limiter:", contradicting what RateLimitOptions.Prefix documents and
// disagreeing with the memory store, which does use "limiter:". Switching
// cfg.Store then silently reset every counter.
func TestNewRateLimitWith_NoPrefixUsesTheLibraryDefault(t *testing.T) {
	mr := miniredis.RunT(t)
	host, port, err := net.SplitHostPort(mr.Addr())
	require.NoError(t, err)

	m, err := NewRateLimitWith(config.RateLimitConfig{
		Rate:  "100-S",
		Store: "redis",
		Redis: config.RedisConfig{Host: host, Port: port},
	}, RateLimitOptions{})
	require.NoError(t, err)

	serveOnce(t, m)

	keys := mr.Keys()
	require.NotEmpty(t, keys)
	for _, key := range keys {
		assert.True(t, strings.HasPrefix(key, limiter.DefaultPrefix+":"),
			"key %q is not under the library's default prefix", key)
	}
}

// The prefix is visible in the keyspace, so the tests above can see it. The
// other two fields are not: MaxRetry only shows up as occasional lost
// increments when two requests race on one key, and CleanUpInterval not at all
// on this store. They are asserted here, where the options are built.
func TestRedisStoreOptions_StartFromTheLibraryDefaults(t *testing.T) {
	got := redisStoreOptions(RateLimitOptions{})

	assert.Equal(t, limiter.DefaultPrefix, got.Prefix,
		"an unset prefix must not put counters at the root of the keyspace")
	assert.Equal(t, limiter.DefaultMaxRetry, got.MaxRetry,
		"MaxRetry 0 drops the retry the store performs under contention")
	assert.Equal(t, limiter.DefaultCleanUpInterval, got.CleanUpInterval)
}

func TestRedisStoreOptions_PrefixOverridesOnlyThePrefix(t *testing.T) {
	// The override must not take the other defaults down with it, which is what
	// building the struct fresh around opts.Prefix would do.
	got := redisStoreOptions(RateLimitOptions{Prefix: "ratelimit:orders:graphql"})

	assert.Equal(t, "ratelimit:orders:graphql", got.Prefix)
	assert.Equal(t, limiter.DefaultMaxRetry, got.MaxRetry)
	assert.Equal(t, limiter.DefaultCleanUpInterval, got.CleanUpInterval)
}

// Two services on one Redis, counting separately. Without the prefix these
// would be the same key for the same client, and the second service's traffic
// would spend the first service's budget.
func TestNewRateLimitWith_DifferentPrefixesDoNotShareCounters(t *testing.T) {
	mr := miniredis.RunT(t)
	host, port, err := net.SplitHostPort(mr.Addr())
	require.NoError(t, err)

	cfg := config.RateLimitConfig{
		Rate:  "100-S",
		Store: "redis",
		Redis: config.RedisConfig{Host: host, Port: port},
	}

	orders, err := NewRateLimitWith(cfg, RateLimitOptions{Prefix: "ratelimit:orders"})
	require.NoError(t, err)
	billing, err := NewRateLimitWith(cfg, RateLimitOptions{Prefix: "ratelimit:billing"})
	require.NoError(t, err)

	serveOnce(t, orders)
	serveOnce(t, billing)

	var sawOrders, sawBilling bool
	for _, key := range mr.Keys() {
		switch {
		case strings.HasPrefix(key, "ratelimit:orders"):
			sawOrders = true
		case strings.HasPrefix(key, "ratelimit:billing"):
			sawBilling = true
		}
	}
	assert.True(t, sawOrders && sawBilling,
		"the same client from two services must produce two counters, got keys %v", mr.Keys())
}
