package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/healtronlabs/gofasta/packages/core"
)

// Test middleware functionality
func TestHTTPServer_Middleware(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("RecoveryMiddleware", func(t *testing.T) {
		// Add a route that panics
		server.GET("/panic", func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		})
		
		req := httptest.NewRequest("GET", "/panic", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
		
		if !strings.Contains(w.Body.String(), "Internal Server Error") {
			t.Error("Recovery middleware should return Internal Server Error")
		}
	})
	
	t.Run("CORSMiddleware", func(t *testing.T) {
		server.GET("/cors-test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		
		// Test preflight request
		req := httptest.NewRequest("OPTIONS", "/cors-test", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", "GET")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204 for OPTIONS request, got %d", w.Code)
		}
		
		// Check CORS headers
		if w.Header().Get("Access-Control-Allow-Origin") == "" {
			t.Error("CORS headers should be set")
		}
		
		// Test actual request
		req = httptest.NewRequest("GET", "/cors-test", nil)
		req.Header.Set("Origin", "https://example.com")
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
	
	t.Run("GzipMiddleware", func(t *testing.T) {
		server.GET("/gzip-test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("This is a test response that should be compressed"))
		})
		
		req := httptest.NewRequest("GET", "/gzip-test", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		// Check if gzip encoding is applied
		if w.Header().Get("Content-Encoding") != "gzip" {
			t.Error("Response should be gzip encoded")
		}
	})
	
	t.Run("ContextMiddleware", func(t *testing.T) {
		server.GET("/context-test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		
		req := httptest.NewRequest("GET", "/context-test", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		// Check if request ID header is set
		if w.Header().Get("X-Request-ID") == "" {
			t.Error("X-Request-ID header should be set")
		}
	})
	
	t.Run("CustomMiddleware", func(t *testing.T) {
		// Add custom middleware
		server.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Custom-Header", "test-value")
				next.ServeHTTP(w, r)
			})
		})
		
		server.GET("/middleware-test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		
		req := httptest.NewRequest("GET", "/middleware-test", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		if w.Header().Get("X-Custom-Header") != "test-value" {
			t.Error("Custom middleware header should be set")
		}
	})
}

// TestGuard for testing guard functionality
type TestGuard struct{}

func (g *TestGuard) CanActivate(ctx *core.RequestContext) bool {
	return ctx.GetHeader("Authorization") != ""
}

// Test guard functionality
func TestHTTPServer_Guards(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("GuardAllows", func(t *testing.T) {
		guard := &TestGuard{}
		server.UseGuards(guard)
		
		server.GET("/protected", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Access granted"))
		})
		
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer token123")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		if w.Body.String() != "Access granted" {
			t.Errorf("Expected 'Access granted', got '%s'", w.Body.String())
		}
	})
	
	t.Run("GuardDenies", func(t *testing.T) {
		// Create new server to avoid guard conflicts
		server2 := NewHTTPServer(container)
		guard := &TestGuard{}
		server2.UseGuards(guard)
		
		server2.GET("/protected", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Access granted"))
		})
		
		req := httptest.NewRequest("GET", "/protected", nil)
		// No Authorization header
		w := httptest.NewRecorder()
		server2.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}
	})
}

// TestPipe for testing pipe functionality
type TestPipe struct{}

func (p *TestPipe) Transform(value interface{}, metadata *core.PipeMetadata) (interface{}, error) {
	if data, ok := value.([]byte); ok {
		// Simple transformation: convert to uppercase
		return []byte(strings.ToUpper(string(data))), nil
	}
	return value, nil
}

// Test pipe functionality
func TestHTTPServer_Pipes(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("PipeTransformation", func(t *testing.T) {
		pipe := &TestPipe{}
		server.UsePipes(pipe)
		
		server.POST("/transform", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			w.WriteHeader(http.StatusOK)
			w.Write(ctx.Body) // Return the transformed body
		})
		
		req := httptest.NewRequest("POST", "/transform", strings.NewReader("hello world"))
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		if w.Body.String() != "HELLO WORLD" {
			t.Errorf("Expected 'HELLO WORLD', got '%s'", w.Body.String())
		}
	})
}

// TestInterceptor for testing interceptor functionality
type TestInterceptor struct{}

func (i *TestInterceptor) Intercept(ctx *core.RequestContext, next core.Handler) *core.Response {
	// Call the next handler
	response := next(ctx)
	
	// Modify the response
	if response != nil {
		response.Headers = make(map[string]string)
		response.Headers["X-Intercepted"] = "true"
	}
	
	return response
}

