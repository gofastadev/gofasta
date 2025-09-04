// Cache efficiency integration tests for Phase 1.1 caching components
package core

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
	"time"

	"github.com/healtronlabs/gofasta/tools/transpiler/core"
)

// TestCacheEfficiencyIntegration tests caching efficiency across components
func TestCacheEfficiencyIntegration(t *testing.T) {
	// Initialize all caching components
	astCache := core.NewASTCache(&core.ASTCacheConfig{
		MaxEntries:    50,
		TTL:           time.Hour,
		MaxMemoryMB:   100,
		EnableMetrics: true,
	})
	
	tokenPool := core.NewTokenPool(&core.TokenPoolConfig{
		InitialSize:   5,
		MaxSize:       10,
		EnableMetrics: true,
	})
	
	typeChecker := core.NewIncrementalTypeChecker(&core.TypeCheckerConfig{
		EnableCaching:    true,
		CacheTTL:         30 * time.Minute,
		MaxCacheEntries:  20,
		ParallelChecking: true,
		MaxWorkers:       2,
		EnableMetrics:    true,
	})
	
	importer := core.NewCachedImporter(&core.ImportCacheConfig{
		MaxEntries:    30,
		TTL:           time.Hour,
		EnableMetrics: true,
		MaxMemoryMB:   50,
	})
	
	// Create test data
	fset := tokenPool.Get()
	defer tokenPool.Put(fset)
	
	file, err := parser.ParseFile(fset, "cache_test.go", `
package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println(time.Now())
}`, parser.ParseComments)
	
	if err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}
	
	modTime := time.Now()
	size := int64(200)
	
	// First iteration - populate caches
	astCache.Put("cache_test.go", file, fset, modTime, size)
	
	ctx := context.Background()
	typeResult1, _ := typeChecker.CheckPackage(ctx, "main", []*ast.File{file}, fset)
	
	// Import standard packages (first time)
	for _, pkg := range []string{"fmt", "time"} {
		importer.Import(pkg)
	}
	
	// Import same packages again to generate cache hits
	for _, pkg := range []string{"fmt", "time"} {
		importer.Import(pkg)
	}
	
	// Second iteration - should hit all caches
	cachedFile, cachedFset, astHit := astCache.Get("cache_test.go", modTime, size)
	if !astHit {
		t.Error("Expected AST cache hit on second access")
	}
	
	typeResult2, _ := typeChecker.CheckPackage(ctx, "main", []*ast.File{cachedFile}, cachedFset)
	
	// Verify cache efficiency
	astStats := astCache.GetStatistics()
	typeStats := typeChecker.GetStatistics()
	importStats := importer.GetStatistics()
	poolStats := tokenPool.GetStatistics()
	
	// Expect cache hits
	if astStats["hits"].(int64) == 0 {
		t.Error("Expected AST cache hits")
	}
	
	if typeStats["cache_hits"].(int64) == 0 {
		t.Error("Expected type checker cache hits")
	}
	
	if importStats["hits"].(int64) == 0 {
		t.Error("Expected import cache hits")
	}
	
	if poolStats["reused"].(int64) == 0 {
		t.Error("Expected token pool reuse")
	}
	
	// Verify results consistency
	if typeResult1 != nil && typeResult2 != nil {
		if typeResult1.Package != nil && typeResult2.Package != nil {
			if typeResult1.Package.Name() != typeResult2.Package.Name() {
				t.Error("Expected consistent type checking results")
			}
		}
	}
	
	t.Logf("Cache Efficiency Results:")
	t.Logf("  AST Cache hit ratio: %.1f%%", astStats["hit_ratio"])
	t.Logf("  Type Checker hit ratio: %.1f%%", typeStats["hit_ratio"])
	t.Logf("  Import Cache hit ratio: %.1f%%", importStats["hit_ratio"])
	t.Logf("  Token Pool reused: %v", poolStats["reused"])
}

// TestComponentInteroperability tests how components work together
func TestComponentInteroperability(t *testing.T) {
	// Create shared components
	astCache := core.NewASTCache(core.DefaultASTCacheConfig())
	tokenPool := core.NewTokenPool(core.DefaultTokenPoolConfig())
	typeChecker := core.NewIncrementalTypeChecker(core.DefaultTypeCheckerConfig())
	formatter := core.NewBatchFormatter(core.DefaultBatchFormatterConfig())
	importer := core.NewCachedImporter(core.DefaultImportCacheConfig())
	
	// Warm up components
	tokenPool.WarmUp(3)
	importer.WarmUp()
	
	// Create test data
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", `
package test

import "fmt"

func Hello() {
	fmt.Println("Hello from integration test")
}
`, parser.ParseComments)
	
	if err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}
	
	// Test workflow: Cache -> Type Check -> Format
	modTime := time.Now()
	size := int64(100)
	
	// Step 1: Cache the AST
	astCache.Put("test.go", file, fset, modTime, size)
	
	// Step 2: Type check with caching
	ctx := context.Background()
	typeResult, typeErr := typeChecker.CheckPackage(ctx, "test", []*ast.File{file}, fset)
	if typeErr != nil {
		t.Logf("Type checking completed: %v", typeErr)
	}
	
	if typeResult == nil {
		t.Error("Expected type check result")
	}
	
	// Step 3: Format the file
	formatResult := formatter.FormatFile("test.go", file, fset)
	if formatResult.Error != nil {
		t.Errorf("Formatting failed: %v", formatResult.Error)
	}
	
	// Step 4: Verify cache hits on second iteration
	// Retrieve from AST cache
	cachedFile, cachedFset, hit := astCache.Get("test.go", modTime, size)
	if !hit {
		t.Error("Expected AST cache hit")
	}
	
	if cachedFile == nil || cachedFset == nil {
		t.Error("Expected cached AST and fileset")
	}
	
	// Type check again (should hit cache)
	typeResult2, typeErr2 := typeChecker.CheckPackage(ctx, "test", []*ast.File{file}, fset)
	if typeErr2 != nil {
		t.Logf("Second type check completed: %v", typeErr2)
	}
	
	if typeResult2 == nil {
		t.Error("Expected second type check result")
	}
	
	// Verify performance improvements from caching
	astStats := astCache.GetStatistics()
	typeStats := typeChecker.GetStatistics()
	formatStats := formatter.GetStatistics()
	importStats := importer.GetStatistics()
	
	if astStats["hits"].(int64) == 0 {
		t.Error("Expected AST cache hits")
	}
	
	if typeStats["cache_hits"].(int64) == 0 {
		t.Error("Expected type checker cache hits")
	}
	
	t.Logf("Component interoperability metrics:")
	t.Logf("  AST Cache hits: %v", astStats["hits"])
	t.Logf("  Type Checker cache hits: %v", typeStats["cache_hits"])
	t.Logf("  Files formatted: %v", formatStats["files_formatted"])
	t.Logf("  Import cache entries: %v", importStats["entries"])
}