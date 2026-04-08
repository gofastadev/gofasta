package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/stretchr/testify/assert"
)

// ---------- helpers ----------

var noopHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

var echoHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(r.Method + " " + r.URL.Path))
})

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// ---------- Chain ----------

func TestChain_AppliesMiddlewaresInOrder(t *testing.T) {
	var order []string

	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw1-before")
			next.ServeHTTP(w, r)
			order = append(order, "mw1-after")
		})
	}
	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw2-before")
			next.ServeHTTP(w, r)
			order = append(order, "mw2-after")
		})
	}

	handler := Chain(noopHandler, mw1, mw2)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, []string{"mw1-before", "mw2-before", "mw2-after", "mw1-after"}, order)
}

func TestChain_NoMiddlewares(t *testing.T) {
	handler := Chain(echoHandler)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/test", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "GET /test", rec.Body.String())
}

// ---------- CORS ----------

func TestCORS_SetsHeaders(t *testing.T) {
	origins := []string{"https://example.com", "https://other.com"}
	handler := CORS(origins)(noopHandler)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, "https://example.com, https://other.com", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS, PATCH", rec.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type, Authorization, X-Request-ID", rec.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "86400", rec.Header().Get("Access-Control-Max-Age"))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCORS_PreflightReturnsNoContent(t *testing.T) {
	handler := CORS([]string{"*"})(noopHandler)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, "/api/resource", nil))

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestCORS_NonPreflightCallsNext(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := CORS([]string{"*"})(inner)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/resource", nil))

	assert.True(t, called)
}

func TestCORS_SingleOrigin(t *testing.T) {
	handler := CORS([]string{"https://only.com"})(noopHandler)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, "https://only.com", rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_EmptyOrigins(t *testing.T) {
	handler := CORS([]string{})(noopHandler)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, "", rec.Header().Get("Access-Control-Allow-Origin"))
}

// ---------- ContentType ----------

func TestContentType_SetsJSONHeader(t *testing.T) {
	handler := ContentType()(noopHandler)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ---------- RequestID ----------

func TestRequestID_GeneratesNewID(t *testing.T) {
	handler := RequestID()(noopHandler)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	id := rec.Header().Get("X-Request-ID")
	assert.NotEmpty(t, id)
	assert.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, id)
}

func TestRequestID_UsesExistingID(t *testing.T) {
	handler := RequestID()(noopHandler)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "custom-id-123")
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "custom-id-123", rec.Header().Get("X-Request-ID"))
}

func TestRequestID_StoresInContext(t *testing.T) {
	var ctxID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxID, _ = r.Context().Value(RequestIDKey).(string)
		w.WriteHeader(http.StatusOK)
	})
	handler := RequestID()(inner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "ctx-test-id")
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "ctx-test-id", ctxID)
}

func TestRequestID_GeneratesUniqueIDs(t *testing.T) {
	handler := RequestID()(noopHandler)
	ids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		id := rec.Header().Get("X-Request-ID")
		assert.False(t, ids[id], "duplicate request ID: %s", id)
		ids[id] = true
	}
}

// ---------- Recovery ----------

func TestRecovery_CatchesPanics(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	})
	handler := Recovery(testLogger())(panicHandler)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Internal Server Error")
}

func TestRecovery_PassesThroughNormally(t *testing.T) {
	handler := Recovery(testLogger())(echoHandler)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthy", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "GET /healthy", rec.Body.String())
}

func TestRecovery_CatchesNonStringPanic(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(42)
	})
	handler := Recovery(testLogger())(panicHandler)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRecovery_DoesNotPanicOnNilPanic(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(nil)
	})
	handler := Recovery(testLogger())(panicHandler)

	rec := httptest.NewRecorder()
	assert.NotPanics(t, func() {
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	})
}

// ---------- RequestLogging ----------

func TestRequestLogging_CallsNextHandler(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	})
	handler := RequestLogging(testLogger())(inner)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/items", nil))

	assert.True(t, called)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestRequestLogging_CapturesStatusCode(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	handler := RequestLogging(testLogger())(inner)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/missing", nil))

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestRequestLogging_DefaultsTo200(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	handler := RequestLogging(testLogger())(inner)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequestLogging_WithRequestIDInContext(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := Chain(inner, RequestID(), RequestLogging(testLogger()))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/with-id", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))
}

// ---------- StatusRecorder ----------

func TestStatusRecorder_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	sr := &statusRecorder{ResponseWriter: rec, statusCode: http.StatusOK}

	sr.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, sr.statusCode)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------- SecurityHeaders ----------

func TestSecurityHeaders_SetsHeaders(t *testing.T) {
	cfg := config.SecurityConfig{
		HSTS:               true,
		HSTSMaxAge:         31536000,
		FrameDeny:          true,
		ContentTypeNosniff: true,
		BrowserXSSFilter:   true,
		ReferrerPolicy:     "strict-origin-when-cross-origin",
	}
	handler := SecurityHeaders(cfg)(noopHandler)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
}

func TestSecurityHeaders_CallsNextHandler(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	cfg := config.SecurityConfig{
		ContentTypeNosniff: true,
	}
	handler := SecurityHeaders(cfg)(inner)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.True(t, called)
}

// ---------- RateLimit ----------

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

// ---------- Context Keys ----------

func TestContextKeys_Values(t *testing.T) {
	assert.Equal(t, contextKey("requestID"), RequestIDKey)
	assert.Equal(t, contextKey("userClaims"), UserClaimsKey)
}

func TestContextKey_TypeSafety(t *testing.T) {
	ctx := context.WithValue(context.Background(), RequestIDKey, "test-id")

	val, ok := ctx.Value(RequestIDKey).(string)
	assert.True(t, ok)
	assert.Equal(t, "test-id", val)

	// Plain string key should not match the typed contextKey
	val2, ok2 := ctx.Value("requestID").(string)
	assert.False(t, ok2)
	assert.Empty(t, val2)
}

// ---------- Integration: Chain with multiple middlewares ----------

func TestChain_Integration(t *testing.T) {
	var gotRequestID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequestID, _ = r.Context().Value(RequestIDKey).(string)
		w.WriteHeader(http.StatusOK)
	})

	handler := Chain(inner, RequestID(), ContentType())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/test", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.NotEmpty(t, gotRequestID)
	assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))
}
