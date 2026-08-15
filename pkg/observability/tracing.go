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
// The labeller is called before the handler runs, because a span has to be
// named when it starts. That rules out a router that only records its matched
// route while serving; supply a labeller that can answer from the request
// alone, or accept the path-named default.
//
// A labeller that returns "" falls back to the literal path, so an unmatched
// request still produces a span with something readable in it.
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

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
