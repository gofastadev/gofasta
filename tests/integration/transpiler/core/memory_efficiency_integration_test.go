// Memory efficiency integration tests for Phase 1.1 components
package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/healtronlabs/gofasta/transpiler/core"
)

// TestMemoryEfficiencyIntegration tests memory usage across components
func TestMemoryEfficiencyIntegration(t *testing.T) {
	// Create components with memory tracking
	astCache := core.NewASTCache(&core.ASTCacheConfig{
		MaxEntries:    10,
		TTL:           time.Hour,
		MaxMemoryMB:   50,
		EnableMetrics: true,
	})

	tokenPool := core.NewTokenPool(&core.TokenPoolConfig{
		InitialSize:   3,
		MaxSize:       5,
		EnableMetrics: true,
	})

	importer := core.NewCachedImporter(&core.ImportCacheConfig{
		MaxEntries:    10,
		TTL:           time.Hour,
		EnableMetrics: true,
		MaxMemoryMB:   20,
	})

	// Process multiple files to test memory management
	tempDir := t.TempDir()

	for i := 0; i < 15; i++ {
		content := fmt.Sprintf("package test%d\nfunc Test%d() {}", i, i)
		filename := filepath.Join(tempDir, fmt.Sprintf("test%d.go", i))
		os.WriteFile(filename, []byte(content), 0644)
	}

	// Parse and cache files
	parser := core.NewParallelParser(core.DefaultConfig())
	ctx := context.Background()
	results, err := parser.ParseDirectory(ctx, tempDir)
	if err != nil {
		t.Fatalf("Failed to parse directory: %v", err)
	}

	// Cache all parsed files (should trigger evictions)
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

	// Import multiple packages (should trigger evictions)
	packages := []string{"fmt", "os", "io", "time", "context", "sync", "net/http", "encoding/json", "strings", "strconv", "errors"}
	for _, pkg := range packages {
		importer.Import(pkg)
	}

	// Verify memory constraints are respected
	astStats := astCache.GetStatistics()
	importStats := importer.GetStatistics()
	poolStats := tokenPool.GetStatistics()

	// Should have triggered evictions due to entry limits
	if astStats["evictions"].(int64) == 0 {
		t.Error("Expected AST cache evictions due to entry limit")
	}

	if astStats["entries"].(int) > 10 {
		t.Error("AST cache exceeded MaxEntries limit")
	}

	if importStats["entries"].(int) > 10 {
		t.Error("Import cache exceeded MaxEntries limit")
	}

	t.Logf("Memory Efficiency Results:")
	t.Logf("  AST Cache: %v entries, %v evictions", astStats["entries"], astStats["evictions"])
	t.Logf("  Import Cache: %v entries, %v evictions", importStats["entries"], importStats["evictions"])
	t.Logf("  Token Pool: %v created, %v reused", poolStats["created"], poolStats["reused"])
}
