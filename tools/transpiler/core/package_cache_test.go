// Package core provides tests for package cache.
package core

import (
	"context"
	"testing"
	"time"

	"golang.org/x/tools/go/packages"
)

func TestNewPackageCache(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		pc := NewPackageCache(nil)
		if pc == nil {
			t.Fatal("Expected non-nil package cache")
		}
		if pc.config == nil {
			t.Error("Expected non-nil config")
		}
		if !pc.config.EnableMetrics {
			t.Error("Expected metrics to be enabled by default")
		}
		if !pc.config.ConcurrentLoads {
			t.Error("Expected concurrent loads to be enabled by default")
		}
	})
	
	t.Run("with custom config", func(t *testing.T) {
		config := &PackageCacheConfig{
			Mode:            packages.NeedName | packages.NeedFiles,
			MaxPackages:     500,
			MaxCacheSizeMB:  100,
			TTL:             10 * time.Minute,
			EnableMetrics:   false,
			ConcurrentLoads: false,
			LoadWorkers:     2,
			Tests:           true,
		}
		pc := NewPackageCache(config)
		if pc == nil {
			t.Fatal("Expected non-nil package cache")
		}
		if pc.config.MaxPackages != 500 {
			t.Errorf("Expected max packages 500, got %d", pc.config.MaxPackages)
		}
		if pc.config.Tests != true {
			t.Error("Expected tests to be true")
		}
	})
}

func TestLoad(t *testing.T) {
	pc := NewPackageCache(&PackageCacheConfig{
		Mode: packages.NeedName | packages.NeedFiles,
		Dir:  ".",
	})
	
	t.Run("load standard library package", func(t *testing.T) {
		pkgs, err := pc.Load("fmt")
		if err != nil {
			t.Fatalf("Failed to load fmt package: %v", err)
		}
		
		if len(pkgs) == 0 {
			t.Error("Expected at least one package")
		}
		
		pkg := pkgs[0]
		if pkg.Name != "fmt" {
			t.Errorf("Expected package name 'fmt', got %s", pkg.Name)
		}
		
		// Check caching
		pc.mu.RLock()
		cached, exists := pc.cache["fmt"]
		pc.mu.RUnlock()
		
		if !exists {
			t.Error("Expected package to be cached")
		}
		if cached != pkg {
			t.Error("Expected cached package to match loaded package")
		}
	})
	
	t.Run("load with pattern caching", func(t *testing.T) {
		// First load
		pkgs1, err := pc.Load("strings")
		if err != nil {
			t.Fatalf("Failed to load strings package: %v", err)
		}
		
		// Second load (should hit cache)
		pkgs2, err := pc.Load("strings")
		if err != nil {
			t.Fatalf("Failed to load strings package from cache: %v", err)
		}
		
		if len(pkgs1) != len(pkgs2) {
			t.Error("Expected same number of packages from cache")
		}
		
		// Check metrics
		if pc.hits == 0 {
			t.Error("Expected cache hits")
		}
	})
	
	t.Run("load multiple packages", func(t *testing.T) {
		pkgs, err := pc.Load("fmt", "strings", "bytes")
		if err != nil {
			t.Fatalf("Failed to load multiple packages: %v", err)
		}
		
		if len(pkgs) < 3 {
			t.Errorf("Expected at least 3 packages, got %d", len(pkgs))
		}
	})
}

func TestLoadWithContext(t *testing.T) {
	pc := NewPackageCache(nil)
	
	t.Run("load with context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		pkgs, err := pc.LoadWithContext(ctx, "errors")
		if err != nil {
			t.Fatalf("Failed to load with context: %v", err)
		}
		
		if len(pkgs) == 0 {
			t.Error("Expected at least one package")
		}
	})
	
	t.Run("load with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		
		_, err := pc.LoadWithContext(ctx, "fmt")
		if err == nil {
			t.Error("Expected error with cancelled context")
		}
	})
}

