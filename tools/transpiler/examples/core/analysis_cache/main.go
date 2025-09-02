// Example demonstrating golang.org/x/tools/go/analysis with analysis cache (Phase 1.2f)
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"log"

	"github.com/healtronlabs/gofasta/tools/transpiler/core"
	"golang.org/x/tools/go/analysis"
)

func main() {
	// Create analysis cache with custom configuration
	config := &core.AnalysisCacheConfig{
		MaxResults:         500,
		EnableMetrics:      true,
		EnableAllPasses:    true,  // Register all standard analyzers
		ConcurrentAnalysis: true,
		AnalysisWorkers:    4,
		FactCaching:        true,
		DiagnosticCaching:  true,
		SkipPasses:         []string{"buildtag"}, // Skip specific passes if needed
	}
	cache := core.NewAnalysisCache(config)

	// Sample Go source code with potential issues
	src := `package main

import (
	"fmt"
	"strings"
)

func main() {
	// Potential issue: always true comparison
	x := 42
	if x == x {
		fmt.Println("Always true")
	}
	
	// Potential issue: unreachable code
	return
	fmt.Println("Unreachable")
	
	// Potential issue: printf format mismatch
	name := "Alice"
	fmt.Printf("Age: %d\n", name)
	
	// Potential issue: unused variable
	unused := "not used"
	_ = unused
	
	// String concatenation
	result := strings.Join([]string{"Hello", "World"}, " ")
	fmt.Println(result)
}

func helper(x int) int {
	// Potential issue: comparing with itself
	if x != x {
		return -1
	}
	return x * 2
}

func emptyFunc() {
	// Empty function
}

type Person struct {
	Name string ` + "`json:\"name\"`" + `
	Age  int    ` + "`json:\"age\"`" + `
}

func (p *Person) String() string {
	// Potential issue: wrong signature for String method
	return fmt.Sprintf("%s (%d)", p.Name, p.Age)
}`

	// Parse the source code
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "example.go", src, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	// Create type information
	conf := types.Config{
		Importer: nil, // Use default importer
		Error: func(err error) {
			// Ignore type errors for this example
		},
	}
	
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	
	pkg, _ := conf.Check("main", fset, []*ast.File{file}, info)

	// Create package for analysis
	analysisPackage := &core.Package{
		Fset:      fset,
		Files:     []*ast.File{file},
		Pkg:       pkg,
		TypesInfo: info,
		Path:      "main",
	}

	fmt.Println("=== Analysis Cache Example ===\n")

	// 1. List registered analyzers
	fmt.Println("1. Registered analyzers:")
	analyzers := cache.ListAnalyzers()
	fmt.Printf("   Total: %d analyzers\n", len(analyzers))
	// Show first 10
	for i, name := range analyzers {
		if i >= 10 {
			fmt.Printf("   ... and %d more\n", len(analyzers)-10)
			break
		}
		if analyzer, exists := cache.GetAnalyzer(name); exists {
			fmt.Printf("   - %s: %s\n", name, truncate(analyzer.Doc, 50))
		}
	}

	// 2. Create and register a custom analyzer
	fmt.Println("\n2. Custom analyzer:")
	customAnalyzer := &analysis.Analyzer{
		Name: "custom_checker",
		Doc:  "Checks for custom patterns in the code",
		Run: func(pass *analysis.Pass) (interface{}, error) {
			count := 0
			for _, file := range pass.Files {
				ast.Inspect(file, func(n ast.Node) bool {
					// Count if statements
					if _, ok := n.(*ast.IfStmt); ok {
						count++
						pass.Report(analysis.Diagnostic{
							Pos:     n.Pos(),
							Message: "Found if statement",
						})
					}
					return true
				})
			}
			return fmt.Sprintf("Found %d if statements", count), nil
		},
	}
	cache.RegisterAnalyzer(customAnalyzer)
	fmt.Printf("   Registered custom analyzer: %s\n", customAnalyzer.Name)

	// 3. Run specific analyzer
	fmt.Println("\n3. Running custom analyzer:")
	pass := &analysis.Pass{
		Analyzer:  customAnalyzer,
		Fset:      fset,
		Files:     []*ast.File{file},
		Pkg:       pkg,
		TypesInfo: info,
		Report: func(diag analysis.Diagnostic) {
			fmt.Printf("   - %s: %s\n", fset.Position(diag.Pos), diag.Message)
		},
	}
	
	result, err := cache.RunAnalysis(customAnalyzer, pass)
	if err != nil {
		log.Printf("Error running analysis: %v", err)
	} else {
		fmt.Printf("   Result: %v\n", result)
	}
	
	// Run again to demonstrate caching
	fmt.Println("\n4. Running analyzer again (should hit cache):")
	result2, err := cache.RunAnalysis(customAnalyzer, pass)
	if err == nil {
		fmt.Printf("   Result (cached): %v\n", result2)
	}

	// 4. Run all analyzers
	fmt.Println("\n5. Running all registered analyzers:")
	results, err := cache.RunAllAnalyzers(analysisPackage)
	if err != nil {
		// Some analyzers might fail, that's okay for demo
		fmt.Printf("   Some analyzers reported errors (expected)\n")
	}
	fmt.Printf("   Completed %d analyses\n", len(results))
	
	// Show some results
	showCount := 0
	for name, res := range results {
		if showCount >= 5 {
			break
		}
		if res != nil {
			fmt.Printf("   - %s: %v\n", name, truncate(fmt.Sprintf("%v", res), 50))
			showCount++
		}
	}

	// 5. Get diagnostics
	fmt.Println("\n6. Diagnostics:")
	diags := cache.GetDiagnostics("custom_checker", analysisPackage)
	fmt.Printf("   Custom checker diagnostics: %d\n", len(diags))
	
	allDiags := cache.GetAllDiagnostics()
	totalDiags := 0
	for _, d := range allDiags {
		totalDiags += len(d)
	}
	fmt.Printf("   Total diagnostics cached: %d\n", totalDiags)

	// 6. Filter diagnostics
	fmt.Println("\n7. Finding specific diagnostics:")
	// Add some test diagnostics
	errorDiags := cache.FindDiagnosticsByType("if")
	fmt.Printf("   Found %d diagnostics containing 'if'\n", len(errorDiags))

	// 7. Generate comprehensive report
	fmt.Println("\n8. Generating analysis report:")
	report, err := cache.GenerateReport(analysisPackage)
	if err != nil {
		fmt.Printf("   Error generating report: %v\n", err)
	} else {
		fmt.Printf("   Package: %s\n", report.Package)
		fmt.Printf("   Analyzers run: %d\n", len(report.Analyzers))
		fmt.Printf("   Results collected: %d\n", len(report.Results))
		fmt.Printf("   Duration: %v\n", report.Duration)
		fmt.Printf("   Report time: %v\n", report.Timestamp.Format("15:04:05"))
	}

	// 8. Warmup cache with multiple packages
	fmt.Println("\n9. Cache warmup:")
	// Create a simple package for warmup
	src2 := `package util
func Double(x int) int { return x * 2 }`
	
	file2, _ := parser.ParseFile(fset, "util.go", src2, 0)
	pkg2, _ := conf.Check("util", fset, []*ast.File{file2}, info)
	
	packages := []*core.Package{
		analysisPackage,
		{
			Fset:  fset,
			Files: []*ast.File{file2},
			Pkg:   pkg2,
			Path:  "util",
		},
	}
	
	err = cache.WarmupCache(packages[:1]) // Warmup with first package only
	if err != nil {
		fmt.Printf("   Warmup error: %v\n", err)
	} else {
		fmt.Printf("   Cache warmed up successfully\n")
	}

	// Show cache statistics
	fmt.Println("\n=== Cache Statistics ===")
	stats := cache.GetStatistics()
	fmt.Printf("Cached results: %d\n", stats["cached_results"])
	fmt.Printf("Registered analyzers: %d\n", stats["registered_analyzers"])
	fmt.Printf("Cache hits: %d\n", stats["cache_hits"])
	fmt.Printf("Cache misses: %d\n", stats["cache_misses"])
	if stats["cache_hits"].(int64)+stats["cache_misses"].(int64) > 0 {
		fmt.Printf("Hit rate: %.2f%%\n", stats["hit_rate"])
	}
	fmt.Printf("Total analyses: %d\n", stats["total_analyses"])
	fmt.Printf("Total diagnostics: %d\n", stats["total_diagnostics"])
	fmt.Printf("Total facts: %d\n", stats["total_facts"])
	fmt.Printf("Average duration: %v\n", stats["avg_duration"])

	// Clear cache
	fmt.Println("\n10. Clearing cache:")
	cache.Clear()
	stats = cache.GetStatistics()
	fmt.Printf("   Cached results after clear: %d\n", stats["cached_results"])
}

// Helper function to truncate strings for display
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}