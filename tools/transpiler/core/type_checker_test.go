package core

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sync"
	"testing"
	"time"
)

func TestDefaultTypeCheckerConfig(t *testing.T) {
	config := DefaultTypeCheckerConfig()
	
	if !config.EnableCaching {
		t.Error("Expected EnableCaching to be true")
	}
	
	if config.CacheTTL != 30*time.Minute {
		t.Errorf("Expected CacheTTL to be 30 minutes, got %v", config.CacheTTL)
	}
	
	if config.MaxCacheEntries != 500 {
		t.Errorf("Expected MaxCacheEntries to be 500, got %d", config.MaxCacheEntries)
	}
	
	if !config.ParallelChecking {
		t.Error("Expected ParallelChecking to be true")
	}
	
	if config.MaxWorkers != 4 {
		t.Errorf("Expected MaxWorkers to be 4, got %d", config.MaxWorkers)
	}
	
	if !config.EnableMetrics {
		t.Error("Expected EnableMetrics to be true")
	}
}

func TestNewIncrementalTypeChecker(t *testing.T) {
	tests := []struct {
		name     string
		config   *TypeCheckerConfig
		expected *TypeCheckerConfig
	}{
		{
			name: "WithCustomConfig",
			config: &TypeCheckerConfig{
				EnableCaching:    false,
				CacheTTL:         10 * time.Minute,
				MaxCacheEntries:  100,
				ParallelChecking: false,
				MaxWorkers:       2,
				EnableMetrics:    false,
			},
			expected: &TypeCheckerConfig{
				EnableCaching:    false,
				CacheTTL:         10 * time.Minute,
				MaxCacheEntries:  100,
				ParallelChecking: false,
				MaxWorkers:       2,
				EnableMetrics:    false,
			},
		},
		{
			name:     "WithNilConfig",
			config:   nil,
			expected: DefaultTypeCheckerConfig(),
		},
		{
			name:     "WithZeroValues",
			config:   &TypeCheckerConfig{},
			expected: &TypeCheckerConfig{CacheTTL: 30 * time.Minute, MaxCacheEntries: 500, MaxWorkers: 4},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewIncrementalTypeChecker(tt.config)
			
			if checker.config.CacheTTL != tt.expected.CacheTTL {
				t.Errorf("Expected CacheTTL %v, got %v", tt.expected.CacheTTL, checker.config.CacheTTL)
			}
			
			if checker.config.MaxCacheEntries != tt.expected.MaxCacheEntries {
				t.Errorf("Expected MaxCacheEntries %d, got %d", tt.expected.MaxCacheEntries, checker.config.MaxCacheEntries)
			}
			
			if checker.config.MaxWorkers != tt.expected.MaxWorkers {
				t.Errorf("Expected MaxWorkers %d, got %d", tt.expected.MaxWorkers, checker.config.MaxWorkers)
			}
			
			if checker.cache == nil {
				t.Error("Expected cache to be initialized")
			}
			
			if checker.dependencies == nil {
				t.Error("Expected dependencies to be initialized")
			}
		})
	}
}

