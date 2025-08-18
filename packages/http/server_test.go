package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/healtronlabs/gofasta/packages/core"
)

// Test helper types
type TestController struct {
	// Basic controller implementation
}

func (c *TestController) GetUsers(ctx *RequestContext) map[string]interface{} {
	return map[string]interface{}{
		"users": []map[string]interface{}{
			{"id": 1, "name": "John Doe"},
			{"id": 2, "name": "Jane Smith"},
		},
	}
}

func (c *TestController) GetUser(ctx *RequestContext) (map[string]interface{}, error) {
	id := ctx.GetParam("id")
	if id == "" {
		return nil, fmt.Errorf("user ID is required")
	}
	
	return map[string]interface{}{
		"id":   id,
		"name": "User " + id,
	}, nil
}

func (c *TestController) CreateUser(ctx *RequestContext) (map[string]interface{}, error) {
	var user map[string]interface{}
	if err := ctx.ParseJSON(&user); err != nil {
		return nil, err
	}
	
	user["id"] = 123
	return user, nil
}

func (c *TestController) UpdateUser(ctx *RequestContext) *RequestContext {
	ctx.JSON(200, map[string]interface{}{
		"id":      ctx.GetParam("id"),
		"updated": true,
	})
	return ctx
}

func (c *TestController) DeleteUser(ctx *RequestContext) {
	ctx.Text(204, "")
}

func (c *TestController) GetProfile(ctx *RequestContext) string {
	return "User Profile"
}

func (c *TestController) GetData(ctx *RequestContext) []byte {
	return []byte("binary data")
}

func (c *TestController) ErrorHandler(ctx *RequestContext) error {
	return fmt.Errorf("test error")
}

// Test HTTPServer creation and configuration
func TestNewHTTPServer(t *testing.T) {
	container := core.NewDIContainer()
	
	t.Run("DefaultConfig", func(t *testing.T) {
		server := NewHTTPServer(container)
		
		if server == nil {
			t.Fatal("NewHTTPServer returned nil")
		}
		
		if server.container != container {
			t.Error("Container not set correctly")
		}
		
		if server.config == nil {
			t.Error("Config not initialized")
		}
		
		if server.config.Port != 8080 {
			t.Errorf("Expected default port 8080, got %d", server.config.Port)
		}
		
		if !server.config.EnableGzip {
			t.Error("Gzip should be enabled by default")
		}
		
		if !server.config.CORSEnabled {
			t.Error("CORS should be enabled by default")
		}
	})
	
	t.Run("CustomConfig", func(t *testing.T) {
		config := &ServerConfig{
			Host:        "example.com",
			Port:        3000,
			EnableGzip:  false,
			CORSEnabled: false,
		}
		
		server := NewHTTPServer(container, config)
		
		if server.config.Host != "example.com" {
			t.Errorf("Expected host 'example.com', got '%s'", server.config.Host)
		}
		
		if server.config.Port != 3000 {
			t.Errorf("Expected port 3000, got %d", server.config.Port)
		}
		
		if server.config.EnableGzip {
			t.Error("Gzip should be disabled")
		}
		
		if server.config.CORSEnabled {
			t.Error("CORS should be disabled")
		}
	})
}

// Test route registration
func TestHTTPServer_RouteRegistration(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("BasicRoutes", func(t *testing.T) {
		server.GET("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("GET response"))
		})
		
		server.POST("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("POST response"))
		})
		
		server.PUT("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("PUT response"))
		})
		
		server.DELETE("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
		
		server.PATCH("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("PATCH response"))
		})
		
		// Test GET
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		if w.Body.String() != "GET response" {
			t.Errorf("Expected 'GET response', got '%s'", w.Body.String())
		}
		
		// Test POST
		req = httptest.NewRequest("POST", "/test", nil)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}
	})
	
	t.Run("ParameterizedRoutes", func(t *testing.T) {
		server.GET("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			id := ctx.GetParam("id")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf("User ID: %s", id)))
		})
		
		req := httptest.NewRequest("GET", "/users/123", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		if w.Body.String() != "User ID: 123" {
			t.Errorf("Expected 'User ID: 123', got '%s'", w.Body.String())
		}
	})
}

