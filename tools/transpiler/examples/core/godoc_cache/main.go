// Example demonstrating go/doc with documentation extraction cache (Phase 1.2i)
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/healtronlabs/gofasta/tools/transpiler/core"
)

func main() {
	// Create godoc cache with custom configuration
	config := &core.GoDocCacheConfig{
		MaxCacheEntries:      500,
		EnableMetrics:        true,
		AllMode:              false, // Don't include unexported
		AllMethods:           true,  // Show all methods
		GenerateHTML:         true,
		GenerateJSON:         true,
		ConcurrentExtraction: true,
		ExtractionWorkers:    4,
		PrecomputeFormats:    true,
	}
	cache := core.NewGoDocCache(config)

	fmt.Println("=== GoDoc Cache Example ===\n")

	// 1. Extract documentation from source code
	fmt.Println("1. Extracting package documentation:")
	
	// Create sample source code
	src := map[string][]byte{
		"math_utils.go": []byte(`// Package mathutils provides mathematical utility functions.
// It includes functions for basic arithmetic operations and
// statistical calculations.
package mathutils

import (
	"math"
)

// Constants for mathematical operations.
const (
	// Pi represents the mathematical constant π.
	Pi = 3.14159265359
	
	// E represents Euler's number.
	E = 2.71828182846
)

// DefaultPrecision is the default decimal precision.
var DefaultPrecision = 2

// Calculator performs mathematical calculations.
type Calculator struct {
	// Precision defines decimal precision for calculations.
	Precision int ` + "`json:\"precision\"`" + `
	
	// History stores calculation history.
	History []string ` + "`json:\"history,omitempty\"`" + `
}

// NewCalculator creates a new calculator with default precision.
func NewCalculator() *Calculator {
	return &Calculator{
		Precision: DefaultPrecision,
		History:   make([]string, 0),
	}
}

// Add performs addition of two numbers.
func (c *Calculator) Add(a, b float64) float64 {
	result := a + b
	c.History = append(c.History, fmt.Sprintf("%.2f + %.2f = %.2f", a, b, result))
	return result
}

// Multiply performs multiplication of two numbers.
func (c *Calculator) Multiply(a, b float64) float64 {
	result := a * b
	c.History = append(c.History, fmt.Sprintf("%.2f * %.2f = %.2f", a, b, result))
	return result
}

// GetHistory returns the calculation history.
func (c *Calculator) GetHistory() []string {
	return c.History
}

// Statistics provides statistical calculations.
type Statistics struct {
	data []float64
}

// NewStatistics creates a new statistics calculator.
func NewStatistics(data []float64) *Statistics {
	return &Statistics{data: data}
}

// Mean calculates the arithmetic mean.
func (s *Statistics) Mean() float64 {
	if len(s.data) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range s.data {
		sum += v
	}
	return sum / float64(len(s.data))
}

// Max returns the maximum value.
func (s *Statistics) Max() float64 {
	if len(s.data) == 0 {
		return 0
	}
	max := s.data[0]
	for _, v := range s.data {
		if v > max {
			max = v
		}
	}
	return max
}

// Sqrt calculates the square root of a number.
// It uses the math package for calculation.
func Sqrt(x float64) float64 {
	return math.Sqrt(x)
}

// Power calculates x raised to the power of y.
func Power(x, y float64) float64 {
	return math.Pow(x, y)
}

// Round rounds a number to specified decimal places.
func Round(val float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return math.Round(val*shift) / shift
}`),
		
		"math_utils_test.go": []byte(`package mathutils_test

import (
	"fmt"
	"mathutils"
)

// Example demonstrates basic calculator usage.
func Example() {
	calc := mathutils.NewCalculator()
	result := calc.Add(10, 20)
	fmt.Printf("Result: %.2f\n", result)
	// Output: Result: 30.00
}

// ExampleCalculator_Add demonstrates the Add method.
func ExampleCalculator_Add() {
	calc := mathutils.NewCalculator()
	sum := calc.Add(5.5, 4.5)
	fmt.Printf("Sum: %.2f\n", sum)
	// Output: Sum: 10.00
}

// ExampleSqrt demonstrates the Sqrt function.
func ExampleSqrt() {
	result := mathutils.Sqrt(16)
	fmt.Printf("Square root of 16: %.0f\n", result)
	// Output: Square root of 16: 4
}`),
	}
	
	pkg, err := cache.ExtractPackageDoc("mathutils", src)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("   Package: %s\n", pkg.Name)
	fmt.Printf("   Import Path: %s\n", pkg.ImportPath)
	fmt.Printf("   Constants: %d\n", len(pkg.Consts))
	fmt.Printf("   Variables: %d\n", len(pkg.Vars))
	fmt.Printf("   Functions: %d\n", len(pkg.Funcs))
	fmt.Printf("   Types: %d\n", len(pkg.Types))

	// 2. Get type documentation
	fmt.Println("\n2. Type documentation:")
	typeDoc := cache.GetTypeDoc(pkg, "Calculator")
	if typeDoc != nil {
		fmt.Printf("   Type: %s\n", typeDoc.Name)
		fmt.Printf("   Methods: %d\n", len(typeDoc.Methods))
		for _, method := range typeDoc.Methods {
			fmt.Printf("     - %s: %s\n", method.Name, truncate(method.Doc, 50))
		}
		fmt.Printf("   Fields: %d\n", len(typeDoc.Fields))
		for _, field := range typeDoc.Fields {
			fmt.Printf("     - %s (%s): %s\n", field.Name, field.Type, truncate(field.Doc, 40))
		}
	}

	// 3. Get function documentation
	fmt.Println("\n3. Function documentation:")
	functions := []string{"Sqrt", "Power", "Round"}
	for _, fname := range functions {
		funcDoc := cache.GetFuncDoc(pkg, fname)
		if funcDoc != nil {
			fmt.Printf("   %s: %s\n", funcDoc.Name, truncate(funcDoc.Doc, 60))
		}
	}

	// 4. Get examples
	fmt.Println("\n4. Examples:")
	examples := cache.GetExamples(pkg, "")
	fmt.Printf("   Package examples: %d\n", len(examples))
	
	calcExamples := cache.GetExamples(pkg, "Calculator_Add")
	fmt.Printf("   Calculator.Add examples: %d\n", len(calcExamples))

	// 5. Export as HTML
	fmt.Println("\n5. Exporting as HTML:")
	var htmlBuf bytes.Buffer
	if err := cache.ExportHTML(pkg, &htmlBuf); err != nil {
		log.Printf("Error exporting HTML: %v", err)
	} else {
		fmt.Printf("   HTML size: %d bytes\n", htmlBuf.Len())
		// Save to file for viewing
		htmlFile := "mathutils_doc.html"
		if err := ioutil.WriteFile(htmlFile, htmlBuf.Bytes(), 0644); err == nil {
			fmt.Printf("   Saved to: %s\n", htmlFile)
			defer os.Remove(htmlFile)
		}
	}

	// 6. Export as JSON
	fmt.Println("\n6. Exporting as JSON:")
	var jsonBuf bytes.Buffer
	if err := cache.ExportJSON(pkg, &jsonBuf); err != nil {
		log.Printf("Error exporting JSON: %v", err)
	} else {
		fmt.Printf("   JSON size: %d bytes\n", jsonBuf.Len())
		// Show a snippet
		jsonStr := jsonBuf.String()
		if len(jsonStr) > 200 {
			fmt.Printf("   JSON snippet: %s...\n", jsonStr[:200])
		}
	}

	// 7. Export as Markdown
	fmt.Println("\n7. Exporting as Markdown:")
	var mdBuf bytes.Buffer
	if err := cache.ExportMarkdown(pkg, &mdBuf); err != nil {
		log.Printf("Error exporting Markdown: %v", err)
	} else {
		fmt.Printf("   Markdown size: %d bytes\n", mdBuf.Len())
		// Save to file
		mdFile := "mathutils_doc.md"
		if err := ioutil.WriteFile(mdFile, mdBuf.Bytes(), 0644); err == nil {
			fmt.Printf("   Saved to: %s\n", mdFile)
			defer os.Remove(mdFile)
		}
	}

	// 8. Search documentation
	fmt.Println("\n8. Searching documentation:")
	searchTerms := []string{"calc", "stat", "math", "sqrt"}
	for _, term := range searchTerms {
		results := cache.SearchDocumentation(term)
		fmt.Printf("   Search '%s': %d results\n", term, len(results))
		for i, result := range results {
			if i >= 3 {
				break // Show only first 3
			}
			fmt.Printf("     - %s (%s): %s\n", result.Name, result.Type, truncate(result.Doc, 40))
		}
	}

	// 9. Batch extraction
	fmt.Println("\n9. Batch extraction:")
	
	// Create multiple packages
	packages := map[string]map[string][]byte{
		"pkg1": {
			"main.go": []byte(`// Package pkg1 provides first package.
package pkg1

// Func1 is a function in pkg1.
func Func1() string { return "pkg1" }`),
		},
		"pkg2": {
			"main.go": []byte(`// Package pkg2 provides second package.
package pkg2

// Func2 is a function in pkg2.
func Func2() string { return "pkg2" }`),
		},
		"pkg3": {
			"main.go": []byte(`// Package pkg3 provides third package.
package pkg3

// Func3 is a function in pkg3.
func Func3() string { return "pkg3" }`),
		},
	}
	
	results := cache.BatchExtract(packages)
	fmt.Printf("   Extracted %d packages:\n", len(results))
	for name, pkg := range results {
		fmt.Printf("     - %s: %s\n", name, truncate(pkg.Doc, 40))
	}

	// 10. Extract from directory
	fmt.Println("\n10. Extracting from directory:")
	
	// Create a temporary directory with Go files
	tmpDir, err := ioutil.TempDir("", "godoc_example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Create Go files
	files := map[string]string{
		"server.go": `// Package server provides HTTP server functionality.
package server

import "net/http"

// Server represents an HTTP server.
type Server struct {
	// Port is the server port.
	Port int
	// Handler is the HTTP handler.
	Handler http.Handler
}

// Start starts the server.
func (s *Server) Start() error {
	return nil
}

// Stop stops the server.
func (s *Server) Stop() error {
	return nil
}`,
		"middleware.go": `package server

// Middleware represents HTTP middleware.
type Middleware func(http.Handler) http.Handler

// Logger is a logging middleware.
func Logger() Middleware {
	return func(next http.Handler) http.Handler {
		return next
	}
}`,
	}
	
	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := ioutil.WriteFile(path, []byte(content), 0644); err != nil {
			log.Fatal(err)
		}
	}
	
	dirPkg, err := cache.ExtractPackageDocFromDir(tmpDir)
	if err != nil {
		log.Printf("Error extracting from dir: %v", err)
	} else {
		fmt.Printf("   Package: %s\n", dirPkg.Name)
		fmt.Printf("   Types: %d\n", len(dirPkg.Types))
		fmt.Printf("   Functions: %d\n", len(dirPkg.Funcs))
	}

	// 11. Cache warmup
	fmt.Println("\n11. Cache warmup:")
	warmupPackages := map[string]map[string][]byte{
		"warmup1": {
			"main.go": []byte(`package warmup1
func W1() {}`),
		},
		"warmup2": {
			"main.go": []byte(`package warmup2
func W2() {}`),
		},
	}
	cache.WarmupCache(warmupPackages)
	fmt.Println("   Cache warmed up with packages")

	// Show cache statistics
	fmt.Println("\n=== Cache Statistics ===")
	stats := cache.GetStatistics()
	fmt.Printf("Package cache size: %d\n", stats["package_cache_size"])
	fmt.Printf("Type cache size: %d\n", stats["type_cache_size"])
	fmt.Printf("Function cache size: %d\n", stats["func_cache_size"])
	fmt.Printf("Example cache size: %d\n", stats["example_cache_size"])
	fmt.Printf("Cache hits: %d\n", stats["cache_hits"])
	fmt.Printf("Cache misses: %d\n", stats["cache_misses"])
	if stats["cache_hits"].(int64)+stats["cache_misses"].(int64) > 0 {
		fmt.Printf("Hit rate: %.2f%%\n", stats["hit_rate"])
	}
	fmt.Printf("Total extractions: %d\n", stats["total_extractions"])
	fmt.Printf("Package count: %d\n", stats["package_count"])
	fmt.Printf("Type count: %d\n", stats["type_count"])
	fmt.Printf("Function count: %d\n", stats["func_count"])

	// Clear cache
	fmt.Println("\n12. Clearing cache:")
	cache.Clear()
	stats = cache.GetStatistics()
	fmt.Printf("   Package cache size after clear: %d\n", stats["package_cache_size"])
}

// Helper function to truncate strings for display
func truncate(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}