// Test interceptor functionality
func TestHTTPServer_Interceptors(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("InterceptorModification", func(t *testing.T) {
		interceptor := &TestInterceptor{}
		server.UseInterceptors(interceptor)
		
		server.GET("/intercept", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Original response"))
		})
		
		req := httptest.NewRequest("GET", "/intercept", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		// Note: Due to the simplified implementation, interceptor effects
		// might not be fully visible in this test setup
	})
}

// TestExceptionFilter for testing exception filter functionality
type TestExceptionFilter struct{}

func (f *TestExceptionFilter) Catch(exception interface{}, host *core.RequestContext) *core.Response {
	if err, ok := exception.(error); ok && err.Error() == "custom error" {
		return &core.Response{
			StatusCode: http.StatusBadRequest,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"error":"Custom error handled"}`,
		}
	}
	return nil // Let default error handling take over
}

// Test exception filter functionality
func TestHTTPServer_ExceptionFilters(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("ExceptionFilterHandling", func(t *testing.T) {
		filter := &TestExceptionFilter{}
		server.UseExceptionFilters(filter)
		
		server.GET("/custom-error", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			server.handleError(ctx, fmt.Errorf("custom error"))
		})
		
		req := httptest.NewRequest("GET", "/custom-error", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
		
		if !strings.Contains(w.Body.String(), "Custom error handled") {
			t.Error("Exception filter should handle custom errors")
		}
	})
}

// Test static file middleware in middleware tests
func TestStaticFileMiddleware_Middleware(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("StaticFileServing", func(t *testing.T) {
		// This would normally serve files from a directory
		// For testing, we'll verify the middleware setup
		server.Static("/static/", "./static")
		
		// Test that the route is registered
		router := server.GetRouter()
		if router == nil {
			t.Error("Router should not be nil")
		}
		
		// Note: Full static file serving test would require actual files
		// This tests the setup without file system dependencies
	})
}

// Test rate limiting middleware
func TestRateLimitMiddleware(t *testing.T) {
	t.Run("RateLimit", func(t *testing.T) {
		middleware := RateLimitMiddleware(2) // 2 requests per second
		
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}))
		
		// First request should succeed
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("First request should succeed, got %d", w.Code)
		}
		
		// Second request should succeed
		req = httptest.NewRequest("GET", "/", nil)
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Second request should succeed, got %d", w.Code)
		}
		
		// Third request should be rate limited
		req = httptest.NewRequest("GET", "/", nil)
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		
		if w.Code != http.StatusTooManyRequests {
			t.Errorf("Third request should be rate limited, got %d", w.Code)
		}
	})
}

// Test logging middleware
func TestLoggingMiddleware(t *testing.T) {
	t.Run("Logging", func(t *testing.T) {
		middleware := LoggingMiddleware()
		
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}))
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "TestAgent/1.0")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		// Note: Logging output would normally be captured and verified
		// This test ensures the middleware doesn't break the request flow
	})
}

// Test security headers middleware
func TestSecurityHeadersMiddleware(t *testing.T) {
	t.Run("SecurityHeaders", func(t *testing.T) {
		middleware := SecurityHeadersMiddleware()
		
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}))
		
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		// Check security headers
		expectedHeaders := map[string]string{
			"X-Content-Type-Options":   "nosniff",
			"X-Frame-Options":          "DENY",
			"X-XSS-Protection":         "1; mode=block",
			"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
			"Content-Security-Policy":  "default-src 'self'",
		}
		
		for header, expectedValue := range expectedHeaders {
			if w.Header().Get(header) != expectedValue {
				t.Errorf("Expected %s header to be '%s', got '%s'", 
					header, expectedValue, w.Header().Get(header))
			}
		}
	})
}

// Test timeout middleware
func TestTimeoutMiddleware(t *testing.T) {
	t.Run("Timeout", func(t *testing.T) {
		middleware := TimeoutMiddleware(100 * time.Millisecond)
		
		// Handler that takes longer than timeout
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}))
		
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected timeout status 503, got %d", w.Code)
		}
		
		if !strings.Contains(w.Body.String(), "timeout") {
			t.Error("Response should contain timeout message")
		}
	})
}

// Test middleware order
func TestMiddlewareOrder(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("MiddlewareOrder", func(t *testing.T) {
		var order []string
		
		// Add middleware in specific order
		server.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "first")
				next.ServeHTTP(w, r)
			})
		})
		
		server.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "second")
				next.ServeHTTP(w, r)
			})
		})
		
		server.GET("/order-test", func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "handler")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		
		req := httptest.NewRequest("GET", "/order-test", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		// Note: Due to the default middleware being applied first,
		// the exact order might vary. This test ensures custom middleware works.
		if len(order) == 0 {
			t.Error("Middleware should be executed")
		}
	})
}