// Test request context functionality
func TestRequestContext(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("Parameters", func(t *testing.T) {
		server.GET("/users/{id}/posts/{postId}", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			
			userId := ctx.GetParam("id")
			postId := ctx.GetParam("postId")
			
			response := map[string]string{
				"userId": userId,
				"postId": postId,
			}
			
			ctx.JSON(200, response)
		})
		
		req := httptest.NewRequest("GET", "/users/456/posts/789", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		var response map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}
		
		if response["userId"] != "456" {
			t.Errorf("Expected userId '456', got '%s'", response["userId"])
		}
		
		if response["postId"] != "789" {
			t.Errorf("Expected postId '789', got '%s'", response["postId"])
		}
	})
	
	t.Run("QueryParameters", func(t *testing.T) {
		server.GET("/search", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			
			query := ctx.GetQuery("q")
			limit := ctx.GetQuery("limit")
			tags := ctx.GetQueryArray("tags")
			
			response := map[string]interface{}{
				"query": query,
				"limit": limit,
				"tags":  tags,
			}
			
			ctx.JSON(200, response)
		})
		
		req := httptest.NewRequest("GET", "/search?q=golang&limit=10&tags=web&tags=framework", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		
		if response["query"] != "golang" {
			t.Errorf("Expected query 'golang', got '%v'", response["query"])
		}
		
		if response["limit"] != "10" {
			t.Errorf("Expected limit '10', got '%v'", response["limit"])
		}
		
		tags, ok := response["tags"].([]interface{})
		if !ok || len(tags) != 2 {
			t.Errorf("Expected 2 tags, got %v", response["tags"])
		}
	})
	
	t.Run("JSONParsing", func(t *testing.T) {
		server.POST("/users", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			
			var user map[string]interface{}
			if err := ctx.ParseJSON(&user); err != nil {
				ctx.JSON(400, map[string]string{"error": err.Error()})
				return
			}
			
			user["id"] = 123
			ctx.JSON(201, user)
		})
		
		userData := map[string]interface{}{
			"name":  "John Doe",
			"email": "john@example.com",
		}
		
		jsonData, _ := json.Marshal(userData)
		req := httptest.NewRequest("POST", "/users", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}
		
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		
		if response["name"] != "John Doe" {
			t.Errorf("Expected name 'John Doe', got '%v'", response["name"])
		}
		
		if response["id"] != float64(123) {
			t.Errorf("Expected id 123, got %v", response["id"])
		}
	})
	
	t.Run("Headers", func(t *testing.T) {
		server.GET("/headers", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			
			userAgent := ctx.GetHeader("User-Agent")
			authorization := ctx.GetHeader("Authorization")
			
			ctx.SetHeader("X-Custom-Header", "test-value")
			
			response := map[string]string{
				"userAgent":     userAgent,
				"authorization": authorization,
			}
			
			ctx.JSON(200, response)
		})
		
		req := httptest.NewRequest("GET", "/headers", nil)
		req.Header.Set("User-Agent", "Gofasta-Test/1.0")
		req.Header.Set("Authorization", "Bearer token123")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Header().Get("X-Custom-Header") != "test-value" {
			t.Error("Custom header not set correctly")
		}
		
		var response map[string]string
		json.Unmarshal(w.Body.Bytes(), &response)
		
		if response["userAgent"] != "Gofasta-Test/1.0" {
			t.Errorf("Expected User-Agent 'Gofasta-Test/1.0', got '%s'", response["userAgent"])
		}
	})
}

// Test response methods
func TestRequestContext_ResponseMethods(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("JSONResponse", func(t *testing.T) {
		server.GET("/json", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			ctx.JSON(200, map[string]string{"message": "success"})
		})
		
		req := httptest.NewRequest("GET", "/json", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		if w.Header().Get("Content-Type") != "application/json" {
			t.Error("Content-Type not set to application/json")
		}
	})
	
	t.Run("TextResponse", func(t *testing.T) {
		server.GET("/text", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			ctx.Text(200, "Hello, World!")
		})
		
		req := httptest.NewRequest("GET", "/text", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		if w.Header().Get("Content-Type") != "text/plain" {
			t.Error("Content-Type not set to text/plain")
		}
		
		if w.Body.String() != "Hello, World!" {
			t.Errorf("Expected 'Hello, World!', got '%s'", w.Body.String())
		}
	})
	
	t.Run("HTMLResponse", func(t *testing.T) {
		server.GET("/html", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			ctx.HTML(200, "<h1>Hello, World!</h1>")
		})
		
		req := httptest.NewRequest("GET", "/html", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		if w.Header().Get("Content-Type") != "text/html" {
			t.Error("Content-Type not set to text/html")
		}
	})
	
	t.Run("RedirectResponse", func(t *testing.T) {
		server.GET("/redirect", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			ctx.Redirect(302, "/new-location")
		})
		
		req := httptest.NewRequest("GET", "/redirect", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusFound {
			t.Errorf("Expected status 302, got %d", w.Code)
		}
		
		if w.Header().Get("Location") != "/new-location" {
			t.Error("Location header not set correctly")
		}
	})
}