func TestTypeCheckerCheckPackage(t *testing.T) {
	checker := NewIncrementalTypeChecker(DefaultTypeCheckerConfig())
	
	// Create test files
	fset := token.NewFileSet()
	file1, err := parser.ParseFile(fset, "main.go", `
package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}
	
	ctx := context.Background()
	files := []*ast.File{file1}
	
	// First check - should perform type checking
	result1, err := checker.CheckPackage(ctx, "main", files, fset)
	if err != nil {
		t.Errorf("Type checking failed: %v", err)
	}
	
	if result1 == nil {
		t.Fatal("Expected result to be non-nil")
	}
	
	if result1.Duration <= 0 {
		t.Error("Expected positive duration")
	}
	
	// Second check - should hit cache
	result2, err := checker.CheckPackage(ctx, "main", files, fset)
	if err != nil {
		t.Errorf("Cached type checking failed: %v", err)
	}
	
	if result2 == nil {
		t.Fatal("Expected cached result to be non-nil")
	}
	
	// Check statistics
	stats := checker.GetStatistics()
	if stats["cache_hits"].(int64) == 0 {
		t.Error("Expected at least one cache hit")
	}
}

func TestTypeCheckerCheckPackagesParallel(t *testing.T) {
	checker := NewIncrementalTypeChecker(&TypeCheckerConfig{
		EnableCaching:    true,
		CacheTTL:         time.Hour,
		MaxCacheEntries:  100,
		ParallelChecking: true,
		MaxWorkers:       2,
		EnableMetrics:    true,
	})
	
	// Create test packages
	fset := token.NewFileSet()
	
	file1, _ := parser.ParseFile(fset, "pkg1/main.go", "package pkg1\nfunc Func1() {}", parser.ParseComments)
	file2, _ := parser.ParseFile(fset, "pkg2/main.go", "package pkg2\nfunc Func2() {}", parser.ParseComments)
	
	packages := map[string][]*ast.File{
		"pkg1": {file1},
		"pkg2": {file2},
	}
	
	ctx := context.Background()
	start := time.Now()
	
	results, err := checker.CheckPackages(ctx, packages, fset)
	duration := time.Since(start)
	
	if err != nil {
		t.Errorf("Parallel type checking failed: %v", err)
	}
	
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	
	if results["pkg1"] == nil || results["pkg2"] == nil {
		t.Error("Expected results for both packages")
	}
	
	t.Logf("Parallel type checking completed in %v", duration)
}

func TestTypeCheckerCheckPackagesSequential(t *testing.T) {
	checker := NewIncrementalTypeChecker(&TypeCheckerConfig{
		EnableCaching:    true,
		CacheTTL:         time.Hour,
		MaxCacheEntries:  100,
		ParallelChecking: false,
		MaxWorkers:       1,
		EnableMetrics:    true,
	})
	
	// Create test packages
	fset := token.NewFileSet()
	
	file1, _ := parser.ParseFile(fset, "pkg1/main.go", "package pkg1\nfunc Func1() {}", parser.ParseComments)
	file2, _ := parser.ParseFile(fset, "pkg2/main.go", "package pkg2\nfunc Func2() {}", parser.ParseComments)
	
	packages := map[string][]*ast.File{
		"pkg1": {file1},
		"pkg2": {file2},
	}
	
	ctx := context.Background()
	
	results, err := checker.CheckPackages(ctx, packages, fset)
	if err != nil {
		t.Errorf("Sequential type checking failed: %v", err)
	}
	
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestTypeCheckerCacheInvalidation(t *testing.T) {
	checker := NewIncrementalTypeChecker(DefaultTypeCheckerConfig())
	
	// Create test files
	fset := token.NewFileSet()
	file1, _ := parser.ParseFile(fset, "pkg1.go", "package pkg1", parser.ParseComments)
	file2, _ := parser.ParseFile(fset, "pkg2.go", "package pkg2", parser.ParseComments)
	
	ctx := context.Background()
	
	// Check packages to populate cache
	checker.CheckPackage(ctx, "pkg1", []*ast.File{file1}, fset)
	checker.CheckPackage(ctx, "pkg2", []*ast.File{file2}, fset)
	
	// Verify cache has entries
	stats := checker.GetStatistics()
	if stats["cached_packages"].(int) != 2 {
		t.Error("Expected 2 cached packages")
	}
	
	// Invalidate pkg1
	checker.InvalidateCache("pkg1")
	
	// Check cache size
	stats = checker.GetStatistics()
	if stats["cached_packages"].(int) != 1 {
		t.Error("Expected 1 cached package after invalidation")
	}
}

func TestTypeCheckerCacheTTL(t *testing.T) {
	checker := NewIncrementalTypeChecker(&TypeCheckerConfig{
		EnableCaching:   true,
		CacheTTL:        10 * time.Millisecond,
		MaxCacheEntries: 100,
		EnableMetrics:   true,
	})
	
	// Create test file
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "test.go", "package test", parser.ParseComments)
	
	ctx := context.Background()
	
	// First check
	checker.CheckPackage(ctx, "test", []*ast.File{file}, fset)
	
	// Wait for TTL to expire
	time.Sleep(15 * time.Millisecond)
	
	// Second check - should not hit cache due to TTL
	checker.CheckPackage(ctx, "test", []*ast.File{file}, fset)
	
	stats := checker.GetStatistics()
	if stats["cache_misses"].(int64) < 2 {
		t.Error("Expected cache miss due to TTL expiration")
	}
}

func TestTypeCheckerContextCancellation(t *testing.T) {
	checker := NewIncrementalTypeChecker(DefaultTypeCheckerConfig())
	
	// Create test files
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "test.go", "package test", parser.ParseComments)
	
	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	
	packages := map[string][]*ast.File{
		"test": {file},
	}
	
	// Should handle context cancellation
	_, err := checker.CheckPackages(ctx, packages, fset)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

func TestTypeCheckerClear(t *testing.T) {
	checker := NewIncrementalTypeChecker(DefaultTypeCheckerConfig())
	
	// Add some cached results
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "test.go", "package test", parser.ParseComments)
	
	ctx := context.Background()
	checker.CheckPackage(ctx, "test", []*ast.File{file}, fset)
	
	// Verify cache has entries
	stats := checker.GetStatistics()
	if stats["cached_packages"].(int) == 0 {
		t.Error("Expected cached packages before clear")
	}
	
	// Clear cache
	checker.Clear()
	
	// Verify cache is empty
	stats = checker.GetStatistics()
	if stats["cached_packages"].(int) != 0 {
		t.Error("Expected no cached packages after clear")
	}
}

func TestTypeCheckerMetricsDisabled(t *testing.T) {
	checker := NewIncrementalTypeChecker(&TypeCheckerConfig{
		EnableMetrics: false,
	})
	
	// Create test file
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "test.go", "package test", parser.ParseComments)
	
	ctx := context.Background()
	
	// Perform operations
	checker.CheckPackage(ctx, "test", []*ast.File{file}, fset)
	checker.CheckPackage(ctx, "test", []*ast.File{file}, fset)
	
	// Metrics should remain zero
	stats := checker.GetStatistics()
	if stats["cache_hits"].(int64) != 0 || stats["cache_misses"].(int64) != 0 {
		t.Error("Expected metrics to be disabled")
	}
}

func TestTypeCheckerEdgeCases(t *testing.T) {
	checker := NewIncrementalTypeChecker(DefaultTypeCheckerConfig())
	ctx := context.Background()
	fset := token.NewFileSet()
	
	t.Run("EmptyFileList", func(t *testing.T) {
		result, err := checker.CheckPackage(ctx, "empty", []*ast.File{}, fset)
		if err == nil {
			t.Error("Expected error for empty package")
		}
		if result == nil {
			t.Error("Expected result even for empty package")
		}
		if result != nil && result.Error == nil {
			t.Error("Expected result to contain error for empty package")
		}
	})
	
	t.Run("NilFiles", func(t *testing.T) {
		files := []*ast.File{nil}
		result, err := checker.CheckPackage(ctx, "nil", files, fset)
		if result == nil {
			t.Error("Expected result even with nil files")
		}
		// Error is expected but shouldn't crash
		_ = err
	})
	
	t.Run("CacheEviction", func(t *testing.T) {
		smallChecker := NewIncrementalTypeChecker(&TypeCheckerConfig{
			EnableCaching:   true,
			MaxCacheEntries: 2,
			CacheTTL:        time.Hour,
		})
		
		// Create multiple packages to trigger eviction
		for i := 0; i < 5; i++ {
			file, _ := parser.ParseFile(fset, "test.go", "package test", parser.ParseComments)
			pkgName := fmt.Sprintf("pkg%d", i)
			smallChecker.CheckPackage(ctx, pkgName, []*ast.File{file}, fset)
		}
		
		stats := smallChecker.GetStatistics()
		if stats["cached_packages"].(int) > 2 {
			t.Error("Expected cache eviction to limit entries")
		}
	})
}

func TestTypeCheckerCacheValidation(t *testing.T) {
	checker := NewIncrementalTypeChecker(DefaultTypeCheckerConfig())
	
	// Create test files
	fset := token.NewFileSet()
	file1, _ := parser.ParseFile(fset, "test1.go", "package test", parser.ParseComments)
	file2, _ := parser.ParseFile(fset, "test2.go", "package test", parser.ParseComments)
	
	files1 := []*ast.File{file1}
	files2 := []*ast.File{file1, file2}
	
	ctx := context.Background()
	
	// Cache result with one file
	checker.CheckPackage(ctx, "test", files1, fset)
	
	// Check with different file list - should miss cache
	result, _ := checker.CheckPackage(ctx, "test", files2, fset)
	
	stats := checker.GetStatistics()
	if stats["cache_misses"].(int64) < 2 {
		t.Error("Expected cache miss due to different file list")
	}
	
	// Verify result is still valid
	if result == nil {
		t.Error("Expected valid result even on cache miss")
	}
}

func TestTypeCheckerPerformance(t *testing.T) {
	checker := NewIncrementalTypeChecker(&TypeCheckerConfig{
		EnableCaching:    true,
		CacheTTL:         time.Hour,
		MaxCacheEntries:  100,
		ParallelChecking: true,
		MaxWorkers:       4,
		EnableMetrics:    true,
	})
	
	// Create multiple test packages
	packages := make(map[string][]*ast.File)
	fset := token.NewFileSet()
	
	for i := 0; i < 10; i++ {
		code := fmt.Sprintf("package pkg%d\nfunc Func%d() {}", i, i)
		file, _ := parser.ParseFile(fset, fmt.Sprintf("pkg%d.go", i), code, parser.ParseComments)
		packages[fmt.Sprintf("pkg%d", i)] = []*ast.File{file}
	}
	
	ctx := context.Background()
	start := time.Now()
	
	// Parallel type checking
	results, err := checker.CheckPackages(ctx, packages, fset)
	duration := time.Since(start)
	
	if err != nil {
		t.Errorf("Parallel type checking failed: %v", err)
	}
	
	if len(results) != 10 {
		t.Errorf("Expected 10 results, got %d", len(results))
	}
	
	t.Logf("Parallel type checking of 10 packages completed in %v", duration)
	
	// Check performance metrics
	stats := checker.GetStatistics()
	checksRun := stats["checks_run"].(int64)
	if checksRun != 10 {
		t.Errorf("Expected 10 checks run, got %d", checksRun)
	}
}

func TestTypeCheckerConcurrency(t *testing.T) {
	checker := NewIncrementalTypeChecker(DefaultTypeCheckerConfig())
	
	// Create test file
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "concurrent.go", "package concurrent", parser.ParseComments)
	files := []*ast.File{file}
	
	ctx := context.Background()
	var wg sync.WaitGroup
	numGoroutines := 5
	
	// Concurrent type checking
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			pkgName := fmt.Sprintf("concurrent%d", id)
			result, err := checker.CheckPackage(ctx, pkgName, files, fset)
			if err != nil {
				t.Errorf("Concurrent type checking failed: %v", err)
				return
			}
			if result == nil {
				t.Error("Expected non-nil result from concurrent checking")
			}
		}(i)
	}
	
	wg.Wait()
	
	// Verify consistent state
	stats := checker.GetStatistics()
	if stats["checks_run"].(int64) != int64(numGoroutines) {
		t.Errorf("Expected %d checks run, got %v", numGoroutines, stats["checks_run"])
	}
}

func TestTypeCheckerCacheOperations(t *testing.T) {
	checker := NewIncrementalTypeChecker(DefaultTypeCheckerConfig())
	
	// Test getCachedResult with non-existent package
	result := checker.getCachedResult("nonexistent", []*ast.File{})
	if result != nil {
		t.Error("Expected nil result for non-existent package")
	}
	
	// Test cacheResult
	testResult := &TypeCheckResult{
		Package:   nil,
		Info:      nil,
		Error:     nil,
		Duration:  time.Millisecond,
		CheckTime: time.Now(),
		FilePaths: []string{"test.go"},
	}
	
	checker.cacheResult("test", testResult)
	
	// Verify it was cached
	stats := checker.GetStatistics()
	if stats["cached_packages"].(int) != 1 {
		t.Error("Expected 1 cached package")
	}
}

func TestTypeCheckerEviction(t *testing.T) {
	checker := NewIncrementalTypeChecker(&TypeCheckerConfig{
		EnableCaching:   true,
		MaxCacheEntries: 2,
		CacheTTL:        time.Hour,
	})
	
	fset := token.NewFileSet()
	ctx := context.Background()
	
	// Add entries to trigger eviction
	for i := 0; i < 5; i++ {
		file, _ := parser.ParseFile(fset, "test.go", "package test", parser.ParseComments)
		pkgName := fmt.Sprintf("pkg%d", i)
		checker.CheckPackage(ctx, pkgName, []*ast.File{file}, fset)
	}
	
	// Should only have MaxCacheEntries
	stats := checker.GetStatistics()
	if stats["cached_packages"].(int) > 2 {
		t.Error("Expected cache eviction to limit entries to 2")
	}
}

func TestTypeCheckerInvalidSyntax(t *testing.T) {
	checker := NewIncrementalTypeChecker(DefaultTypeCheckerConfig())
	
	// Create file with syntax error that results in nil file
	fset := token.NewFileSet()
	_, parseErr := parser.ParseFile(fset, "invalid.go", "package test\nfunc {", parser.ParseComments)
	
	ctx := context.Background()
	
	// Test with nil file (simulating complete parse failure)
	result, err := checker.CheckPackage(ctx, "test", []*ast.File{nil}, fset)
	
	// Should get a result even with nil files
	if result == nil {
		t.Error("Expected result even with nil files")
	}
	
	// Error is expected for nil files  
	if err == nil && result.Error == nil {
		t.Error("Expected error for nil files")
	}
	
	// Verify parse error occurred
	if parseErr == nil {
		t.Error("Expected parse error for invalid syntax")
	}
}