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

	"github.com/healtronlabs/gofasta/transpiler/core"

	// Import fault tolerance decorators to ensure they're compiled into the binary
	_ "github.com/healtronlabs/gofasta/transpiler/decorators/faulttolerance/actor"
	_ "github.com/healtronlabs/gofasta/transpiler/decorators/faulttolerance/actorref"
	_ "github.com/healtronlabs/gofasta/transpiler/decorators/faulttolerance/actorsystem"
	_ "github.com/healtronlabs/gofasta/transpiler/decorators/faulttolerance/supervisor"
)

const (
	version = "v1.0.0"
	banner  = `
 ██████   ██████  ███████  █████  ███████ ████████  █████ 
██       ██    ██ ██      ██   ██ ██         ██    ██   ██
██   ███ ██    ██ █████   ███████ ███████    ██    ███████
██    ██ ██    ██ ██      ██   ██      ██    ██    ██   ██
 ██████   ██████  ██      ██   ██ ███████    ██    ██   ██
                                                          
Gofasta Enterprise Backend Framework - Transpiler %s
`
)

type Config struct {
	InputDir    string
	OutputDir   string
	Pattern     string
	Verbose     bool
	Force       bool
	DryRun      bool
	Watch       bool
	ShowVersion bool
	ShowHelp    bool
	Parallel    int
	CacheDir    string
	EnableCache bool
	LogLevel    string
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
		fmt.Printf("Gofasta Transpiler %s\n", version)
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
		fmt.Printf("🚀 Starting Gofasta transpilation\n")
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

	// Find Gofasta files
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

	// Use the global decorator registry
	cli.registry = core.GlobalRegistry

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
				InputFile: file,
				Success:   false,
				Error:     err,
				StartTime: time.Now(),
				Duration:  0,
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

	// Step 1: Extract decorators from source code FIRST (before Go parsing)
	extractionResult, err := cli.extractor.Extract(sourceCode)
	if err != nil {
		result.Error = fmt.Errorf("failed to extract decorators: %v", err)
		return result, err
	}

	decorators := extractionResult.Decorators

	// Step 2: Transform .gofa to .go by processing decorators and generating code
	transformedCode, err := cli.transformGofaToGo(string(sourceCode), decorators)
	if err != nil {
		result.Error = fmt.Errorf("failed to transform .gofa to .go: %v", err)
		return result, err
	}

	// Step 3: Parse the transformed Go code (not the original .gofa) - validation only
	_, err = parser.ParseFile(cli.fileSet, inputFile, transformedCode, parser.ParseComments)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse transformed Go code: %v", err)
		return result, err
	}

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

	// Context creation removed - using transformed code directly

	// Use the transformed code as the generated output
	generatedCode := transformedCode

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
	header := fmt.Sprintf("// Code generated by Gofasta %s; DO NOT EDIT.\n\n", version)

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

// transformGofaToGo transforms .gofa source code with decorators into functional Go code
func (cli *CLI) transformGofaToGo(sourceCode string, decorators []core.Decorator) (string, error) {
	// Start with the original source code
	result := sourceCode

	// Step 1: Remove decorator lines (they'll be replaced with generated code)
	result = cli.removeDecoratorLines(result, decorators)

	// Step 2: Generate code for each decorator and inject it
	for _, decorator := range decorators {
		generatedCode, err := cli.generateCodeForDecorator(decorator)
		if err != nil {
			return "", fmt.Errorf("failed to generate code for decorator %s: %v", decorator.Name, err)
		}

		// Step 3: Inject the generated code at the appropriate location
		result, err = cli.injectGeneratedCode(result, decorator, generatedCode)
		if err != nil {
			return "", fmt.Errorf("failed to inject code for decorator %s: %v", decorator.Name, err)
		}
	}

	// Step 4: Add necessary imports for generated code
	result = cli.addNecessaryImports(result, decorators)

	return result, nil
}

// removeDecoratorLines removes @Decorator lines from the source code
func (cli *CLI) removeDecoratorLines(sourceCode string, decorators []core.Decorator) string {
	// Use the decorator's raw text to remove the entire multi-line decorator block
	result := sourceCode

	for _, decorator := range decorators {
		if decorator.Raw != "" {
			// Remove the exact decorator text that was captured
			result = strings.Replace(result, decorator.Raw, "", 1)
		}
	}

	// Clean up any extra blank lines that might result from removal
	lines := strings.Split(result, "\n")
	var cleanedLines []string

	for i, line := range lines {
		// Skip multiple consecutive blank lines, but keep one
		if strings.TrimSpace(line) == "" {
			// Only add blank line if previous line wasn't blank
			if i == 0 || strings.TrimSpace(lines[i-1]) != "" {
				cleanedLines = append(cleanedLines, line)
			}
		} else {
			cleanedLines = append(cleanedLines, line)
		}
	}

	return strings.Join(cleanedLines, "\n")
}