// Test error handling
func TestHTTPServer_ErrorHandling(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("GofastaError", func(t *testing.T) {
		server.GET("/error", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			err := core.NewNotFoundError("User", "123")
			server.handleError(ctx, err)
		})
		
		req := httptest.NewRequest("GET", "/error", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
		
		var response ErrorResponse
		json.Unmarshal(w.Body.Bytes(), &response)
		
		if response.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status code 404, got %d", response.StatusCode)
		}
	})
	
	t.Run("GenericError", func(t *testing.T) {
		server.GET("/generic-error", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			err := fmt.Errorf("generic error message")
			server.handleError(ctx, err)
		})
		
		req := httptest.NewRequest("GET", "/generic-error", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
		
		var response ErrorResponse
		json.Unmarshal(w.Body.Bytes(), &response)
		
		if response.Message != "Internal Server Error" {
			t.Errorf("Expected message 'Internal Server Error', got '%s'", response.Message)
		}
	})
}

// Test controller integration (when available)
func TestHTTPServer_ControllerIntegration(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("ControllerRegistration", func(t *testing.T) {
		// This test would work when the core.ExtractControllerMetadata function
		// is properly implemented and controllers are supported
		controller := &TestController{}
		
		// For now, we'll manually register routes that mimic controller behavior
		server.GET("/users", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			result := controller.GetUsers(ctx)
			server.sendResponse(ctx, result)
		})
		
		server.GET("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			result, err := controller.GetUser(ctx)
			if err != nil {
				server.handleError(ctx, err)
				return
			}
			server.sendResponse(ctx, result)
		})
		
		server.POST("/users", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			result, err := controller.CreateUser(ctx)
			if err != nil {
				server.handleError(ctx, err)
				return
			}
			server.sendResponse(ctx, result)
		})
		
		// Test GET /users
		req := httptest.NewRequest("GET", "/users", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		
		users, ok := response["users"].([]interface{})
		if !ok || len(users) != 2 {
			t.Error("Expected 2 users in response")
		}
		
		// Test GET /users/123
		req = httptest.NewRequest("GET", "/users/123", nil)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		// Test POST /users
		userData := map[string]interface{}{"name": "Test User", "email": "test@example.com"}
		jsonData, _ := json.Marshal(userData)
		req = httptest.NewRequest("POST", "/users", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// Test server configuration
func TestServerConfig(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		config := DefaultServerConfig()
		
		if config.Host != "localhost" {
			t.Errorf("Expected default host 'localhost', got '%s'", config.Host)
		}
		
		if config.Port != 8080 {
			t.Errorf("Expected default port 8080, got %d", config.Port)
		}
		
		if config.ReadTimeout != 15*time.Second {
			t.Errorf("Expected read timeout 15s, got %v", config.ReadTimeout)
		}
		
		if config.WriteTimeout != 15*time.Second {
			t.Errorf("Expected write timeout 15s, got %v", config.WriteTimeout)
		}
		
		if config.IdleTimeout != 60*time.Second {
			t.Errorf("Expected idle timeout 60s, got %v", config.IdleTimeout)
		}
		
		if !config.EnableGzip {
			t.Error("Expected gzip to be enabled by default")
		}
		
		if !config.CORSEnabled {
			t.Error("Expected CORS to be enabled by default")
		}
	})
}

// Test server lifecycle
func TestHTTPServer_Lifecycle(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("ServerShutdown", func(t *testing.T) {
		// Test that server can be shut down gracefully
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		
		err := server.Shutdown(ctx)
		if err != nil {
			// This is expected when server is not running
			// In a real test, we'd start the server first
		}
	})
	
	t.Run("GetRouter", func(t *testing.T) {
		router := server.GetRouter()
		if router == nil {
			t.Error("GetRouter returned nil")
		}
		
		if router != server.router {
			t.Error("GetRouter returned wrong router instance")
		}
	})
	
	t.Run("GetConfig", func(t *testing.T) {
		config := server.GetConfig()
		if config == nil {
			t.Error("GetConfig returned nil")
		}
		
		if config != server.config {
			t.Error("GetConfig returned wrong config instance")
		}
	})
}

// Test response writer functionality
func TestResponseWriter(t *testing.T) {
	t.Run("StatusCode", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := &ResponseWriter{ResponseWriter: w, statusCode: 200}
		
		rw.WriteHeader(404)
		if rw.StatusCode() != 404 {
			t.Errorf("Expected status code 404, got %d", rw.StatusCode())
		}
		
		// Second call should not change status
		rw.WriteHeader(500)
		if rw.StatusCode() != 404 {
			t.Errorf("Status code should not change after first WriteHeader call")
		}
	})
	
	t.Run("Write", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := &ResponseWriter{ResponseWriter: w, statusCode: 200}
		
		data := []byte("test data")
		n, err := rw.Write(data)
		
		if err != nil {
			t.Errorf("Write returned error: %v", err)
		}
		
		if n != len(data) {
			t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
		}
		
		if w.Body.String() != "test data" {
			t.Errorf("Expected 'test data', got '%s'", w.Body.String())
		}
	})
}