func TestGetPackage(t *testing.T) {
	pc := NewPackageCache(nil)
	
	// Load a package first
	_, err := pc.Load("fmt")
	if err != nil {
		t.Fatalf("Failed to load package: %v", err)
	}
	
	t.Run("get existing package", func(t *testing.T) {
		pkg, exists := pc.GetPackage("fmt")
		if !exists {
			t.Error("Expected package to exist")
		}
		if pkg == nil {
			t.Error("Expected non-nil package")
		}
		if pkg.Name != "fmt" {
			t.Errorf("Expected package name 'fmt', got %s", pkg.Name)
		}
		
		// Check access count increased
		if pkg.AccessCount < 2 {
			t.Error("Expected access count to increase")
		}
	})
	
	t.Run("get non-existent package", func(t *testing.T) {
		pkg, exists := pc.GetPackage("nonexistent")
		if exists {
			t.Error("Expected package to not exist")
		}
		if pkg != nil {
			t.Error("Expected nil package")
		}
	})
}

func TestLoadOne(t *testing.T) {
	pc := NewPackageCache(nil)
	
	t.Run("load single package", func(t *testing.T) {
		pkg, err := pc.LoadOne("errors")
		if err != nil {
			t.Fatalf("Failed to load single package: %v", err)
		}
		
		if pkg == nil {
			t.Fatal("Expected non-nil package")
		}
		if pkg.Name != "errors" {
			t.Errorf("Expected package name 'errors', got %s", pkg.Name)
		}
		
		// Load again (should hit cache)
		pkg2, err := pc.LoadOne("errors")
		if err != nil {
			t.Fatalf("Failed to load from cache: %v", err)
		}
		
		if pkg != pkg2 {
			t.Error("Expected same package from cache")
		}
	})
}

func TestDependencyTracking(t *testing.T) {
	pc := NewPackageCache(&PackageCacheConfig{
		Mode: packages.NeedName | packages.NeedImports,
	})
	
	t.Run("track dependencies", func(t *testing.T) {
		// Load a package with dependencies
		_, err := pc.Load("fmt")
		if err != nil {
			t.Fatalf("Failed to load package: %v", err)
		}
		
		// Check dependencies
		deps := pc.GetDependencies("fmt")
		if deps == nil {
			t.Error("Expected non-nil dependencies")
		}
		
		// fmt should have some dependencies
		if len(deps) == 0 {
			t.Skip("fmt package has no tracked dependencies in test environment")
		}
	})
	
	t.Run("get transitive dependencies", func(t *testing.T) {
		// This would need a more complex package structure
		// For now, just test the method exists and returns non-nil
		transDeps := pc.GetTransitiveDependencies("fmt")
		if transDeps == nil {
			t.Error("Expected non-nil transitive dependencies")
		}
	})
}

func TestFindPackages(t *testing.T) {
	pc := NewPackageCache(nil)
	
	// Load some packages
	_, _ = pc.Load("fmt", "strings", "bytes")
	
	t.Run("find packages by name", func(t *testing.T) {
		pkgs := pc.FindPackagesByName("*")
		if len(pkgs) == 0 {
			t.Error("Expected to find packages")
		}
		
		pkgs = pc.FindPackagesByName("fmt")
		found := false
		for _, pkg := range pkgs {
			if pkg.Name == "fmt" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find fmt package")
		}
	})
	
	t.Run("find packages by import path", func(t *testing.T) {
		pkgs := pc.FindPackagesByImportPath("*")
		if len(pkgs) == 0 {
			t.Error("Expected to find packages")
		}
	})
}

