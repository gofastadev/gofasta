package core

import (
	"go/token"
	"sync"
	"testing"
	"time"
)

func TestDefaultTokenPoolConfig(t *testing.T) {
	config := DefaultTokenPoolConfig()
	
	if config.InitialSize != 10 {
		t.Errorf("Expected InitialSize to be 10, got %d", config.InitialSize)
	}
	
	if config.MaxSize != 100 {
		t.Errorf("Expected MaxSize to be 100, got %d", config.MaxSize)
	}
	
	if !config.EnableMetrics {
		t.Error("Expected EnableMetrics to be true")
	}
}

func TestNewTokenPool(t *testing.T) {
	tests := []struct {
		name     string
		config   *TokenPoolConfig
		expected *TokenPoolConfig
	}{
		{
			name:     "WithCustomConfig",
			config:   &TokenPoolConfig{InitialSize: 5, MaxSize: 20, EnableMetrics: false},
			expected: &TokenPoolConfig{InitialSize: 5, MaxSize: 20, EnableMetrics: false},
		},
		{
			name:     "WithNilConfig",
			config:   nil,
			expected: DefaultTokenPoolConfig(),
		},
		{
			name:     "WithZeroValues",
			config:   &TokenPoolConfig{},
			expected: &TokenPoolConfig{InitialSize: 10, MaxSize: 100, EnableMetrics: false},
		},
		{
			name:     "MaxSizeSmallerThanInitial",
			config:   &TokenPoolConfig{InitialSize: 20, MaxSize: 10},
			expected: &TokenPoolConfig{InitialSize: 20, MaxSize: 20},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewTokenPool(tt.config)
			
			if pool.config.InitialSize != tt.expected.InitialSize {
				t.Errorf("Expected InitialSize %d, got %d", tt.expected.InitialSize, pool.config.InitialSize)
			}
			
			if pool.config.MaxSize != tt.expected.MaxSize {
				t.Errorf("Expected MaxSize %d, got %d", tt.expected.MaxSize, pool.config.MaxSize)
			}
			
			if pool.config.EnableMetrics != tt.expected.EnableMetrics {
				t.Errorf("Expected EnableMetrics %v, got %v", tt.expected.EnableMetrics, pool.config.EnableMetrics)
			}
			
			if pool.pool == nil {
				t.Error("Expected pool channel to be initialized")
			}
			
			// Check initial population
			if cap(pool.pool) != tt.expected.MaxSize {
				t.Errorf("Expected pool capacity %d, got %d", tt.expected.MaxSize, cap(pool.pool))
			}
		})
	}
}

func TestTokenPoolGetPut(t *testing.T) {
	pool := NewTokenPool(&TokenPoolConfig{InitialSize: 2, MaxSize: 5, EnableMetrics: true})
	
	// Get should return a FileSet
	fset1 := pool.Get()
	if fset1 == nil {
		t.Error("Expected Get to return a FileSet")
	}
	
	// Get another one
	fset2 := pool.Get()
	if fset2 == nil {
		t.Error("Expected Get to return a second FileSet")
	}
	
	// Put back
	pool.Put(fset1)
	pool.Put(fset2)
	
	// Get again - should reuse from pool
	fset3 := pool.Get()
	if fset3 == nil {
		t.Error("Expected Get to return a reused FileSet")
	}
	
	// Check statistics
	stats := pool.GetStatistics()
	if stats["reused"].(int64) == 0 {
		t.Error("Expected some reuse to occur")
	}
}

func TestTokenPoolOverflow(t *testing.T) {
	pool := NewTokenPool(&TokenPoolConfig{InitialSize: 1, MaxSize: 2, EnableMetrics: true})
	
	// Fill the pool beyond capacity
	fset1 := token.NewFileSet()
	fset2 := token.NewFileSet()
	fset3 := token.NewFileSet()
	
	pool.Put(fset1)
	pool.Put(fset2)
	pool.Put(fset3) // This should be discarded
	
	stats := pool.GetStatistics()
	if stats["discarded"].(int64) == 0 {
		t.Error("Expected some FileSets to be discarded")
	}
}

