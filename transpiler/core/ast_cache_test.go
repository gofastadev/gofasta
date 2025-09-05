package core

import (
	"go/parser"
	"go/token"
	"testing"
	"time"
)

func TestDefaultASTCacheConfig(t *testing.T) {
	config := DefaultASTCacheConfig()
	
	if config.MaxEntries != 1000 {
		t.Errorf("Expected MaxEntries to be 1000, got %d", config.MaxEntries)
	}
	
	if config.TTL != time.Hour {
		t.Errorf("Expected TTL to be 1 hour, got %v", config.TTL)
	}
	
	if config.MaxMemoryMB != 512 {
		t.Errorf("Expected MaxMemoryMB to be 512, got %d", config.MaxMemoryMB)
	}
	
	if !config.EnableMetrics {
		t.Error("Expected EnableMetrics to be true")
	}
}

func TestNewASTCache(t *testing.T) {
	tests := []struct {
		name     string
		config   *ASTCacheConfig
		expected *ASTCacheConfig
	}{
		{
			name:   "WithCustomConfig",
			config: &ASTCacheConfig{MaxEntries: 500, TTL: 30 * time.Minute, MaxMemoryMB: 256},
			expected: &ASTCacheConfig{MaxEntries: 500, TTL: 30 * time.Minute, MaxMemoryMB: 256},
		},
		{
			name:     "WithNilConfig",
			config:   nil,
			expected: DefaultASTCacheConfig(),
		},
		{
			name:     "WithZeroValues",
			config:   &ASTCacheConfig{},
			expected: &ASTCacheConfig{MaxEntries: 1000, TTL: time.Hour, MaxMemoryMB: 512},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewASTCache(tt.config)
			
			if cache.config.MaxEntries != tt.expected.MaxEntries {
				t.Errorf("Expected MaxEntries %d, got %d", tt.expected.MaxEntries, cache.config.MaxEntries)
			}
			
			if cache.config.TTL != tt.expected.TTL {
				t.Errorf("Expected TTL %v, got %v", tt.expected.TTL, cache.config.TTL)
			}
			
			if cache.config.MaxMemoryMB != tt.expected.MaxMemoryMB {
				t.Errorf("Expected MaxMemoryMB %d, got %d", tt.expected.MaxMemoryMB, cache.config.MaxMemoryMB)
			}
			
			if cache.cache == nil {
				t.Error("Expected cache map to be initialized")
			}
			
			if cache.lruList == nil {
				t.Error("Expected LRU list to be initialized")
			}
		})
	}
}

func TestASTCachePutGet(t *testing.T) {
	cache := NewASTCache(DefaultASTCacheConfig())
	
	// Create test AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", "package main\nfunc main() {}", parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}
	
	modTime := time.Now()
	size := int64(100)
	
	// Test Put
	cache.Put("test.go", file, fset, modTime, size)
	
	// Test Get - should hit
	gotFile, gotFileSet, hit := cache.Get("test.go", modTime, size)
	if !hit {
		t.Error("Expected cache hit")
	}
	
	if gotFile == nil {
		t.Error("Expected file to be returned")
	}
	
	if gotFileSet == nil {
		t.Error("Expected fileset to be returned")
	}
	
	// Test Get with different modTime - should miss
	newModTime := modTime.Add(time.Second)
	_, _, hit = cache.Get("test.go", newModTime, size)
	if hit {
		t.Error("Expected cache miss due to different modTime")
	}
	
	// Test Get with different size - should miss
	_, _, hit = cache.Get("test.go", modTime, size+1)
	if hit {
		t.Error("Expected cache miss due to different size")
	}
}

func TestASTCacheTTL(t *testing.T) {
	config := &ASTCacheConfig{
		MaxEntries:  10,
		TTL:         10 * time.Millisecond,
		MaxMemoryMB: 100,
	}
	cache := NewASTCache(config)
	
	// Create test AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", "package main", parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}
	
	modTime := time.Now()
	size := int64(50)
	
	// Put entry
	cache.Put("test.go", file, fset, modTime, size)
	
	// Should hit immediately
	_, _, hit := cache.Get("test.go", modTime, size)
	if !hit {
		t.Error("Expected cache hit")
	}
	
	// Wait for TTL to expire
	time.Sleep(15 * time.Millisecond)
	
	// Should miss after TTL
	_, _, hit = cache.Get("test.go", modTime, size)
	if hit {
		t.Error("Expected cache miss after TTL expiration")
	}
}

