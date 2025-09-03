package core

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"fmt"
	"strings"
	"golang.org/x/tools/go/analysis"
	"testing"
	"time"
)

func TestNewAnalysisCache(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		cache := NewAnalysisCache(nil)
		if cache == nil {
			t.Fatal("Expected non-nil cache")
		}
		if cache.config == nil {
			t.Fatal("Expected non-nil config")
		}
		
		// Check that standard analyzers are registered
		analyzers := cache.ListAnalyzers()
		if len(analyzers) == 0 {
			t.Error("Expected standard analyzers to be registered")
		}
	})
	
	t.Run("custom config", func(t *testing.T) {
		config := &AnalysisCacheConfig{
			MaxResults:      500,
			TTL:            10 * time.Minute,
			EnableAllPasses: false,
		}
		cache := NewAnalysisCache(config)
		if cache.config.MaxResults != 500 {
			t.Errorf("Expected MaxResults=500, got %d", cache.config.MaxResults)
		}
		
		// No standard analyzers should be registered
		analyzers := cache.ListAnalyzers()
		if len(analyzers) != 0 {
			t.Error("Expected no analyzers when EnableAllPasses=false")
		}
	})
}

func TestRegisterAnalyzer(t *testing.T) {
	cache := NewAnalysisCache(&AnalysisCacheConfig{
		EnableAllPasses: false,
	})
	
	// Create a custom analyzer
	customAnalyzer := &analysis.Analyzer{
		Name: "custom",
		Doc:  "Custom analyzer for testing",
		Run: func(pass *analysis.Pass) (interface{}, error) {
			return "custom result", nil
		},
	}
	
	cache.RegisterAnalyzer(customAnalyzer)
	
	analyzer, exists := cache.GetAnalyzer("custom")
	if !exists {
		t.Fatal("Custom analyzer not found")
	}
	if analyzer.Name != "custom" {
		t.Errorf("Expected analyzer name 'custom', got '%s'", analyzer.Name)
	}
}

func TestRunAnalysis(t *testing.T) {
	src := `package main

func main() {
	x := 42
	if x == x {
		println("always true")
	}
}`
	
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	
	// Create a simple analyzer
	testAnalyzer := &analysis.Analyzer{
		Name: "test",
		Doc:  "Test analyzer",
		Run: func(pass *analysis.Pass) (interface{}, error) {
			count := 0
			for _, f := range pass.Files {
				ast.Inspect(f, func(n ast.Node) bool {
					if _, ok := n.(*ast.IfStmt); ok {
						count++
					}
					return true
				})
			}
			return count, nil
		},
	}
	
	cache := NewAnalysisCache(&AnalysisCacheConfig{
		EnableAllPasses: false,
	})
	cache.RegisterAnalyzer(testAnalyzer)
	
	// Create a pass
	pass := &analysis.Pass{
		Analyzer: testAnalyzer,
		Fset:     fset,
		Files:    []*ast.File{file},
		Report: func(diag analysis.Diagnostic) {
			// Test report function
		},
	}
	
	// First run - should miss cache
	result1, err := cache.RunAnalysis(testAnalyzer, pass)
	if err != nil {
		t.Fatal(err)
	}
	if result1.(int) != 1 {
		t.Errorf("Expected 1 if statement, got %d", result1.(int))
	}
	
	// Second run - should hit cache
	result2, err := cache.RunAnalysis(testAnalyzer, pass)
	if err != nil {
		t.Fatal(err)
	}
	if result2.(int) != result1.(int) {
		t.Error("Cached result differs from original")
	}
	
	stats := cache.GetStatistics()
	if stats["cache_hits"].(int64) < 1 {
		t.Error("Expected at least one cache hit")
	}
}

