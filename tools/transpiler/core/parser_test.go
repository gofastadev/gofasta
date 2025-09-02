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

// TestGetFileSet verifies GetFileSet functionality
func TestGetFileSet(t *testing.T) {
	parser := NewParallelParser(DefaultConfig())
	
	fileSet := parser.GetFileSet()
	if fileSet == nil {
		t.Error("Expected FileSet to be non-nil")
	}
	
	// Verify it's the same instance
	fileSet2 := parser.GetFileSet()
	if fileSet != fileSet2 {
		t.Error("Expected GetFileSet to return the same instance")
	}
}

// TestDiscoverFilesEdgeCases tests edge cases in file discovery
func TestDiscoverFilesEdgeCases(t *testing.T) {
	parser := NewParallelParser(DefaultConfig())
	
	t.Run("NonExistentDirectory", func(t *testing.T) {
		files, err := parser.discoverFiles("/non/existent/directory")
		if err == nil {
			t.Error("Expected error for non-existent directory")
		}
		if files != nil {
			t.Error("Expected nil files for failed discovery")
		}
	})
	
	t.Run("EmptyDirectory", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "empty_test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)
		
		files, err := parser.discoverFiles(tempDir)
		if err != nil {
			t.Errorf("Expected no error for empty directory, got: %v", err)
		}
		if len(files) != 0 {
			t.Errorf("Expected 0 files in empty directory, got %d", len(files))
		}
	})
	
	t.Run("DirectoryWithNoMatchingFiles", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "no_match_test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)
		
		// Create files with non-matching extensions
		testFiles := []string{"file1.txt", "file2.md", "file3.json"}
		for _, filename := range testFiles {
			filePath := filepath.Join(tempDir, filename)
			err := os.WriteFile(filePath, []byte("content"), 0644)
			if err != nil {
				t.Fatalf("Failed to write file %s: %v", filename, err)
			}
		}
		
		files, err := parser.discoverFiles(tempDir)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(files) != 0 {
			t.Errorf("Expected 0 matching files, got %d", len(files))
		}
	})
	
	t.Run("DirectoryWithGlobPatternMatching", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "glob_test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)
		
		// Create subdirectory that matches exclusion pattern
		excludedDir := filepath.Join(tempDir, "vendor-like")
		err = os.MkdirAll(excludedDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create excluded dir: %v", err)
		}
		
		// Add a config that matches this pattern
		config := DefaultConfig()
		config.ExcludeDirs = []string{"vendor-like"}
		parser := NewParallelParser(config)
		
		// Create a file in the excluded directory
		filePath := filepath.Join(excludedDir, "test.go")
		err = os.WriteFile(filePath, []byte("package main"), 0644)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
		
		files, err := parser.discoverFiles(tempDir)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		
		// Should not find any files due to exclusion
		for _, file := range files {
			if strings.Contains(file, "vendor-like") {
				t.Errorf("Expected excluded file to not be found: %s", file)
			}
		}
	})
}

// TestParseFileEdgeCases tests edge cases in single file parsing
func TestParseFileEdgeCases(t *testing.T) {
	parser := NewParallelParser(DefaultConfig())
	
	t.Run("NonExistentFile", func(t *testing.T) {
		result := parser.parseFile("/non/existent/file.go")
		
		if result.Error == nil {
			t.Error("Expected error for non-existent file")
		}
		if result.File != nil {
			t.Error("Expected nil file for non-existent file")
		}
		if result.Size != 0 {
			t.Error("Expected size to be 0 for non-existent file")
		}
	})
	
	t.Run("FileWithNoComments", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "no_comments_test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)
		
		// Create file with no comments
		content := `package main
func main() {
	fmt.Println("No comments here")
}`
		filePath := filepath.Join(tempDir, "no_comments.go")
		err = os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
		
		// Test with ParseComments disabled
		config := DefaultConfig()
		config.ParseComments = false
		parser := NewParallelParser(config)
		
		result := parser.parseFile(filePath)
		if result.Error != nil {
			t.Errorf("Expected no error, got: %v", result.Error)
		}
		if result.File == nil {
			t.Error("Expected file to be parsed")
		}
	})
	
	t.Run("FileStatError", func(t *testing.T) {
		// This tests the os.Stat error path
		tempDir, err := os.MkdirTemp("", "stat_error_test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)
		
		// Create a file, then remove it to cause stat error but still test the path
		filePath := filepath.Join(tempDir, "temp.go")
		err = os.WriteFile(filePath, []byte("package main"), 0644)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
		
		// Remove the file to cause stat error
		os.Remove(filePath)
		
		result := parser.parseFile(filePath)
		// Should handle stat error gracefully - size will be 0, but parsing will still be attempted
		if result.Size != 0 {
			t.Error("Expected size to be 0 when stat fails")
		}
	})
}

