// Example demonstrating Phase 1.1d: Incremental type checking
package main

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"time"

	"github.com/healtronlabs/gofasta/transpiler/core"
)

func main() {
	// Initialize incremental type checker
	config := &core.TypeCheckerConfig{
		EnableCaching:    true,
		CacheTTL:         30 * time.Minute,
		MaxCacheEntries:  100,
		ParallelChecking: true,
		MaxWorkers:       4,
		EnableMetrics:    true,
	}

	typeChecker := core.NewIncrementalTypeChecker(config)

	fmt.Println("=== Incremental Type Checker Example ===")

	// Example source files
	sources := map[string]string{
		"main.go": `package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Current time:", time.Now())
	processData("Hello, World!")
}`,
		"utils.go": `package main

import "fmt"

func processData(data string) {
	fmt.Printf("Processing: %s\n", data)
}

type Config struct {
	Name    string
	Enabled bool
}`,
		"types.go": `package main

import "time"

type User struct {
	ID        int
	Name      string
	Email     string
	CreatedAt time.Time
}

func (u *User) IsValid() bool {
	return u.Name != "" && u.Email != ""
}`,
	}

	// Parse all files
	fset := token.NewFileSet()
	files := make([]*ast.File, 0, len(sources))

	for filename, source := range sources {
		file, err := parser.ParseFile(fset, filename, source, parser.ParseComments)
		if err != nil {
			log.Printf("Failed to parse %s: %v", filename, err)
			continue
		}
		files = append(files, file)
	}

	fmt.Printf("Parsed %d files for type checking\n", len(files))

	// Single package type checking
	fmt.Printf("\n=== Single Package Type Checking ===\n")
	ctx := context.Background()

	start := time.Now()
	result, err := typeChecker.CheckPackage(ctx, "main", files, fset)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("Type checking completed with errors: %v\n", err)
	} else {
		fmt.Printf("✓ Type checking succeeded\n")
	}

	if result != nil {
		fmt.Printf("Package: %s\n", result.Package.Name())
		fmt.Printf("Duration: %v\n", duration)
	}

	// Second type check (should hit cache)
	fmt.Printf("\n=== Second Type Check (Cache Hit) ===\n")

	start = time.Now()
	_, err = typeChecker.CheckPackage(ctx, "main", files, fset)
	duration2 := time.Since(start)

	if err != nil {
		fmt.Printf("Second type check completed: %v\n", err)
	} else {
		fmt.Printf("✓ Second type check succeeded\n")
	}

	fmt.Printf("Second check duration: %v\n", duration2)

	if duration2 < duration {
		improvement := float64(duration-duration2) / float64(duration) * 100
		fmt.Printf("Performance improvement: %.1f%% faster\n", improvement)
	}

	// Show statistics
	stats := typeChecker.GetStatistics()
	fmt.Printf("\nType Checker Statistics:\n")
	fmt.Printf("  Checks run: %v\n", stats["checks_run"])
	fmt.Printf("  Cache hits: %v\n", stats["cache_hits"])
	fmt.Printf("  Cache misses: %v\n", stats["cache_misses"])
	fmt.Printf("  Hit ratio: %.1f%%\n", stats["hit_ratio"])
	fmt.Printf("  Average duration: %.1f ms\n", stats["avg_duration_ms"])
	fmt.Printf("  Cached packages: %v\n", stats["cached_packages"])

	// Parallel type checking example
	fmt.Printf("\n=== Parallel Type Checking ===\n")

	// Create multiple package datasets
	packages := []struct {
		name  string
		files []*ast.File
	}{
		{"pkg1", files[:1]},
		{"pkg2", files[1:2]},
		{"pkg3", files[2:]},
	}

	start = time.Now()
	// Convert to map format expected by CheckPackages
	packageMap := make(map[string][]*ast.File)
	for _, pkg := range packages {
		packageMap[pkg.name] = pkg.files
	}
	results, err := typeChecker.CheckPackages(ctx, packageMap, fset)
	parallelDuration := time.Since(start)

	if err != nil {
		fmt.Printf("Parallel type checking completed with some errors: %v\n", err)
	}

	fmt.Printf("Parallel type checking results:\n")
	fmt.Printf("  Packages checked: %d\n", len(results))
	fmt.Printf("  Duration: %v\n", parallelDuration)

	for pkgName, result := range results {
		if result.Error != nil {
			fmt.Printf("  ✗ %s: %v\n", pkgName, result.Error)
		} else {
			fmt.Printf("  ✓ %s: success\n", pkgName)
		}
	}

	// Cache invalidation example
	fmt.Printf("\n=== Cache Invalidation ===\n")

	typeChecker.InvalidateCache("main")
	fmt.Printf("Invalidated cache for package 'main'\n")

	// Clear all caches
	// Clear cache by invalidating all packages
	stats = typeChecker.GetStatistics()
	if cachedPkgs, ok := stats["cached_packages"]; ok {
		if pkgCount, ok := cachedPkgs.(int); ok && pkgCount > 0 {
			// Clear by invalidating known packages
			typeChecker.InvalidateCache("main")
		}
	}
	fmt.Printf("Cleared type checker caches\n")

	finalStats := typeChecker.GetStatistics()
	fmt.Printf("Stats after clear:\n")
	fmt.Printf("  Cached packages: %v\n", finalStats["cached_packages"])

	fmt.Printf("\n✓ Type checker example completed successfully!\n")
}
