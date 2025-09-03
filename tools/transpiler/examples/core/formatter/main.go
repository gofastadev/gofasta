// Example demonstrating Phase 1.1e: Batched formatting with parallel processing
package main

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"time"

	"github.com/healtronlabs/gofasta/tools/transpiler/core"
)

func main() {
	// Initialize batch formatter with custom configuration
	config := &core.BatchFormatterConfig{
		BatchSize:     5,
		MaxWorkers:    3,
		EnableMetrics: true,
		FormatOptions: &core.FormatOptions{
			TabWidth:    4,
			UseSpaces:   false,
			SortImports: true,
		},
	}
	
	formatter := core.NewBatchFormatter(config)
	
	fmt.Println("=== Batch Formatter Example ===")
	fmt.Printf("Configuration:\n")
	fmt.Printf("  Batch Size: %d\n", config.BatchSize)
	fmt.Printf("  Max Workers: %d\n", config.MaxWorkers)
	fmt.Printf("  Tab Width: %d\n", config.FormatOptions.TabWidth)
	fmt.Printf("  Sort Imports: %v\n", config.FormatOptions.SortImports)
	
	// Example source files with poor formatting
	sources := map[string]string{
		"main.go": `package main
import(
"fmt"
"os"
"time"
)
func main(){
fmt.Println("Hello, World!")
if len(os.Args)>1{
fmt.Printf("Args: %v\n",os.Args[1:])
}
fmt.Printf("Current time: %v\n",time.Now())
}`,
		"utils.go": `package main
import "fmt"
func helper(data string)string{
return fmt.Sprintf("Processed: %s",data)
}
type Config struct{
Name string
Enabled bool
}
func(c*Config)IsValid()bool{
return len(c.Name)>0
}`,
		"constants.go": `package main
const(
DefaultName="default"
MaxRetries=3
Timeout=30*time.Second
)
var(
GlobalConfig=&Config{Name:DefaultName,Enabled:true}
)`,
	}
	
	// Parse all source files
	fset := token.NewFileSet()
	files := make(map[string]*ast.File)
	
	for filename, source := range sources {
		file, err := parser.ParseFile(fset, filename, source, parser.ParseComments)
		if err != nil {
			log.Printf("Failed to parse %s: %v", filename, err)
			continue
		}
		files[filename] = file
	}
	
	fmt.Printf("\nFormatting %d files...\n", len(files))
	
	// Format files using batch formatter
	ctx := context.Background()
	start := time.Now()
	
	results, err := formatter.FormatFiles(ctx, files, fset)
	duration := time.Since(start)
	
	if err != nil {
		fmt.Printf("Formatting completed with some errors: %v\n", err)
	} else {
		fmt.Printf("✓ All files formatted successfully\n")
	}
	
	fmt.Printf("Formatting completed in %v\n", duration)
	
	// Show results
	fmt.Printf("\nFormatting Results:\n")
	for filename, result := range results {
		if result.Error != nil {
			fmt.Printf("  ✗ %s: %v\n", filename, result.Error)
		} else {
			fmt.Printf("  ✓ %s: %d bytes -> %d bytes (%v)\n", 
				filename, result.OriginalSize, result.FormattedSize, result.Duration)
		}
	}
	
	// Show performance statistics
	stats := formatter.GetStatistics()
	fmt.Printf("\nFormatter Statistics:\n")
	fmt.Printf("  Files formatted: %v\n", stats["files_formatted"])
	fmt.Printf("  Total duration: %v ms\n", stats["total_duration_ms"])
	fmt.Printf("  Average duration: %.2f ms\n", stats["avg_duration_ms"])
	fmt.Printf("  Files per second: %.0f\n", stats["files_per_second"])
	fmt.Printf("  Bytes processed: %v\n", stats["bytes_processed"])
	fmt.Printf("  Bytes per second: %.0f\n", stats["bytes_per_second"])
	fmt.Printf("  Success rate: %.1f%%\n", stats["success_rate"])
	
	// Demonstrate single file formatting
	fmt.Printf("\n=== Single File Formatting Example ===\n")
	
	singleSource := `package example
import("fmt";"time")
func messy(   ){
fmt.Printf("This is poorly formatted code: %v\n",time.Now())
}`
	
	file, err := parser.ParseFile(fset, "messy.go", singleSource, parser.ParseComments)
	if err != nil {
		log.Printf("Failed to parse messy file: %v", err)
		return
	}
	
	fmt.Printf("Formatting single file...\n")
	singleResult := formatter.FormatFile("messy.go", file, fset)
	
	if singleResult.Error != nil {
		fmt.Printf("✗ Single file formatting failed: %v\n", singleResult.Error)
	} else {
		fmt.Printf("✓ Single file formatted successfully\n")
		fmt.Printf("  Original size: %d bytes\n", len(singleSource))
		fmt.Printf("  Formatted size: %d bytes\n", singleResult.FormattedSize)
		fmt.Printf("  Duration: %v\n", singleResult.Duration)
		
		// Show formatted output
		fmt.Printf("\nFormatted output:\n")
		fmt.Printf("```go\n%s```\n", string(singleResult.Output))
	}
	
	// Demonstrate memory usage estimation
	fmt.Printf("\n=== Memory Usage Estimation ===\n")
	estimatedMB := formatter.EstimateMemoryUsage(files)
	fmt.Printf("Estimated memory usage: %d KB\n", estimatedMB)
	
	// Demonstrate formatting options
	fmt.Printf("\n=== Formatting Options Example ===\n")
	
	// Create formatter with spaces instead of tabs
	spaceFormatter := core.NewBatchFormatter(&core.BatchFormatterConfig{
		BatchSize:  2,
		MaxWorkers: 2,
		FormatOptions: &core.FormatOptions{
			TabWidth:    2,
			UseSpaces:   true,
			SortImports: true,
		},
	})
	
	// Format with different options
	spaceResult := spaceFormatter.FormatFile("example.go", file, fset)
	if spaceResult.Error == nil {
		fmt.Printf("✓ Formatted with spaces (2-space indentation)\n")
		fmt.Printf("Formatted size: %d bytes\n", spaceResult.FormattedSize)
	}
	
	// Get and modify formatting options
	currentOptions := formatter.GetFormattingOptions()
	fmt.Printf("\nCurrent formatting options:\n")
	fmt.Printf("  Tab Width: %d\n", currentOptions.TabWidth)
	fmt.Printf("  Use Spaces: %v\n", currentOptions.UseSpaces)
	fmt.Printf("  Sort Imports: %v\n", currentOptions.SortImports)
	
	// Reset formatter statistics
	fmt.Printf("\nResetting formatter statistics...\n")
	formatter.Reset()
	
	resetStats := formatter.GetStatistics()
	fmt.Printf("Statistics after reset:\n")
	fmt.Printf("  Files formatted: %v\n", resetStats["files_formatted"])
	fmt.Printf("  Total duration: %v ms\n", resetStats["total_duration_ms"])
	
	fmt.Printf("\n✓ Formatter example completed successfully!\n")
}