func TestRunAllAnalyzers(t *testing.T) {
	src := `package main

func main() {
	fmt.Printf("%d", "string") // Type mismatch
}`
	
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	
	// Create type info
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	
	pkg := &Package{
		Fset:      fset,
		Files:     []*ast.File{file},
		Pkg:       types.NewPackage("main", "main"),
		TypesInfo: info,
		Path:      "main",
	}
	
	cache := NewAnalysisCache(&AnalysisCacheConfig{
		EnableAllPasses:    false,
		ConcurrentAnalysis: false,
	})
	
	// Register a simple analyzer
	analyzer1 := &analysis.Analyzer{
		Name: "analyzer1",
		Doc:  "Test analyzer 1",
		Run: func(pass *analysis.Pass) (interface{}, error) {
			return "result1", nil
		},
	}
	
	analyzer2 := &analysis.Analyzer{
		Name: "analyzer2",
		Doc:  "Test analyzer 2",
		Run: func(pass *analysis.Pass) (interface{}, error) {
			return "result2", nil
		},
	}
	
	cache.RegisterAnalyzer(analyzer1)
	cache.RegisterAnalyzer(analyzer2)
	
	results, err := cache.RunAllAnalyzers(pkg)
	if err != nil {
		t.Fatal(err)
	}
	
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	
	if results["analyzer1"] != "result1" {
		t.Errorf("Expected result1 for analyzer1, got %v", results["analyzer1"])
	}
	
	if results["analyzer2"] != "result2" {
		t.Errorf("Expected result2 for analyzer2, got %v", results["analyzer2"])
	}
}

func TestConcurrentAnalysis(t *testing.T) {
	src := `package main; func main() {}`
	
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	
	pkg := &Package{
		Fset:  fset,
		Files: []*ast.File{file},
		Pkg:   types.NewPackage("main", "main"),
		Path:  "main",
	}
	
	cache := NewAnalysisCache(&AnalysisCacheConfig{
		EnableAllPasses:    false,
		ConcurrentAnalysis: true,
		AnalysisWorkers:   2,
	})
	
	// Register multiple analyzers
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("analyzer%d", i)
		result := fmt.Sprintf("result%d", i)
		analyzer := &analysis.Analyzer{
			Name: name,
			Doc:  "Test analyzer",
			Run: func(res string) func(*analysis.Pass) (interface{}, error) {
				return func(pass *analysis.Pass) (interface{}, error) {
					time.Sleep(10 * time.Millisecond) // Simulate work
					return res, nil
				}
			}(result),
		}
		cache.RegisterAnalyzer(analyzer)
	}
	
	start := time.Now()
	results, err := cache.RunAllAnalyzers(pkg)
	duration := time.Since(start)
	
	if err != nil {
		t.Fatal(err)
	}
	
	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}
	
	// With concurrency, should be faster than sequential
	// 5 analyzers * 10ms = 50ms sequential
	// With 2 workers, should be around 30ms
	if duration > 40*time.Millisecond {
		t.Logf("Concurrent analysis might not be working efficiently: %v", duration)
	}
}

func TestDiagnosticsCaching(t *testing.T) {
	src := `package main

func main() {
	x := 42
}`
	
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	
	// Create an analyzer that reports diagnostics
	diagAnalyzer := &analysis.Analyzer{
		Name: "diag",
		Doc:  "Diagnostic analyzer",
		Run: func(pass *analysis.Pass) (interface{}, error) {
			for _, f := range pass.Files {
				ast.Inspect(f, func(n ast.Node) bool {
					if lit, ok := n.(*ast.BasicLit); ok {
						pass.Report(analysis.Diagnostic{
							Pos:     lit.Pos(),
							Message: fmt.Sprintf("Found literal: %s", lit.Value),
						})
					}
					return true
				})
			}
			return nil, nil
		},
	}
	
	cache := NewAnalysisCache(&AnalysisCacheConfig{
		EnableAllPasses:   false,
		DiagnosticCaching: true,
	})
	cache.RegisterAnalyzer(diagAnalyzer)
	
	pkg := &Package{
		Fset:  fset,
		Files: []*ast.File{file},
		Pkg:   types.NewPackage("main", "main"),
		Path:  "main",
	}
	
	// Run analysis
	pass := cache.createPass(diagAnalyzer, pkg)
	_, err = cache.RunAnalysis(diagAnalyzer, pass)
	if err != nil {
		t.Fatal(err)
	}
	
	// Check diagnostics
	diags := cache.GetDiagnostics("diag", pkg)
	if len(diags) == 0 {
		t.Error("Expected diagnostics to be cached")
	}
	
	// Get all diagnostics
	allDiags := cache.GetAllDiagnostics()
	if len(allDiags) == 0 {
		t.Error("Expected to get all diagnostics")
	}
}

// TestFact is a test implementation of analysis.Fact
type TestFact struct {
	Value string
}

// AFact implements the analysis.Fact interface
func (*TestFact) AFact() {}

