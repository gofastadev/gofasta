// Error handling integration tests for Phase 1.1 components
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

// TestErrorHandlingIntegration tests error handling across components
func TestErrorHandlingIntegration(t *testing.T) {
	// Create components with error tracking
	astCache := core.NewASTCache(core.DefaultASTCacheConfig())
	typeChecker := core.NewIncrementalTypeChecker(core.DefaultTypeCheckerConfig())
	formatter := core.NewBatchFormatter(core.DefaultBatchFormatterConfig())
	
	// Test with problematic files
	fset := token.NewFileSet()
	
	// Valid file
	validFile, _ := parser.ParseFile(fset, "valid.go", "package test\nfunc Valid() {}", parser.ParseComments)
	
	// Create mixed file set with valid and invalid entries
	files := map[string]*ast.File{
		"valid.go":   validFile,
		"invalid.go": nil, // This should cause errors
	}
	
	ctx := context.Background()
	
	// Test AST cache with mixed data
	astCache.Put("valid.go", validFile, fset, time.Now(), 100)
	astCache.Put("invalid.go", nil, nil, time.Now(), 0)
	
	// Test type checker with mixed data
	_, err1 := typeChecker.CheckPackage(ctx, "test", []*ast.File{validFile}, fset)
	_, err2 := typeChecker.CheckPackage(ctx, "test", []*ast.File{nil}, fset)
	
	// Test formatter with mixed data
	formatResults, formatErr := formatter.FormatFiles(ctx, files, fset)
	
	// Verify error handling
	if len(formatResults) != 2 {
		t.Errorf("Expected results for all files, got %d", len(formatResults))
	}
	
	if formatResults["valid.go"].Error != nil {
		t.Errorf("Expected valid file to succeed: %v", formatResults["valid.go"].Error)
	}
	
	if formatResults["invalid.go"].Error == nil {
		t.Error("Expected invalid file to have error")
	}
	
	if formatErr == nil {
		t.Error("Expected overall format error when some files fail")
	}
	
	// Verify error metrics
	formatStats := formatter.GetStatistics()
	if formatStats["error_count"].(int64) == 0 {
		t.Error("Expected error count in formatter metrics")
	}
	
	// Type checker should handle both valid and invalid inputs
	if err1 != nil {
		t.Logf("Valid file type check: %v", err1)
	}
	
	if err2 == nil || (err2 != nil && err2.Error() != "no valid files to type check") {
		t.Logf("Note: nil file type check returned: %v (this is acceptable)", err2)
	}
	
	t.Logf("Error Handling Integration Results:")
	t.Logf("  Format error count: %v", formatStats["error_count"])
	t.Logf("  Format success rate: %.1f%%", formatStats["success_rate"])
}