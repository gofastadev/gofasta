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

	"github.com/healtronlabs/gofasta/tools/transpiler/core"
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

func validateInput(input string) bool {
	return len(input) > 0
}`,
	}
	
	// Parse all source files
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
	
	ctx := context.Background()
	
	// First type check - will populate cache
	fmt.Printf("\nFirst type check (cache miss expected):\n")
	start := time.Now()
	
	result1, err := typeChecker.CheckPackage(ctx, "main", files, fset)
	duration1 := time.Since(start)
	
	if err != nil {
		fmt.Printf("Type checking completed with errors: %v\n", err)
	} else {
		fmt.Printf("✓ Type checking succeeded\n")
	}
	
	if result1 != nil {
		fmt.Printf("✓ Package: %s\n", result1.Package.Name())
		fmt.Printf("  Duration: %v\n", duration1)
	}
	
	// Second type check - should hit cache
	fmt.Printf("\nSecond type check (cache hit expected):\n")
	start = time.Now()
	
	result2, err := typeChecker.CheckPackage(ctx, "main", files, fset)
	duration2 := time.Since(start)
	
	if err != nil {
		fmt.Printf("Type checking completed with errors: %v\n", err)
	} else {
		fmt.Printf("✓ Type checking succeeded\n")
	}
	
	if result2 != nil {
		fmt.Printf("✓ Package: %s\n", result2.Package.Name())
		fmt.Printf("  Duration: %v\n", duration2)
	}
	
	// Show performance improvement
	if duration1 > duration2 {
		improvement := float64(duration1-duration2) / float64(duration1) * 100
		fmt.Printf("✓ Cache improved performance by %.1f%%\n", improvement)
	}
	
	// Show cache statistics
	stats := typeChecker.GetStatistics()
	fmt.Printf("\nType Checker Statistics:\n")
	fmt.Printf("  Cached packages: %v\n", stats["cached_packages"])
	fmt.Printf("  Cache hits: %v\n", stats["cache_hits"])
	fmt.Printf("  Cache misses: %v\n", stats["cache_misses"])
	fmt.Printf("  Hit ratio: %.1f%%\n", stats["hit_ratio"])
	fmt.Printf("  Checks run: %v\n", stats["checks_run"])
	fmt.Printf("  Average duration: %.1f ms\n", stats["avg_duration_ms"])
	
	// Demonstrate parallel type checking
	fmt.Printf("\n=== Parallel Type Checking Example ===\n")
	
	// Create multiple packages
	packages := make(map[string][]*ast.File)
	packageSources := map[string]string{
		"pkg1": `package pkg1
func Function1() string { return "pkg1" }`,
		"pkg2": `package pkg2  
func Function2() int { return 42 }`,
		"pkg3": `package pkg3
import "time"
func Function3() time.Time { return time.Now() }`,
		"pkg4": `package pkg4
import "fmt"
func Function4(msg string) { fmt.Println(msg) }`,
	}
	
	for pkgName, source := range packageSources {
		file, err := parser.ParseFile(fset, pkgName+".go", source, parser.ParseComments)
		if err != nil {
			log.Printf("Failed to parse %s: %v", pkgName, err)
			continue
		}
		packages[pkgName] = []*ast.File{file}
	}
	
	// Parallel type checking
	fmt.Printf("Type checking %d packages in parallel...\n", len(packages))
	start = time.Now()
	
	results, err := typeChecker.CheckPackages(ctx, packages, fset)
	parallelDuration := time.Since(start)
	
	if err != nil {
		fmt.Printf("Parallel type checking completed with some errors: %v\n", err)
	}
	
	fmt.Printf("✓ Parallel type checking completed in %v\n", parallelDuration)
	fmt.Printf("✓ Results for %d packages\n", len(results))
	
	for pkgName, result := range results {
		if result.Error == nil {
			fmt.Printf("  ✓ %s: %v\n", pkgName, result.Duration)
		} else {
			fmt.Printf("  ✗ %s: %v\n", pkgName, result.Error)
		}
	}
	
	// Final statistics
	finalStats := typeChecker.GetStatistics()
	fmt.Printf("\nFinal Statistics:\n")
	fmt.Printf("  Total checks run: %v\n", finalStats["checks_run"])
	fmt.Printf("  Total cache hits: %v\n", finalStats["cache_hits"])
	fmt.Printf("  Overall hit ratio: %.1f%%\n", finalStats["hit_ratio"])
	
	// Demonstrate cache invalidation
	fmt.Printf("\n=== Cache Invalidation Example ===\n")
	fmt.Printf("Invalidating cache for pkg1...\n")
	typeChecker.InvalidateCache("pkg1")
	
	invalidationStats := typeChecker.GetStatistics()
	fmt.Printf("Cached packages after invalidation: %v\n", invalidationStats["cached_packages"])
	
	fmt.Printf("\n✓ Type checker example completed successfully!\n")
}

// demonstrateTypeCheckerFeatures shows advanced features
func demonstrateTypeCheckerFeatures() {
	fmt.Printf("\n=== Advanced Type Checker Features ===\n")
	
	typeChecker := core.NewIncrementalTypeChecker(core.DefaultTypeCheckerConfig())
	
	// Example with type information
	source := `package example

import "fmt"

type User struct {
	ID   int
	Name string
}

func (u *User) String() string {
	return fmt.Sprintf("User{ID: %d, Name: %s}", u.ID, u.Name)
}

func NewUser(id int, name string) *User {
	return &User{ID: id, Name: name}
}

func main() {
	user := NewUser(1, "Alice")
	fmt.Println(user)
}`
	
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "example.go", source, parser.ParseComments)
	if err != nil {
		log.Printf("Failed to parse example: %v", err)
		return
	}
	
	ctx := context.Background()
	result, err := typeChecker.CheckPackage(ctx, "example", []*ast.File{file}, fset)
	
	if err != nil {
		fmt.Printf("Type checking result: %v\n", err)
	}
	
	if result != nil && result.Package != nil {
		fmt.Printf("✓ Successfully type checked package: %s\n", result.Package.Name())
		fmt.Printf("  Package path: %s\n", result.Package.Path())
		fmt.Printf("  Scope entries: %d\n", result.Package.Scope().Len())
		
		// Show type information if available
		if result.Info != nil {
			fmt.Printf("  Type information collected:\n")
			fmt.Printf("    Definitions: %d\n", len(result.Info.Defs))
			fmt.Printf("    Uses: %d\n", len(result.Info.Uses))
			fmt.Printf("    Types: %d\n", len(result.Info.Types))
		}
	}
}