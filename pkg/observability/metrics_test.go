package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
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

// countFor reads the http_requests_total sample for one label set. The metrics
// are package-level and shared across tests, so every assertion here is a delta
// against a value read just before the request rather than an absolute count.
func countFor(t *testing.T, method, path, status string) float64 {
	t.Helper()
	m, err := httpRequestsTotal.GetMetricWithLabelValues(method, path, status)
	if err != nil {
		t.Fatalf("GetMetricWithLabelValues: %v", err)
	}
	return testutil.ToFloat64(m)
}

func TestMetricsMiddlewareWith_UsesTheLabeller(t *testing.T) {
	// The point of the labeller: a route with a parameter must produce one
	// series for the pattern, not one per parameter value.
	const pattern = "/keys/{version}"

	before := countFor(t, http.MethodGet, pattern, "200")

	mw := MetricsMiddlewareWith(func(*http.Request) string { return pattern })
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, version := range []string{"1", "2", "3"} {
		handler.ServeHTTP(httptest.NewRecorder(),
			httptest.NewRequest(http.MethodGet, "/keys/"+version, nil))
	}

	if got := countFor(t, http.MethodGet, pattern, "200") - before; got != 3 {
		t.Errorf("pattern series counted %v of 3 requests", got)
	}
	// The literal paths must not have become series of their own.
	for _, version := range []string{"1", "2", "3"} {
		if got := countFor(t, http.MethodGet, "/keys/"+version, "200"); got != 0 {
			t.Errorf("/keys/%s got its own series (%v)", version, got)
		}
	}
}

func TestMetricsMiddlewareWith_LabellerRunsAfterTheHandler(t *testing.T) {
	// chi only records the matched route pattern while serving, so the
	// labeller has to be called after next.ServeHTTP returns. This asserts
	// that ordering directly: the labeller reads a value the handler sets.
	var handlerRan bool
	var labeledAfterHandler bool

	mw := MetricsMiddlewareWith(func(*http.Request) string {
		labeledAfterHandler = handlerRan
		return "/labeled-late"
	})
	handler := mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		handlerRan = true
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/anything", nil))

	if !labeledAfterHandler {
		t.Fatal("labeller ran before the handler; a router's route pattern would not be set yet")
	}
}

func TestMetricsMiddlewareWith_EmptyLabelFallsBackToThePath(t *testing.T) {
	// An unmatched request has no pattern to name. Dropping the observation
	// would hide a 404 flood, so the literal path is used instead.
	before := countFor(t, http.MethodGet, "/no-such-route", "404")

	mw := MetricsMiddlewareWith(func(*http.Request) string { return "" })
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/no-such-route", nil))

	if got := countFor(t, http.MethodGet, "/no-such-route", "404") - before; got != 1 {
		t.Errorf("fallback series counted %v, want 1", got)
	}
}

func TestMetricsMiddlewareWith_NilLabeller(t *testing.T) {
	before := countFor(t, http.MethodGet, "/nil-labeller", "200")

	handler := MetricsMiddlewareWith(nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/nil-labeller", nil))

	if got := countFor(t, http.MethodGet, "/nil-labeller", "200") - before; got != 1 {
		t.Errorf("nil labeller counted %v, want 1 request labeled by path", got)
	}
}

func TestMetricsMiddleware_StillLabelsByPath(t *testing.T) {
	// The original entry point keeps its behavior: existing callers that
	// upgrade must not see their series change name.
	before := countFor(t, http.MethodPost, "/legacy", "201")

	handler := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/legacy", nil))

	if got := countFor(t, http.MethodPost, "/legacy", "201") - before; got != 1 {
		t.Errorf("MetricsMiddleware counted %v, want 1", got)
	}
}
