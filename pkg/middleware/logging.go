package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// statusRecorder wraps http.ResponseWriter to capture the status code.
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.statusCode = code
	sr.ResponseWriter.WriteHeader(code)
}

// RequestLogging logs each HTTP request with method, path, status, and duration.
func RequestLogging(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sr := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(sr, r)

			requestID, _ := r.Context().Value(RequestIDKey).(string)
			logger.Info("request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", sr.statusCode,
				"duration", time.Since(start).String(),
				"request_id", requestID,
			)
		})
	}
}