func TestFactCaching(t *testing.T) {
	cache := NewAnalysisCache(&AnalysisCacheConfig{
		EnableAllPasses: false,
		FactCaching:    true,
	})
	
	// Create a dummy object and fact
	obj := types.NewVar(token.NoPos, nil, "test", types.Typ[types.Int])
	
	fact := &TestFact{Value: "test"}
	
	// Export fact
	cache.exportFact(obj, fact)
	
	// Import fact
	found := cache.importFact(obj, fact)
	if !found {
		t.Error("Expected to find exported fact")
	}
	
	// Test package fact
	pkg := types.NewPackage("test", "test")
	cache.exportPackageFact(pkg, fact)
	
	found = cache.importPackageFact(pkg, fact)
	if !found {
		t.Error("Expected to find exported package fact")
	}
	
	stats := cache.GetStatistics()
	if stats["total_facts"].(int64) < 2 {
		t.Error("Expected at least 2 facts")
	}
}

func TestGenerateReport(t *testing.T) {
	src := `package main

func main() {
	x := 42
	println(x)
}`
	
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	
	pkg := &Package{
		Fset:  fset,
		Files: []*ast.File{file},
		Pkg:   types.NewPackage("main", "main"),
		Path:  "main",
	}
	
	cache := NewAnalysisCache(&AnalysisCacheConfig{
		EnableAllPasses: false,
	})
	
	// Register a simple analyzer
	analyzer := &analysis.Analyzer{
		Name: "test",
		Doc:  "Test analyzer",
		Run: func(pass *analysis.Pass) (interface{}, error) {
			return "test result", nil
		},
	}
	cache.RegisterAnalyzer(analyzer)
	
	report, err := cache.GenerateReport(pkg)
	if err != nil {
		t.Fatal(err)
	}
	
	if report.Package != "main" {
		t.Errorf("Expected package 'main', got '%s'", report.Package)
	}
	
	if len(report.Analyzers) != 1 {
		t.Errorf("Expected 1 analyzer, got %d", len(report.Analyzers))
	}
	
	if report.Results["test"] != "test result" {
		t.Errorf("Expected 'test result', got %v", report.Results["test"])
	}
}

func TestFilterDiagnostics(t *testing.T) {
	cache := NewAnalysisCache(&AnalysisCacheConfig{
		EnableAllPasses: false,
	})
	
	// Manually add diagnostics for testing
	cache.mu.Lock()
	cache.diagnostics["test1"] = []*analysis.Diagnostic{
		{Message: "Error: something went wrong"},
		{Message: "Warning: potential issue"},
	}
	cache.diagnostics["test2"] = []*analysis.Diagnostic{
		{Message: "Error: another error"},
		{Message: "Info: just information"},
	}
	cache.mu.Unlock()
	
	// Filter by severity
	errors := cache.FilterDiagnostics("Error")
	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errors))
	}
	
	warnings := cache.FilterDiagnostics("Warning")
	if len(warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(warnings))
	}
}

func TestAnalysisCacheEviction(t *testing.T) {
	cache := NewAnalysisCache(&AnalysisCacheConfig{
		MaxResults:      2,
		EnableAllPasses: false,
	})
	
	// Create analyzer
	analyzer := &analysis.Analyzer{
		Name: "test",
		Doc:  "Test",
		Run: func(pass *analysis.Pass) (interface{}, error) {
			return pass.Files[0].Name.Name, nil
		},
	}
	cache.RegisterAnalyzer(analyzer)
	
	fset := token.NewFileSet()
	
	// Create multiple packages to trigger eviction
	for i := 0; i < 3; i++ {
		src := fmt.Sprintf("package pkg%d", i)
		file, _ := parser.ParseFile(fset, fmt.Sprintf("test%d.go", i), src, 0)
		
		pass := &analysis.Pass{
			Analyzer: analyzer,
			Fset:     fset,
			Files:    []*ast.File{file},
			Report:   func(analysis.Diagnostic) {},
		}
		
		cache.RunAnalysis(analyzer, pass)
	}
	
	stats := cache.GetStatistics()
	cachedResults := stats["cached_results"].(int)
	if cachedResults > 2 {
		t.Errorf("Expected at most 2 cached results, got %d", cachedResults)
	}
}