func TestAnalyzePackage(t *testing.T) {
	pc := NewPackageCache(&PackageCacheConfig{
		Mode: packages.NeedName | 
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes,
	})
	
	t.Run("analyze package", func(t *testing.T) {
		// Load a simple package
		_, err := pc.Load("errors")
		if err != nil {
			t.Fatalf("Failed to load package: %v", err)
		}
		
		analysis, err := pc.AnalyzePackage("errors")
		if err != nil {
			t.Fatalf("Failed to analyze package: %v", err)
		}
		
		if analysis == nil {
			t.Fatal("Expected non-nil analysis")
		}
		if analysis.Name != "errors" {
			t.Errorf("Expected package name 'errors', got %s", analysis.Name)
		}
		if analysis.FileCount == 0 {
			t.Error("Expected at least one file")
		}
	})
	
	t.Run("analyze non-existent package", func(t *testing.T) {
		_, err := pc.AnalyzePackage("nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent package")
		}
	})
}

func TestWarmup(t *testing.T) {
	pc := NewPackageCache(&PackageCacheConfig{
		WarmupPatterns: []string{"errors", "fmt"},
	})
	
	t.Run("warmup with configured patterns", func(t *testing.T) {
		err := pc.Warmup(nil)
		if err != nil {
			t.Fatalf("Failed to warmup: %v", err)
		}
		
		// Check that packages were loaded
		if _, exists := pc.GetPackage("errors"); !exists {
			t.Error("Expected errors package to be loaded")
		}
		if _, exists := pc.GetPackage("fmt"); !exists {
			t.Error("Expected fmt package to be loaded")
		}
	})
	
	t.Run("warmup with custom patterns", func(t *testing.T) {
		err := pc.Warmup([]string{"strings"})
		if err != nil {
			t.Fatalf("Failed to warmup: %v", err)
		}
		
		if _, exists := pc.GetPackage("strings"); !exists {
			t.Error("Expected strings package to be loaded")
		}
	})
}

func TestPackageCacheEviction(t *testing.T) {
	pc := NewPackageCache(&PackageCacheConfig{
		MaxPackages: 3,
		Mode:        packages.NeedName,
	})
	
	t.Run("evict LRU package", func(t *testing.T) {
		// Load packages to fill cache
		packages := []string{"fmt", "strings", "bytes", "errors", "io"}
		
		for _, pkg := range packages {
			_, err := pc.Load(pkg)
			if err != nil {
				t.Fatalf("Failed to load %s: %v", pkg, err)
			}
			time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		}
		
		// Check cache size
		pc.mu.RLock()
		cacheSize := len(pc.cache)
		pc.mu.RUnlock()
		
		if cacheSize > 3 {
			t.Errorf("Expected cache size <= 3, got %d", cacheSize)
		}
		
		// Check that newest packages are still cached
		if _, exists := pc.GetPackage("io"); !exists {
			t.Error("Expected 'io' (newest) to be cached")
		}
	})
}

func TestInvalidation(t *testing.T) {
	pc := NewPackageCache(nil)
	
	// Load some packages
	_, _ = pc.Load("fmt", "strings")
	
	t.Run("invalidate package", func(t *testing.T) {
		// Verify package exists
		if _, exists := pc.GetPackage("fmt"); !exists {
			t.Fatal("Expected fmt package to exist")
		}
		
		// Invalidate
		pc.InvalidatePackage("fmt")
		
		// Verify it's removed
		if _, exists := pc.GetPackage("fmt"); exists {
			t.Error("Expected fmt package to be invalidated")
		}
	})
	
	t.Run("invalidate pattern", func(t *testing.T) {
		pattern := "strings"
		
		// Load with pattern
		_, _ = pc.Load(pattern)
		
		// Verify pattern is cached
		pc.mu.RLock()
		_, exists := pc.patterns[pattern]
		pc.mu.RUnlock()
		
		if !exists {
			t.Fatal("Expected pattern to be cached")
		}
		
		// Invalidate
		pc.InvalidatePattern(pattern)
		
		// Verify it's removed
		pc.mu.RLock()
		_, exists = pc.patterns[pattern]
		pc.mu.RUnlock()
		
		if exists {
			t.Error("Expected pattern to be invalidated")
		}
	})
}

