package http

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/healtronlabs/gofasta/packages/core"
)

// recoveryMiddleware provides panic recovery
func (s *HTTPServer) recoveryMiddleware() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Log the panic
					stack := make([]byte, 4096)
					length := runtime.Stack(stack, false)
					fmt.Printf("PANIC: %v\n%s\n", err, stack[:length])
					
					// Send error response
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			
			next.ServeHTTP(w, r)
		})
	}
}

// corsMiddleware handles CORS
func (s *HTTPServer) corsMiddleware() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			
			// Set CORS headers
			if s.allowedOrigin(origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(s.config.CORSMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(s.config.CORSHeaders, ", "))
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")
			
			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// allowedOrigin checks if origin is allowed
func (s *HTTPServer) allowedOrigin(origin string) bool {
	for _, allowed := range s.config.CORSOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

// gzipMiddleware provides response compression
func (s *HTTPServer) gzipMiddleware() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if client accepts gzip
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}
			
			// Create gzip writer
			gz := gzip.NewWriter(w)
			defer gz.Close()
			
			// Set headers
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Vary", "Accept-Encoding")
			
			// Wrap response writer
			gzw := &gzipResponseWriter{
				ResponseWriter: w,
				Writer:         gz,
			}
			
			next.ServeHTTP(gzw, r)
		})
	}
}

// gzipResponseWriter wraps response writer with gzip compression
type gzipResponseWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// contextMiddleware adds request context
func (s *HTTPServer) contextMiddleware() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add request ID
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}
			w.Header().Set("X-Request-ID", requestID)
			
			// Add timing
			start := time.Now()
			
			// Wrap response writer to capture status
			wrapped := &statusResponseWriter{
				ResponseWriter: w,
				statusCode:     200,
			}
			
			next.ServeHTTP(wrapped, r)
			
			// Log request
			duration := time.Since(start)
			fmt.Printf("%s %s %d %v\n", r.Method, r.URL.Path, wrapped.statusCode, duration)
		})
	}
}

// statusResponseWriter captures response status
type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// staticFileMiddleware handles static file serving with caching
func (s *HTTPServer) staticFileMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set cache headers
		if s.config.StaticFileCache > 0 {
			w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(s.config.StaticFileCache.Seconds())))
			w.Header().Set("Expires", time.Now().Add(s.config.StaticFileCache).Format(http.TimeFormat))
		}
		
		// Set content type based on file extension
		ext := filepath.Ext(r.URL.Path)
		if contentType := getContentType(ext); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		
		next.ServeHTTP(w, r)
	})
}

// wrapWithGuard wraps handler with guard
func (s *HTTPServer) wrapWithGuard(handler http.HandlerFunc, guard core.Guard) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create request context for guard
		reqCtx := &core.RequestContext{
			Request: r,
		}
		
		// Check guard
		allowed := guard.CanActivate(reqCtx)
		if !allowed {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		
		handler(w, r)
	}
}

// wrapWithPipe wraps handler with pipe
func (s *HTTPServer) wrapWithPipe(handler http.HandlerFunc, pipe core.Pipe) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := s.createRequestContext(w, r)
		
		// Transform request
		transformed, err := pipe.Transform(ctx.Body, &core.PipeMetadata{
			Type: "request",
			Data: make(map[string]interface{}),
		})
		if err != nil {
			s.handleError(ctx, err)
			return
		}
		
		// Update context with transformed data
		if transformedBytes, ok := transformed.([]byte); ok {
			ctx.Body = transformedBytes
		}
		
		handler(w, r)
	}
}

// wrapWithInterceptor wraps handler with interceptor
func (s *HTTPServer) wrapWithInterceptor(handler http.HandlerFunc, interceptor core.Interceptor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create request context for interceptor
		reqCtx := &core.RequestContext{
			Request: r,
		}
		
		// Call interceptor
		result := interceptor.Intercept(reqCtx, func(ctx *core.RequestContext) *core.Response {
			// Call the actual handler
			handler(w, r)
			return &core.Response{
				StatusCode: 200,
				Headers:    make(map[string]string),
				Body:       nil,
			}
		})
		
		// Handle interceptor result if needed
		_ = result
	}
}

// Utility functions

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// getContentType returns content type for file extension
func getContentType(ext string) string {
	contentTypes := map[string]string{
		".html": "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".svg":  "image/svg+xml",
		".ico":  "image/x-icon",
		".pdf":  "application/pdf",
		".txt":  "text/plain",
		".xml":  "application/xml",
		".zip":  "application/zip",
	}
	
	return contentTypes[strings.ToLower(ext)]
}

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(requestsPerSecond int) MiddlewareFunc {
	tokens := make(chan struct{}, requestsPerSecond)
	
	// Fill the bucket
	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(requestsPerSecond))
		defer ticker.Stop()
		
		for range ticker.C {
			select {
			case tokens <- struct{}{}:
			default:
			}
		}
	}()
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-tokens:
				next.ServeHTTP(w, r)
			default:
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			}
		})
	}
}

// LoggingMiddleware creates a request logging middleware
func LoggingMiddleware() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			wrapped := &statusResponseWriter{
				ResponseWriter: w,
				statusCode:     200,
			}
			
			next.ServeHTTP(wrapped, r)
			
			duration := time.Since(start)
			fmt.Printf("[HTTP] %s %s %d %v - %s\n", 
				r.Method, r.URL.Path, wrapped.statusCode, duration, r.UserAgent())
		})
	}
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			w.Header().Set("Content-Security-Policy", "default-src 'self'")
			
			next.ServeHTTP(w, r)
		})
	}
}

// TimeoutMiddleware creates a request timeout middleware
func TimeoutMiddleware(timeout time.Duration) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, timeout, "Request timeout")
	}
}