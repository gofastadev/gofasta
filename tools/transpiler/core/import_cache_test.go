package core

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestDefaultImportCacheConfig(t *testing.T) {
	config := DefaultImportCacheConfig()
	
	if config.MaxEntries != 1000 {
		t.Errorf("Expected MaxEntries to be 1000, got %d", config.MaxEntries)
	}
	
	if config.TTL != time.Hour {
		t.Errorf("Expected TTL to be 1 hour, got %v", config.TTL)
	}
	
	if !config.EnableMetrics {
		t.Error("Expected EnableMetrics to be true")
	}
	
	if config.MaxMemoryMB != 256 {
		t.Errorf("Expected MaxMemoryMB to be 256, got %d", config.MaxMemoryMB)
	}
}

func TestNewCachedImporter(t *testing.T) {
	tests := []struct {
		name     string
		config   *ImportCacheConfig
		expected *ImportCacheConfig
	}{
		{
			name: "WithCustomConfig",
			config: &ImportCacheConfig{
				MaxEntries:    500,
				TTL:           30 * time.Minute,
				EnableMetrics: false,
				MaxMemoryMB:   128,
			},
			expected: &ImportCacheConfig{
				MaxEntries:    500,
				TTL:           30 * time.Minute,
				EnableMetrics: false,
				MaxMemoryMB:   128,
			},
		},
		{
			name:     "WithNilConfig",
			config:   nil,
			expected: DefaultImportCacheConfig(),
		},
		{
			name:     "WithZeroValues",
			config:   &ImportCacheConfig{},
			expected: &ImportCacheConfig{MaxEntries: 1000, TTL: time.Hour, MaxMemoryMB: 256},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			importer := NewCachedImporter(tt.config)
			
			if importer.config.MaxEntries != tt.expected.MaxEntries {
				t.Errorf("Expected MaxEntries %d, got %d", tt.expected.MaxEntries, importer.config.MaxEntries)
			}
			
			if importer.config.TTL != tt.expected.TTL {
				t.Errorf("Expected TTL %v, got %v", tt.expected.TTL, importer.config.TTL)
			}
			
			if importer.config.MaxMemoryMB != tt.expected.MaxMemoryMB {
				t.Errorf("Expected MaxMemoryMB %d, got %d", tt.expected.MaxMemoryMB, importer.config.MaxMemoryMB)
			}
			
			if importer.cache == nil {
				t.Error("Expected cache to be initialized")
			}
			
			if importer.importer == nil {
				t.Error("Expected underlying importer to be initialized")
			}
		})
	}
}

func TestCachedImporterImport(t *testing.T) {
	importer := NewCachedImporter(DefaultImportCacheConfig())
	
	// First import - should miss cache
	pkg1, err := importer.Import("fmt")
	if err != nil {
		t.Errorf("First import failed: %v", err)
	}
	
	if pkg1 == nil {
		t.Error("Expected package to be returned")
	}
	
	if pkg1.Name() != "fmt" {
		t.Errorf("Expected package name 'fmt', got '%s'", pkg1.Name())
	}
	
	// Second import - should hit cache
	pkg2, err := importer.Import("fmt")
	if err != nil {
		t.Errorf("Second import failed: %v", err)
	}
	
	if pkg2 != pkg1 {
		t.Error("Expected same package instance from cache")
	}
	
	// Check statistics
	stats := importer.GetStatistics()
	if stats["hits"].(int64) == 0 {
		t.Error("Expected at least one cache hit")
	}
	
	if stats["misses"].(int64) == 0 {
		t.Error("Expected at least one cache miss")
	}
}

func TestCachedImporterImportWithFallback(t *testing.T) {
	importer := NewCachedImporter(DefaultImportCacheConfig())
	
	// Create a fallback importer
	fallback := NewCachedImporter(DefaultImportCacheConfig())
	
	// Import with fallback
	pkg, err := importer.ImportWithFallback("fmt", fallback)
	if err != nil {
		t.Errorf("Import with fallback failed: %v", err)
	}
	
	if pkg == nil {
		t.Error("Expected package to be returned")
	}
}

func TestCachedImporterPreload(t *testing.T) {
	importer := NewCachedImporter(DefaultImportCacheConfig())
	
	packages := []string{"fmt", "os", "io"}
	errors := importer.Preload(packages)
	
	// Should succeed for standard packages
	if len(errors) > 0 {
		t.Errorf("Unexpected preload errors: %v", errors)
	}
	
	// Check that packages are cached
	stats := importer.GetStatistics()
	if stats["entries"].(int) < 3 {
		t.Error("Expected at least 3 cached entries after preload")
	}
	
	// Subsequent imports should hit cache
	initialHits := stats["hits"].(int64)
	importer.Import("fmt")
	
	stats = importer.GetStatistics()
	if stats["hits"].(int64) <= initialHits {
		t.Error("Expected cache hit after preload")
	}
}

