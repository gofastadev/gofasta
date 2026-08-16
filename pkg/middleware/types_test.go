package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

var noopHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

var echoHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(r.Method + " " + r.URL.Path))
})

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
