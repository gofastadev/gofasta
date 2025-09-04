// Example demonstrating all Phase 1.1 components working together in a complete pipeline
package main

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/healtronlabs/gofasta/transpiler/core"
)

func main() {
	fmt.Println("=== Complete Phase 1.1 Pipeline Example ===")

	// Setup: Create a sample Go project
	projectDir := setupSampleProject()
	defer os.RemoveAll(projectDir)

	// Initialize all Phase 1.1 components with optimized settings
	pipeline := &GoFastaPipeline{
		Parser: core.NewParallelParser(&core.ParserConfig{
			MaxWorkers:    4,
			ParseComments: true,
			AllowErrors:   true,
		}),
		ASTCache: core.NewASTCache(&core.ASTCacheConfig{
			MaxEntries:    200,
			TTL:           time.Hour,
			MaxMemoryMB:   200,
			EnableMetrics: true,
		}),
		TokenPool: core.NewTokenPool(&core.TokenPoolConfig{
			InitialSize:   10,
			MaxSize:       50,
			EnableMetrics: true,
		}),
		TypeChecker: core.NewIncrementalTypeChecker(&core.TypeCheckerConfig{
			EnableCaching:    true,
			CacheTTL:         45 * time.Minute,
			MaxCacheEntries:  100,
			ParallelChecking: true,
			MaxWorkers:       4,
			EnableMetrics:    true,
		}),
		Formatter: core.NewBatchFormatter(&core.BatchFormatterConfig{
			BatchSize:     8,
			MaxWorkers:    4,
			EnableMetrics: true,
			FormatOptions: &core.FormatOptions{
				TabWidth:    4,
				UseSpaces:   false,
				SortImports: true,
			},
		}),
		ImportCache: core.NewCachedImporter(&core.ImportCacheConfig{
			MaxEntries:    100,
			TTL:           2 * time.Hour,
			EnableMetrics: true,
			MaxMemoryMB:   150,
		}),
	}

	// Warm up components
	fmt.Printf("Warming up components...\n")
	pipeline.TokenPool.WarmUp(20)
	pipeline.ImportCache.WarmUp()

	// Process the project
	ctx := context.Background()
	results := pipeline.ProcessProject(ctx, projectDir)

	// Display comprehensive results
	pipeline.ShowResults(results)
}

// GoFastaPipeline represents a complete Phase 1.1 processing pipeline
type GoFastaPipeline struct {
	Parser      *core.ParallelParser
	ASTCache    *core.ASTCache
	TokenPool   *core.TokenPool
	TypeChecker *core.IncrementalTypeChecker
	Formatter   *core.BatchFormatter
	ImportCache *core.CachedImporter
}

// PipelineResults contains results from processing a project
type PipelineResults struct {
	ParseResults  []*core.ParseResult
	TypeResults   map[string]*core.TypeCheckResult
	FormatResults map[string]*core.FormatResult
	TotalDuration time.Duration
	Statistics    map[string]interface{}
}

