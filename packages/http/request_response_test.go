package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/healtronlabs/gofasta/packages/core"
)

// Test request context creation and functionality
func TestRequestContextCreation(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("BasicContextCreation", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		
		ctx := server.createRequestContext(w, req)
		
		if ctx == nil {
			t.Fatal("Request context should not be nil")
		}
		
		if ctx.Request != req {
			t.Error("Request should be set in context")
		}
		
		if ctx.Response != w {
			t.Error("Response writer should be set in context")
		}
		
		if ctx.Params == nil {
			t.Error("Params map should be initialized")
		}
		
		if ctx.Query == nil {
			t.Error("Query map should be initialized")
		}
		
		// Headers are accessed via Request.Header, not a separate map
		if ctx.Request.Header == nil {
			t.Error("Request headers should be accessible")
		}
	})
	
	t.Run("ContextWithParameters", func(t *testing.T) {
		// Register a route with parameters
		server.GET("/users/{id}/posts/{postId}", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			
			userID := ctx.GetParam("id")
			postID := ctx.GetParam("postId")
			
			response := map[string]string{
				"userId": userID,
				"postId": postID,
			}
			
			ctx.JSON(200, response)
		})
		
		req := httptest.NewRequest("GET", "/users/123/posts/456", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}
		
		if response["userId"] != "123" {
			t.Errorf("Expected userId '123', got '%s'", response["userId"])
		}
		
		if response["postId"] != "456" {
			t.Errorf("Expected postId '456', got '%s'", response["postId"])
		}
	})
	
	t.Run("ContextWithQueryParameters", func(t *testing.T) {
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
		
		req := httptest.NewRequest("GET", "/search?q=golang&limit=10&tags=web&tags=framework&tags=http", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}
		
		if response["query"] != "golang" {
			t.Errorf("Expected query 'golang', got '%v'", response["query"])
		}
		
		if response["limit"] != "10" {
			t.Errorf("Expected limit '10', got '%v'", response["limit"])
		}
		
		tags, ok := response["tags"].([]interface{})
		if !ok {
			t.Fatal("Tags should be an array")
		}
		
		if len(tags) != 3 {
			t.Errorf("Expected 3 tags, got %d", len(tags))
		}
		
		expectedTags := []string{"web", "framework", "http"}
		for i, expectedTag := range expectedTags {
			if i < len(tags) && tags[i] != expectedTag {
				t.Errorf("Expected tag[%d] to be '%s', got '%v'", i, expectedTag, tags[i])
			}
		}
	})
}

// Test request body parsing
func TestRequestBodyParsing(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("JSONParsing", func(t *testing.T) {
		server.POST("/json", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			
			var data map[string]interface{}
			err := ctx.ParseJSON(&data)
			if err != nil {
				ctx.JSON(400, map[string]string{"error": err.Error()})
				return
			}
			
			data["processed"] = true
			ctx.JSON(200, data)
		})
		
		requestData := map[string]interface{}{
			"name":  "John Doe",
			"email": "john@example.com",
			"age":   30,
		}
		
		jsonData, _ := json.Marshal(requestData)
		req := httptest.NewRequest("POST", "/json", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to parse response JSON: %v", err)
		}
		
		if response["name"] != "John Doe" {
			t.Errorf("Expected name 'John Doe', got '%v'", response["name"])
		}
		
		if response["processed"] != true {
			t.Error("Expected processed to be true")
		}
	})
	
	t.Run("FormParsing", func(t *testing.T) {
		server.POST("/form", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			
			err := r.ParseForm()
			if err != nil {
				ctx.JSON(400, map[string]string{"error": err.Error()})
				return
			}
			
			data := map[string]string{
				"name":  r.FormValue("name"),
				"email": r.FormValue("email"),
			}
			
			ctx.JSON(200, data)
		})
		
		formData := url.Values{}
		formData.Set("name", "Jane Doe")
		formData.Set("email", "jane@example.com")
		
		req := httptest.NewRequest("POST", "/form", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to parse response JSON: %v", err)
		}
		
		if response["name"] != "Jane Doe" {
			t.Errorf("Expected name 'Jane Doe', got '%s'", response["name"])
		}
		
		if response["email"] != "jane@example.com" {
			t.Errorf("Expected email 'jane@example.com', got '%s'", response["email"])
		}
	})
	
	t.Run("RawBodyAccess", func(t *testing.T) {
		server.POST("/raw", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			
			body, err := io.ReadAll(r.Body)
			if err != nil {
				ctx.JSON(400, map[string]string{"error": err.Error()})
				return
			}
			
			response := map[string]interface{}{
				"received": string(body),
				"length":   len(body),
			}
			
			ctx.JSON(200, response)
		})
		
		testData := "This is raw text data"
		req := httptest.NewRequest("POST", "/raw", strings.NewReader(testData))
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to parse response JSON: %v", err)
		}
		
		if response["received"] != testData {
			t.Errorf("Expected received data '%s', got '%v'", testData, response["received"])
		}
		
		if int(response["length"].(float64)) != len(testData) {
			t.Errorf("Expected length %d, got %v", len(testData), response["length"])
		}
	})
}

