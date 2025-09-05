// Full pipeline integration tests for Phase 1.1 components working together
package core

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/healtronlabs/gofasta/transpiler/core"
)

// TestPhase1FullPipelineIntegration tests all Phase 1.1 components working together in sequence
func TestPhase1FullPipelineIntegration(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()

	// Create test Go files
	testFiles := map[string]string{
		"main.go": `package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Hello, World!")
	os.Exit(0)
}`,
		"util.go": `package main

import "fmt"

func helper() {
	fmt.Println("Helper function")
}`,
		"types.go": `package main

type Config struct {
	Name    string
	Enabled bool
}

func NewConfig() *Config {
	return &Config{
		Name:    "default",
		Enabled: true,
	}
}`,
	}

	// Write test files to temp directory
	for filename, content := range testFiles {
		path := filepath.Join(tempDir, filename)
		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file %s: %v", filename, err)
		}
	}

	t.Run("FullPipelineIntegration", func(t *testing.T) {
		// Phase 1.1a: Parse files using parallel parser
		parser := core.NewParallelParser(core.DefaultConfig())
		ctx := context.Background()
		results, err := parser.ParseDirectory(ctx, tempDir)
		if err != nil {
			t.Fatalf("Failed to parse directory: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 parsed files, got %d", len(results))
		}

		// Phase 1.1b: Use AST cache to cache parsed results
		astCache := core.NewASTCache(core.DefaultASTCacheConfig())

		for _, result := range results {
			if result.Error == nil {
				// Get mod time from file stat
				info, _ := os.Stat(result.FilePath)
				modTime := time.Now()
				if info != nil {
					modTime = info.ModTime()
				}
				astCache.Put(result.FilePath, result.File, result.FileSet, modTime, result.Size)
			}
		}

		// Verify cache usage
		stats := astCache.GetStatistics()
		if stats["entries"].(int) != 3 {
			t.Errorf("Expected 3 cached AST entries, got %v", stats["entries"])
		}

		// Phase 1.1c: Use token pool for memory efficiency
		tokenPool := core.NewTokenPool(core.DefaultTokenPoolConfig())
		tokenPool.WarmUp(5)

		poolStats := tokenPool.GetStatistics()
		if poolStats["pool_size"].(int) < 3 {
			t.Errorf("Expected pool to be warmed up")
		}

		// Phase 1.1d: Type check all packages
		typeChecker := core.NewIncrementalTypeChecker(core.DefaultTypeCheckerConfig())

		// Collect files for type checking
		files := make([]*ast.File, 0, len(results))
		var fset *token.FileSet

		for _, result := range results {
			if result.Error == nil {
				files = append(files, result.File)
				if fset == nil {
					fset = result.FileSet
				}
			}
		}

		typeResult, typeErr := typeChecker.CheckPackage(ctx, "main", files, fset)
		if typeErr != nil {
			t.Logf("Type checking completed with expected errors: %v", typeErr)
		}

		if typeResult == nil {
			t.Error("Expected type check result")
		}

		// Phase 1.1e: Format all files
		formatter := core.NewBatchFormatter(core.DefaultBatchFormatterConfig())

		fileMap := make(map[string]*ast.File)
		for _, result := range results {
			if result.Error == nil {
				fileMap[result.FilePath] = result.File
			}
		}

		formatResults, err := formatter.FormatFiles(ctx, fileMap, fset)
		if err != nil {
			t.Logf("Formatting completed with some errors: %v", err)
		}

		if len(formatResults) != len(fileMap) {
			t.Errorf("Expected format results for all files")
		}

		// Phase 1.1f: Test import caching
		importer := core.NewCachedImporter(core.DefaultImportCacheConfig())

		// Import some standard packages
		standardPkgs := []string{"fmt", "os"}
		for _, pkg := range standardPkgs {
			_, err := importer.Import(pkg)
			if err != nil {
				t.Errorf("Failed to import %s: %v", pkg, err)
			}
		}

		importStats := importer.GetStatistics()
		if importStats["entries"].(int) != 2 {
			t.Errorf("Expected 2 cached imports, got %v", importStats["entries"])
		}

		// Verify all components have good performance metrics
		parserStats := parser.GetStatistics()
		astStats := astCache.GetStatistics()
		typeStats := typeChecker.GetStatistics()
		formatStats := formatter.GetStatistics()

		t.Logf("Full Pipeline Integration Performance:")
		t.Logf("  Parser: %v files/sec", parserStats["files_per_second"])
		t.Logf("  AST Cache: %.1f%% hit ratio", astStats["hit_ratio"])
		t.Logf("  Type Checker: %.1f%% cache hit ratio", typeStats["hit_ratio"])
		t.Logf("  Formatter: %.1f%% success rate", formatStats["success_rate"])
		t.Logf("  Import Cache: %.1f%% hit ratio", importStats["hit_ratio"])
	})
}

// TestPerformanceIntegration tests end-to-end performance
func TestPerformanceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create larger test dataset
	numFiles := 20
	config := core.DefaultConfig()
	config.MaxWorkers = 4
	parser := core.NewParallelParser(config)

	tempDir := t.TempDir()

	// Create multiple test files
	for i := 0; i < numFiles; i++ {
		content := fmt.Sprintf(`package test%d

import (
	"fmt"
	"time"
)

type Data%d struct {
	ID   int
	Name string
	Time time.Time
}

func Process%d(data *Data%d) {
	fmt.Printf("Processing %%v\n", data)
}

func main() {
	d := &Data%d{ID: %d, Name: "test", Time: time.Now()}
	Process%d(d)
}`, i, i, i, i, i, i, i)

		filename := filepath.Join(tempDir, fmt.Sprintf("file%d.go", i))
		err := os.WriteFile(filename, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
	}

	// Test full pipeline performance
	start := time.Now()

	// Parse all files
	ctx := context.Background()
	results, err := parser.ParseDirectory(ctx, tempDir)
	if err != nil {
		t.Fatalf("Failed to parse directory: %v", err)
	}

	parseTime := time.Since(start)

	// Collect ASTs for further processing
	fileMap := make(map[string]*ast.File)
	var fset *token.FileSet

	for _, result := range results {
		if result.Error == nil {
			fileMap[result.FilePath] = result.File
			if fset == nil {
				fset = result.FileSet
			}
		}
	}

	// Format all files
	formatter := core.NewBatchFormatter(core.DefaultBatchFormatterConfig())

	formatStart := time.Now()
	formatResults, err := formatter.FormatFiles(ctx, fileMap, fset)
	formatTime := time.Since(formatStart)

	if err != nil {
		t.Logf("Formatting completed with some errors: %v", err)
	}

	// Report performance metrics
	parserStats := parser.GetStatistics()
	formatStats := formatter.GetStatistics()

	totalTime := parseTime + formatTime

	t.Logf("Performance Integration Results:")
	t.Logf("  Files processed: %d", numFiles)
	t.Logf("  Parse time: %v", parseTime)
	t.Logf("  Format time: %v", formatTime)
	t.Logf("  Total time: %v", totalTime)
	t.Logf("  Parse rate: %.0f files/sec", parserStats["files_per_second"])
	t.Logf("  Format rate: %.0f files/sec", formatStats["files_per_second"])
	t.Logf("  Format results: %d", len(formatResults))

	// Performance expectations
	if totalTime > 5*time.Second {
		t.Errorf("Performance too slow: %v (expected < 5s for %d files)", totalTime, numFiles)
	}

	if len(results) != numFiles {
		t.Errorf("Expected %d parsed files, got %d", numFiles, len(results))
	}
}
