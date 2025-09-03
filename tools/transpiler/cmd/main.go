package main

import (
	"context"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/healtronlabs/gofasta/tools/transpiler/core"
)

const (
	version = "v1.0.0"
	banner  = `
 ██████   ██████  ███████  █████  ███████ ████████  █████ 
██       ██    ██ ██      ██   ██ ██         ██    ██   ██
██   ███ ██    ██ █████   ███████ ███████    ██    ███████
██    ██ ██    ██ ██      ██   ██      ██    ██    ██   ██
 ██████   ██████  ██      ██   ██ ███████    ██    ██   ██
                                                          
GoFasta Enterprise Backend Framework - Transpiler %s
`
)

type Config struct {
	InputDir     string
	OutputDir    string
	Pattern      string
	Verbose      bool
	Force        bool
	DryRun       bool
	Watch        bool
	ShowVersion  bool
	ShowHelp     bool
	Parallel     int
	CacheDir     string
	EnableCache  bool
	LogLevel     string
}

type CLI struct {
	config    *Config
	extractor *core.DecoratorExtractor
	generator *core.CodeGenerator
	registry  *core.DecoratorRegistry
	fileSet   *token.FileSet
}

func main() {
	cli := &CLI{
		config:  parseFlags(),
		fileSet: token.NewFileSet(),
	}

	if cli.config.ShowVersion {
		fmt.Printf("GoFasta Transpiler %s\n", version)
		return
	}

	if cli.config.ShowHelp {
		showHelp()
		return
	}

	fmt.Printf(banner, version)
	
	if err := cli.run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func parseFlags() *Config {
	return parseFlagsFromArgs(os.Args)
}

func parseFlagsFromArgs(args []string) *Config {
	config := &Config{}
	
	// Create a new FlagSet to avoid redefinition issues in tests
	fs := flag.NewFlagSet("gofasta", flag.ExitOnError)
	
	fs.StringVar(&config.InputDir, "input", ".", "Input directory containing .gofa files")
	fs.StringVar(&config.InputDir, "i", ".", "Input directory containing .gofa files (short)")
	fs.StringVar(&config.OutputDir, "output", ".", "Output directory for generated .go files")
	fs.StringVar(&config.OutputDir, "o", ".", "Output directory for generated .go files (short)")
	fs.StringVar(&config.Pattern, "pattern", "*.gofa", "File pattern to match")
	fs.StringVar(&config.Pattern, "p", "*.gofa", "File pattern to match (short)")
	fs.BoolVar(&config.Verbose, "verbose", false, "Verbose output")
	fs.BoolVar(&config.Verbose, "v", false, "Verbose output (short)")
	fs.BoolVar(&config.Force, "force", false, "Force overwrite existing files")
	fs.BoolVar(&config.Force, "f", false, "Force overwrite existing files (short)")
	fs.BoolVar(&config.DryRun, "dry-run", false, "Show what would be done without executing")
	fs.BoolVar(&config.DryRun, "n", false, "Show what would be done without executing (short)")
	fs.BoolVar(&config.Watch, "watch", false, "Watch for file changes and auto-transpile")
	fs.BoolVar(&config.Watch, "w", false, "Watch for file changes and auto-transpile (short)")
	fs.BoolVar(&config.ShowVersion, "version", false, "Show version information")
	fs.BoolVar(&config.ShowHelp, "help", false, "Show help information")
	fs.BoolVar(&config.ShowHelp, "h", false, "Show help information (short)")
	fs.IntVar(&config.Parallel, "parallel", 4, "Number of parallel workers")
	fs.StringVar(&config.CacheDir, "cache-dir", ".gofasta-cache", "Cache directory")
	fs.BoolVar(&config.EnableCache, "cache", true, "Enable caching")
	fs.StringVar(&config.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	
	if len(args) > 1 {
		fs.Parse(args[1:])
	}
	return config
}

func (cli *CLI) run() error {
	start := time.Now()
	
	if cli.config.Verbose {
		fmt.Printf("🚀 Starting GoFasta transpilation\n")
		fmt.Printf("   Input:  %s\n", cli.config.InputDir)
		fmt.Printf("   Output: %s\n", cli.config.OutputDir)
		fmt.Printf("   Pattern: %s\n", cli.config.Pattern)
		fmt.Printf("   Parallel: %d workers\n", cli.config.Parallel)
		fmt.Printf("   Cache: %v (dir: %s)\n", cli.config.EnableCache, cli.config.CacheDir)
		fmt.Println()
	}

	// Initialize core components
	if err := cli.initializeComponents(); err != nil {
		return fmt.Errorf("failed to initialize components: %v", err)
	}

	// Find GoFasta files
	files, err := cli.findGofastaFiles()
	if err != nil {
		return fmt.Errorf("failed to find .gofa files: %v", err)
	}

	if len(files) == 0 {
		fmt.Printf("⚠️  No .gofa files found in %s\n", cli.config.InputDir)
		return nil
	}

	if cli.config.Verbose {
		fmt.Printf("📁 Found %d .gofa files:\n", len(files))
		for _, file := range files {
			relPath, _ := filepath.Rel(cli.config.InputDir, file)
			fmt.Printf("   • %s\n", relPath)
		}
		fmt.Println()
	}

	// Process files
	if cli.config.DryRun {
		return cli.dryRun(files)
	}

	if cli.config.Watch {
		return cli.watchMode(files)
	}

	// Transpile files
	results, err := cli.transpileFiles(files)
	if err != nil {
		return err
	}

	// Show results
	cli.showResults(results, time.Since(start))
	return nil
}

func (cli *CLI) initializeComponents() error {
	// Initialize decorator extractor
	extractorConfig := &core.ExtractorConfig{
		EnableCache:        cli.config.EnableCache,
		MaxCacheEntries:    1000,
		ParallelExtraction: true,
		WorkerCount:        cli.config.Parallel,
		EnableMetrics:      cli.config.Verbose,
	}
	cli.extractor = core.NewDecoratorExtractor(extractorConfig)

	// Initialize code generator
	generatorConfig := &core.GeneratorConfig{
		EnableCache:        cli.config.EnableCache,
		ConcurrentGenerate: true,
		WorkerCount:        cli.config.Parallel,
		EnableMetrics:      cli.config.Verbose,
		FormatOutput:       true,
		AddHeaders:         true,
	}
	cli.generator = core.NewCodeGenerator(generatorConfig)

	// Initialize decorator registry
	registryConfig := &core.RegistryConfig{
		EnableHotReload:    false,
		ParallelLoading:    true,
		LoadWorkers:        cli.config.Parallel,
		MaxDecorators:      1000,
		EnableMetrics:      cli.config.Verbose,
	}
	cli.registry = core.NewDecoratorRegistry(registryConfig)

	return nil
}

func (cli *CLI) findGofastaFiles() ([]string, error) {
	var files []string
	
	err := filepath.WalkDir(cli.config.InputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() {
			return nil
		}
		
		if matched, _ := filepath.Match(cli.config.Pattern, filepath.Base(path)); matched {
			files = append(files, path)
		}
		
		return nil
	})
	
	return files, err
}

func (cli *CLI) transpileFiles(files []string) ([]*TranspileResult, error) {
	ctx := context.Background()
	results := make([]*TranspileResult, 0, len(files))
	
	for i, file := range files {
		if cli.config.Verbose {
			fmt.Printf("[%d/%d] 📝 Processing %s...\n", i+1, len(files), filepath.Base(file))
		}
		
		result, err := cli.transpileFile(ctx, file)
		if err != nil {
			result = &TranspileResult{
				InputFile:  file,
				Success:    false,
				Error:      err,
				StartTime:  time.Now(),
				Duration:   0,
			}
		}
		
		results = append(results, result)
		
		if cli.config.Verbose && result.Success {
			fmt.Printf("   ✅ Generated %s (%d bytes → %d bytes) in %v\n", 
				filepath.Base(result.OutputFile), result.InputSize, result.OutputSize, result.Duration)
		} else if cli.config.Verbose && !result.Success {
			fmt.Printf("   ❌ Failed: %v\n", result.Error)
		}
	}
	
	return results, nil
}

func (cli *CLI) transpileFile(ctx context.Context, inputFile string) (*TranspileResult, error) {
	start := time.Now()
	result := &TranspileResult{
		InputFile: inputFile,
		StartTime: start,
	}

	// Read input file
	sourceCode, err := os.ReadFile(inputFile)
	if err != nil {
		result.Error = fmt.Errorf("failed to read file: %v", err)
		return result, err
	}
	result.InputSize = int64(len(sourceCode))

	// Parse file
	file, err := parser.ParseFile(cli.fileSet, inputFile, string(sourceCode), parser.ParseComments)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse file: %v", err)
		return result, err
	}

	// Extract decorators from source code
	extractionResult, err := cli.extractor.Extract(sourceCode)
	if err != nil {
		result.Error = fmt.Errorf("failed to extract decorators: %v", err)
		return result, err
	}
	
	decorators := extractionResult.Decorators

	if cli.config.Verbose && len(decorators) > 0 {
		fmt.Printf("   🎯 Found %d decorators: ", len(decorators))
		decoratorTypes := make([]string, 0, len(decorators))
		for _, d := range decorators {
			decoratorTypes = append(decoratorTypes, d.Name)
		}
		fmt.Printf("%s\n", strings.Join(decoratorTypes, ", "))
	}

	// Generate code
	outputFile := cli.getOutputPath(inputFile)
	result.OutputFile = outputFile

	// Create generation context
	generationContext := &core.GenerationContext{
		PackageName: file.Name.Name,
		Decorators:  decorators,
		Metadata: map[string]interface{}{
			"sourceFile": inputFile,
			"outputFile": outputFile,
			"fileSet":    cli.fileSet,
		},
	}

	// Generate using a basic template (for now, just return formatted source)
	var generatedCode string
	generationResult, err := cli.generator.Generate("basic", generationContext)
	if err != nil {
		// Fallback: just return formatted original source with decorator comments
		generatedCode = cli.generateFallback(string(sourceCode), decorators)
	} else {
		generatedCode = generationResult.Code
	}

	// Check if output file exists and force flag
	if !cli.config.Force {
		if _, err := os.Stat(outputFile); err == nil {
			result.Error = fmt.Errorf("output file exists (use -force to overwrite): %s", outputFile)
			return result, result.Error
		}
	}

	// Write output file
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		result.Error = fmt.Errorf("failed to create output directory: %v", err)
		return result, err
	}

	if err := os.WriteFile(outputFile, []byte(generatedCode), 0644); err != nil {
		result.Error = fmt.Errorf("failed to write output file: %v", err)
		return result, err
	}

	result.OutputSize = int64(len(generatedCode))
	result.Duration = time.Since(start)
	result.Success = true
	result.GeneratedCode = generatedCode
	result.Decorators = decorators

	return result, nil
}

