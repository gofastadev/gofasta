package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

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

func TestInitTracer_RecordsTheServiceName(t *testing.T) {
	// The service name is the only way a span backend can tell one service's
	// spans from another's, so it has to reach the provider's resource rather
	// than only a log line.
	shutdown := InitTracer("discovery")
	defer shutdown()

	tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	if !ok {
		t.Fatal("global tracer provider is not an SDK provider; spans would be discarded")
	}

	exporter := tracetest.NewInMemoryExporter()
	tp.RegisterSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter))

	_, span := tp.Tracer("test").Start(context.Background(), "unit")
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("exported %d spans, want 1", len(spans))
	}

	var name string
	for _, attr := range spans[0].Resource.Attributes() {
		if attr.Key == semconv.ServiceNameKey {
			name = attr.Value.AsString()
		}
	}
	if name != "discovery" {
		t.Errorf("service.name = %q, want %q", name, "discovery")
	}
}

func TestInitTracer_ReplacesTheNoOpProvider(t *testing.T) {
	// A project whose repositories already call otel.Tracer(...) gets nothing
	// until this runs: the default global provider records no spans at all.
	shutdown := InitTracer("discovery")
	defer shutdown()

	if _, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); !ok {
		t.Fatal("InitTracer did not install an SDK tracer provider")
	}
}

func TestTracingMiddlewareWith_NamesSpansFromTheLabeller(t *testing.T) {
	// One span name per route, not per resource id — the reason http.route
	// exists as a convention.
	shutdown := InitTracer("discovery")
	defer shutdown()

	tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	if !ok {
		t.Fatal("global tracer provider is not an SDK provider")
	}
	exporter := tracetest.NewInMemoryExporter()
	tp.RegisterSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter))

	mw := TracingMiddlewareWith("discovery", func(*http.Request) string { return "/keys/{version}" })
	handler := mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	for _, version := range []string{"1", "2"} {
		handler.ServeHTTP(httptest.NewRecorder(),
			httptest.NewRequest(http.MethodGet, "/keys/"+version, nil))
	}

	spans := exporter.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("exported %d spans, want 2", len(spans))
	}
	for _, span := range spans {
		if span.Name != "GET /keys/{version}" {
			t.Errorf("span name = %q, want the route pattern", span.Name)
		}
	}
}

func TestTracingMiddlewareWith_EmptyLabelFallsBackToThePath(t *testing.T) {
	shutdown := InitTracer("discovery")
	defer shutdown()

	tp, _ := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	exporter := tracetest.NewInMemoryExporter()
	tp.RegisterSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter))

	mw := TracingMiddlewareWith("discovery", func(*http.Request) string { return "" })
	mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).
		ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/unmatched", nil))

	spans := exporter.GetSpans()
	if len(spans) != 1 || spans[0].Name != "GET /unmatched" {
		t.Fatalf("got %d spans, first named %q", len(spans), spans[0].Name)
	}
}

func TestTracingMiddlewareWith_NilLabellerFallsBackToPath(t *testing.T) {
	shutdown := InitTracer("discovery")
	defer shutdown()

	tp, _ := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	exporter := tracetest.NewInMemoryExporter()
	tp.RegisterSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter))

	mw := TracingMiddlewareWith("discovery", nil)
	mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).
		ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/nil-labeller", nil))

	spans := exporter.GetSpans()
	if len(spans) != 1 || spans[0].Name != "GET /nil-labeller" {
		t.Fatalf("got %d spans, first named %q", len(spans), spans[0].Name)
	}
}