func TestAnalysisWarmupCache(t *testing.T) {
	cache := NewAnalysisCache(&AnalysisCacheConfig{
		EnableAllPasses: false,
	})
	
	// Register analyzer
	analyzer := &analysis.Analyzer{
		Name: "warmup",
		Doc:  "Warmup analyzer",
		Run: func(pass *analysis.Pass) (interface{}, error) {
			return "warmed up", nil
		},
	}
	cache.RegisterAnalyzer(analyzer)
	
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "test.go", "package main", 0)
	
	packages := []*Package{
		{
			Fset:  fset,
			Files: []*ast.File{file},
			Pkg:   types.NewPackage("pkg1", "pkg1"),
			Path:  "pkg1",
		},
		{
			Fset:  fset,
			Files: []*ast.File{file},
			Pkg:   types.NewPackage("pkg2", "pkg2"),
			Path:  "pkg2",
		},
	}
	
	err := cache.WarmupCache(packages)
	if err != nil {
		t.Fatal(err)
	}
	
	stats := cache.GetStatistics()
	if stats["cached_results"].(int) != 2 {
		t.Errorf("Expected 2 cached results after warmup, got %d", stats["cached_results"].(int))
	}
}

func TestAnalysisCacheClear(t *testing.T) {
	cache := NewAnalysisCache(nil)
	
	// Add some data
	cache.mu.Lock()
	cache.resultCache["test"] = &CachedResult{}
	cache.diagnostics["test"] = []*analysis.Diagnostic{}
	cache.factCache["test"] = make(map[analysis.Fact]bool)
	cache.hits = 10
	cache.misses = 5
	cache.mu.Unlock()
	
	// Clear cache
	cache.Clear()
	
	// Verify everything is cleared
	stats := cache.GetStatistics()
	if stats["cached_results"].(int) != 0 {
		t.Error("Expected 0 cached results after clear")
	}
	if stats["cache_hits"].(int64) != 0 {
		t.Error("Expected 0 cache hits after clear")
	}
	if stats["cache_misses"].(int64) != 0 {
		t.Error("Expected 0 cache misses after clear")
	}
}

func TestSkipPasses(t *testing.T) {
	config := &AnalysisCacheConfig{
		EnableAllPasses: true,
		SkipPasses:     []string{"printf", "atomic"},
	}
	cache := NewAnalysisCache(config)
	
	// Check that skipped passes are not registered
	_, exists := cache.GetAnalyzer("printf")
	if exists {
		t.Error("printf analyzer should be skipped")
	}
	
	_, exists = cache.GetAnalyzer("atomic")
	if exists {
		t.Error("atomic analyzer should be skipped")
	}
	
	// Check that other passes are registered
	_, exists = cache.GetAnalyzer("bools")
	if !exists {
		t.Error("bools analyzer should be registered")
	}
}

// TestSetupPackageFacts tests package fact initialization
func TestSetupPackageFacts(t *testing.T) {
	cache := NewAnalysisCache(nil)
	
	// Create a mock package
	pkg := createSimpleTestPackage(t)
	
	t.Run("setup with valid package", func(t *testing.T) {
		// Create a mock analyzer and pass
		analyzer := &analysis.Analyzer{
			Name: "test-analyzer",
			Doc:  "Test analyzer for package facts",
			Run: func(pass *analysis.Pass) (interface{}, error) {
				return nil, nil
			},
		}
		
		pass := &analysis.Pass{
			Analyzer:  analyzer,
			Fset:      pkg.Fset,
			Files:     pkg.Files,
			Pkg:       pkg.Pkg,
			TypesInfo: pkg.TypesInfo,
		}
		
		// SetupPackageFacts should not panic and should complete without error
		cache.SetupPackageFacts(pass, pkg)
		
		// Since the function is currently a no-op, we just verify it doesn't crash
		// In a full implementation, we would verify that package facts are properly initialized
	})
	
	t.Run("setup with nil pass", func(t *testing.T) {
		// Should handle nil pass gracefully
		cache.SetupPackageFacts(nil, pkg)
	})
	
	t.Run("setup with nil package", func(t *testing.T) {
		analyzer := &analysis.Analyzer{Name: "test"}
		pass := &analysis.Pass{Analyzer: analyzer}
		
		// Should handle nil package gracefully  
		cache.SetupPackageFacts(pass, nil)
	})
	
	t.Run("setup with both nil", func(t *testing.T) {
		// Should handle both nil parameters gracefully
		cache.SetupPackageFacts(nil, nil)
	})
}