func (cli *CLI) getOutputPath(inputFile string) string {
	// Convert .gofa to .go
	base := filepath.Base(inputFile)
	if strings.HasSuffix(base, ".gofa") {
		base = strings.TrimSuffix(base, ".gofa") + ".go"
	}
	
	// Handle relative paths
	if cli.config.OutputDir == "." {
		return filepath.Join(filepath.Dir(inputFile), base)
	}
	
	// Handle absolute output directory
	relPath, _ := filepath.Rel(cli.config.InputDir, inputFile)
	relDir := filepath.Dir(relPath)
	if relDir == "." {
		return filepath.Join(cli.config.OutputDir, base)
	}
	
	return filepath.Join(cli.config.OutputDir, relDir, base)
}

func (cli *CLI) dryRun(files []string) error {
	fmt.Printf("🔍 Dry run mode - showing what would be done:\n\n")
	
	for _, file := range files {
		outputFile := cli.getOutputPath(file)
		fmt.Printf("📝 %s → %s\n", file, outputFile)
	}
	
	fmt.Printf("\n✨ Would process %d files\n", len(files))
	return nil
}

func (cli *CLI) watchMode(files []string) error {
	fmt.Printf("👁️  Watch mode not implemented yet\n")
	return fmt.Errorf("watch mode not implemented")
}

