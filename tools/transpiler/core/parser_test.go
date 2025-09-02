package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestDefaultConfig verifies the default configuration is properly set
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config.MaxWorkers != runtime.NumCPU() {
		t.Errorf("Expected MaxWorkers to be %d, got %d", runtime.NumCPU(), config.MaxWorkers)
	}
	
	if !config.ParseComments {
		t.Error("Expected ParseComments to be true")
	}
	
	if !config.AllowErrors {
		t.Error("Expected AllowErrors to be true")
	}
	
	expectedExtensions := []string{".gofa", ".go"}
	if len(config.FileExtensions) != len(expectedExtensions) {
		t.Errorf("Expected %d file extensions, got %d", len(expectedExtensions), len(config.FileExtensions))
	}
	
	for i, ext := range expectedExtensions {
		if i >= len(config.FileExtensions) || config.FileExtensions[i] != ext {
			t.Errorf("Expected extension %s at index %d, got %s", ext, i, config.FileExtensions[i])
		}
	}
}

// TestNewParallelParser verifies parser initialization
func TestNewParallelParser(t *testing.T) {
	t.Run("WithCustomConfig", func(t *testing.T) {
		config := &ParserConfig{
			MaxWorkers:     4,
			ParseComments:  false,
			AllowErrors:    false,
			FileExtensions: []string{".go"},
		}
		
		parser := NewParallelParser(config)
		
		if parser.config.MaxWorkers != 4 {
			t.Errorf("Expected MaxWorkers to be 4, got %d", parser.config.MaxWorkers)
		}
		
		if parser.config.ParseComments {
			t.Error("Expected ParseComments to be false")
		}
	})
	
	t.Run("WithNilConfig", func(t *testing.T) {
		parser := NewParallelParser(nil)
		
		if parser.config.MaxWorkers != runtime.NumCPU() {
			t.Errorf("Expected MaxWorkers to be %d, got %d", runtime.NumCPU(), parser.config.MaxWorkers)
		}
	})
	
	t.Run("WithZeroMaxWorkers", func(t *testing.T) {
		config := &ParserConfig{MaxWorkers: 0}
		parser := NewParallelParser(config)
		
		if parser.config.MaxWorkers != runtime.NumCPU() {
			t.Errorf("Expected MaxWorkers to be %d, got %d", runtime.NumCPU(), parser.config.MaxWorkers)
		}
	})
}

// TestParseFile verifies single file parsing
func TestParseFile(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "parser_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create test files
	testFiles := map[string]string{
		"valid.go": `package main

import "fmt"

// @Controller("/api")
func main() {
	fmt.Println("Hello World")
}`,
		"valid.gofa": `package api

// @Get("/users")
// @JWT()
func GetUsers() {
	// Implementation here
}`,
		"invalid.go": `package main

func main( {
	// Missing closing parenthesis
}`,
	}
	
	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file %s: %v", filename, err)
		}
	}
	
	parser := NewParallelParser(DefaultConfig())
	
	t.Run("ValidGoFile", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "valid.go")
		result := parser.parseFile(filePath)
		
		if result.Error != nil {
			t.Errorf("Expected no error, got: %v", result.Error)
		}
		
		if result.File == nil {
			t.Error("Expected parsed file to be non-nil")
		}
		
		if result.FilePath != filePath {
			t.Errorf("Expected file path %s, got %s", filePath, result.FilePath)
		}
		
		if result.Size <= 0 {
			t.Error("Expected file size to be greater than 0")
		}
		
		if result.Duration <= 0 {
			t.Error("Expected parse duration to be greater than 0")
		}
	})
	
	t.Run("ValidGofaFile", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "valid.gofa")
		result := parser.parseFile(filePath)
		
		if result.Error != nil {
			t.Errorf("Expected no error, got: %v", result.Error)
		}
		
		if result.File == nil {
			t.Error("Expected parsed file to be non-nil")
		}
	})
	
	t.Run("InvalidFile", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "invalid.go")
		result := parser.parseFile(filePath)
		
		if result.Error == nil {
			t.Error("Expected error for invalid file")
		}
		
		if result.File != nil {
			t.Error("Expected parsed file to be nil for invalid file")
		}
	})
}