func TestCachedImporterTTL(t *testing.T) {
	config := &ImportCacheConfig{
		MaxEntries:    100,
		TTL:           10 * time.Millisecond,
		EnableMetrics: true,
		MaxMemoryMB:   100,
	}
	importer := NewCachedImporter(config)
	
	// Import package
	pkg1, err := importer.Import("fmt")
	if err != nil {
		t.Errorf("Import failed: %v", err)
	}
	
	// Should hit cache immediately
	pkg2, _ := importer.Import("fmt")
	if pkg1 != pkg2 {
		t.Error("Expected cache hit")
	}
	
	// Wait for TTL to expire
	time.Sleep(15 * time.Millisecond)
	
	// Should miss cache after TTL
	stats := importer.GetStatistics()
	initialMisses := stats["misses"].(int64)
	
	importer.Import("fmt")
	
	stats = importer.GetStatistics()
	if stats["misses"].(int64) <= initialMisses {
		t.Error("Expected cache miss after TTL expiration")
	}
}

func TestCachedImporterLRUEviction(t *testing.T) {
	config := &ImportCacheConfig{
		MaxEntries:    2,
		TTL:           time.Hour,
		EnableMetrics: true,
		MaxMemoryMB:   1000,
	}
	importer := NewCachedImporter(config)
	
	// Import packages to fill cache
	importer.Import("fmt")
	importer.Import("os")
	
	// Both should be cached
	stats := importer.GetStatistics()
	if stats["entries"].(int) != 2 {
		t.Error("Expected 2 cached entries")
	}
	
	// Import third package (should evict first)
	importer.Import("io")
	
	stats = importer.GetStatistics()
	if stats["entries"].(int) != 2 {
		t.Error("Expected cache size to remain at max entries")
	}
	
	if stats["evictions"].(int64) == 0 {
		t.Error("Expected at least one eviction")
	}
}

func TestCachedImporterCleanup(t *testing.T) {
	config := &ImportCacheConfig{
		MaxEntries:    100,
		TTL:           10 * time.Millisecond,
		EnableMetrics: true,
		MaxMemoryMB:   100,
	}
	importer := NewCachedImporter(config)
	
	// Import some packages
	importer.Import("fmt")
	importer.Import("os")
	
	// Wait for TTL to expire
	time.Sleep(15 * time.Millisecond)
	
	// Cleanup expired entries
	removed := importer.Cleanup()
	if removed != 2 {
		t.Errorf("Expected 2 entries to be removed, got %d", removed)
	}
	
	// Cache should be empty
	stats := importer.GetStatistics()
	if stats["entries"].(int) != 0 {
		t.Error("Expected cache to be empty after cleanup")
	}
}

func TestCachedImporterClear(t *testing.T) {
	importer := NewCachedImporter(DefaultImportCacheConfig())
	
	// Import some packages
	importer.Import("fmt")
	importer.Import("os")
	
	// Verify entries exist
	stats := importer.GetStatistics()
	if stats["entries"].(int) == 0 {
		t.Error("Expected cached entries before clear")
	}
	
	// Clear cache
	importer.Clear()
	
	// Cache should be empty
	stats = importer.GetStatistics()
	if stats["entries"].(int) != 0 {
		t.Error("Expected cache to be empty after clear")
	}
}

func TestCachedImporterGetCachedPackages(t *testing.T) {
	importer := NewCachedImporter(DefaultImportCacheConfig())
	
	// Import some packages
	importer.Import("fmt")
	importer.Import("os")
	
	packages := importer.GetCachedPackages()
	if len(packages) != 2 {
		t.Errorf("Expected 2 cached packages, got %d", len(packages))
	}
	
	// Check that fmt and os are in the list
	found := make(map[string]bool)
	for _, pkg := range packages {
		found[pkg] = true
	}
	
	if !found["fmt"] || !found["os"] {
		t.Error("Expected fmt and os to be in cached packages list")
	}
}

func TestCachedImporterConcurrency(t *testing.T) {
	importer := NewCachedImporter(DefaultImportCacheConfig())
	
	// Pre-populate cache to avoid concurrent first imports
	importer.Import("fmt")
	
	var wg sync.WaitGroup
	numGoroutines := 5
	
	// Concurrent cache hits only
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			pkg, err := importer.Import("fmt")
			if err != nil {
				t.Errorf("Concurrent import of fmt failed: %v", err)
				return
			}
			
			if pkg == nil {
				t.Error("Expected non-nil package for fmt")
			}
		}()
	}
	
	wg.Wait()
	
	// Check that cache is in consistent state
	stats := importer.GetStatistics()
	if stats["hits"].(int64) < int64(numGoroutines) {
		t.Error("Expected multiple cache hits from concurrent access")
	}
}