// generateCodeForDecorator generates Go code for a specific decorator
func (cli *CLI) generateCodeForDecorator(decorator core.Decorator) (string, error) {
	switch decorator.Name {
	case "Supervisor":
		return cli.generateSupervisorCode(decorator)
	case "Actor":
		return cli.generateActorCode(decorator)
	case "ActorRef":
		return cli.generateActorRefCode(decorator)
	case "ActorSystem":
		return cli.generateActorSystemCode(decorator)
	case "SupervisionStrategy":
		return cli.generateSupervisorCode(decorator)
	default:
		// Return error for unknown decorators to prevent silent failures
		return "", fmt.Errorf("unknown decorator '@%s' - supported decorators: @Supervisor, @Actor, @ActorRef, @ActorSystem", decorator.Name)
	}
}

// generateSupervisorCode generates fault tolerance supervision code
func (cli *CLI) generateSupervisorCode(decorator core.Decorator) (string, error) {
	// Extract parameters from decorator
	strategy := "OneForOne"
	maxRetries := 3
	retryInterval := "1s"

	if len(decorator.Arguments) > 0 {
		strategy = strings.Trim(decorator.Arguments[0], "\"")
	}

	// Parse properties
	for key, value := range decorator.Properties {
		switch key {
		case "maxRetries":
			if v, ok := value.(int); ok {
				maxRetries = v
			}
		case "retryInterval":
			if v, ok := value.(string); ok {
				retryInterval = strings.Trim(v, "\"")
			}
		}
	}

	return fmt.Sprintf(`
// Generated type definitions for supervisor
type SupervisorState struct {
	strategy      string
	maxRetries    int
	retryInterval string
	children      map[string]*ChildState
}

type ChildState struct {
	name       string
	restarts   int
	lastRestart time.Time
}

// Generated supervisor code for %s strategy
var supervisorState = &SupervisorState{
	strategy: "%s",
	maxRetries: %d,
	retryInterval: "%s",
	children: make(map[string]*ChildState),
}

func initSupervisor() {
	// Initialize supervision with %s strategy
	log.Printf("Initializing supervisor with strategy: %s")
}
`, strategy, strategy, maxRetries, retryInterval, strategy, strategy), nil
}

// generateActorCode generates actor system code
func (cli *CLI) generateActorCode(decorator core.Decorator) (string, error) {
	mailboxSize := 1000
	poolSize := 10
	supervised := true

	// Parse properties
	for key, value := range decorator.Properties {
		switch key {
		case "mailboxSize":
			if v, ok := value.(int); ok {
				mailboxSize = v
			}
		case "poolSize":
			if v, ok := value.(int); ok {
				poolSize = v
			}
		case "supervised":
			if v, ok := value.(bool); ok {
				supervised = v
			}
		}
	}

	return fmt.Sprintf(`
// Generated actor system code
type ActorMailbox struct {
	messages chan interface{}
	size     int
}

var actorMailbox = &ActorMailbox{
	messages: make(chan interface{}, %d),
	size:     %d,
}

var actorPool = &ActorPool{
	workers:    %d,
	supervised: %t,
}

func initActor() {
	log.Printf("Initializing actor with mailbox size: %d, pool size: %d, supervised: %t")
}
`, mailboxSize, mailboxSize, poolSize, supervised, mailboxSize, poolSize, supervised), nil
}

// generateActorRefCode generates actor reference code
func (cli *CLI) generateActorRefCode(decorator core.Decorator) (string, error) {
	fastLookup := true
	cacheEnabled := true

	// Parse properties
	for key, value := range decorator.Properties {
		switch key {
		case "fastLookup":
			if v, ok := value.(bool); ok {
				fastLookup = v
			}
		case "cacheEnabled":
			if v, ok := value.(bool); ok {
				cacheEnabled = v
			}
		}
	}

	return fmt.Sprintf(`
// Generated actor reference code
type ActorRefSystem struct {
	lookup map[string]interface{}
	cache  bool
}

var actorRefSystem = &ActorRefSystem{
	lookup: make(map[string]interface{}),
	cache:  %t,
}

func initActorRef() {
	log.Printf("Initializing actor ref system with fast lookup: %t, cache: %t")
}
`, cacheEnabled, fastLookup, cacheEnabled), nil
}

// generateActorSystemCode generates actor system management code
func (cli *CLI) generateActorSystemCode(decorator core.Decorator) (string, error) {
	systemName := "DefaultSystem"
	parallelStartup := true
	maxActors := 10000
	clustering := false

	if len(decorator.Arguments) > 0 {
		systemName = strings.Trim(decorator.Arguments[0], "\"")
	}

	// Parse properties
	for key, value := range decorator.Properties {
		switch key {
		case "parallelStartup":
			if v, ok := value.(bool); ok {
				parallelStartup = v
			}
		case "maxActors":
			if v, ok := value.(int); ok {
				maxActors = v
			}
		case "clustering":
			if v, ok := value.(bool); ok {
				clustering = v
			}
		}
	}

	return fmt.Sprintf(`
// Generated actor system code for %s
type ActorSystemConfig struct {
	Name            string
	ParallelStartup bool
	MaxActors       int
	Clustering      bool
}

var actorSystem = &ActorSystemConfig{
	Name:            "%s",
	ParallelStartup: %t,
	MaxActors:       %d,
	Clustering:      %t,
}

func initActorSystem() {
	log.Printf("Initializing actor system: %s (parallel: %t, max actors: %d, clustering: %t)")
}
`, systemName, systemName, parallelStartup, maxActors, clustering, systemName, parallelStartup, maxActors, clustering), nil
}

