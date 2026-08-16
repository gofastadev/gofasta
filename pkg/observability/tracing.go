package observability

import (
	"context"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// InitTracer sets up the OpenTelemetry tracer provider.
// In production, add an exporter (OTLP, Jaeger, Zipkin).
// By default, uses a no-op exporter for zero overhead when unconfigured.
//
// The service name is recorded as the provider's `service.name` resource
// attribute, so it travels with every span once an exporter is attached — a
// backend receiving spans from several services has no other way to tell them
// apart.
//
// Until an exporter is registered the spans stay in-process. That is not
// wasted: it is what makes the tracer provider a real *sdktrace.TracerProvider
// rather than the no-op default, which is the difference between a project's
// existing otel.Tracer(...) calls being recorded and being discarded.
func InitTracer(serviceName string) func() {
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	slog.Info("tracer initialized", "service", serviceName)
	return func() {
		_ = tp.Shutdown(context.Background())
	}
}

// TracingMiddleware creates spans for HTTP requests and propagates trace
// context, naming each span after the request's literal path.
//
// Fine when every route is a static path. With path parameters this produces a
// distinct span name per parameter value, which is what OTel's `http.route`
// convention exists to avoid — use TracingMiddlewareWith and a labeller that
// returns the route pattern.
func TracingMiddleware(serviceName string) func(http.Handler) http.Handler {
	return TracingMiddlewareWith(serviceName, rawPath)
}

// TracingMiddlewareWith creates the same spans, naming them from the supplied
// labeller instead of the request path.
//
// The labeller is consulted twice, because the routers this exists for record
// their matched route only while serving. It runs once before the handler, so
// a span that never completes still carries the best name available; then
// again after, renaming the span if the router has since supplied a pattern.
// Renaming a span before End is explicitly allowed by the OTel spec and is how
// `http.route` gets onto a span at all.
//
// With chi that means:
//
//	observability.TracingMiddlewareWith(name, func(r *http.Request) string {
//	    return chi.RouteContext(r.Context()).RoutePattern()
//	})
//
// A labeller that returns "" at both points leaves the span named after the
// literal path, which is the right answer for a request that matched no route.
func TracingMiddlewareWith(serviceName string, label PathLabeller) func(http.Handler) http.Handler {
	if label == nil {
		label = rawPath
	}
	tracer := otel.Tracer(serviceName)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			name := label(r)
			if name == "" {
				name = r.URL.Path
			}

			ctx, span := tracer.Start(ctx, r.Method+" "+name, trace.WithSpanKind(trace.SpanKindServer))
			defer span.End()

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)

			// The router has routed by now. Rename only on a real improvement:
			// a labeller that still answers "" must not overwrite the path
			// name with nothing.
			if routed := label(r); routed != "" && routed != name {
				span.SetName(r.Method + " " + routed)
			}
		})
	}
}