// Test response generation
func TestResponseGeneration(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("JSONResponse", func(t *testing.T) {
		server.GET("/api/json", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			
			data := map[string]interface{}{
				"message": "success",
				"data": map[string]interface{}{
					"id":   1,
					"name": "Test Item",
				},
				"timestamp": time.Now().Unix(),
			}
			
			ctx.JSON(200, data)
		})
		
		req := httptest.NewRequest("GET", "/api/json", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected Content-Type to contain application/json, got %s", contentType)
		}
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}
		
		if response["message"] != "success" {
			t.Errorf("Expected message 'success', got '%v'", response["message"])
		}
	})
	
	t.Run("TextResponse", func(t *testing.T) {
		server.GET("/api/text", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			ctx.Text(200, "Hello, World! This is a text response.")
		})
		
		req := httptest.NewRequest("GET", "/api/text", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/plain") {
			t.Errorf("Expected Content-Type to contain text/plain, got %s", contentType)
		}
		
		expected := "Hello, World! This is a text response."
		if w.Body.String() != expected {
			t.Errorf("Expected body '%s', got '%s'", expected, w.Body.String())
		}
	})
	
	t.Run("HTMLResponse", func(t *testing.T) {
		server.GET("/api/html", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			html := `<html><head><title>Test</title></head><body><h1>Hello, World!</h1></body></html>`
			ctx.HTML(200, html)
		})
		
		req := httptest.NewRequest("GET", "/api/html", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/html") {
			t.Errorf("Expected Content-Type to contain text/html, got %s", contentType)
		}
		
		if !strings.Contains(w.Body.String(), "<h1>Hello, World!</h1>") {
			t.Error("HTML content not found in response")
		}
	})
	
	t.Run("RedirectResponse", func(t *testing.T) {
		server.GET("/api/redirect", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			ctx.Redirect(302, "/api/target")
		})
		
		req := httptest.NewRequest("GET", "/api/redirect", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusFound {
			t.Errorf("Expected status 302, got %d", w.Code)
		}
		
		location := w.Header().Get("Location")
		if location != "/api/target" {
			t.Errorf("Expected Location '/api/target', got '%s'", location)
		}
	})
	
	t.Run("CustomHeaders", func(t *testing.T) {
		server.GET("/api/headers", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			
			ctx.SetHeader("X-Custom-Header", "custom-value")
			ctx.SetHeader("X-Request-ID", "req-123")
			ctx.SetHeader("X-API-Version", "1.0")
			
			ctx.JSON(200, map[string]string{"message": "headers set"})
		})
		
		req := httptest.NewRequest("GET", "/api/headers", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		expectedHeaders := map[string]string{
			"X-Custom-Header": "custom-value",
			"X-Request-ID":    "req-123",
			"X-API-Version":   "1.0",
		}
		
		for header, expectedValue := range expectedHeaders {
			actualValue := w.Header().Get(header)
			if actualValue != expectedValue {
				t.Errorf("Expected header %s to be '%s', got '%s'", header, expectedValue, actualValue)
			}
		}
	})
}