// ProcessProject runs the complete pipeline on a Go project
func (p *GoFastaPipeline) ProcessProject(ctx context.Context, projectDir string) *PipelineResults {
	start := time.Now()

	fmt.Printf("\n=== Pipeline Processing: %s ===\n", projectDir)

	// Step 1: Parse all Go files
	fmt.Printf("Step 1: Parsing files...\n")
	parseStart := time.Now()

	parseResults, err := p.Parser.ParseDirectory(ctx, projectDir)
	if err != nil {
		log.Printf("Parse error: %v", err)
	}

	parseTime := time.Since(parseStart)
	fmt.Printf("✓ Parsed %d files in %v\n", len(parseResults), parseTime)

	// Step 2: Cache ASTs for future use
	fmt.Printf("\nStep 2: Caching ASTs...\n")
	cacheStart := time.Now()

	validFiles := make(map[string]*ast.File)
	var mainFset *token.FileSet

	for _, result := range parseResults {
		if result.Error == nil {
			// Get file modification time
			info, _ := os.Stat(result.FilePath)
			modTime := time.Now()
			if info != nil {
				modTime = info.ModTime()
			}

			// Cache the AST
			p.ASTCache.Put(result.FilePath, result.File, result.FileSet, modTime, result.Size)
			validFiles[result.FilePath] = result.File

			if mainFset == nil {
				mainFset = result.FileSet
			}
		}
	}

	cacheTime := time.Since(cacheStart)
	fmt.Printf("✓ Cached %d ASTs in %v\n", len(validFiles), cacheTime)

	// Step 3: Type check packages
	fmt.Printf("\nStep 3: Type checking...\n")
	typeStart := time.Now()

	// Group files by package (simplified - assumes all files are in main package)
	files := make([]*ast.File, 0, len(validFiles))
	for _, file := range validFiles {
		files = append(files, file)
	}

	typeResult, err := p.TypeChecker.CheckPackage(ctx, "main", files, mainFset)
	if err != nil {
		fmt.Printf("Type checking completed with errors: %v\n", err)
	} else {
		fmt.Printf("✓ Type checking succeeded\n")
	}

	typeTime := time.Since(typeStart)
	fmt.Printf("✓ Type checking completed in %v\n", typeTime)

	// Step 4: Format all files
	fmt.Printf("\nStep 4: Formatting files...\n")
	formatStart := time.Now()

	formatResults, err := p.Formatter.FormatFiles(ctx, validFiles, mainFset)
	if err != nil {
		fmt.Printf("Formatting completed with some errors: %v\n", err)
	}

	formatTime := time.Since(formatStart)
	fmt.Printf("✓ Formatted %d files in %v\n", len(formatResults), formatTime)

	totalDuration := time.Since(start)

	// Collect statistics from all components
	statistics := map[string]interface{}{
		"parse":        p.Parser.GetStatistics(),
		"ast_cache":    p.ASTCache.GetStatistics(),
		"token_pool":   p.TokenPool.GetStatistics(),
		"type_check":   p.TypeChecker.GetStatistics(),
		"formatter":    p.Formatter.GetStatistics(),
		"import_cache": p.ImportCache.GetStatistics(),
		"timings": map[string]interface{}{
			"parse_time":  parseTime,
			"cache_time":  cacheTime,
			"type_time":   typeTime,
			"format_time": formatTime,
			"total_time":  totalDuration,
		},
	}

	typeResults := map[string]*core.TypeCheckResult{
		"main": typeResult,
	}

	return &PipelineResults{
		ParseResults:  parseResults,
		TypeResults:   typeResults,
		FormatResults: formatResults,
		TotalDuration: totalDuration,
		Statistics:    statistics,
	}
}

// ShowResults displays comprehensive pipeline results
func (p *GoFastaPipeline) ShowResults(results *PipelineResults) {
	fmt.Printf("\n=== Pipeline Results Summary ===\n")

	timings := results.Statistics["timings"].(map[string]interface{})

	fmt.Printf("Performance Timings:\n")
	fmt.Printf("  Parse Time: %v\n", timings["parse_time"])
	fmt.Printf("  Cache Time: %v\n", timings["cache_time"])
	fmt.Printf("  Type Check Time: %v\n", timings["type_time"])
	fmt.Printf("  Format Time: %v\n", timings["format_time"])
	fmt.Printf("  Total Time: %v\n", timings["total_time"])

	// Component statistics
	parseStats := results.Statistics["parse"].(map[string]interface{})
	astStats := results.Statistics["ast_cache"].(map[string]interface{})
	poolStats := results.Statistics["token_pool"].(map[string]interface{})
	typeStats := results.Statistics["type_check"].(map[string]interface{})
	formatStats := results.Statistics["formatter"].(map[string]interface{})
	importStats := results.Statistics["import_cache"].(map[string]interface{})

	fmt.Printf("\nComponent Performance:\n")
	fmt.Printf("  Parser Rate: %.0f files/sec\n", parseStats["files_per_second"])
	fmt.Printf("  AST Cache Hit Ratio: %.1f%%\n", astStats["hit_ratio"])
	fmt.Printf("  Token Pool Reuse: %v/%v\n", poolStats["reused"], poolStats["created"])
	fmt.Printf("  Type Check Hit Ratio: %.1f%%\n", typeStats["hit_ratio"])
	fmt.Printf("  Formatter Rate: %.0f files/sec\n", formatStats["files_per_second"])
	fmt.Printf("  Import Cache Hit Ratio: %.1f%%\n", importStats["hit_ratio"])

	fmt.Printf("\nAccuracy Metrics:\n")
	fmt.Printf("  Parse Success Rate: %.1f%%\n", parseStats["success_rate"])
	fmt.Printf("  Format Success Rate: %.1f%%\n", formatStats["success_rate"])
	fmt.Printf("  Type Check Errors: %v\n", len(results.TypeResults))

	fmt.Printf("\nMemory Efficiency:\n")
	fmt.Printf("  AST Cache Memory: %v MB\n", astStats["memory_mb"])
	fmt.Printf("  Import Cache Memory: %v MB\n", importStats["memory_mb"])
	fmt.Printf("  Token Pool Size: %v\n", poolStats["pool_size"])

	// Show potential optimizations
	p.suggestOptimizations(results)
}

