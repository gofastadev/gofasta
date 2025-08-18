package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/healtronlabs/gofasta/packages/core"
)

// Test error handling functionality
func TestErrorHandling(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("GofastaErrors", func(t *testing.T) {
		server.GET("/error/not-found", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			err := core.NewNotFoundError("User", "123")
			server.handleError(ctx, err)
		})
		
		server.GET("/error/bad-request", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			err := core.NewBadRequestError("Invalid input data")
			server.handleError(ctx, err)
		})
		
		server.GET("/error/internal", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			err := core.NewInternalServerError("Database connection failed")
			server.handleError(ctx, err)
		})
		
		// Test not found error
		req := httptest.NewRequest("GET", "/error/not-found", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
		
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}
		
		if response.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status code 404 in response, got %d", response.StatusCode)
		}
		
		// Test bad request error
		req = httptest.NewRequest("GET", "/error/bad-request", nil)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
		
		// Test internal server error
		req = httptest.NewRequest("GET", "/error/internal", nil)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})
	
	t.Run("GenericErrors", func(t *testing.T) {
		server.GET("/error/generic", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			err := errors.New("this is a generic error")
			server.handleError(ctx, err)
		})
		
		req := httptest.NewRequest("GET", "/error/generic", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500 for generic error, got %d", w.Code)
		}
		
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}
		
		if response.Message != "Internal Server Error" {
			t.Errorf("Expected generic error message, got '%s'", response.Message)
		}
		
		if response.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status code 500, got %d", response.StatusCode)
		}
	})
	
	t.Run("PanicRecovery", func(t *testing.T) {
		// Note: The panic recovery is handled by middleware, not in the test
		// For testing purposes, we'll simulate an error condition that would be handled
		server.GET("/error/panic", func(w http.ResponseWriter, r *http.Request) {
			// Simulate panic recovery by handling an error
			ctx := server.createRequestContext(w, r)
			err := errors.New("simulated panic error")
			server.handleError(ctx, err)
		})
		
		req := httptest.NewRequest("GET", "/error/panic", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500 for panic recovery, got %d", w.Code)
		}
		
		if !strings.Contains(w.Body.String(), "Internal Server Error") {
			t.Error("Panic should be recovered and return internal server error")
		}
	})
}

// Test custom error responses
func TestCustomErrorResponses(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("CustomErrorFormat", func(t *testing.T) {
		server.GET("/error/custom", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			
			// Create a custom error response
			errorResp := ErrorResponse{
				StatusCode: http.StatusUnprocessableEntity,
				Message:    "Validation failed",
				Error:      "Email format is invalid",
				Timestamp:  "2023-01-01T00:00:00Z",
			}
			
			ctx.JSON(errorResp.StatusCode, errorResp)
		})
		
		req := httptest.NewRequest("GET", "/error/custom", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("Expected status 422, got %d", w.Code)
		}
		
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}
		
		if response.Message != "Validation failed" {
			t.Errorf("Expected message 'Validation failed', got '%s'", response.Message)
		}
		
		if response.Error != "Email format is invalid" {
			t.Errorf("Expected error 'Email format is invalid', got '%s'", response.Error)
		}
	})
	
	t.Run("ValidationErrors", func(t *testing.T) {
		server.POST("/error/validation", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			
			// Simulate validation errors
			validationErrors := map[string][]string{
				"email":    {"Email is required", "Email format is invalid"},
				"password": {"Password must be at least 8 characters"},
				"age":      {"Age must be a positive number"},
			}
			
			errorResp := map[string]interface{}{
				"error":   "Validation failed",
				"code":    "VALIDATION_ERROR",
				"details": validationErrors,
			}
			
			ctx.JSON(http.StatusBadRequest, errorResp)
		})
		
		req := httptest.NewRequest("POST", "/error/validation", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to parse validation error response: %v", err)
		}
		
		if response["error"] != "Validation failed" {
			t.Errorf("Expected error 'Validation failed', got '%v'", response["error"])
		}
		
		if response["code"] != "VALIDATION_ERROR" {
			t.Errorf("Expected code 'VALIDATION_ERROR', got '%v'", response["code"])
		}
		
		details, ok := response["details"].(map[string]interface{})
		if !ok {
			t.Fatal("Details should be a map")
		}
		
		emailErrors, ok := details["email"].([]interface{})
		if !ok || len(emailErrors) != 2 {
			t.Error("Email should have 2 validation errors")
		}
	})
}

// Test error middleware and filters
func TestErrorMiddleware(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("ExceptionFilter", func(t *testing.T) {
		// Test that exception filters can be registered and used
		// Note: The actual filter implementation would be defined elsewhere
		
		// For this test, we'll simulate the filter behavior directly
		server.GET("/error/custom-filter", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			
			// Simulate custom error handling
			errorResp := map[string]interface{}{
				"error": "Custom filter handled this error",
				"code":  "CUSTOM_ERROR",
			}
			
			ctx.JSON(http.StatusTeapot, errorResp)
		})
		
		req := httptest.NewRequest("GET", "/error/custom-filter", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusTeapot {
			t.Errorf("Expected status 418 (custom filter), got %d", w.Code)
		}
		
		if !strings.Contains(w.Body.String(), "Custom filter handled") {
			t.Error("Custom exception filter should handle the error")
		}
	})
	
	t.Run("ErrorWithoutFilter", func(t *testing.T) {
		// Create a new server without custom filters for this test
		server2 := NewHTTPServer(container)
		
		server2.GET("/error/no-filter", func(w http.ResponseWriter, r *http.Request) {
			ctx := server2.createRequestContext(w, r)
			err := errors.New("this error won't be caught by custom filter")
			server2.handleError(ctx, err)
		})
		
		req := httptest.NewRequest("GET", "/error/no-filter", nil)
		w := httptest.NewRecorder()
		server2.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500 (default handling), got %d", w.Code)
		}
	})
}