// TestFindDiagnosticsByType tests type-specific diagnostic filtering
func TestFindDiagnosticsByType(t *testing.T) {
	cache := NewAnalysisCache(nil)
	
	// Add some test diagnostics to the cache
	testDiags := []*analysis.Diagnostic{
		{Message: "unused variable detected"},
		{Message: "ineffective assignment found"},
		{Message: "variable not used properly"},
		{Message: "function parameter unused"},
		{Message: "deadcode detected in function"},
	}
	
	// Manually populate cache diagnostics for testing
	cache.diagnostics["test-key-1"] = testDiags[:2]
	cache.diagnostics["test-key-2"] = testDiags[2:4]
	cache.diagnostics["test-key-3"] = testDiags[4:]
	
	t.Run("find unused diagnostics", func(t *testing.T) {
		result := cache.FindDiagnosticsByType("unused")
		
		// Based on our test data: "unused variable detected" and "function parameter unused"
		expectedCount := 2  
		if len(result) != expectedCount {
			t.Errorf("FindDiagnosticsByType('unused') = %d diagnostics, want %d", len(result), expectedCount)
		}
		
		// Verify all returned diagnostics contain "unused"
		for _, diag := range result {
			if !strings.Contains(diag.Message, "unused") {
				t.Errorf("Diagnostic message %q should contain 'unused'", diag.Message)
			}
		}
	})
	
	t.Run("find variable diagnostics", func(t *testing.T) {
		result := cache.FindDiagnosticsByType("variable")
		
		expectedCount := 2 // "unused variable" and "variable not used"
		if len(result) != expectedCount {
			t.Errorf("FindDiagnosticsByType('variable') = %d diagnostics, want %d", len(result), expectedCount)
		}
	})
	
	t.Run("find non-existent type", func(t *testing.T) {
		result := cache.FindDiagnosticsByType("nonexistent")
		
		if len(result) != 0 {
			t.Errorf("FindDiagnosticsByType('nonexistent') = %d diagnostics, want 0", len(result))
		}
	})
	
	t.Run("find with empty string", func(t *testing.T) {
		result := cache.FindDiagnosticsByType("")
		
		// All diagnostics should contain empty string
		expectedCount := len(testDiags)
		if len(result) != expectedCount {
			t.Errorf("FindDiagnosticsByType('') = %d diagnostics, want %d", len(result), expectedCount)
		}
	})
	
	t.Run("concurrent access", func(t *testing.T) {
		done := make(chan bool, 10)
		
		// Start multiple goroutines accessing diagnostics
		for i := 0; i < 10; i++ {
			go func() {
				_ = cache.FindDiagnosticsByType("unused")
				done <- true
			}()
		}
		
		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// TestCreateReportFunc tests report function creation and execution
func TestCreateReportFunc(t *testing.T) {
	cache := NewAnalysisCache(nil)
	
	t.Run("create and use report function", func(t *testing.T) {
		key := "test-report-key"
		reportFunc := cache.createReportFunc(key)
		
		if reportFunc == nil {
			t.Fatal("createReportFunc() returned nil function")
		}
		
		// Create a test diagnostic
		testDiag := analysis.Diagnostic{
			Message: "test diagnostic message",
			Pos:     token.NoPos,
		}
		
		// Initial state - no diagnostics
		initialCount := len(cache.diagnostics[key])
		
		// Use the report function
		reportFunc(testDiag)
		
		// Verify diagnostic was added
		if len(cache.diagnostics[key]) != initialCount+1 {
			t.Errorf("Expected diagnostic count to increase by 1, got %d -> %d", 
				initialCount, len(cache.diagnostics[key]))
		}
		
		// Verify the diagnostic content
		storedDiag := cache.diagnostics[key][len(cache.diagnostics[key])-1]
		if storedDiag.Message != testDiag.Message {
			t.Errorf("Stored diagnostic message = %q, want %q", storedDiag.Message, testDiag.Message)
		}
		
		// Verify total diagnostics counter
		if cache.totalDiags == 0 {
			t.Error("Expected totalDiags to be incremented")
		}
	})
	
	t.Run("multiple reports to same key", func(t *testing.T) {
		key := "multi-report-key"
		reportFunc := cache.createReportFunc(key)
		
		// Report multiple diagnostics
		diag1 := analysis.Diagnostic{Message: "first diagnostic"}
		diag2 := analysis.Diagnostic{Message: "second diagnostic"}
		diag3 := analysis.Diagnostic{Message: "third diagnostic"}
		
		reportFunc(diag1)
		reportFunc(diag2)
		reportFunc(diag3)
		
		// Verify all diagnostics are stored
		if len(cache.diagnostics[key]) != 3 {
			t.Errorf("Expected 3 diagnostics for key %q, got %d", key, len(cache.diagnostics[key]))
		}
		
		// Verify order is preserved
		messages := []string{"first diagnostic", "second diagnostic", "third diagnostic"}
		for i, expectedMsg := range messages {
			if cache.diagnostics[key][i].Message != expectedMsg {
				t.Errorf("Diagnostic[%d].Message = %q, want %q", i, cache.diagnostics[key][i].Message, expectedMsg)
			}
		}
	})
	
	t.Run("different keys have separate diagnostics", func(t *testing.T) {
		key1 := "separate-key-1"
		key2 := "separate-key-2"
		
		reportFunc1 := cache.createReportFunc(key1)
		reportFunc2 := cache.createReportFunc(key2)
		
		// Report to different keys
		reportFunc1(analysis.Diagnostic{Message: "key1 diagnostic"})
		reportFunc2(analysis.Diagnostic{Message: "key2 diagnostic"})
		
		// Verify separation
		if len(cache.diagnostics[key1]) != 1 {
			t.Errorf("Expected 1 diagnostic for key1, got %d", len(cache.diagnostics[key1]))
		}
		if len(cache.diagnostics[key2]) != 1 {
			t.Errorf("Expected 1 diagnostic for key2, got %d", len(cache.diagnostics[key2]))
		}
		
		// Verify correct content
		if cache.diagnostics[key1][0].Message != "key1 diagnostic" {
			t.Error("key1 diagnostic has wrong message")
		}
		if cache.diagnostics[key2][0].Message != "key2 diagnostic" {
			t.Error("key2 diagnostic has wrong message")
		}
	})
	
	t.Run("concurrent reporting", func(t *testing.T) {
		key := "concurrent-key"
		reportFunc := cache.createReportFunc(key)
		
		done := make(chan bool, 100)
		
		// Start 100 concurrent reports
		for i := 0; i < 100; i++ {
			go func(id int) {
				diag := analysis.Diagnostic{Message: fmt.Sprintf("concurrent diagnostic %d", id)}
				reportFunc(diag)
				done <- true
			}(i)
		}
		
		// Wait for all reports
		for i := 0; i < 100; i++ {
			<-done
		}
		
		// Verify all diagnostics were stored
		if len(cache.diagnostics[key]) != 100 {
			t.Errorf("Expected 100 diagnostics after concurrent reporting, got %d", len(cache.diagnostics[key]))
		}
	})
}

// TestInvalidateResult tests result invalidation and cleanup
func TestInvalidateResult(t *testing.T) {
	cache := NewAnalysisCache(nil)
	
	t.Run("invalidate existing result", func(t *testing.T) {
		key := "test-invalidate-key"
		
		// Set up some cached data
		cache.resultCache[key] = &CachedResult{
			Result: "test result",
			Facts:  make(map[analysis.Fact]bool),
		}
		cache.diagnostics[key] = []*analysis.Diagnostic{
			{Message: "test diagnostic"},
		}
		cache.factCache[key] = make(map[analysis.Fact]bool)
		
		// Verify data exists before invalidation
		if _, exists := cache.resultCache[key]; !exists {
			t.Fatal("Test setup failed - result should exist")
		}
		if _, exists := cache.diagnostics[key]; !exists {
			t.Fatal("Test setup failed - diagnostics should exist") 
		}
		if _, exists := cache.factCache[key]; !exists {
			t.Fatal("Test setup failed - facts should exist")
		}
		
		// Invalidate the result
		cache.InvalidateResult(key)
		
		// Verify all data was removed
		if _, exists := cache.resultCache[key]; exists {
			t.Error("Result cache should be cleared after invalidation")
		}
		if _, exists := cache.diagnostics[key]; exists {
			t.Error("Diagnostics should be cleared after invalidation")
		}
		if _, exists := cache.factCache[key]; exists {
			t.Error("Fact cache should be cleared after invalidation")
		}
	})
	
	t.Run("invalidate non-existent result", func(t *testing.T) {
		key := "non-existent-key"
		
		// Should not panic when invalidating non-existent key
		cache.InvalidateResult(key)
		
		// Verify no data exists (should be safe to check)
		if _, exists := cache.resultCache[key]; exists {
			t.Error("Non-existent key should not have result cache entry")
		}
	})
	
	t.Run("invalidate partial data", func(t *testing.T) {
		key := "partial-data-key"
		
		// Set up only some cached data
		cache.resultCache[key] = &CachedResult{Result: "test"}
		// No diagnostics or facts for this key
		
		// Should handle partial data safely
		cache.InvalidateResult(key)
		
		// Verify result was removed
		if _, exists := cache.resultCache[key]; exists {
			t.Error("Result cache should be cleared even for partial data")
		}
	})
	
	t.Run("invalidate preserves other keys", func(t *testing.T) {
		key1 := "preserve-key-1"
		key2 := "preserve-key-2"
		keyToInvalidate := "invalidate-this-key"
		
		// Set up data for multiple keys
		cache.resultCache[key1] = &CachedResult{Result: "result1"}
		cache.resultCache[key2] = &CachedResult{Result: "result2"}
		cache.resultCache[keyToInvalidate] = &CachedResult{Result: "to-be-invalidated"}
		
		cache.diagnostics[key1] = []*analysis.Diagnostic{{Message: "diag1"}}
		cache.diagnostics[key2] = []*analysis.Diagnostic{{Message: "diag2"}}
		cache.diagnostics[keyToInvalidate] = []*analysis.Diagnostic{{Message: "diag-invalidate"}}
		
		// Invalidate only one key
		cache.InvalidateResult(keyToInvalidate)
		
		// Verify other keys are preserved
		if _, exists := cache.resultCache[key1]; !exists {
			t.Error("key1 result should be preserved")
		}
		if _, exists := cache.resultCache[key2]; !exists {
			t.Error("key2 result should be preserved")
		}
		if _, exists := cache.diagnostics[key1]; !exists {
			t.Error("key1 diagnostics should be preserved")
		}
		if _, exists := cache.diagnostics[key2]; !exists {
			t.Error("key2 diagnostics should be preserved")
		}
		
		// Verify invalidated key is gone
		if _, exists := cache.resultCache[keyToInvalidate]; exists {
			t.Error("Invalidated key should be removed")
		}
		if _, exists := cache.diagnostics[keyToInvalidate]; exists {
			t.Error("Invalidated key diagnostics should be removed")
		}
	})
	
	t.Run("concurrent invalidation", func(t *testing.T) {
		// Set up multiple keys for concurrent invalidation
		keys := make([]string, 50)
		for i := 0; i < 50; i++ {
			key := fmt.Sprintf("concurrent-invalidate-%d", i)
			keys[i] = key
			cache.resultCache[key] = &CachedResult{Result: fmt.Sprintf("result-%d", i)}
			cache.diagnostics[key] = []*analysis.Diagnostic{{Message: fmt.Sprintf("diag-%d", i)}}
		}
		
		done := make(chan bool, 50)
		
		// Invalidate concurrently
		for _, key := range keys {
			go func(k string) {
				cache.InvalidateResult(k)
				done <- true
			}(key)
		}
		
		// Wait for all invalidations
		for i := 0; i < 50; i++ {
			<-done
		}
		
		// Verify all keys were invalidated
		for _, key := range keys {
			if _, exists := cache.resultCache[key]; exists {
				t.Errorf("Key %q should be invalidated", key)
			}
		}
	})
}

// TestGetCachedResult tests cache result retrieval
func TestGetCachedResult(t *testing.T) {
	cache := NewAnalysisCache(nil)
	
	t.Run("get existing cached result", func(t *testing.T) {
		key := "existing-result-key"
		expectedResult := &CachedResult{
			Result: "test cached result",
			Facts:  make(map[analysis.Fact]bool),
		}
		
		// Add result to cache
		cache.resultCache[key] = expectedResult
		
		// Retrieve the result
		result, exists := cache.GetCachedResult(key)
		
		if !exists {
			t.Error("GetCachedResult() exists = false, want true for existing key")
		}
		
		if result != expectedResult {
			t.Error("GetCachedResult() returned different result object")
		}
		
		if result.Result != expectedResult.Result {
			t.Errorf("GetCachedResult().Result = %v, want %v", result.Result, expectedResult.Result)
		}
	})
	
	t.Run("get non-existent cached result", func(t *testing.T) {
		key := "non-existent-key"
		
		result, exists := cache.GetCachedResult(key)
		
		if exists {
			t.Error("GetCachedResult() exists = true, want false for non-existent key")
		}
		
		if result != nil {
			t.Error("GetCachedResult() result should be nil for non-existent key")
		}
	})
	
	t.Run("get multiple different results", func(t *testing.T) {
		keys := []string{"result-1", "result-2", "result-3"}
		expectedResults := []*CachedResult{
			{Result: "first result"},
			{Result: "second result"}, 
			{Result: "third result"},
		}
		
		// Add multiple results
		for i, key := range keys {
			cache.resultCache[key] = expectedResults[i]
		}
		
		// Verify each result can be retrieved correctly
		for i, key := range keys {
			result, exists := cache.GetCachedResult(key)
			
			if !exists {
				t.Errorf("GetCachedResult(%q) exists = false, want true", key)
			}
			
			if result != expectedResults[i] {
				t.Errorf("GetCachedResult(%q) returned wrong result object", key)
			}
			
			if result.Result != expectedResults[i].Result {
				t.Errorf("GetCachedResult(%q).Result = %v, want %v", key, result.Result, expectedResults[i].Result)
			}
		}
	})
	
	t.Run("get result with complex data", func(t *testing.T) {
		key := "complex-result-key"
		expectedResult := &CachedResult{
			Result: map[string]interface{}{
				"findings": []string{"issue1", "issue2"},
				"score":    85,
				"metadata": map[string]string{"analyzer": "test", "version": "1.0"},
			},
			Facts: make(map[analysis.Fact]bool),
		}
		
		// Add complex result to cache
		cache.resultCache[key] = expectedResult
		
		result, exists := cache.GetCachedResult(key)
		
		if !exists {
			t.Error("GetCachedResult() exists = false, want true for complex result")
		}
		
		// Verify complex data structure is preserved
		resultData, ok := result.Result.(map[string]interface{})
		if !ok {
			t.Error("Result data should be preserved as map[string]interface{}")
		}
		
		if len(resultData) != 3 {
			t.Errorf("Expected 3 fields in result data, got %d", len(resultData))
		}
	})
	
	t.Run("concurrent access to cached results", func(t *testing.T) {
		key := "concurrent-access-key"
		expectedResult := &CachedResult{
			Result: "concurrent test result",
		}
		
		cache.resultCache[key] = expectedResult
		
		done := make(chan bool, 100)
		
		// Start 100 concurrent reads
		for i := 0; i < 100; i++ {
			go func() {
				result, exists := cache.GetCachedResult(key)
				if !exists {
					t.Error("Concurrent access should find existing result")
				}
				if result == nil {
					t.Error("Concurrent access should return non-nil result")
				}
				done <- true
			}()
		}
		
		// Wait for all reads
		for i := 0; i < 100; i++ {
			<-done
		}
	})
	
	t.Run("get after invalidation", func(t *testing.T) {
		key := "invalidation-test-key"
		
		// Add result
		cache.resultCache[key] = &CachedResult{Result: "to be invalidated"}
		
		// Verify it exists
		_, exists := cache.GetCachedResult(key)
		if !exists {
			t.Fatal("Result should exist before invalidation")
		}
		
		// Invalidate
		cache.InvalidateResult(key)
		
		// Verify it's gone
		result, exists := cache.GetCachedResult(key)
		if exists {
			t.Error("GetCachedResult() exists = true after invalidation, want false")
		}
		if result != nil {
			t.Error("GetCachedResult() should return nil after invalidation")
		}
	})
}

// createSimpleTestPackage creates a simple test package for analysis
func createSimpleTestPackage(t *testing.T) *Package {
	const src = `package test

func ExampleFunc() {
	var x int
	return
}
`
	
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}
	
	// Create types package and info
	pkg := types.NewPackage("test", "test")
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	
	// Create the Package struct
	testPkg := &Package{
		Path:      "test",
		Fset:      fset,
		Files:     []*ast.File{file},
		Pkg:       pkg,
		TypesInfo: info,
	}
	
	return testPkg
}