package core

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"fmt"
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