// suggestOptimizations analyzes results and suggests improvements
func (p *GoFastaPipeline) suggestOptimizations(results *PipelineResults) {
	fmt.Printf("\n=== Optimization Suggestions ===\n")

	astStats := results.Statistics["ast_cache"].(map[string]interface{})
	typeStats := results.Statistics["type_check"].(map[string]interface{})
	formatStats := results.Statistics["formatter"].(map[string]interface{})
	importStats := results.Statistics["import_cache"].(map[string]interface{})

	// Analyze cache hit ratios
	astHitRatio := astStats["hit_ratio"].(float64)
	typeHitRatio := typeStats["hit_ratio"].(float64)
	importHitRatio := importStats["hit_ratio"].(float64)

	if astHitRatio < 80.0 {
		fmt.Printf("• Consider increasing AST cache TTL or size (current hit ratio: %.1f%%)\n", astHitRatio)
	}

	if typeHitRatio < 70.0 {
		fmt.Printf("• Consider increasing type checker cache TTL (current hit ratio: %.1f%%)\n", typeHitRatio)
	}

	if importHitRatio < 90.0 {
		fmt.Printf("• Consider pre-warming import cache (current hit ratio: %.1f%%)\n", importHitRatio)
	}

	// Analyze performance
	formatRate := formatStats["files_per_second"].(float64)
	if formatRate < 1000 {
		fmt.Printf("• Consider increasing formatter batch size or workers (current rate: %.0f files/sec)\n", formatRate)
	}

	// Analyze memory usage
	astMemory := astStats["memory_mb"].(int64)
	importMemory := importStats["memory_mb"].(int64)

	if astMemory > 100 {
		fmt.Printf("• Consider reducing AST cache size to lower memory usage (%d MB)\n", astMemory)
	}

	if importMemory > 50 {
		fmt.Printf("• Consider reducing import cache size (%d MB)\n", importMemory)
	}

	fmt.Printf("✓ Analysis complete\n")
}

// setupSampleProject creates a sample Go project for demonstration
func setupSampleProject() string {
	tempDir := "/tmp/gofasta_example_" + fmt.Sprintf("%d", time.Now().UnixNano())
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}

	// Sample project files
	files := map[string]string{
		"main.go": `package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: program <name>")
	}
	
	name := os.Args[1]
	greeting := createGreeting(name)
	fmt.Println(greeting)
}`,
		"greetings.go": `package main

import (
	"fmt"
	"strings"
	"time"
)

func createGreeting(name string) string {
	cleanName := strings.TrimSpace(name)
	timeOfDay := getTimeOfDay()
	return fmt.Sprintf("%s, %s! Welcome to Gofasta.", timeOfDay, cleanName)
}

func getTimeOfDay() string {
	hour := time.Now().Hour()
	switch {
	case hour < 12:
		return "Good morning"
	case hour < 18:
		return "Good afternoon"
	default:
		return "Good evening"
	}
}`,
		"utils.go": `package main

import (
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	AppName    string
	Version    string
	Debug      bool
	LogLevel   int
}

func (c *Config) String() string {
	return fmt.Sprintf("Config{App: %s, Version: %s, Debug: %v}",
		c.AppName, c.Version, c.Debug)
}

func loadConfig() *Config {
	return &Config{
		AppName:  "Gofasta Example",
		Version:  "1.0.0",
		Debug:    true,
		LogLevel: 1,
	}
}

func getExecutablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}`,
		"data.go": `package main

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type User struct {
	ID        int       ` + "`json:\"id\"`" + `
	Name      string    ` + "`json:\"name\"`" + `
	Email     string    ` + "`json:\"email\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
}

func (u *User) Validate() error {
	if u.Name == "" {
		return fmt.Errorf("name is required")
	}
	if u.Email == "" {
		return fmt.Errorf("email is required")
	}
	return nil
}

func parseUser(data io.Reader) (*User, error) {
	var user User
	decoder := json.NewDecoder(data)
	if err := decoder.Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user: %w", err)
	}
	
	if err := user.Validate(); err != nil {
		return nil, fmt.Errorf("invalid user: %w", err)
	}
	
	return &user, nil
}

func (u *User) ToJSON() ([]byte, error) {
	return json.Marshal(u)
}`,
	}

	// Write all files
	for filename, content := range files {
		path := filepath.Join(tempDir, filename)
		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			log.Printf("Failed to write %s: %v", filename, err)
		}
	}

	fmt.Printf("Created sample project in: %s\n", tempDir)
	fmt.Printf("Project contains %d Go files\n", len(files))

	return tempDir
}
