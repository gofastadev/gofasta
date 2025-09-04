package core

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDefaultBatchFormatterConfig(t *testing.T) {
	config := DefaultBatchFormatterConfig()
	
	if config.BatchSize != 10 {
		t.Errorf("Expected BatchSize to be 10, got %d", config.BatchSize)
	}
	
	if config.MaxWorkers != 4 {
		t.Errorf("Expected MaxWorkers to be 4, got %d", config.MaxWorkers)
	}
	
	if !config.EnableMetrics {
		t.Error("Expected EnableMetrics to be true")
	}
	
	if config.FormatOptions == nil {
		t.Error("Expected FormatOptions to be initialized")
	}
	
	if config.FormatOptions.TabWidth != 8 {
		t.Errorf("Expected TabWidth to be 8, got %d", config.FormatOptions.TabWidth)
	}
}

func TestNewBatchFormatter(t *testing.T) {
	tests := []struct {
		name     string
		config   *BatchFormatterConfig
		expected *BatchFormatterConfig
	}{
		{
			name: "WithCustomConfig",
			config: &BatchFormatterConfig{
				BatchSize:     5,
				MaxWorkers:    2,
				EnableMetrics: false,
				FormatOptions: &FormatOptions{TabWidth: 4, UseSpaces: true},
			},
			expected: &BatchFormatterConfig{
				BatchSize:     5,
				MaxWorkers:    2,
				EnableMetrics: false,
			},
		},
		{
			name:     "WithNilConfig",
			config:   nil,
			expected: DefaultBatchFormatterConfig(),
		},
		{
			name:     "WithZeroValues",
			config:   &BatchFormatterConfig{},
			expected: &BatchFormatterConfig{BatchSize: 10, MaxWorkers: 4, EnableMetrics: false},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewBatchFormatter(tt.config)
			
			if formatter.config.BatchSize != tt.expected.BatchSize {
				t.Errorf("Expected BatchSize %d, got %d", tt.expected.BatchSize, formatter.config.BatchSize)
			}
			
			if formatter.config.MaxWorkers != tt.expected.MaxWorkers {
				t.Errorf("Expected MaxWorkers %d, got %d", tt.expected.MaxWorkers, formatter.config.MaxWorkers)
			}
			
			if formatter.config.FormatOptions == nil {
				t.Error("Expected FormatOptions to be initialized")
			}
		})
	}
}