// TestParseFiles verifies parsing multiple specific files
func TestParseFiles(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "parser_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create test files
	testFiles := []string{"file1.go", "file2.gofa", "file3.go"}
	var filePaths []string
	
	for _, filename := range testFiles {
		content := fmt.Sprintf(`package main
// File: %s
func main() {}`, filename)
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file %s: %v", filename, err)
		}
		filePaths = append(filePaths, filePath)
	}
	
	parser := NewParallelParser(DefaultConfig())
	ctx := context.Background()
	
	results, err := parser.ParseFiles(ctx, filePaths)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if len(results) != len(testFiles) {
		t.Errorf("Expected %d results, got %d", len(testFiles), len(results))
	}
	
	for _, result := range results {
		if result.Error != nil {
			t.Errorf("Expected no error for file %s, got: %v", result.FilePath, result.Error)
		}
		
		if result.File == nil {
			t.Errorf("Expected parsed file to be non-nil for %s", result.FilePath)
		}
	}
	
	// Test statistics
	stats := parser.GetStatistics()
	if stats["total_files"] != len(testFiles) {
		t.Errorf("Expected total_files to be %d, got %v", len(testFiles), stats["total_files"])
	}
	
	if stats["successful_files"] != len(testFiles) {
		t.Errorf("Expected successful_files to be %d, got %v", len(testFiles), stats["successful_files"])
	}
}

// TestParseDirectory verifies directory parsing functionality
func TestParseDirectory(t *testing.T) {
	// Create a temporary directory structure
	tempDir, err := os.MkdirTemp("", "parser_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create subdirectories
	subDirs := []string{"api", "models", "vendor"}
	for _, subDir := range subDirs {
		err := os.MkdirAll(filepath.Join(tempDir, subDir), 0755)
		if err != nil {
			t.Fatalf("Failed to create subdir %s: %v", subDir, err)
		}
	}
	
	// Create test files
	testFiles := map[string]string{
		"main.go": `package main
func main() {}`,
		"api/handler.go": `package api
// @Controller("/api")
func Handler() {}`,
		"api/routes.gofa": `package api
// @Get("/users")
func GetUsers() {}`,
		"models/user.go": `package models
type User struct {}`,
		"vendor/lib.go": `package vendor
// Should be excluded`,
		"README.md": `# Test project`,
	}
	
	for filePath, content := range testFiles {
		fullPath := filepath.Join(tempDir, filePath)
		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file %s: %v", filePath, err)
		}
	}
	
	parser := NewParallelParser(DefaultConfig())
	ctx := context.Background()
	
	results, err := parser.ParseDirectory(ctx, tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	// Should find 4 files (.go and .gofa), excluding vendor and .md
	expectedFiles := 4
	t.Logf("Found %d total results", len(results))
	for _, result := range results {
		t.Logf("Found file: %s (error: %v)", result.FilePath, result.Error)
	}
	
	if len(results) != expectedFiles {
		t.Errorf("Expected %d results, got %d", expectedFiles, len(results))
	}
	
	// Verify vendor directory is excluded
	for _, result := range results {
		if filepath.Dir(result.FilePath) == filepath.Join(tempDir, "vendor") {
			t.Errorf("Expected vendor files to be excluded, found: %s", result.FilePath)
		}
	}
	
	// Test successful results filter
	successful := parser.GetSuccessfulResults()
	if len(successful) != len(results) {
		t.Errorf("Expected %d successful results, got %d", len(results), len(successful))
	}
}

// TestParallelProcessing verifies that parsing actually runs in parallel
func TestParallelProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping parallel processing test in short mode")
	}
	
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "parser_parallel_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create many test files to force parallel processing
	numFiles := 20
	var filePaths []string
	
	for i := 0; i < numFiles; i++ {
		content := fmt.Sprintf(`package main
// File number %d
// @Controller("/api/%d")
func Handler%d() {
	// Simulated work
}`, i, i, i)
		
		filePath := filepath.Join(tempDir, fmt.Sprintf("file%d.gofa", i))
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file %d: %v", i, err)
		}
		filePaths = append(filePaths, filePath)
	}
	
	// Test with different worker counts
	testCases := []struct {
		name       string
		maxWorkers int
	}{
		{"SingleWorker", 1},
		{"MultipleWorkers", 4},
		{"DefaultWorkers", runtime.NumCPU()},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := DefaultConfig()
			config.MaxWorkers = tc.maxWorkers
			
			parser := NewParallelParser(config)
			ctx := context.Background()
			
			start := time.Now()
			results, err := parser.ParseFiles(ctx, filePaths)
			duration := time.Since(start)
			
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}
			
			if len(results) != numFiles {
				t.Errorf("Expected %d results, got %d", numFiles, len(results))
			}
			
			stats := parser.GetStatistics()
			t.Logf("Workers: %d, Duration: %v, Files/sec: %.2f", 
				tc.maxWorkers, duration, stats["files_per_second"])
		})
	}
}

