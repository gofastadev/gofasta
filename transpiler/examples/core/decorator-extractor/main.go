// Package main demonstrates usage of the decorator extraction engine.
// This example shows how to extract decorators from Go source code.
package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/healtronlabs/gofasta/transpiler/core"
)

func main() {
	fmt.Println("=== Decorator Extraction Engine Example ===")
	fmt.Println()

	// Example 1: Basic extraction with default config
	basicExample()

	// Example 2: Custom patterns
	customPatternExample()

	// Example 3: Parallel extraction
	parallelExample()

	// Example 4: Context capture
	contextExample()

	// Example 5: Performance comparison
	performanceExample()
}

func basicExample() {
	fmt.Println("Example 1: Basic Decorator Extraction")
	fmt.Println(strings.Repeat("-", 40))

	// Create extractor with default config
	extractor := core.NewDecoratorExtractor(nil)

	// Sample Go code with decorators
	source := []byte(`
package api

import "net/http"

// UserController handles user-related endpoints
type UserController struct{}

// GetUsers returns all users
@GET("/api/users")
@Auth
@RateLimit(100)
func (c *UserController) GetUsers(w http.ResponseWriter, r *http.Request) {
	// Implementation
}

// CreateUser creates a new user
@POST("/api/users")
@Auth
@Validate
func (c *UserController) CreateUser(w http.ResponseWriter, r *http.Request) {
	// Implementation
}

// User represents a user entity
@Entity("users")
type User struct {
	@Required
	@MinLength(3)
	Name string ` + "`json:\"name\"`" + `

	@Email
	@Required
	Email string ` + "`json:\"email\"`" + `

	@Pattern("^[0-9]{10}$")
	Phone string ` + "`json:\"phone\"`" + `
}
`)

	// Extract decorators
	result, err := extractor.Extract(source)
	if err != nil {
		log.Fatal(err)
	}

	// Display results
	fmt.Printf("Found %d decorators:\n", len(result.Decorators))
	for _, dec := range result.Decorators {
		fmt.Printf("  - %s (type: %s) at line %d\n", dec.Name, dec.Type, dec.Line)
		if len(dec.Arguments) > 0 {
			fmt.Printf("    Arguments: %v\n", dec.Arguments)
		}
	}

	fmt.Printf("\nExtraction took: %v\n", result.Duration)
	fmt.Println()
}

func customPatternExample() {
	fmt.Println("Example 2: Custom Pattern Extraction")
	fmt.Println(strings.Repeat("-", 40))

	// Create extractor with custom patterns
	config := &core.ExtractorConfig{
		CustomPatterns: map[string]string{
			"deprecated": `@Deprecated\((.*?)\)`,
			"since":      `@Since\((.*?)\)`,
			"todo":       `@TODO\((.*?)\)`,
		},
	}
	extractor := core.NewDecoratorExtractor(config)

	// Add more custom patterns
	extractor.AddPattern("feature", `@Feature\((.*?)\)`, 10)

	source := []byte(`
package legacy

// OldAPI is deprecated
@Deprecated("Use NewAPI instead")
@Since("v1.0")
func OldAPI() {}

// NewAPI is the replacement
@Since("v2.0")
@Feature("improved-performance")
func NewAPI() {}

// WorkInProgress needs completion
@TODO("implement caching")
func WorkInProgress() {}
`)

	result, err := extractor.Extract(source)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d custom decorators:\n", len(result.Decorators))
	for _, dec := range result.Decorators {
		fmt.Printf("  - %s: %s\n", dec.Name, dec.Raw)
	}
	fmt.Println()
}

func parallelExample() {
	fmt.Println("Example 3: Parallel Extraction")
	fmt.Println(strings.Repeat("-", 40))

	config := &core.ExtractorConfig{
		ParallelExtraction: true,
		WorkerCount:        4,
	}
	extractor := core.NewDecoratorExtractor(config)

	// Multiple source files
	sources := map[string][]byte{
		"controller.go": []byte(`
@RestController("/api")
type Controller struct{}

@GET("/users")
func GetUsers() {}
`),
		"model.go": []byte(`
@Entity("users")
type User struct {
	@Required
	Name string
}
`),
		"service.go": []byte(`
@Service
type UserService struct{}

@Transaction
func (s *UserService) CreateUser() {}
`),
	}

	start := time.Now()
	results, err := extractor.ExtractParallel(sources)
	if err != nil {
		log.Fatal(err)
	}
	duration := time.Since(start)

	fmt.Printf("Processed %d files in %v:\n", len(results), duration)
	for file, result := range results {
		fmt.Printf("  %s: %d decorators\n", file, len(result.Decorators))
	}
	fmt.Println()
}

func contextExample() {
	fmt.Println("Example 4: Context Capture")
	fmt.Println(strings.Repeat("-", 40))

	config := &core.ExtractorConfig{
		CaptureContext: true,
		ContextLines:   2,
	}
	extractor := core.NewDecoratorExtractor(config)

	source := []byte(`
package api

// Important: This endpoint requires authentication
// It returns paginated results
@GET("/api/products")
@Auth
@Paginated(50)
func GetProducts() {
	// Implementation here
}
`)

	result, err := extractor.Extract(source)
	if err != nil {
		log.Fatal(err)
	}

	for _, dec := range result.Decorators {
		fmt.Printf("Decorator: %s\n", dec.Name)
		if len(dec.Context) > 0 {
			fmt.Println("Context:")
			for _, line := range dec.Context {
				fmt.Printf("  | %s\n", line)
			}
		}
		fmt.Println()
	}
}

func performanceExample() {
	fmt.Println("Example 5: Performance Comparison")
	fmt.Println(strings.Repeat("-", 40))

	// Large source with many decorators
	var builder strings.Builder
	for i := 0; i < 100; i++ {
		builder.WriteString(fmt.Sprintf(`
@GET("/endpoint%d")
@Auth
@Cache(%d)
func Handler%d() {}
`, i, i*60, i))
	}
	source := []byte(builder.String())

	// Test with byte scanning
	config1 := &core.ExtractorConfig{
		UseByteScanning: true,
		EnableCache:     false,
	}
	extractor1 := core.NewDecoratorExtractor(config1)

	start := time.Now()
	result1, _ := extractor1.Extract(source)
	duration1 := time.Since(start)

	// Test with regex
	config2 := &core.ExtractorConfig{
		UseByteScanning: false,
		EnableCache:     false,
	}
	extractor2 := core.NewDecoratorExtractor(config2)

	start = time.Now()
	result2, _ := extractor2.Extract(source)
	duration2 := time.Since(start)

	// Test with caching
	config3 := &core.ExtractorConfig{
		EnableCache: true,
	}
	extractor3 := core.NewDecoratorExtractor(config3)

	// First extraction (cache miss)
	extractor3.Extract(source)
	// Second extraction (cache hit)
	start = time.Now()
	extractor3.Extract(source)
	duration3 := time.Since(start)

	fmt.Printf("Performance Comparison:\n")
	fmt.Printf("  Byte Scanning: %v (%d decorators)\n", duration1, len(result1.Decorators))
	fmt.Printf("  Regex:         %v (%d decorators)\n", duration2, len(result2.Decorators))
	fmt.Printf("  Cached:        %v\n", duration3)

	// Display statistics
	stats := extractor3.GetStatistics()
	fmt.Printf("\nCache Statistics:\n")
	fmt.Printf("  Hits: %d\n", stats["cache_hits"])
	fmt.Printf("  Misses: %d\n", stats["cache_misses"])
	fmt.Printf("  Hit Rate: %.2f%%\n", stats["cache_hit_rate"])
}