func TestASTCacheLRUEviction(t *testing.T) {
	config := &ASTCacheConfig{
		MaxEntries:  2,
		TTL:         time.Hour,
		MaxMemoryMB: 1000,
	}
	cache := NewASTCache(config)
	
	// Create test ASTs
	fset := token.NewFileSet()
	file1, _ := parser.ParseFile(fset, "test1.go", "package main", parser.ParseComments)
	file2, _ := parser.ParseFile(fset, "test2.go", "package main", parser.ParseComments)
	file3, _ := parser.ParseFile(fset, "test3.go", "package main", parser.ParseComments)
	
	modTime := time.Now()
	size := int64(50)
	
	// Put first two entries
	cache.Put("test1.go", file1, fset, modTime, size)
	cache.Put("test2.go", file2, fset, modTime, size)
	
	// Both should be available
	_, _, hit1 := cache.Get("test1.go", modTime, size)
	_, _, hit2 := cache.Get("test2.go", modTime, size)
	if !hit1 || !hit2 {
		t.Error("Expected both entries to be cached")
	}
	
	// Put third entry (should evict first)
	cache.Put("test3.go", file3, fset, modTime, size)
	
	// First should be evicted, others should be available
	_, _, hit1 = cache.Get("test1.go", modTime, size)
	_, _, hit2 = cache.Get("test2.go", modTime, size)
	_, _, hit3 := cache.Get("test3.go", modTime, size)
	
	if hit1 {
		t.Error("Expected first entry to be evicted")
	}
	if !hit2 {
		t.Error("Expected second entry to be cached")
	}
	if !hit3 {
		t.Error("Expected third entry to be cached")
	}
}

func TestASTCacheCleanup(t *testing.T) {
	config := &ASTCacheConfig{
		MaxEntries:  10,
		TTL:         10 * time.Millisecond,
		MaxMemoryMB: 100,
	}
	cache := NewASTCache(config)
	
	// Create test AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", "package main", parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}
	
	modTime := time.Now()
	size := int64(50)
	
	// Put entry
	cache.Put("test.go", file, fset, modTime, size)
	
	// Wait for TTL to expire
	time.Sleep(15 * time.Millisecond)
	
	// Cleanup expired entries
	removed := cache.Cleanup()
	if removed != 1 {
		t.Errorf("Expected 1 entry to be removed, got %d", removed)
	}
	
	// Entry should no longer be available
	_, _, hit := cache.Get("test.go", modTime, size)
	if hit {
		t.Error("Expected cache miss after cleanup")
	}
}

func TestASTCacheStatistics(t *testing.T) {
	config := &ASTCacheConfig{
		MaxEntries:    10,
		TTL:           time.Hour,
		MaxMemoryMB:   100,
		EnableMetrics: true,
	}
	cache := NewASTCache(config)
	
	// Create test AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", "package main\nfunc main() {}", parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}
	
	modTime := time.Now()
	size := int64(50)
	
	// Initial stats
	stats := cache.GetStatistics()
	if stats["entries"] != 0 {
		t.Error("Expected 0 entries initially")
	}
	
	// Put entry
	cache.Put("test.go", file, fset, modTime, size)
	
	// Get hit
	cache.Get("test.go", modTime, size)
	
	// Get miss
	cache.Get("nonexistent.go", modTime, size)
	
	// Check stats
	stats = cache.GetStatistics()
	if stats["entries"] != 1 {
		t.Errorf("Expected 1 entry, got %v", stats["entries"])
	}
	
	if stats["hits"] != int64(1) {
		t.Errorf("Expected 1 hit, got %v", stats["hits"])
	}
	
	if stats["misses"] != int64(1) {
		t.Errorf("Expected 1 miss, got %v", stats["misses"])
	}
	
	hitRatio := stats["hit_ratio"].(float64)
	if hitRatio != 50.0 {
		t.Errorf("Expected 50%% hit ratio, got %.1f", hitRatio)
	}
}

func TestASTCacheClear(t *testing.T) {
	cache := NewASTCache(DefaultASTCacheConfig())
	
	// Create test AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", "package main", parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}
	
	modTime := time.Now()
	size := int64(50)
	
	// Put entry
	cache.Put("test.go", file, fset, modTime, size)
	
	// Verify entry exists
	_, _, hit := cache.Get("test.go", modTime, size)
	if !hit {
		t.Error("Expected cache hit before clear")
	}
	
	// Clear cache
	cache.Clear()
	
	// Entry should no longer exist
	_, _, hit = cache.Get("test.go", modTime, size)
	if hit {
		t.Error("Expected cache miss after clear")
	}
	
	// Stats should be reset
	stats := cache.GetStatistics()
	if stats["entries"] != 0 {
		t.Errorf("Expected 0 entries after clear, got %v", stats["entries"])
	}
}

