package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
