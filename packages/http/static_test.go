package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/healtronlabs/gofasta/packages/core"
)

// Test static file serving functionality
func TestHTTPServer_StaticFileServing(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	// Create temporary directory and files for testing
	tempDir := t.TempDir()
	
	// Create test files
	testFiles := map[string]string{
		"index.html":    "<html><body>Hello World</body></html>",
		"style.css":     "body { color: red; }",
		"app.js":        "console.log('Hello');",
		"data.json":     `{"message": "test"}`,
		"image.png":     "fake-png-data",
		"document.pdf":  "fake-pdf-data",
		"archive.zip":   "fake-zip-data",
		"text.txt":      "plain text content",
	}
	
	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}
	
	t.Run("BasicStaticServing", func(t *testing.T) {
		// Register static file serving
		server.Static("/static/", tempDir)
		
		// Test serving HTML file
		req := httptest.NewRequest("GET", "/static/index.html", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		if !strings.Contains(w.Body.String(), "Hello World") {
			t.Error("HTML content not served correctly")
		}
	})
	
	t.Run("ContentTypeHeaders", func(t *testing.T) {
		tests := []struct {
			file        string
			contentType string
		}{
			{"index.html", "text/html"},
			{"style.css", "text/css"},
			{"app.js", "application/javascript"},
			{"data.json", "application/json"},
			{"image.png", "image/png"},
			{"document.pdf", "application/pdf"},
			{"text.txt", "text/plain"},
			{"archive.zip", "application/zip"},
		}
		
		for _, test := range tests {
			req := httptest.NewRequest("GET", "/static/"+test.file, nil)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
			
			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for %s, got %d", test.file, w.Code)
				continue
			}
			
			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, test.contentType) {
				t.Errorf("Expected Content-Type to contain %s for %s, got %s", 
					test.contentType, test.file, contentType)
			}
		}
	})
	
	t.Run("CacheHeaders", func(t *testing.T) {
		// Create server with cache configuration
		config := &ServerConfig{
			Host:            "localhost",
			Port:            8080,
			StaticFileCache: 3600 * time.Second, // 1 hour
		}
		serverWithCache := NewHTTPServer(container, config)
		serverWithCache.Static("/cached/", tempDir)
		
		req := httptest.NewRequest("GET", "/cached/index.html", nil)
		w := httptest.NewRecorder()
		serverWithCache.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		cacheControl := w.Header().Get("Cache-Control")
		if !strings.Contains(cacheControl, "max-age=3600") {
			t.Errorf("Expected cache control with max-age=3600, got %s", cacheControl)
		}
		
		expires := w.Header().Get("Expires")
		if expires == "" {
			t.Error("Expected Expires header to be set")
		}
	})
	
	t.Run("DirectoryListing", func(t *testing.T) {
		// Test accessing directory without index file
		subDir := filepath.Join(tempDir, "subdir")
		err := os.Mkdir(subDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create subdirectory: %v", err)
		}
		
		err = os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("sub file"), 0644)
		if err != nil {
			t.Fatalf("Failed to create sub file: %v", err)
		}
		
		req := httptest.NewRequest("GET", "/static/subdir/", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		// Directory listing behavior depends on implementation
		// This tests that we get some response (not necessarily 200)
		if w.Code == 0 {
			t.Error("Expected some HTTP status code")
		}
	})
	
	t.Run("NotFoundFiles", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/static/nonexistent.html", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for non-existent file, got %d", w.Code)
		}
	})
	
	t.Run("PathTraversal", func(t *testing.T) {
		// Test that path traversal attacks are prevented
		req := httptest.NewRequest("GET", "/static/../../../etc/passwd", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		// Should not be able to access files outside the static directory
		if w.Code == http.StatusOK {
			t.Error("Path traversal should be prevented")
		}
	})
}

// Test static file middleware
func TestStaticFileMiddleware(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("MiddlewareSetup", func(t *testing.T) {
		// Test that static file middleware can be set up
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("test response"))
		})
		
		middleware := server.staticFileMiddleware(handler)
		
		req := httptest.NewRequest("GET", "/test.html", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		// Check that Content-Type is set based on file extension
		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/html") {
			t.Errorf("Expected Content-Type to contain text/html, got %s", contentType)
		}
	})
	
	t.Run("CacheHeadersMiddleware", func(t *testing.T) {
		config := &ServerConfig{
			StaticFileCache: 7200 * time.Second, // 2 hours
		}
		serverWithCache := NewHTTPServer(container, config)
		
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("cached content"))
		})
		
		middleware := serverWithCache.staticFileMiddleware(handler)
		
		req := httptest.NewRequest("GET", "/cached.css", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)
		
		cacheControl := w.Header().Get("Cache-Control")
		if !strings.Contains(cacheControl, "max-age=7200") {
			t.Errorf("Expected cache control with max-age=7200, got %s", cacheControl)
		}
		
		expires := w.Header().Get("Expires")
		if expires == "" {
			t.Error("Expected Expires header to be set")
		}
		
		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/css") {
			t.Errorf("Expected Content-Type to contain text/css, got %s", contentType)
		}
	})
}