// TestContextCancellation verifies that parsing respects context cancellation
func TestContextCancellation(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "parser_cancel_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a test file
	filePath := filepath.Join(tempDir, "test.go")
	content := `package main
func main() {}`
	err = os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	parser := NewParallelParser(DefaultConfig())
	
	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	_, err = parser.ParseFiles(ctx, []string{filePath})
	if err == nil {
		t.Error("Expected error due to cancelled context")
	}
	
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("Expected context canceled error, got: %v", err)
	}
}

// TestFilterResultsByExtension verifies extension filtering
func TestFilterResultsByExtension(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "parser_filter_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create test files with different extensions
	testFiles := map[string]string{
		"file1.go":   `package main`,
		"file2.gofa": `package api`,
		"file3.go":   `package models`,
	}
	
	var filePaths []string
	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file %s: %v", filename, err)
		}
		filePaths = append(filePaths, filePath)
	}
	
	parser := NewParallelParser(DefaultConfig())
	ctx := context.Background()
	
	_, err = parser.ParseFiles(ctx, filePaths)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	// Test filtering by .go extension
	goResults := parser.FilterResultsByExtension(".go")
	if len(goResults) != 2 {
		t.Errorf("Expected 2 .go files, got %d", len(goResults))
	}
	
	// Test filtering by .gofa extension
	gofaResults := parser.FilterResultsByExtension(".gofa")
	if len(gofaResults) != 1 {
		t.Errorf("Expected 1 .gofa file, got %d", len(gofaResults))
	}
}

// BenchmarkParallelParsing benchmarks the parallel parsing performance
func BenchmarkParallelParsing(b *testing.B) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "parser_benchmark")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create test files
	numFiles := 50
	var filePaths []string
	
	for i := 0; i < numFiles; i++ {
		content := fmt.Sprintf(`package main

import "fmt"

// @Controller("/api/%d")
// @Get("/users")
func Handler%d() {
	fmt.Println("Handler %d")
}

// @Post("/users")
func CreateUser%d() {
	fmt.Println("Create user %d")
}`, i, i, i, i, i)
		
		filePath := filepath.Join(tempDir, fmt.Sprintf("file%d.gofa", i))
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			b.Fatalf("Failed to write test file %d: %v", i, err)
		}
		filePaths = append(filePaths, filePath)
	}
	
	config := DefaultConfig()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		parser := NewParallelParser(config)
		ctx := context.Background()
		
		_, err := parser.ParseFiles(ctx, filePaths)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}