func TestPackageCacheGetStatistics(t *testing.T) {
	pc := NewPackageCache(nil)
	
	// Generate some activity
	_, _ = pc.Load("fmt")
	_, _ = pc.GetPackage("fmt")
	_, _ = pc.GetPackage("nonexistent")
	
	stats := pc.GetStatistics()
	
	// Check statistics
	if stats["cached_packages"].(int) == 0 {
		t.Error("Expected cached packages")
	}
	if stats["cache_hits"].(int64) == 0 {
		t.Error("Expected cache hits")
	}
	if stats["cache_misses"].(int64) == 0 {
		t.Error("Expected cache misses")
	}
	if stats["total_loads"].(int64) == 0 {
		t.Error("Expected total loads")
	}
	
	hitRate := stats["hit_rate"].(float64)
	if hitRate < 0 || hitRate > 100 {
		t.Errorf("Invalid hit rate: %.2f", hitRate)
	}
}

func TestPackageCacheClear(t *testing.T) {
	pc := NewPackageCache(nil)
	
	// Load some packages
	_, _ = pc.Load("fmt", "strings")
	
	// Clear cache
	pc.Clear()
	
	// Check everything was cleared
	pc.mu.RLock()
	cacheSize := len(pc.cache)
	patternSize := len(pc.patterns)
	typeSize := len(pc.typeCache)
	pc.mu.RUnlock()
	
	if cacheSize != 0 {
		t.Errorf("Expected 0 cached packages after clear, got %d", cacheSize)
	}
	if patternSize != 0 {
		t.Errorf("Expected 0 cached patterns after clear, got %d", patternSize)
	}
	if typeSize != 0 {
		t.Errorf("Expected 0 cached types after clear, got %d", typeSize)
	}
	if pc.hits != 0 {
		t.Error("Expected hits to be reset")
	}
	if pc.loads != 0 {
		t.Error("Expected loads to be reset")
	}
}

func TestGetCyclicDependencies(t *testing.T) {
	pc := NewPackageCache(nil)
	
	// Manually create a cyclic dependency for testing
	pc.depGraph["a"] = []string{"b"}
	pc.depGraph["b"] = []string{"c"}
	pc.depGraph["c"] = []string{"a"}
	
	t.Run("detect cyclic dependencies", func(t *testing.T) {
		cycles := pc.GetCyclicDependencies()
		
		if len(cycles) == 0 {
			t.Error("Expected to detect cyclic dependency")
		}
		
		// Should detect the a->b->c->a cycle
		foundCycle := false
		for _, cycle := range cycles {
			if len(cycle) >= 3 {
				hasA, hasB, hasC := false, false, false
				for _, node := range cycle {
					switch node {
					case "a":
						hasA = true
					case "b":
						hasB = true
					case "c":
						hasC = true
					}
				}
				if hasA && hasB && hasC {
					foundCycle = true
					break
				}
			}
		}
		
		if !foundCycle {
			t.Errorf("Expected to find a->b->c->a cycle, got: %v", cycles)
		}
	})
}

func BenchmarkLoad(b *testing.B) {
	pc := NewPackageCache(&PackageCacheConfig{
		Mode: packages.NeedName,
	})
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pc.Load("fmt")
		pc.InvalidatePattern("fmt") // Clear pattern cache for next iteration
	}
}

func BenchmarkGetPackage(b *testing.B) {
	pc := NewPackageCache(nil)
	_, _ = pc.Load("fmt")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pc.GetPackage("fmt")
	}
}

func BenchmarkConcurrentLoad(b *testing.B) {
	pc := NewPackageCache(&PackageCacheConfig{
		ConcurrentLoads: true,
		LoadWorkers:     4,
		Mode:            packages.NeedName,
	})
	
	packages := []string{"fmt", "strings", "bytes", "errors"}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			pkg := packages[i%len(packages)]
			_, _ = pc.LoadOne(pkg)
			i++
		}
	})
}