// Example demonstrating Phase 1.1f: Import caching with fallback support
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/healtronlabs/gofasta/tools/transpiler/core"
)

func main() {
	// Initialize cached importer with custom configuration
	config := &core.ImportCacheConfig{
		MaxEntries:    50,
		TTL:           time.Hour,
		EnableMetrics: true,
		MaxMemoryMB:   100,
	}
	
	importer := core.NewCachedImporter(config)
	
	fmt.Println("=== Cached Importer Example ===")
	fmt.Printf("Configuration:\n")
	fmt.Printf("  Max Entries: %d\n", config.MaxEntries)
	fmt.Printf("  TTL: %v\n", config.TTL)
	fmt.Printf("  Max Memory: %d MB\n", config.MaxMemoryMB)
	
	// Standard packages to import
	packages := []string{
		"fmt",
		"os",
		"io",
		"time",
		"context",
		"sync",
		"net/http",
		"encoding/json",
		"strings",
		"strconv",
	}
	
	fmt.Printf("\n=== First Import Round (Cache Misses) ===\n")
	start := time.Now()
	
	for _, pkgPath := range packages {
		pkg, err := importer.Import(pkgPath)
		if err != nil {
			fmt.Printf("✗ Failed to import %s: %v\n", pkgPath, err)
		} else {
			fmt.Printf("✓ Imported %s (name: %s)\n", pkgPath, pkg.Name())
		}
	}
	
	firstRoundDuration := time.Since(start)
	
	// Show statistics after first round
	stats := importer.GetStatistics()
	fmt.Printf("\nFirst Round Statistics:\n")
	fmt.Printf("  Duration: %v\n", firstRoundDuration)
	fmt.Printf("  Cached entries: %v\n", stats["entries"])
	fmt.Printf("  Cache misses: %v\n", stats["misses"])
	fmt.Printf("  Cache hits: %v\n", stats["hits"])
	fmt.Printf("  Memory usage: %v MB\n", stats["memory_mb"])
	
	fmt.Printf("\n=== Second Import Round (Cache Hits) ===\n")
	start = time.Now()
	
	for _, pkgPath := range packages {
		pkg, err := importer.Import(pkgPath)
		if err != nil {
			fmt.Printf("✗ Failed to import %s: %v\n", pkgPath, err)
		} else {
			fmt.Printf("✓ Imported %s (cached)\n", pkgPath)
		}
	}
	
	secondRoundDuration := time.Since(start)
	
	// Show statistics after second round
	stats = importer.GetStatistics()
	fmt.Printf("\nSecond Round Statistics:\n")
	fmt.Printf("  Duration: %v\n", secondRoundDuration)
	fmt.Printf("  Cache hits: %v\n", stats["hits"])
	fmt.Printf("  Hit ratio: %.1f%%\n", stats["hit_ratio"])
	
	// Show performance improvement
	if firstRoundDuration > secondRoundDuration {
		improvement := float64(firstRoundDuration-secondRoundDuration) / float64(firstRoundDuration) * 100
		fmt.Printf("  Performance improvement: %.1f%% faster\n", improvement)
	}
	
	// Demonstrate preloading
	fmt.Printf("\n=== Preloading Example ===\n")
	
	newImporter := core.NewCachedImporter(core.DefaultImportCacheConfig())
	
	preloadPackages := []string{"fmt", "os", "io", "time", "context"}
	fmt.Printf("Preloading packages: %v\n", preloadPackages)
	
	start = time.Now()
	errors := newImporter.Preload(preloadPackages)
	preloadDuration := time.Since(start)
	
	if len(errors) > 0 {
		fmt.Printf("Preload errors: %v\n", errors)
	} else {
		fmt.Printf("✓ All packages preloaded successfully in %v\n", preloadDuration)
	}
	
	preloadStats := newImporter.GetStatistics()
	fmt.Printf("Preload statistics:\n")
	fmt.Printf("  Entries: %v\n", preloadStats["entries"])
	fmt.Printf("  Memory usage: %v MB\n", preloadStats["memory_mb"])
	
	// Test imports after preload (should be fast)
	start = time.Now()
	for _, pkg := range preloadPackages {
		newImporter.Import(pkg)
	}
	preloadTestDuration := time.Since(start)
	
	preloadTestStats := newImporter.GetStatistics()
	fmt.Printf("Post-preload import test:\n")
	fmt.Printf("  Duration: %v\n", preloadTestDuration)
	fmt.Printf("  Hit ratio: %.1f%%\n", preloadTestStats["hit_ratio"])
	
	// Demonstrate warm-up
	fmt.Printf("\n=== Warm-Up Example ===\n")
	
	warmImporter := core.NewCachedImporter(core.DefaultImportCacheConfig())
	
	fmt.Printf("Warming up importer...\n")
	start = time.Now()
	err := warmImporter.WarmUp()
	warmUpDuration := time.Since(start)
	
	if err != nil {
		fmt.Printf("Warm-up failed: %v\n", err)
	} else {
		fmt.Printf("✓ Warm-up completed in %v\n", warmUpDuration)
	}
	
	warmStats := warmImporter.GetStatistics()
	fmt.Printf("Warm-up statistics:\n")
	fmt.Printf("  Pre-cached entries: %v\n", warmStats["entries"])
	
	// Test standard package import after warm-up
	pkg, err := warmImporter.Import("fmt")
	if err != nil {
		fmt.Printf("✗ Failed to import fmt after warm-up: %v\n", err)
	} else {
		fmt.Printf("✓ Successfully imported fmt: %s\n", pkg.Name())
	}
	
	postWarmStats := warmImporter.GetStatistics()
	fmt.Printf("  Hit ratio after test: %.1f%%\n", postWarmStats["hit_ratio"])
	
	// Demonstrate cache management
	fmt.Printf("\n=== Cache Management Example ===\n")
	
	// Show cached packages
	cachedPackages := importer.GetCachedPackages()
	fmt.Printf("Currently cached packages (%d):\n", len(cachedPackages))
	for i, pkg := range cachedPackages {
		fmt.Printf("  %d. %s\n", i+1, pkg)
	}
	
	// Cleanup expired entries
	fmt.Printf("\nCleaning up expired entries...\n")
	removed := importer.Cleanup()
	fmt.Printf("Removed %d expired entries\n", removed)
	
	// Clear entire cache
	fmt.Printf("\nClearing entire cache...\n")
	importer.Clear()
	
	clearStats := importer.GetStatistics()
	fmt.Printf("Cache after clear:\n")
	fmt.Printf("  Entries: %v\n", clearStats["entries"])
	
	// Demonstrate fallback mechanism
	fmt.Printf("\n=== Fallback Mechanism Example ===\n")
	
	primaryImporter := core.NewCachedImporter(&core.ImportCacheConfig{
		MaxEntries:  10,
		TTL:         time.Minute,
		MaxMemoryMB: 10,
	})
	
	fallbackImporter := core.NewCachedImporter(core.DefaultImportCacheConfig())
	
	// Try to import with fallback
	pkg, err = primaryImporter.ImportWithFallback("fmt", fallbackImporter)
	if err != nil {
		fmt.Printf("✗ Failed to import with fallback: %v\n", err)
	} else {
		fmt.Printf("✓ Successfully imported with fallback: %s\n", pkg.Name())
	}
	
	// Get underlying importer
	underlying := importer.GetLoadedImporter()
	if underlying != nil {
		fmt.Printf("✓ Retrieved underlying importer\n")
		
		// Use underlying importer directly
		directPkg, err := underlying.Import("os")
		if err != nil {
			fmt.Printf("✗ Direct import failed: %v\n", err)
		} else {
			fmt.Printf("✓ Direct import succeeded: %s\n", directPkg.Name())
		}
	}
	
	fmt.Printf("\n✓ Import cache example completed successfully!\n")
}