func TestASTCacheEdgeCases(t *testing.T) {
	cache := NewASTCache(DefaultASTCacheConfig())
	
	t.Run("GetNonExistentKey", func(t *testing.T) {
		_, _, hit := cache.Get("nonexistent.go", time.Now(), 100)
		if hit {
			t.Error("Expected cache miss for non-existent key")
		}
	})
	
	t.Run("PutNilFile", func(t *testing.T) {
		modTime := time.Now()
		cache.Put("nil.go", nil, nil, modTime, 0)
		
		file, fset, hit := cache.Get("nil.go", modTime, 0)
		if !hit {
			t.Error("Expected cache hit for nil file")
		}
		if file != nil {
			t.Error("Expected nil file")
		}
		if fset != nil {
			t.Error("Expected nil fileset")
		}
	})
	
	t.Run("AccessCountIncrement", func(t *testing.T) {
		fset := token.NewFileSet()
		file, _ := parser.ParseFile(fset, "test.go", "package main", parser.ParseComments)
		
		modTime := time.Now()
		size := int64(50)
		
		cache.Put("access.go", file, fset, modTime, size)
		
		// Access multiple times (5 gets + 1 put = 6 total)
		for i := 0; i < 5; i++ {
			cache.Get("access.go", modTime, size)
		}
		
		cache.mu.RLock()
		entry := cache.cache["access.go"]
		accessCount := entry.AccessCount
		cache.mu.RUnlock()
		
		if accessCount != 6 {
			t.Errorf("Expected access count 6 (1 put + 5 gets), got %d", accessCount)
		}
	})
}

func TestASTCacheMemoryManagement(t *testing.T) {
	config := &ASTCacheConfig{
		MaxEntries:    2, // Low entry limit to test eviction
		TTL:           time.Hour,
		MaxMemoryMB:   1000, // High memory limit so entry limit triggers first
		EnableMetrics: true,
	}
	cache := NewASTCache(config)
	
	// Create test ASTs
	fset := token.NewFileSet()
	file1, _ := parser.ParseFile(fset, "test1.go", "package main\nfunc test1() {}", parser.ParseComments)
	file2, _ := parser.ParseFile(fset, "test2.go", "package main\nfunc test2() {}", parser.ParseComments)
	file3, _ := parser.ParseFile(fset, "test3.go", "package main\nfunc test3() {}", parser.ParseComments)
	
	modTime := time.Now()
	size := int64(50)
	
	// Put entries to exceed MaxEntries
	cache.Put("test1.go", file1, fset, modTime, size)
	cache.Put("test2.go", file2, fset, modTime, size)
	cache.Put("test3.go", file3, fset, modTime, size) // Should trigger eviction
	
	// Entry-based eviction should have occurred
	stats := cache.GetStatistics()
	if stats["entries"].(int) > 2 {
		t.Error("Expected cache size to be limited by MaxEntries")
	}
}