func (cli *CLI) showResults(results []*TranspileResult, totalDuration time.Duration) {
	successCount := 0
	errorCount := 0
	totalInputSize := int64(0)
	totalOutputSize := int64(0)

	fmt.Printf("\n📊 Transpilation Results:\n")
	fmt.Printf("═══════════════════════════════════════\n")

	for _, result := range results {
		if result.Success {
			successCount++
			totalInputSize += result.InputSize
			totalOutputSize += result.OutputSize
		} else {
			errorCount++
			fmt.Printf("❌ %s: %v\n", filepath.Base(result.InputFile), result.Error)
		}
	}

	fmt.Printf("\n📈 Summary:\n")
	fmt.Printf("   ✅ Successful: %d files\n", successCount)
	if errorCount > 0 {
		fmt.Printf("   ❌ Failed: %d files\n", errorCount)
	}
	fmt.Printf("   📦 Input size: %d bytes\n", totalInputSize)
	fmt.Printf("   📦 Output size: %d bytes\n", totalOutputSize)
	fmt.Printf("   ⚡ Total duration: %v\n", totalDuration)
	
	if successCount > 0 {
		avgDuration := totalDuration / time.Duration(successCount)
		fmt.Printf("   📊 Average per file: %v\n", avgDuration)
		
		if totalDuration > 0 {
			filesPerSecond := float64(successCount) / totalDuration.Seconds()
			fmt.Printf("   🚀 Files per second: %.1f\n", filesPerSecond)
		}
	}

	if errorCount == 0 {
		fmt.Printf("\n🎉 All files transpiled successfully!\n")
	} else {
		fmt.Printf("\n⚠️  %d files had errors\n", errorCount)
	}
}

