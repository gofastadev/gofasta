// Example demonstrating Phase 1.1b: AST caching system usage
package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"time"

	"github.com/healtronlabs/gofasta/transpiler/core"
)

func main() {
	// Initialize AST cache with custom configuration
	config := &core.ASTCacheConfig{
		MaxEntries:    100,
		TTL:           30 * time.Minute,
		MaxMemoryMB:   200,
		EnableMetrics: true,
	}

	cache := core.NewASTCache(config)

	fmt.Println("=== AST Cache Example ===")
	fmt.Printf("Configuration:\n")
	fmt.Printf("  Max Entries: %d\n", config.MaxEntries)
	fmt.Printf("  TTL: %v\n", config.TTL)
	fmt.Printf("  Max Memory: %d MB\n", config.MaxMemoryMB)

	// Example source code to parse and cache
	source := `package example

import (
	"fmt"
	"time"
)

// @Controller("/api/example")
// @UseGuards("jwt")
type ExampleController struct{}

// @Get("/")
// @Cache(ttl: "5m")
func (ec *ExampleController) GetExample() (*Response, error) {
	return &Response{
		Message: "Hello from cached AST!",
		Time:    time.Now(),
	}, nil
}

type Response struct {
	Message string    ` + "`json:\"message\"`" + `
	Time    time.Time ` + "`json:\"time\"`" + `
}`

	// Parse the source
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "example.go", source, parser.ParseComments)
	if err != nil {
		log.Fatalf("Failed to parse source: %v", err)
	}

	fmt.Printf("\nCaching AST for file: example.go\n")

	// Cache the parsed AST
	modTime := time.Now()
	size := int64(len(source))
	cache.Put("example.go", file, fset, modTime, size)

	// Retrieve from cache (should be a hit)
	cachedFile, cachedFset, hit := cache.Get("example.go", modTime, size)
	if hit {
		fmt.Printf("✓ Cache hit! Retrieved AST from cache\n")
	} else {
		fmt.Printf("✗ Cache miss\n")
	}

	// Verify cached AST is valid
	if cachedFile != nil && cachedFset != nil {
		fmt.Printf("✓ Cached AST is valid\n")
	} else {
		fmt.Printf("✗ Cached AST is invalid\n")
	}

	// Try with different modification time (should be a miss)
	differentModTime := modTime.Add(-time.Hour)
	_, _, hit2 := cache.Get("example.go", differentModTime, size)
	if !hit2 {
		fmt.Printf("✓ Cache miss for different modification time (expected)\n")
	}

	// Show cache statistics
	stats := cache.GetStatistics()
	fmt.Printf("\nCache Statistics:\n")
	fmt.Printf("  Entries: %v\n", stats["entries"])
	fmt.Printf("  Hits: %v\n", stats["hits"])
	fmt.Printf("  Misses: %v\n", stats["misses"])
	fmt.Printf("  Hit Ratio: %.1f%%\n", stats["hit_ratio"])
	fmt.Printf("  Memory Usage: %v MB\n", stats["memory_mb"])

	// Demonstrate cleanup
	fmt.Printf("\nCleaning up expired entries...\n")
	removed := cache.Cleanup()
	fmt.Printf("Removed %d expired entries\n", removed)

	// Performance test
	fmt.Printf("\nPerformance Test:\n")
	numLookups := 1000
	start := time.Now()

	for i := 0; i < numLookups; i++ {
		cache.Get("example.go", modTime, size)
	}

	duration := time.Since(start)
	fmt.Printf("Performed %d cache lookups in %v\n", numLookups, duration)
	fmt.Printf("Average lookup time: %v\n", duration/time.Duration(numLookups))

	finalStats := cache.GetStatistics()
	fmt.Printf("Final hit ratio: %.1f%%\n", finalStats["hit_ratio"])

	fmt.Printf("\n✓ AST cache example completed successfully!\n")
}