func TestTokenPoolConcurrency(t *testing.T) {
	pool := NewTokenPool(&TokenPoolConfig{InitialSize: 5, MaxSize: 20, EnableMetrics: true})
	
	var wg sync.WaitGroup
	numGoroutines := 10
	operationsPerGoroutine := 100
	
	// Concurrent get/put operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for j := 0; j < operationsPerGoroutine; j++ {
				fset := pool.Get()
				if fset == nil {
					t.Error("Get should never return nil")
					return
				}
				
				// Simulate some work
				time.Sleep(time.Microsecond)
				
				pool.Put(fset)
			}
		}()
	}
	
	wg.Wait()
	
	// Check that pool is in consistent state
	stats := pool.GetStatistics()
	if stats["total_requests"].(int64) <= 0 {
		t.Error("Expected some requests to be processed")
	}
}

func TestTokenPoolStatistics(t *testing.T) {
	pool := NewTokenPool(&TokenPoolConfig{InitialSize: 2, MaxSize: 5, EnableMetrics: true})
	
	// Initial stats
	stats := pool.GetStatistics()
	if stats["pool_size"].(int) != 2 {
		t.Errorf("Expected initial pool size 2, got %v", stats["pool_size"])
	}
	
	if stats["max_size"].(int) != 5 {
		t.Errorf("Expected max size 5, got %v", stats["max_size"])
	}
	
	// Get some FileSets
	fset1 := pool.Get()
	fset2 := pool.Get()
	fset3 := pool.Get() // This should create new one
	
	// Check utilization
	stats = pool.GetStatistics()
	utilization := stats["utilization"].(float64)
	if utilization <= 0 {
		t.Error("Expected positive utilization")
	}
	
	// Put back
	pool.Put(fset1)
	pool.Put(fset2)
	pool.Put(fset3)
	
	// Check reuse rate
	stats = pool.GetStatistics()
	reuseRate := stats["reuse_rate"].(float64)
	if reuseRate <= 0 {
		t.Error("Expected positive reuse rate")
	}
}

func TestTokenPoolDrain(t *testing.T) {
	pool := NewTokenPool(&TokenPoolConfig{InitialSize: 3, MaxSize: 10})
	
	// Drain the pool
	drained := pool.Drain()
	if drained != 3 {
		t.Errorf("Expected to drain 3 FileSets, got %d", drained)
	}
	
	// Pool should be empty
	stats := pool.GetStatistics()
	if stats["pool_size"].(int) != 0 {
		t.Errorf("Expected pool to be empty after drain, got %v", stats["pool_size"])
	}
	
	// Get should still work (creates new)
	fset := pool.Get()
	if fset == nil {
		t.Error("Expected Get to work even after drain")
	}
}

func TestTokenPoolResize(t *testing.T) {
	pool := NewTokenPool(&TokenPoolConfig{InitialSize: 3, MaxSize: 5})
	
	// Fill pool
	pool.Put(token.NewFileSet())
	pool.Put(token.NewFileSet())
	
	initialStats := pool.GetStatistics()
	initialSize := initialStats["pool_size"].(int)
	
	// Resize to larger
	pool.Resize(10)
	
	stats := pool.GetStatistics()
	if stats["max_size"].(int) != 10 {
		t.Errorf("Expected max size 10 after resize, got %v", stats["max_size"])
	}
	
	// Pool should retain existing entries
	if stats["pool_size"].(int) != initialSize {
		t.Errorf("Expected pool size to be preserved during resize")
	}
	
	// Resize to smaller
	pool.Resize(2)
	
	stats = pool.GetStatistics()
	if stats["max_size"].(int) != 2 {
		t.Errorf("Expected max size 2 after resize, got %v", stats["max_size"])
	}
}

