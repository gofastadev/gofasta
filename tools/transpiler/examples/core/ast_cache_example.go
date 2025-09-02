// Example demonstrating Phase 1.1b: AST caching system usage
package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"time"

	"github.com/healtronlabs/gofasta/tools/transpiler/core"
)

func main() {
	// Initialize AST cache with custom configuration
	config := &core.ASTCacheConfig{
		MaxEntries:    100,
		TTL:           30 * time.Minute,
		MaxMemoryMB:   50,
		EnableMetrics: true,
	}
	
	astCache := core.NewASTCache(config)
	
	// Example Go source code to parse and cache
	source := `package example

import "fmt"

func Hello(name string) {
	fmt.Printf("Hello, %s!\n", name)
}

func main() {
	Hello("World")
}`
	
	// Parse the source code
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "example.go", source, parser.ParseComments)
	if err != nil {
		log.Fatalf("Failed to parse source: %v", err)
	}
	
	// Cache the AST
	modTime := time.Now()
	size := int64(len(source))
	
	fmt.Println("=== AST Cache Example ===")
	fmt.Printf("Caching AST for file: example.go\n")
	
	astCache.Put("example.go", file, fset, modTime, size)
	
	// Retrieve from cache (should hit)
	cachedFile, cachedFset, hit := astCache.Get("example.go", modTime, size)
	if hit {
		fmt.Printf("✓ Cache hit! Retrieved AST from cache\n")
		if cachedFile != nil && cachedFset != nil {
			fmt.Printf("✓ Cached AST is valid\n")
		}
	} else {
		fmt.Printf("✗ Cache miss\n")
	}
	
	// Try with different modification time (should miss)
	newModTime := modTime.Add(time.Second)
	_, _, hit = astCache.Get("example.go", newModTime, size)
	if !hit {
		fmt.Printf("✓ Cache miss for different modification time (expected)\n")
	}
	
	// Show cache statistics
	stats := astCache.GetStatistics()
	fmt.Printf("\nCache Statistics:\n")
	fmt.Printf("  Entries: %v\n", stats["entries"])
	fmt.Printf("  Hits: %v\n", stats["hits"])
	fmt.Printf("  Misses: %v\n", stats["misses"])
	fmt.Printf("  Hit Ratio: %.1f%%\n", stats["hit_ratio"])
	fmt.Printf("  Memory Usage: %v MB\n", stats["memory_mb"])
	
	// Demonstrate cache cleanup
	fmt.Printf("\nCleaning up expired entries...\n")
	removed := astCache.Cleanup()
	fmt.Printf("Removed %d expired entries\n", removed)
	
	// Performance demonstration
	fmt.Printf("\nPerformance Test:\n")
	start := time.Now()
	iterations := 1000
	
	for i := 0; i < iterations; i++ {
		astCache.Get("example.go", modTime, size)
	}
	
	duration := time.Since(start)
	fmt.Printf("Performed %d cache lookups in %v\n", iterations, duration)
	fmt.Printf("Average lookup time: %v\n", duration/time.Duration(iterations))
	
	finalStats := astCache.GetStatistics()
	fmt.Printf("Final hit ratio: %.1f%%\n", finalStats["hit_ratio"])
}