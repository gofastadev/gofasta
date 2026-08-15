package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "path", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	httpRequestsInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_requests_in_flight",
		Help: "Number of HTTP requests currently being processed",
	})
)

type metricsRecorder struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the HTTP status before forwarding it to the underlying writer.
func (m *metricsRecorder) WriteHeader(code int) {
	m.statusCode = code
	m.ResponseWriter.WriteHeader(code)
}

// PathLabeller decides the `path` label for a request.
//
// It runs after the handler, so a router that records the matched route on the
// request context has already done so by the time it is called.
type PathLabeller func(*http.Request) string

// rawPath labels by the request's literal path.
//
// Correct only for an API whose routes are all static. Anything with a path
// parameter — /users/{id}, /keys/{version} — produces one Prometheus time
// series per distinct value, which is the classic way to make a metrics
// backend fall over. Use MetricsMiddlewareWith and a labeller that returns the
// route pattern instead.
func rawPath(r *http.Request) string {
	return r.URL.Path
}

// MetricsMiddleware records HTTP request metrics (count, duration, in-flight),
// labeling by the request's literal path.
//
// Safe when every route is a static path. If any route has a parameter, use
// MetricsMiddlewareWith with a labeller that returns the route pattern —
// otherwise each distinct parameter value becomes its own time series.
func MetricsMiddleware(next http.Handler) http.Handler {
	return MetricsMiddlewareWith(rawPath)(next)
}

// MetricsMiddlewareWith records the same metrics, taking the `path` label from
// the supplied labeller.
//
// This exists so a project can label by route pattern rather than by URL,
// keeping label cardinality bounded by the size of the route table instead of
// by the number of distinct resources ever requested. The labeller is
// router-specific, so it is supplied by the caller rather than assumed here —
// with chi, for example:
//
//	observability.MetricsMiddlewareWith(func(r *http.Request) string {
//	    return chi.RouteContext(r.Context()).RoutePattern()
//	})
//
// A labeller that returns "" falls back to the literal path, which is what
// happens for a request that matched no route: there is no pattern to name,
// and dropping the observation entirely would hide 404 floods.
func MetricsMiddlewareWith(label PathLabeller) func(http.Handler) http.Handler {
	if label == nil {
		label = rawPath
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			httpRequestsInFlight.Inc()
			defer httpRequestsInFlight.Dec()

			rec := &metricsRecorder{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rec, r)

			path := label(r)
			if path == "" {
				path = r.URL.Path
			}

			duration := time.Since(start).Seconds()
			httpRequestsTotal.WithLabelValues(r.Method, path, strconv.Itoa(rec.statusCode)).Inc()
			httpRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
		})
	}
}

// MetricsHandler returns the Prometheus metrics HTTP handler.
// Mount at "/metrics" in your router.
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