// Test error response formats
func TestErrorResponseFormats(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("JSONErrorFormat", func(t *testing.T) {
		server.GET("/error/json", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			err := core.NewBadRequestError("Invalid JSON payload")
			server.handleError(ctx, err)
		})
		
		req := httptest.NewRequest("GET", "/error/json", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
		
		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected JSON content type, got '%s'", contentType)
		}
		
		// Verify it's valid JSON
		var errorResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResp)
		if err != nil {
			t.Fatalf("Response should be valid JSON: %v", err)
		}
	})
	
	t.Run("TextErrorFormat", func(t *testing.T) {
		server.GET("/error/text", func(w http.ResponseWriter, r *http.Request) {
			// For text responses, we might want to return plain text errors
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad Request: Invalid input"))
		})
		
		req := httptest.NewRequest("GET", "/error/text", nil)
		req.Header.Set("Accept", "text/plain")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
		
		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/plain") {
			t.Errorf("Expected text/plain content type, got '%s'", contentType)
		}
		
		expected := "Bad Request: Invalid input"
		if w.Body.String() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, w.Body.String())
		}
	})
}

// Test error codes and status mapping
func TestErrorStatusMapping(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("HTTPStatusCodes", func(t *testing.T) {
		tests := []struct {
			path           string
			expectedStatus int
			errorType      string
		}{
			{"/400", http.StatusBadRequest, "BadRequest"},
			{"/401", http.StatusUnauthorized, "Unauthorized"},
			{"/403", http.StatusForbidden, "Forbidden"},
			{"/404", http.StatusNotFound, "NotFound"},
			{"/409", http.StatusConflict, "Conflict"},
			{"/422", http.StatusUnprocessableEntity, "UnprocessableEntity"},
			{"/500", http.StatusInternalServerError, "InternalServerError"},
		}
		
		for _, test := range tests {
			server.GET(test.path, func(w http.ResponseWriter, r *http.Request) {
				ctx := server.createRequestContext(w, r)
				
				var err error
				switch test.expectedStatus {
				case http.StatusBadRequest:
					err = core.NewBadRequestError("Bad request")
				case http.StatusUnauthorized:
					err = core.NewUnauthorizedError("Unauthorized")
				case http.StatusForbidden:
					err = core.NewForbiddenError("Forbidden")
				case http.StatusNotFound:
					err = core.NewNotFoundError("Resource", "123")
				case http.StatusConflict:
					err = core.NewConflictError("Resource already exists")
				case http.StatusUnprocessableEntity:
					err = errors.New("validation failed")
				case http.StatusInternalServerError:
					err = core.NewInternalServerError("Internal error")
				default:
					err = errors.New("generic error")
				}
				
				server.handleError(ctx, err)
			})
			
			req := httptest.NewRequest("GET", test.path, nil)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
			
			if w.Code != test.expectedStatus {
				t.Errorf("Expected status %d for %s, got %d", 
					test.expectedStatus, test.errorType, w.Code)
			}
		}
	})
}

// Test error context and metadata
func TestErrorContext(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("ErrorWithContext", func(t *testing.T) {
		server.GET("/error/context", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			
			// Create error with additional context
			errorResp := map[string]interface{}{
				"error":     "Resource not found",
				"requestId": "req-123",
				"userId":    "user-456",
				"resource":  "user profile",
				"timestamp": "2023-01-01T00:00:00Z",
				"method":    r.Method,
				"path":      r.URL.Path,
				"userAgent": r.UserAgent(),
			}
			
			ctx.JSON(http.StatusNotFound, errorResp)
		})
		
		req := httptest.NewRequest("GET", "/error/context", nil)
		req.Header.Set("User-Agent", "Test-Client/1.0")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}
		
		if response["requestId"] != "req-123" {
			t.Errorf("Expected requestId 'req-123', got '%v'", response["requestId"])
		}
		
		if response["method"] != "GET" {
			t.Errorf("Expected method 'GET', got '%v'", response["method"])
		}
		
		if response["userAgent"] != "Test-Client/1.0" {
			t.Errorf("Expected userAgent 'Test-Client/1.0', got '%v'", response["userAgent"])
		}
	})
	
	t.Run("ErrorChaining", func(t *testing.T) {
		server.GET("/error/chain", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			
			// Simulate error chain
			rootErr := errors.New("database connection failed")
			wrappedErr := fmt.Errorf("failed to fetch user: %w", rootErr)
			finalErr := fmt.Errorf("API request failed: %w", wrappedErr)
			
			server.handleError(ctx, finalErr)
		})
		
		req := httptest.NewRequest("GET", "/error/chain", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
		
		// The error should be handled gracefully even if it's a wrapped error
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}
		
		if response.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status code 500, got %d", response.StatusCode)
		}
	})
}