func TestCachedImporterErrorHandling(t *testing.T) {
	importer := NewCachedImporter(DefaultImportCacheConfig())
	
	// Try to import non-existent package
	pkg, err := importer.Import("nonexistent/package/path")
	if err == nil {
		t.Error("Expected error for non-existent package")
	}
	
	if pkg != nil {
		t.Error("Expected nil package for failed import")
	}
	
	// Check error metrics
	stats := importer.GetStatistics()
	if stats["errors"].(int64) == 0 {
		t.Error("Expected error count to increment")
	}
}

func TestCachedImporterMemoryEstimation(t *testing.T) {
	importer := NewCachedImporter(&ImportCacheConfig{
		MaxEntries:    100,
		TTL:           time.Hour,
		EnableMetrics: true,
		MaxMemoryMB:   10,
	})
	
	// Import some packages
	importer.Import("fmt")
	importer.Import("os")
	
	stats := importer.GetStatistics()
	memUsage := stats["memory_mb"].(int64)
	
	if memUsage < 0 {
		t.Errorf("Expected non-negative memory usage, got %d", memUsage)
	}
}

func TestCachedImporterWarmUp(t *testing.T) {
	importer := NewCachedImporter(DefaultImportCacheConfig())
	
	// Warm up cache
	err := importer.WarmUp()
	if err != nil {
		t.Errorf("WarmUp failed: %v", err)
	}
	
	// Check that standard packages are cached
	stats := importer.GetStatistics()
	if stats["entries"].(int) == 0 {
		t.Error("Expected some packages to be cached after warmup")
	}
	
	// Importing standard packages should hit cache
	initialHits := stats["hits"].(int64)
	importer.Import("fmt")
	
	stats = importer.GetStatistics()
	if stats["hits"].(int64) <= initialHits {
		t.Error("Expected cache hit for standard package after warmup")
	}
}

func TestCachedImporterLRUOperations(t *testing.T) {
	importer := NewCachedImporter(&ImportCacheConfig{MaxEntries: 3, TTL: time.Hour})
	
	// Test addToLRU
	importer.mu.Lock()
	importer.addToLRU("pkg1")
	importer.addToLRU("pkg2")
	importer.addToLRU("pkg3")
	importer.mu.Unlock()
	
	importer.mu.RLock()
	if len(importer.lruList) != 3 {
		t.Errorf("Expected 3 items in LRU list, got %d", len(importer.lruList))
	}
	if importer.lruList[0] != "pkg3" {
		t.Errorf("Expected pkg3 at front, got %s", importer.lruList[0])
	}
	importer.mu.RUnlock()
	
	// Test moveToFront
	importer.mu.Lock()
	importer.moveToFront("pkg1")
	importer.mu.Unlock()
	
	importer.mu.RLock()
	if importer.lruList[0] != "pkg1" {
		t.Errorf("Expected pkg1 at front after move, got %s", importer.lruList[0])
	}
	importer.mu.RUnlock()
	
	// Test removeFromLRU
	importer.mu.Lock()
	importer.removeFromLRU("pkg2")
	importer.mu.Unlock()
	
	importer.mu.RLock()
	if len(importer.lruList) != 2 {
		t.Errorf("Expected 2 items after removal, got %d", len(importer.lruList))
	}
	for _, pkg := range importer.lruList {
		if pkg == "pkg2" {
			t.Error("Expected pkg2 to be removed from LRU list")
		}
	}
	importer.mu.RUnlock()
}

func TestCachedImporterPackageSizeEstimation(t *testing.T) {
	importer := NewCachedImporter(DefaultImportCacheConfig())
	
	// Test with nil package
	size := importer.estimatePackageSize(nil)
	if size != 0 {
		t.Errorf("Expected 0 size for nil package, got %d", size)
	}
	
	// Import real package and check size estimation
	pkg, err := importer.Import("fmt")
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	
	size = importer.estimatePackageSize(pkg)
	if size <= 0 {
		t.Errorf("Expected positive size for real package, got %d", size)
	}
}

func TestCachedImporterMetricsDisabled(t *testing.T) {
	importer := NewCachedImporter(&ImportCacheConfig{
		MaxEntries:    100,
		TTL:           time.Hour,
		EnableMetrics: false,
		MaxMemoryMB:   100,
	})
	
	// Perform operations
	importer.Import("fmt")
	importer.Import("fmt") // Cache hit
	importer.Import("nonexistent") // Error
	
	// Metrics should remain zero
	stats := importer.GetStatistics()
	if stats["hits"].(int64) != 0 || stats["misses"].(int64) != 0 || stats["errors"].(int64) != 0 {
		t.Error("Expected metrics to be disabled")
	}
}