// Test header handling
func TestHeaderHandling(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("RequestHeaders", func(t *testing.T) {
		server.GET("/headers/request", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			
			userAgent := ctx.GetHeader("User-Agent")
			authorization := ctx.GetHeader("Authorization")
			customHeader := ctx.GetHeader("X-Custom-Request")
			
			response := map[string]string{
				"userAgent":    userAgent,
				"authorization": authorization,
				"custom":       customHeader,
			}
			
			ctx.JSON(200, response)
		})
		
		req := httptest.NewRequest("GET", "/headers/request", nil)
		req.Header.Set("User-Agent", "Gofasta-Test/1.0")
		req.Header.Set("Authorization", "Bearer token123")
		req.Header.Set("X-Custom-Request", "test-value")
		
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}
		
		if response["userAgent"] != "Gofasta-Test/1.0" {
			t.Errorf("Expected User-Agent 'Gofasta-Test/1.0', got '%s'", response["userAgent"])
		}
		
		if response["authorization"] != "Bearer token123" {
			t.Errorf("Expected Authorization 'Bearer token123', got '%s'", response["authorization"])
		}
		
		if response["custom"] != "test-value" {
			t.Errorf("Expected custom header 'test-value', got '%s'", response["custom"])
		}
	})
	
	t.Run("ResponseHeaders", func(t *testing.T) {
		server.GET("/headers/response", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			
			// Set various response headers
			ctx.SetHeader("Cache-Control", "no-cache")
			ctx.SetHeader("X-Frame-Options", "DENY")
			ctx.SetHeader("X-Content-Type-Options", "nosniff")
			
			ctx.Text(200, "Headers set")
		})
		
		req := httptest.NewRequest("GET", "/headers/response", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		expectedHeaders := map[string]string{
			"Cache-Control":            "no-cache",
			"X-Frame-Options":          "DENY",
			"X-Content-Type-Options":   "nosniff",
		}
		
		for header, expectedValue := range expectedHeaders {
			actualValue := w.Header().Get(header)
			if actualValue != expectedValue {
				t.Errorf("Expected header %s to be '%s', got '%s'", header, expectedValue, actualValue)
			}
		}
	})
}

// Test cookie handling
func TestCookieHandling(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("SetCookies", func(t *testing.T) {
		server.GET("/cookies/set", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			
			// Set various types of cookies
			sessionCookie := &http.Cookie{
				Name:  "session_id",
				Value: "abc123",
				Path:  "/",
			}
			
			persistentCookie := &http.Cookie{
				Name:     "user_pref",
				Value:    "dark_mode",
				Path:     "/",
				Expires:  time.Now().Add(24 * time.Hour),
				HttpOnly: true,
				Secure:   true,
			}
			
			http.SetCookie(w, sessionCookie)
			http.SetCookie(w, persistentCookie)
			
			ctx.JSON(200, map[string]string{"message": "cookies set"})
		})
		
		req := httptest.NewRequest("GET", "/cookies/set", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		cookies := w.Result().Cookies()
		if len(cookies) != 2 {
			t.Errorf("Expected 2 cookies, got %d", len(cookies))
		}
		
		cookieMap := make(map[string]*http.Cookie)
		for _, cookie := range cookies {
			cookieMap[cookie.Name] = cookie
		}
		
		if sessionCookie, ok := cookieMap["session_id"]; ok {
			if sessionCookie.Value != "abc123" {
				t.Errorf("Expected session_id value 'abc123', got '%s'", sessionCookie.Value)
			}
		} else {
			t.Error("session_id cookie not found")
		}
		
		if userPrefCookie, ok := cookieMap["user_pref"]; ok {
			if userPrefCookie.Value != "dark_mode" {
				t.Errorf("Expected user_pref value 'dark_mode', got '%s'", userPrefCookie.Value)
			}
			if !userPrefCookie.HttpOnly {
				t.Error("user_pref cookie should be HttpOnly")
			}
		} else {
			t.Error("user_pref cookie not found")
		}
	})
	
	t.Run("ReadCookies", func(t *testing.T) {
		server.GET("/cookies/read", func(w http.ResponseWriter, r *http.Request) {
			ctx := server.createRequestContext(w, r)
			
			sessionCookie, err := r.Cookie("session_id")
			var sessionValue string
			if err == nil {
				sessionValue = sessionCookie.Value
			}
			
			userPrefCookie, err := r.Cookie("user_pref")
			var userPrefValue string
			if err == nil {
				userPrefValue = userPrefCookie.Value
			}
			
			response := map[string]string{
				"session":   sessionValue,
				"user_pref": userPrefValue,
			}
			
			ctx.JSON(200, response)
		})
		
		req := httptest.NewRequest("GET", "/cookies/read", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "xyz789"})
		req.AddCookie(&http.Cookie{Name: "user_pref", Value: "light_mode"})
		
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}
		
		if response["session"] != "xyz789" {
			t.Errorf("Expected session 'xyz789', got '%s'", response["session"])
		}
		
		if response["user_pref"] != "light_mode" {
			t.Errorf("Expected user_pref 'light_mode', got '%s'", response["user_pref"])
		}
	})
}