func TestASTCacheConcurrency(t *testing.T) {
	cache := NewASTCache(DefaultASTCacheConfig())
	
	// Create test AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", "package main", parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}
	
	modTime := time.Now()
	size := int64(50)
	
	// Concurrent puts and gets
	done := make(chan bool, 10)
	
	// Concurrent puts
	for i := 0; i < 5; i++ {
		go func(id int) {
			key := "concurrent" + string(rune('0'+id)) + ".go"
			cache.Put(key, file, fset, modTime, size)
			done <- true
		}(i)
	}
	
	// Concurrent gets
	for i := 0; i < 5; i++ {
		go func(id int) {
			key := "concurrent" + string(rune('0'+id)) + ".go"
			cache.Get(key, modTime, size)
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify cache is in consistent state
	stats := cache.GetStatistics()
	if stats["entries"].(int) < 0 {
		t.Error("Expected non-negative entry count")
	}
}

func TestASTCacheNodeCounting(t *testing.T) {
	cache := NewASTCache(DefaultASTCacheConfig())
	
	// Test with nil node
	count := cache.countASTNodes(nil)
	if count != 0 {
		t.Errorf("Expected 0 nodes for nil, got %d", count)
	}
	
	// Test with simple AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", "package main\nfunc main() {}", parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}
	
	count = cache.countASTNodes(file)
	if count <= 0 {
		t.Errorf("Expected positive node count, got %d", count)
	}
}

func TestASTCacheLRUOperations(t *testing.T) {
	cache := NewASTCache(&ASTCacheConfig{MaxEntries: 3, TTL: time.Hour, MaxMemoryMB: 100})
	
	// Test addToLRU
	cache.mu.Lock()
	cache.addToLRU("key1")
	cache.addToLRU("key2")
	cache.addToLRU("key3")
	cache.mu.Unlock()
	
	cache.mu.RLock()
	if len(cache.lruList) != 3 {
		t.Errorf("Expected 3 items in LRU list, got %d", len(cache.lruList))
	}
	if cache.lruList[0] != "key3" {
		t.Errorf("Expected key3 at front, got %s", cache.lruList[0])
	}
	cache.mu.RUnlock()
	
	// Test moveToFront
	cache.mu.Lock()
	cache.moveToFront("key1")
	cache.mu.Unlock()
	
	cache.mu.RLock()
	if cache.lruList[0] != "key1" {
		t.Errorf("Expected key1 at front after move, got %s", cache.lruList[0])
	}
	cache.mu.RUnlock()
	
	// Test removeFromLRU
	cache.mu.Lock()
	cache.removeFromLRU("key2")
	cache.mu.Unlock()
	
	cache.mu.RLock()
	if len(cache.lruList) != 2 {
		t.Errorf("Expected 2 items after removal, got %d", len(cache.lruList))
	}
	for _, key := range cache.lruList {
		if key == "key2" {
			t.Error("Expected key2 to be removed from LRU list")
		}
	}
	cache.mu.RUnlock()
}

func TestASTCacheMemoryEstimation(t *testing.T) {
	config := &ASTCacheConfig{
		MaxEntries:    10,
		TTL:           time.Hour,
		MaxMemoryMB:   100,
		EnableMetrics: true,
	}
	cache := NewASTCache(config)
	
	// Put entry and check memory estimation
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", "package main\nfunc main() {}", parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}
	
	cache.Put("test.go", file, fset, time.Now(), 100)
	
	stats := cache.GetStatistics()
	memoryMB := stats["memory_mb"].(int64)
	if memoryMB < 0 {
		t.Errorf("Expected non-negative memory usage, got %d", memoryMB)
	}
}

func TestASTCacheValidation(t *testing.T) {
	cache := NewASTCache(DefaultASTCacheConfig())
	
	baseTime := time.Now()
	entry := &ASTCacheEntry{
		ModTime:   baseTime,
		Size:      100,
		CacheTime: baseTime,
	}
	
	t.Run("ValidEntry", func(t *testing.T) {
		if !cache.isValidEntry(entry, baseTime, 100) {
			t.Error("Expected entry to be valid")
		}
	})
	
	t.Run("ExpiredEntry", func(t *testing.T) {
		cache.config.TTL = 10 * time.Millisecond
		time.Sleep(15 * time.Millisecond)
		
		if cache.isValidEntry(entry, baseTime, 100) {
			t.Error("Expected entry to be expired")
		}
	})
	
	t.Run("ModifiedFile", func(t *testing.T) {
		cache.config.TTL = time.Hour
		newTime := baseTime.Add(time.Second)
		
		if cache.isValidEntry(entry, newTime, 100) {
			t.Error("Expected entry to be invalid due to modification time")
		}
	})
	
	t.Run("DifferentSize", func(t *testing.T) {
		if cache.isValidEntry(entry, baseTime, 200) {
			t.Error("Expected entry to be invalid due to size change")
		}
	})
}

func TestASTCacheDisabledMetrics(t *testing.T) {
	config := &ASTCacheConfig{
		MaxEntries:    10,
		TTL:           time.Hour,
		MaxMemoryMB:   100,
		EnableMetrics: false,
	}
	cache := NewASTCache(config)
	
	// Create test AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", "package main", parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}
	
	modTime := time.Now()
	size := int64(50)
	
	// Put and get
	cache.Put("test.go", file, fset, modTime, size)
	cache.Get("test.go", modTime, size)
	cache.Get("missing.go", modTime, size)
	
	// Metrics should not be tracked
	stats := cache.GetStatistics()
	if stats["hits"] != int64(0) || stats["misses"] != int64(0) {
		t.Error("Expected metrics to be disabled")
	}
}