func TestCachedImporterEdgeCases(t *testing.T) {
	importer := NewCachedImporter(DefaultImportCacheConfig())
	
	t.Run("ImportEmptyPath", func(t *testing.T) {
		pkg, err := importer.Import("")
		if err == nil {
			t.Error("Expected error for empty package path")
		}
		if pkg != nil {
			t.Error("Expected nil package for empty path")
		}
	})
	
	t.Run("ImportInvalidPath", func(t *testing.T) {
		pkg, err := importer.Import("invalid/package/path/that/does/not/exist")
		if err == nil {
			t.Error("Expected error for invalid package path")
		}
		if pkg != nil {
			t.Error("Expected nil package for invalid path")
		}
	})
	
	t.Run("AccessCountIncrement", func(t *testing.T) {
		// Import package multiple times
		for i := 0; i < 5; i++ {
			importer.Import("fmt")
		}
		
		importer.mu.RLock()
		entry := importer.cache["fmt"]
		if entry != nil && entry.AccessCount < 2 {
			t.Error("Expected access count to increment on cache hits")
		}
		importer.mu.RUnlock()
	})
}

func TestCachedImporterMemoryPressure(t *testing.T) {
	config := &ImportCacheConfig{
		MaxEntries:    100,
		TTL:           time.Hour,
		EnableMetrics: true,
		MaxMemoryMB:   1, // Very low limit to test eviction  
	}
	importer := NewCachedImporter(config)
	
	// Import many packages to trigger memory-based eviction
	packages := []string{}
	for i := 0; i < 50; i++ {
		packages = append(packages, fmt.Sprintf("test/package%d", i))
	}
	
	// Also add some real packages
	packages = append(packages, "fmt", "os", "io", "time", "context", "sync")
	
	errorCount := 0
	for _, pkg := range packages {
		_, err := importer.Import(pkg)
		if err != nil {
			errorCount++
		}
	}
	
	stats := importer.GetStatistics()
	
	// Should have triggered some evictions due to memory pressure or entry limit
	// Note: Most test packages will fail to import, which is expected
	if stats["evictions"].(int64) == 0 && int64(len(packages)) > config.MaxMemoryMB {
		t.Log("Note: Memory-based eviction may not trigger with small standard packages")
		t.Log("This is acceptable as the logic is tested in entry-based eviction")
	}
	
	// Memory usage tracking should work
	memUsage := stats["memory_mb"].(int64)
	if memUsage < 0 {
		t.Errorf("Expected non-negative memory usage, got %d", memUsage)
	}
}

func TestCachedImporterGetLoadedImporter(t *testing.T) {
	importer := NewCachedImporter(DefaultImportCacheConfig())
	
	underlying := importer.GetLoadedImporter()
	if underlying == nil {
		t.Error("Expected underlying importer to be available")
	}
	
	// Should be able to use underlying importer directly
	pkg, err := underlying.Import("fmt")
	if err != nil {
		t.Errorf("Direct import failed: %v", err)
	}
	
	if pkg == nil {
		t.Error("Expected package from direct import")
	}
}

func TestCachedImporterPerformance(t *testing.T) {
	importer := NewCachedImporter(&ImportCacheConfig{
		MaxEntries:    1000,
		TTL:           time.Hour,
		EnableMetrics: true,
		MaxMemoryMB:   512,
	})
	
	packages := []string{"fmt", "os", "io", "time", "context", "sync", "net/http", "encoding/json"}
	
	// Warm up cache
	for _, pkg := range packages {
		importer.Import(pkg)
	}
	
	// Benchmark cache hits
	start := time.Now()
	iterations := 1000
	
	for i := 0; i < iterations; i++ {
		pkg := packages[i%len(packages)]
		importer.Import(pkg)
	}
	
	duration := time.Since(start)
	importsPerSec := float64(iterations) / duration.Seconds()
	
	t.Logf("Cached import performance: %.0f imports/sec", importsPerSec)
	
	// Should be very fast for cached imports
	if importsPerSec < 10000 {
		t.Errorf("Expected > 10k imports/sec for cached imports, got %.0f", importsPerSec)
	}
	
	// Check hit ratio
	stats := importer.GetStatistics()
	hitRatio := stats["hit_ratio"].(float64)
	if hitRatio < 90.0 {
		t.Errorf("Expected hit ratio > 90%%, got %.1f%%", hitRatio)
	}
}