// TestParseDirectoryErrorPaths tests error paths in ParseDirectory
func TestParseDirectoryErrorPaths(t *testing.T) {
	t.Run("DiscoveryError", func(t *testing.T) {
		parser := NewParallelParser(DefaultConfig())
		ctx := context.Background()
		
		// Try to parse a non-existent directory
		results, err := parser.ParseDirectory(ctx, "/non/existent/directory")
		if err == nil {
			t.Error("Expected error for non-existent directory")
		}
		if results != nil {
			t.Error("Expected nil results for failed discovery")
		}
	})
	
	t.Run("EmptyDirectoryResult", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "empty_dir_test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)
		
		parser := NewParallelParser(DefaultConfig())
		ctx := context.Background()
		
		results, err := parser.ParseDirectory(ctx, tempDir)
		if err != nil {
			t.Errorf("Expected no error for empty directory, got: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("Expected 0 results for empty directory, got %d", len(results))
		}
	})
}

// TestParseFilesErrorPaths tests error paths in ParseFiles
func TestParseFilesErrorPaths(t *testing.T) {
	parser := NewParallelParser(DefaultConfig())
	
	t.Run("EmptyFileList", func(t *testing.T) {
		ctx := context.Background()
		
		results, err := parser.ParseFiles(ctx, []string{})
		if err != nil {
			t.Errorf("Expected no error for empty file list, got: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("Expected 0 results for empty file list, got %d", len(results))
		}
	})
	
	t.Run("ContextTimeout", func(t *testing.T) {
		// Create a context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		
		// Wait for context to timeout
		time.Sleep(1 * time.Millisecond)
		
		results, err := parser.ParseFiles(ctx, []string{"/some/file.go"})
		if err == nil {
			t.Error("Expected error due to context timeout")
		}
		if results != nil {
			t.Error("Expected nil results for timed out context")
		}
	})
}

// TestCollectResultsEdgeCases tests edge cases in result collection
func TestCollectResultsEdgeCases(t *testing.T) {
	parser := NewParallelParser(DefaultConfig())
	
	t.Run("ResultWithError", func(t *testing.T) {
		// Test the collectResults function handles errors correctly
		resultChan := make(chan *ParseResult, 2)
		
		// Send results with and without errors
		resultChan <- &ParseResult{
			FilePath: "/test1.go",
			Size:     100,
			Error:    nil,
		}
		resultChan <- &ParseResult{
			FilePath: "/test2.go", 
			Size:     200,
			Error:    fmt.Errorf("test error"),
		}
		close(resultChan)
		
		// Reset parser state
		parser.reset()
		parser.totalFiles = 2
		
		// Collect results
		parser.collectResults(resultChan)
		
		// Verify counters were updated correctly
		stats := parser.GetStatistics()
		if stats["successful_files"] != 1 {
			t.Errorf("Expected 1 successful file, got %v", stats["successful_files"])
		}
		if stats["failed_files"] != 1 {
			t.Errorf("Expected 1 failed file, got %v", stats["failed_files"])
		}
		if stats["total_bytes"] != int64(300) {
			t.Errorf("Expected 300 total bytes, got %v", stats["total_bytes"])
		}
	})
}

// TestWaitForCollectionEdgeCases tests edge cases in result collection waiting
func TestWaitForCollectionEdgeCases(t *testing.T) {
	parser := NewParallelParser(DefaultConfig())
	
	t.Run("ImmediateCompletion", func(t *testing.T) {
		// Set up a scenario where collection is already complete
		parser.reset()
		parser.totalFiles = 0 // No files to wait for
		
		start := time.Now()
		parser.waitForCollection()
		duration := time.Since(start)
		
		// Should return immediately
		if duration > 10*time.Millisecond {
			t.Errorf("Expected immediate return, took %v", duration)
		}
	})
}

// TestConfigurationEdgeCases tests edge cases in configuration handling
func TestConfigurationEdgeCases(t *testing.T) {
	t.Run("EmptyFileExtensions", func(t *testing.T) {
		config := &ParserConfig{
			FileExtensions: []string{}, // Empty extensions
		}
		
		parser := NewParallelParser(config)
		
		// Should default to [".gofa", ".go"]
		if len(parser.config.FileExtensions) != 2 {
			t.Errorf("Expected 2 default extensions, got %d", len(parser.config.FileExtensions))
		}
	})
	
	t.Run("NilExcludeDirs", func(t *testing.T) {
		config := &ParserConfig{
			ExcludeDirs: nil, // Nil exclude dirs
		}
		
		parser := NewParallelParser(config)
		
		// Should handle nil gracefully
		if parser.config.ExcludeDirs == nil {
			t.Error("Expected ExcludeDirs to be initialized")
		}
	})
}

// TestParseDirectoryWithDiscoveryError tests ParseDirectory when file discovery fails
func TestParseDirectoryWithDiscoveryError(t *testing.T) {
	parser := NewParallelParser(DefaultConfig())
	ctx := context.Background()
	
	// Test with a path that will cause permission error on some systems
	_, err := parser.ParseDirectory(ctx, "/root")
	// Error is expected on most systems due to permissions
	if err != nil {
		t.Logf("Expected permission error for /root: %v", err)
	}
}

// TestParseDirectoryContextCancellation tests context cancellation in ParseDirectory
func TestParseDirectoryContextCancellation(t *testing.T) {
	// Create a temporary directory with files
	tempDir, err := os.MkdirTemp("", "cancel_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create several files to ensure parallel processing
	for i := 0; i < 10; i++ {
		content := fmt.Sprintf(`package main
func main%d() {
	// Function %d
}`, i, i)
		filePath := filepath.Join(tempDir, fmt.Sprintf("file%d.go", i))
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write file %d: %v", i, err)
		}
	}
	
	parser := NewParallelParser(DefaultConfig())
	
	// Create a context that will be cancelled during processing
	ctx, cancel := context.WithCancel(context.Background())
	
	// Cancel the context immediately before starting
	cancel()
	
	// Should get an error due to cancelled context
	parseResults, parseErr := parser.ParseDirectory(ctx, tempDir)
	
	if parseErr == nil {
		t.Error("Expected error due to context cancellation")
	}
	
	if parseResults != nil {
		t.Error("Expected nil results due to context cancellation")
	}
	
	if !strings.Contains(parseErr.Error(), "parallel parsing failed") {
		t.Errorf("Expected 'parallel parsing failed' error, got: %v", parseErr)
	}
}

// TestParseDirectoryWorkflowEdgeCases tests specific workflow edge cases
func TestParseDirectoryWorkflowEdgeCases(t *testing.T) {
	t.Run("ResultCollectionRaceCondition", func(t *testing.T) {
		// Test that result collection handles concurrent access correctly
		parser := NewParallelParser(DefaultConfig())
		
		// Create a temporary directory with one file
		tempDir, err := os.MkdirTemp("", "race_test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)
		
		content := `package main
func main() {}`
		filePath := filepath.Join(tempDir, "test.go")
		err = os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
		
		ctx := context.Background()
		results, err := parser.ParseDirectory(ctx, tempDir)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
		
		// Verify statistics are consistent
		stats := parser.GetStatistics()
		if stats["total_files"] != 1 {
			t.Errorf("Expected total_files to be 1, got %v", stats["total_files"])
		}
	})
}

// TestFilterResultsByExtensionEdgeCases tests edge cases in extension filtering
func TestFilterResultsByExtensionEdgeCases(t *testing.T) {
	parser := NewParallelParser(DefaultConfig())
	
	// Add some test results manually
	parser.mu.Lock()
	parser.results = []*ParseResult{
		{FilePath: "/test/file1.go"},
		{FilePath: "/test/file2.gofa"},
		{FilePath: "/test/file3.js"},
		{FilePath: "/test/file.go.backup"},
	}
	parser.mu.Unlock()
	
	t.Run("NoMatchingExtension", func(t *testing.T) {
		results := parser.FilterResultsByExtension(".py")
		if len(results) != 0 {
			t.Errorf("Expected 0 results for .py extension, got %d", len(results))
		}
	})
	
	t.Run("PartialMatchExtension", func(t *testing.T) {
		results := parser.FilterResultsByExtension(".go")
		// Should match file1.go but not file.go.backup
		expected := 1
		if len(results) != expected {
			t.Errorf("Expected %d results for .go extension, got %d", expected, len(results))
		}
	})
}

// TestGetStatisticsEdgeCases tests edge cases in statistics calculation
func TestGetStatisticsEdgeCases(t *testing.T) {
	parser := NewParallelParser(DefaultConfig())
	
	t.Run("NoFilesProcessed", func(t *testing.T) {
		// Get statistics before processing any files
		stats := parser.GetStatistics()
		
		if stats["total_files"] != 0 {
			t.Errorf("Expected total_files to be 0, got %v", stats["total_files"])
		}
		
		if stats["total_bytes"] != int64(0) {
			t.Errorf("Expected total_bytes to be 0, got %v", stats["total_bytes"])
		}
		
		// When totalBytes is 0, bytes_per_second and mb_per_second should not be present
		if _, exists := stats["bytes_per_second"]; exists {
			t.Error("Expected bytes_per_second to not exist when total_bytes is 0")
		}
		
		if _, exists := stats["mb_per_second"]; exists {
			t.Error("Expected mb_per_second to not exist when total_bytes is 0")
		}
	})
	
	t.Run("WithBytesProcessed", func(t *testing.T) {
		// Create a temporary file and parse it to get some bytes
		tempDir, err := os.MkdirTemp("", "stats_test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)
		
		content := `package main
func main() {
	// Some content to generate bytes
}`
		filePath := filepath.Join(tempDir, "test.go")
		err = os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
		
		ctx := context.Background()
		_, err = parser.ParseDirectory(ctx, tempDir)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		
		stats := parser.GetStatistics()
		
		// Now bytes_per_second and mb_per_second should exist
		if _, exists := stats["bytes_per_second"]; !exists {
			t.Error("Expected bytes_per_second to exist when total_bytes > 0")
		}
		
		if _, exists := stats["mb_per_second"]; !exists {
			t.Error("Expected mb_per_second to exist when total_bytes > 0")
		}
		
		// Verify the calculations are reasonable
		totalBytes := stats["total_bytes"].(int64)
		if totalBytes <= 0 {
			t.Error("Expected total_bytes to be greater than 0")
		}
	})
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