// injectGeneratedCode injects generated code at the appropriate location in the source
func (cli *CLI) injectGeneratedCode(sourceCode string, decorator core.Decorator, generatedCode string) (string, error) {
	lines := strings.Split(sourceCode, "\n")

	// Find the function/struct that this decorator should be applied to
	decoratorTarget := cli.findDecoratorTarget(lines, decorator)
	if decoratorTarget == -1 {
		// If we can't find a specific target, add at the beginning after package/imports
		return cli.injectAtBeginning(sourceCode, generatedCode), nil
	}

	// Insert the generated code before the target
	result := make([]string, 0, len(lines)+strings.Count(generatedCode, "\n")+1)
	result = append(result, lines[:decoratorTarget]...)
	result = append(result, strings.Split(generatedCode, "\n")...)
	result = append(result, lines[decoratorTarget:]...)

	return strings.Join(result, "\n"), nil
}

// findDecoratorTarget finds the line where the decorator should be applied
func (cli *CLI) findDecoratorTarget(lines []string, decorator core.Decorator) int {
	// Look for function definitions, struct definitions, etc.
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "func ") ||
			strings.HasPrefix(trimmed, "type ") ||
			strings.HasPrefix(trimmed, "var ") {
			return i
		}
	}
	return -1
}

// injectAtBeginning injects code at the beginning after package and imports
func (cli *CLI) injectAtBeginning(sourceCode, generatedCode string) string {
	lines := strings.Split(sourceCode, "\n")

	// Find where to inject (after package and imports)
	injectIndex := 0
	inImportBlock := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "package ") {
			injectIndex = i + 1
		} else if strings.HasPrefix(trimmed, "import ") {
			inImportBlock = true
		} else if inImportBlock && (trimmed == "" || (!strings.HasPrefix(trimmed, "import") && !strings.Contains(trimmed, `"`))) {
			inImportBlock = false
			injectIndex = i
			break
		}
	}

	// Insert generated code
	result := make([]string, 0, len(lines)+strings.Count(generatedCode, "\n")+1)
	result = append(result, lines[:injectIndex]...)
	result = append(result, "")
	result = append(result, strings.Split(generatedCode, "\n")...)
	result = append(result, "")
	result = append(result, lines[injectIndex:]...)

	return strings.Join(result, "\n")
}

// addNecessaryImports adds required imports for the generated code
func (cli *CLI) addNecessaryImports(sourceCode string, decorators []core.Decorator) string {
	requiredImports := make(map[string]bool)

	// Determine what imports we need based on decorators
	for _, decorator := range decorators {
		switch decorator.Name {
		case "Supervisor", "Actor", "ActorRef", "ActorSystem", "SupervisionStrategy":
			requiredImports["log"] = true
			requiredImports["time"] = true
		}
	}

	if len(requiredImports) == 0 {
		return sourceCode
	}

	// Check if imports already exist
	lines := strings.Split(sourceCode, "\n")
	hasImport := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "import ") {
			hasImport = true
			break
		}
	}

	// Add missing imports
	if !hasImport {
		// Find package line and add imports after it
		for i, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "package ") {
				importLines := []string{"", "import ("}
				for imp := range requiredImports {
					importLines = append(importLines, fmt.Sprintf("\t\"%s\"", imp))
				}
				importLines = append(importLines, ")")

				result := make([]string, 0, len(lines)+len(importLines))
				result = append(result, lines[:i+1]...)
				result = append(result, importLines...)
				result = append(result, lines[i+1:]...)
				return strings.Join(result, "\n")
			}
		}
	} else {
		// Merge with existing imports
		lines := strings.Split(sourceCode, "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "import (") {
				// Find the closing parenthesis and collect existing imports
				existingImports := make(map[string]bool)
				for j := i + 1; j < len(lines); j++ {
					importLine := strings.TrimSpace(lines[j])
					if importLine == ")" {
						// Filter out imports that already exist
						newImports := make([]string, 0)
						for imp := range requiredImports {
							quotedImp := fmt.Sprintf("\"%s\"", imp)
							if !existingImports[quotedImp] {
								newImports = append(newImports, fmt.Sprintf("\t\"%s\"", imp))
							}
						}

						if len(newImports) > 0 {
							result := make([]string, 0, len(lines)+len(newImports))
							result = append(result, lines[:j]...)
							result = append(result, newImports...)
							result = append(result, lines[j:]...)
							return strings.Join(result, "\n")
						}
						return sourceCode // No new imports needed
					}
					if strings.Contains(importLine, "\"") {
						existingImports[importLine] = true
					}
				}
			}
		}
	}

	return sourceCode
}