func TestTokenPoolWarmUp(t *testing.T) {
	pool := NewTokenPool(&TokenPoolConfig{InitialSize: 1, MaxSize: 10, EnableMetrics: true})
	
	// Warm up to 5
	pool.WarmUp(5)
	
	stats := pool.GetStatistics()
	poolSize := stats["pool_size"].(int)
	if poolSize < 5 {
		t.Errorf("Expected pool size >= 5 after warmup, got %d", poolSize)
	}
	
	// Warm up beyond max size (should cap at max)
	pool.WarmUp(15)
	
	stats = pool.GetStatistics()
	poolSize = stats["pool_size"].(int)
	if poolSize > 10 {
		t.Errorf("Expected pool size <= 10 after warmup, got %d", poolSize)
	}
}

func TestTokenPoolPutNil(t *testing.T) {
	pool := NewTokenPool(DefaultTokenPoolConfig())
	
	initialStats := pool.GetStatistics()
	initialSize := initialStats["pool_size"].(int)
	
	// Put nil should not affect pool
	pool.Put(nil)
	
	stats := pool.GetStatistics()
	if stats["pool_size"].(int) != initialSize {
		t.Error("Putting nil should not change pool size")
	}
}

func TestTokenPoolMetricsDisabled(t *testing.T) {
	pool := NewTokenPool(&TokenPoolConfig{
		InitialSize:   2,
		MaxSize:       5,
		EnableMetrics: false,
	})
	
	// Perform operations
	fset := pool.Get()
	pool.Put(fset)
	
	// Metrics should remain zero
	stats := pool.GetStatistics()
	if stats["created"].(int64) != 0 || stats["reused"].(int64) != 0 {
		t.Error("Expected metrics to be disabled")
	}
}

func TestTokenPoolEdgeCases(t *testing.T) {
	t.Run("InvalidResizeSize", func(t *testing.T) {
		pool := NewTokenPool(DefaultTokenPoolConfig())
		initialMaxSize := pool.config.MaxSize
		
		pool.Resize(0)
		pool.Resize(-5)
		
		if pool.config.MaxSize != initialMaxSize {
			t.Error("Pool max size should not change for invalid resize values")
		}
	})
	
	t.Run("WarmUpInvalidSize", func(t *testing.T) {
		pool := NewTokenPool(&TokenPoolConfig{InitialSize: 2, MaxSize: 5})
		initialSize := len(pool.pool)
		
		pool.WarmUp(0)
		pool.WarmUp(-5)
		
		// Pool size should not change for invalid warmup sizes
		if len(pool.pool) != initialSize {
			t.Error("Pool size should not change for invalid warmup sizes")
		}
	})
	
	t.Run("EmptyPoolGet", func(t *testing.T) {
		pool := NewTokenPool(&TokenPoolConfig{InitialSize: 0, MaxSize: 5, EnableMetrics: true})
		
		// Pool starts empty
		fset := pool.Get()
		if fset == nil {
			t.Error("Get should create new FileSet when pool is empty")
		}
		
		stats := pool.GetStatistics()
		if stats["created"].(int64) == 0 {
			t.Error("Expected created count to increment")
		}
	})
}

func TestTokenPoolPerformance(t *testing.T) {
	pool := NewTokenPool(&TokenPoolConfig{InitialSize: 50, MaxSize: 100, EnableMetrics: true})
	
	iterations := 1000
	start := time.Now()
	
	// Benchmark get/put operations
	for i := 0; i < iterations; i++ {
		fset := pool.Get()
		pool.Put(fset)
	}
	
	duration := time.Since(start)
	opsPerSec := float64(iterations*2) / duration.Seconds() // 2 ops per iteration (get + put)
	
	t.Logf("Token pool performance: %.0f ops/sec", opsPerSec)
	
	// Should be very fast
	if opsPerSec < 100000 {
		t.Errorf("Expected > 100k ops/sec, got %.0f", opsPerSec)
	}
	
	// Check reuse rate
	stats := pool.GetStatistics()
	reuseRate := stats["reuse_rate"].(float64)
	if reuseRate < 80.0 {
		t.Errorf("Expected reuse rate > 80%%, got %.1f%%", reuseRate)
	}
}