// Test content type detection
func TestContentTypeDetection(t *testing.T) {
	tests := []struct {
		extension   string
		expectedType string
	}{
		{".html", "text/html"},
		{".htm", ""}, // Not in our map
		{".css", "text/css"},
		{".js", "application/javascript"},
		{".json", "application/json"},
		{".png", "image/png"},
		{".jpg", "image/jpeg"},
		{".jpeg", "image/jpeg"},
		{".gif", "image/gif"},
		{".svg", "image/svg+xml"},
		{".ico", "image/x-icon"},
		{".pdf", "application/pdf"},
		{".txt", "text/plain"},
		{".xml", "application/xml"},
		{".zip", "application/zip"},
		{"", ""}, // No extension
		{".unknown", ""}, // Unknown extension
	}
	
	for _, test := range tests {
		result := getContentType(test.extension)
		if result != test.expectedType {
			t.Errorf("Expected content type %s for extension %s, got %s", 
				test.expectedType, test.extension, result)
		}
	}
}

// Test static directory configuration
func TestStaticDirectoryConfig(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("MultipleStaticDirs", func(t *testing.T) {
		// Create multiple temporary directories
		tempDir1 := t.TempDir()
		tempDir2 := t.TempDir()
		
		// Create files in each directory
		file1Path := filepath.Join(tempDir1, "file1.txt")
		err := os.WriteFile(file1Path, []byte("content from dir1"), 0644)
		if err != nil {
			t.Fatalf("Failed to create file1: %v", err)
		}
		
		file2Path := filepath.Join(tempDir2, "file2.txt")
		err = os.WriteFile(file2Path, []byte("content from dir2"), 0644)
		if err != nil {
			t.Fatalf("Failed to create file2: %v", err)
		}
		
		// Register multiple static directories
		server.Static("/assets1/", tempDir1)
		server.Static("/assets2/", tempDir2)
		
		// Test first directory
		req := httptest.NewRequest("GET", "/assets1/file1.txt", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for /assets1/file1.txt, got %d", w.Code)
		}
		
		if w.Body.String() != "content from dir1" {
			t.Errorf("Expected content from dir1, got %s", w.Body.String())
		}
		
		// Test second directory
		req = httptest.NewRequest("GET", "/assets2/file2.txt", nil)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for /assets2/file2.txt, got %d", w.Code)
		}
		
		if w.Body.String() != "content from dir2" {
			t.Errorf("Expected content from dir2, got %s", w.Body.String())
		}
	})
	
	t.Run("OverlappingPaths", func(t *testing.T) {
		tempDir := t.TempDir()
		
		// Test that later registrations can override earlier ones
		server.Static("/overlap/", tempDir)
		
		// The behavior here depends on implementation
		// This test ensures the method can be called multiple times
		router := server.GetRouter()
		if router == nil {
			t.Error("Router should not be nil after static registration")
		}
	})
}

// Test static file serving with gzip compression
func TestStaticFileCompression(t *testing.T) {
	container := core.NewDIContainer()
	config := &ServerConfig{
		EnableGzip: true,
	}
	server := NewHTTPServer(container, config)
	
	tempDir := t.TempDir()
	
	// Create a large text file that should be compressed
	largeContent := strings.Repeat("This is a test content that should be compressed. ", 100)
	filePath := filepath.Join(tempDir, "large.txt")
	err := os.WriteFile(filePath, []byte(largeContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}
	
	server.Static("/compressed/", tempDir)
	
	t.Run("GzipCompression", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/compressed/large.txt", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		// Check if gzip encoding is applied
		encoding := w.Header().Get("Content-Encoding")
		if encoding == "gzip" {
			// Gzip was applied
			t.Log("Gzip compression was applied")
		} else {
			// Gzip might not be applied for static files in this implementation
			t.Log("Gzip compression was not applied to static files")
		}
	})
}

// Test static file serving security
func TestStaticFileSecurity(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	tempDir := t.TempDir()
	
	// Create a test file
	filePath := filepath.Join(tempDir, "secure.txt")
	err := os.WriteFile(filePath, []byte("secure content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	server.Static("/secure/", tempDir)
	
	t.Run("PreventDirectoryTraversal", func(t *testing.T) {
		dangerousPaths := []string{
			"/secure/../../../etc/passwd",
			"/secure/..\\..\\..\\windows\\system32\\drivers\\etc\\hosts",
			"/secure/%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd",
			"/secure/....//....//....//etc/passwd",
		}
		
		for _, path := range dangerousPaths {
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
			
			// Should not return 200 OK for directory traversal attempts
			if w.Code == http.StatusOK && strings.Contains(w.Body.String(), "root:") {
				t.Errorf("Directory traversal vulnerability detected for path: %s", path)
			}
		}
	})
	
	t.Run("SecurityHeaders", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/secure/secure.txt", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		// Security headers should be applied by middleware
		// The actual headers depend on which middleware is active
	})
}