func showHelp() {
	fmt.Printf(banner, version)
	fmt.Printf(`
Usage: gofasta [options]

OPTIONS:
  -input, -i <dir>        Input directory containing .gofa files (default: ".")
  -output, -o <dir>       Output directory for generated .go files (default: ".")  
  -pattern, -p <pattern>  File pattern to match (default: "*.gofa")
  -verbose, -v            Verbose output
  -force, -f              Force overwrite existing files
  -dry-run, -n            Show what would be done without executing
  -watch, -w              Watch for file changes and auto-transpile
  -parallel <n>           Number of parallel workers (default: 4)
  -cache                  Enable caching (default: true)
  -cache-dir <dir>        Cache directory (default: ".gofasta-cache")
  -log-level <level>      Log level: debug, info, warn, error (default: "info")
  -version                Show version information
  -help, -h               Show this help

EXAMPLES:
  gofasta                           # Transpile .gofa files in current directory
  gofasta -input ./src -output ./dist -verbose
  gofasta -pattern "controller*.gofa" -force
  gofasta -dry-run -verbose         # See what would be done
  gofasta -watch -verbose           # Watch mode (auto-transpile)

For more information, visit: https://github.com/healtronlabs/gofasta
`)
}

type TranspileResult struct {
	InputFile     string
	OutputFile    string
	Success       bool
	Error         error
	StartTime     time.Time
	Duration      time.Duration
	InputSize     int64
	OutputSize    int64
	GeneratedCode string
	Decorators    []core.Decorator
}

// generateFallback creates basic Go code when template generation fails
func (cli *CLI) generateFallback(sourceCode string, decorators []core.Decorator) string {
	// Add header comment
	header := fmt.Sprintf("// Code generated by GoFasta %s; DO NOT EDIT.\n\n", version)
	
	// Add decorator information as comments
	if len(decorators) > 0 {
		header += "// Detected decorators:\n"
		for _, decorator := range decorators {
			header += fmt.Sprintf("// - @%s\n", decorator.Name)
		}
		header += "\n"
	}
	
	return header + sourceCode
}