func TestBatchFormatterFormatFile(t *testing.T) {
	formatter := NewBatchFormatter(DefaultBatchFormatterConfig())
	
	// Create test AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", `package main
func main(){fmt.Println("Hello")}`, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}
	
	// Format file
	result := formatter.FormatFile("test.go", file, fset)
	
	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}
	
	if result.Error != nil {
		t.Errorf("Formatting failed: %v", result.Error)
	}
	
	if result.FilePath != "test.go" {
		t.Errorf("Expected FilePath 'test.go', got '%s'", result.FilePath)
	}
	
	if len(result.Output) == 0 {
		t.Error("Expected formatted output")
	}
	
	if result.Duration <= 0 {
		t.Error("Expected positive duration")
	}
	
	// Check that output is properly formatted
	output := string(result.Output)
	if !strings.Contains(output, "package main") {
		t.Error("Expected formatted output to contain package declaration")
	}
}

func TestBatchFormatterFormatFiles(t *testing.T) {
	formatter := NewBatchFormatter(&BatchFormatterConfig{
		BatchSize:     2,
		MaxWorkers:    2,
		EnableMetrics: true,
	})
	
	// Create test files
	fset := token.NewFileSet()
	
	file1, _ := parser.ParseFile(fset, "file1.go", "package main\nfunc test1(){}", parser.ParseComments)
	file2, _ := parser.ParseFile(fset, "file2.go", "package main\nfunc test2(){}", parser.ParseComments)
	file3, _ := parser.ParseFile(fset, "file3.go", "package main\nfunc test3(){}", parser.ParseComments)
	
	files := map[string]*ast.File{
		"file1.go": file1,
		"file2.go": file2,
		"file3.go": file3,
	}
	
	ctx := context.Background()
	start := time.Now()
	
	results, err := formatter.FormatFiles(ctx, files, fset)
	duration := time.Since(start)
	
	if err != nil {
		t.Errorf("Batch formatting failed: %v", err)
	}
	
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
	
	// Check all files were formatted
	for path := range files {
		if results[path] == nil {
			t.Errorf("Missing result for %s", path)
		}
		if results[path].Error != nil {
			t.Errorf("Formatting error for %s: %v", path, results[path].Error)
		}
	}
	
	t.Logf("Batch formatting of 3 files completed in %v", duration)
}

func TestBatchFormatterEmptyFiles(t *testing.T) {
	formatter := NewBatchFormatter(DefaultBatchFormatterConfig())
	
	ctx := context.Background()
	fset := token.NewFileSet()
	
	results, err := formatter.FormatFiles(ctx, map[string]*ast.File{}, fset)
	if err != nil {
		t.Errorf("Expected no error for empty files, got %v", err)
	}
	
	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty files, got %d", len(results))
	}
}

func TestBatchFormatterContextCancellation(t *testing.T) {
	formatter := NewBatchFormatter(DefaultBatchFormatterConfig())
	
	// Create test files
	fset := token.NewFileSet()
	files := make(map[string]*ast.File)
	
	for i := 0; i < 10; i++ {
		file, _ := parser.ParseFile(fset, fmt.Sprintf("file%d.go", i), "package main", parser.ParseComments)
		files[fmt.Sprintf("file%d.go", i)] = file
	}
	
	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	
	_, err := formatter.FormatFiles(ctx, files, fset)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

func TestBatchFormatterStatistics(t *testing.T) {
	formatter := NewBatchFormatter(&BatchFormatterConfig{
		BatchSize:     5,
		MaxWorkers:    2,
		EnableMetrics: true,
	})
	
	// Create test files
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "test.go", "package main\nfunc main() {}", parser.ParseComments)
	
	// Format multiple files
	for i := 0; i < 3; i++ {
		formatter.FormatFile(fmt.Sprintf("test%d.go", i), file, fset)
	}
	
	stats := formatter.GetStatistics()
	
	if stats["files_formatted"].(int64) != 3 {
		t.Errorf("Expected 3 files formatted, got %v", stats["files_formatted"])
	}
	
	if stats["files_per_second"].(float64) <= 0 {
		t.Error("Expected positive throughput")
	}
	
	if stats["success_rate"].(float64) != 100.0 {
		t.Errorf("Expected 100%% success rate, got %.1f", stats["success_rate"].(float64))
	}
}

func TestBatchFormatterReset(t *testing.T) {
	formatter := NewBatchFormatter(DefaultBatchFormatterConfig())
	
	// Create test file and format it
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "test.go", "package main", parser.ParseComments)
	formatter.FormatFile("test.go", file, fset)
	
	// Verify metrics are non-zero
	stats := formatter.GetStatistics()
	if stats["files_formatted"].(int64) == 0 {
		t.Error("Expected non-zero files formatted before reset")
	}
	
	// Reset
	formatter.Reset()
	
	// Verify metrics are zero
	stats = formatter.GetStatistics()
	if stats["files_formatted"].(int64) != 0 {
		t.Error("Expected zero files formatted after reset")
	}
}

func TestBatchFormatterCreateBatches(t *testing.T) {
	formatter := NewBatchFormatter(&BatchFormatterConfig{BatchSize: 2})
	
	// Create test files
	files := map[string]*ast.File{
		"file1.go": nil,
		"file2.go": nil,
		"file3.go": nil,
		"file4.go": nil,
		"file5.go": nil,
	}
	
	batches := formatter.createBatches(files)
	
	// Should create 3 batches (2+2+1)
	if len(batches) != 3 {
		t.Errorf("Expected 3 batches, got %d", len(batches))
	}
	
	// Check batch sizes
	if len(batches[0]) != 2 || len(batches[1]) != 2 || len(batches[2]) != 1 {
		t.Error("Unexpected batch sizes")
	}
	
	// Verify all files are included
	totalFiles := 0
	for _, batch := range batches {
		totalFiles += len(batch)
	}
	
	if totalFiles != 5 {
		t.Errorf("Expected 5 total files in batches, got %d", totalFiles)
	}
}

func TestBatchFormatterFormatOptions(t *testing.T) {
	options := &FormatOptions{
		TabWidth:    4,
		UseSpaces:   true,
		SortImports: false,
	}
	
	formatter := NewBatchFormatter(&BatchFormatterConfig{
		FormatOptions: options,
	})
	
	// Test GetFormattingOptions
	gotOptions := formatter.GetFormattingOptions()
	if gotOptions.TabWidth != 4 {
		t.Errorf("Expected TabWidth 4, got %d", gotOptions.TabWidth)
	}
	
	if !gotOptions.UseSpaces {
		t.Error("Expected UseSpaces to be true")
	}
	
	if gotOptions.SortImports {
		t.Error("Expected SortImports to be false")
	}
	
	// Test SetFormattingOptions
	newOptions := &FormatOptions{
		TabWidth:    2,
		UseSpaces:   false,
		SortImports: true,
	}
	
	formatter.SetFormattingOptions(newOptions)
	gotOptions = formatter.GetFormattingOptions()
	
	if gotOptions.TabWidth != 2 {
		t.Errorf("Expected TabWidth 2 after update, got %d", gotOptions.TabWidth)
	}
}

func TestBatchFormatterMemoryEstimation(t *testing.T) {
	formatter := NewBatchFormatter(DefaultBatchFormatterConfig())
	
	// Create test files with different sizes
	fset := token.NewFileSet()
	
	smallFile, _ := parser.ParseFile(fset, "small.go", "package main", parser.ParseComments)
	largeFile, _ := parser.ParseFile(fset, "large.go", `
package main
import "fmt"
func main() {
	for i := 0; i < 10; i++ {
		fmt.Println(i)
	}
}`, parser.ParseComments)
	
	files := map[string]*ast.File{
		"small.go": smallFile,
		"large.go": largeFile,
	}
	
	memUsage := formatter.EstimateMemoryUsage(files)
	if memUsage <= 0 {
		t.Error("Expected positive memory usage estimate")
	}
	
	// Larger file should contribute more to memory usage
	smallFiles := map[string]*ast.File{"small.go": smallFile}
	largeFiles := map[string]*ast.File{"large.go": largeFile}
	
	smallMem := formatter.EstimateMemoryUsage(smallFiles)
	largeMem := formatter.EstimateMemoryUsage(largeFiles)
	
	if largeMem <= smallMem {
		t.Error("Expected larger file to have higher memory estimate")
	}
}

func TestBatchFormatterConcurrency(t *testing.T) {
	formatter := NewBatchFormatter(&BatchFormatterConfig{
		BatchSize:     3,
		MaxWorkers:    3,
		EnableMetrics: true,
	})
	
	// Create multiple test files
	fset := token.NewFileSet()
	files := make(map[string]*ast.File)
	
	for i := 0; i < 9; i++ {
		code := fmt.Sprintf("package main\nfunc test%d() {}", i)
		file, _ := parser.ParseFile(fset, fmt.Sprintf("test%d.go", i), code, parser.ParseComments)
		files[fmt.Sprintf("test%d.go", i)] = file
	}
	
	ctx := context.Background()
	var wg sync.WaitGroup
	numConcurrent := 3
	
	// Concurrent formatting operations
	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			results, err := formatter.FormatFiles(ctx, files, fset)
			if err != nil {
				t.Errorf("Concurrent formatting failed: %v", err)
				return
			}
			
			if len(results) != 9 {
				t.Errorf("Expected 9 results, got %d", len(results))
			}
		}()
	}
	
	wg.Wait()
	
	// Check that formatter is in consistent state
	stats := formatter.GetStatistics()
	if stats["files_formatted"].(int64) <= 0 {
		t.Error("Expected some files to be formatted")
	}
}

func TestBatchFormatterErrorHandling(t *testing.T) {
	formatter := NewBatchFormatter(DefaultBatchFormatterConfig())
	
	// Create invalid AST (nil file)
	fset := token.NewFileSet()
	
	result := formatter.FormatFile("invalid.go", nil, fset)
	
	if result == nil {
		t.Fatal("Expected result even for invalid file")
	}
	
	if result.Error == nil {
		t.Error("Expected error for nil file")
	}
	
	if result.FilePath != "invalid.go" {
		t.Errorf("Expected FilePath 'invalid.go', got '%s'", result.FilePath)
	}
}

func TestBatchFormatterPerformance(t *testing.T) {
	formatter := NewBatchFormatter(&BatchFormatterConfig{
		BatchSize:     5,
		MaxWorkers:    4,
		EnableMetrics: true,
	})
	
	// Create multiple test files
	fset := token.NewFileSet()
	files := make(map[string]*ast.File)
	
	for i := 0; i < 20; i++ {
		code := fmt.Sprintf("package main\nimport \"fmt\"\nfunc test%d() {\n\tfmt.Println(%d)\n}", i, i)
		file, _ := parser.ParseFile(fset, fmt.Sprintf("test%d.go", i), code, parser.ParseComments)
		files[fmt.Sprintf("test%d.go", i)] = file
	}
	
	ctx := context.Background()
	start := time.Now()
	
	results, err := formatter.FormatFiles(ctx, files, fset)
	duration := time.Since(start)
	
	if err != nil {
		t.Errorf("Performance test failed: %v", err)
	}
	
	if len(results) != 20 {
		t.Errorf("Expected 20 results, got %d", len(results))
	}
	
	// Check performance metrics
	stats := formatter.GetStatistics()
	throughput := stats["files_per_second"].(float64)
	
	t.Logf("Batch formatting performance: %.0f files/sec (20 files in %v)", throughput, duration)
	
	if throughput < 100 {
		t.Errorf("Expected > 100 files/sec, got %.0f", throughput)
	}
}

func TestBatchFormatterMetricsDisabled(t *testing.T) {
	formatter := NewBatchFormatter(&BatchFormatterConfig{
		EnableMetrics: false,
	})
	
	// Create test file
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "test.go", "package main", parser.ParseComments)
	
	// Format file
	formatter.FormatFile("test.go", file, fset)
	
	// Metrics should remain zero
	stats := formatter.GetStatistics()
	if stats["files_formatted"].(int64) != 0 {
		t.Error("Expected metrics to be disabled")
	}
}

func TestBatchFormatterNodeCounting(t *testing.T) {
	formatter := NewBatchFormatter(DefaultBatchFormatterConfig())
	
	// Test with nil node
	count := formatter.countASTNodes(nil)
	if count != 0 {
		t.Errorf("Expected 0 nodes for nil, got %d", count)
	}
	
	// Test with simple AST
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "test.go", "package main\nfunc main() {}", parser.ParseComments)
	
	count = formatter.countASTNodes(file)
	if count <= 0 {
		t.Errorf("Expected positive node count, got %d", count)
	}
}

func TestBatchFormatterEdgeCases(t *testing.T) {
	formatter := NewBatchFormatter(DefaultBatchFormatterConfig())
	fset := token.NewFileSet()
	
	t.Run("FormatFileWithNilFileSet", func(t *testing.T) {
		file, _ := parser.ParseFile(fset, "test.go", "package main", parser.ParseComments)
		
		result := formatter.FormatFile("test.go", file, nil)
		if result.Error == nil {
			t.Error("Expected error for nil FileSet")
		}
	})
	
	t.Run("SetNilFormatOptions", func(t *testing.T) {
		originalOptions := formatter.GetFormattingOptions()
		
		formatter.SetFormattingOptions(nil)
		
		// Options should remain unchanged
		currentOptions := formatter.GetFormattingOptions()
		if currentOptions != originalOptions {
			t.Error("Expected options to remain unchanged when setting nil")
		}
	})
	
	t.Run("CalculateSuccessRateNoFiles", func(t *testing.T) {
		newFormatter := NewBatchFormatter(DefaultBatchFormatterConfig())
		
		successRate := newFormatter.calculateSuccessRate()
		if successRate != 0.0 {
			t.Errorf("Expected 0.0 success rate for no files, got %.1f", successRate)
		}
	})
	
	t.Run("FormatFileWithEmptyPath", func(t *testing.T) {
		file, _ := parser.ParseFile(fset, "test.go", "package main", parser.ParseComments)
		
		result := formatter.FormatFile("", file, fset)
		if result.FilePath != "" {
			t.Errorf("Expected empty FilePath to be preserved, got '%s'", result.FilePath)
		}
	})
}

func TestBatchFormatterBatchCreation(t *testing.T) {
	t.Run("ExactBatchSize", func(t *testing.T) {
		formatter := NewBatchFormatter(&BatchFormatterConfig{BatchSize: 3})
		
		files := map[string]*ast.File{
			"file1.go": nil,
			"file2.go": nil,
			"file3.go": nil,
		}
		
		batches := formatter.createBatches(files)
		
		if len(batches) != 1 {
			t.Errorf("Expected 1 batch, got %d", len(batches))
		}
		
		if len(batches[0]) != 3 {
			t.Errorf("Expected batch size 3, got %d", len(batches[0]))
		}
	})
	
	t.Run("EmptyFiles", func(t *testing.T) {
		formatter := NewBatchFormatter(&BatchFormatterConfig{BatchSize: 5})
		
		batches := formatter.createBatches(map[string]*ast.File{})
		
		if len(batches) != 0 {
			t.Errorf("Expected 0 batches for empty files, got %d", len(batches))
		}
	})
	
	t.Run("SingleFile", func(t *testing.T) {
		formatter := NewBatchFormatter(&BatchFormatterConfig{BatchSize: 10})
		
		files := map[string]*ast.File{"single.go": nil}
		batches := formatter.createBatches(files)
		
		if len(batches) != 1 {
			t.Errorf("Expected 1 batch for single file, got %d", len(batches))
		}
		
		if len(batches[0]) != 1 {
			t.Errorf("Expected 1 file in batch, got %d", len(batches[0]))
		}
	})
}

func TestBatchFormatterWithErrors(t *testing.T) {
	formatter := NewBatchFormatter(&BatchFormatterConfig{
		BatchSize:     2,
		MaxWorkers:    2,
		EnableMetrics: true,
	})
	
	// Create files with some that will cause errors
	fset := token.NewFileSet()
	
	validFile, _ := parser.ParseFile(fset, "valid.go", "package main", parser.ParseComments)
	
	files := map[string]*ast.File{
		"valid.go":   validFile,
		"invalid.go": nil, // This will cause an error
	}
	
	ctx := context.Background()
	results, err := formatter.FormatFiles(ctx, files, fset)
	
	// Should get results for all files, but some with errors
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	
	// Valid file should succeed
	if results["valid.go"].Error != nil {
		t.Errorf("Expected valid file to format successfully: %v", results["valid.go"].Error)
	}
	
	// Invalid file should have error
	if results["invalid.go"].Error == nil {
		t.Error("Expected invalid file to have error")
	}
	
	// Overall operation should report first error
	if err == nil {
		t.Error("Expected overall error when some files fail")
	}
	
	// Check success rate
	stats := formatter.GetStatistics()
	successRate := stats["success_rate"].(float64)
	if successRate >= 100.0 {
		t.Errorf("Expected success rate < 100%% with errors, got %.1f", successRate)
	}
}