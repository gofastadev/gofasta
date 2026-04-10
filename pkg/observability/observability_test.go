package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetricsMiddleware_CallsNext(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := MetricsMiddleware(inner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("expected inner handler to be called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestMetricsMiddleware_CustomStatus(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	handler := MetricsMiddleware(inner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestMetricsHandler(t *testing.T) {
	handler := MetricsHandler()
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestMetricsRecorder_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	mr := &metricsRecorder{ResponseWriter: rec, statusCode: http.StatusOK}

	mr.WriteHeader(http.StatusBadRequest)

	if mr.statusCode != http.StatusBadRequest {
		t.Errorf("expected statusCode 400, got %d", mr.statusCode)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected underlying recorder to have 400, got %d", rec.Code)
	}
}

func TestInitTracer(t *testing.T) {
	cleanup := InitTracer("test-service")
	if cleanup == nil {
		t.Fatal("expected non-nil cleanup function")
	}
	// Calling cleanup should not panic.
	cleanup()
}

func TestTracingMiddleware_CallsNext(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := TracingMiddleware("test-service")
	handler := middleware(inner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/traced", nil)
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("expected inner handler to be called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
