package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CLI represents the command line interface for the transpiler
type CLI struct {
	version string
}

// NewCLI creates a new CLI instance
func NewCLI(version string) *CLI {
	return &CLI{
		version: version,
	}
}

// Command represents a CLI command
type Command struct {
	Name        string
	Description string
	Usage       string
	Handler     func(args []string) error
}

// TranspileFunc is the function signature for transpile operations
type TranspileFunc func(inputPath string, inputContent string) (string, error)

// BatchTranspiler interface for batch operations
type BatchTranspiler interface {
	TranspileProject(inputDir string) error
}

// ParallelTranspiler interface for parallel operations
type ParallelTranspiler interface {
	FindGofaFiles(inputDir string) ([]string, error)
	GetOutputPath(inputDir, gofaFile string) string
}

// WatchMode interface for watch operations
type WatchMode interface {
	Start() error
	Stop()
}

// TranspileOptions represents options for transpilation
type TranspileOptions struct {
	MaxWorkers     int
	OutputDir      string
	FileExtension  string
	PreserveStruct bool
	Verbose        bool
}

// Dependencies that need to be injected
type Dependencies struct {
	TranspileFile         TranspileFunc
	NewBatchTranspiler    func(opts TranspileOptions) BatchTranspiler
	NewParallelTranspiler func(opts TranspileOptions) ParallelTranspiler
	NewWatchMode          func(opts TranspileOptions, inputDir string, debounce time.Duration) WatchMode
}

// cliWithDeps holds the CLI instance and its injected dependencies
type cliWithDeps struct {
	*CLI
	deps Dependencies
}

// Run runs the CLI with given arguments
func (cli *CLI) Run(args []string, deps Dependencies) error {
	cliWithDeps := &cliWithDeps{CLI: cli, deps: deps}
	
	if len(args) < 2 {
		cliWithDeps.printUsage()
		return nil
	}

	command := args[1]
	commandArgs := args[2:]

	switch command {
	case "transpile", "t":
		return cliWithDeps.transpileCommand(commandArgs)
	case "watch", "w":
		return cliWithDeps.watchCommand(commandArgs)
	case "version", "v":
		return cliWithDeps.versionCommand(commandArgs)
	case "help", "h":
		return cliWithDeps.helpCommand(commandArgs)
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		cliWithDeps.printUsage()
		return fmt.Errorf("unknown command: %s", command)
	}
}

// transpileCommand handles the transpile command
func (cli *cliWithDeps) transpileCommand(args []string) error {
	fs := flag.NewFlagSet("transpile", flag.ContinueOnError)

	var (
		inputDir       = fs.String("input", ".", "Input directory containing .gofa files")
		outputDir      = fs.String("output", "", "Output directory for .go files (default: same as input)")
		maxWorkers     = fs.Int("workers", 0, "Maximum number of worker goroutines (default: number of CPU cores)")
		preserveStruct = fs.Bool("preserve", true, "Preserve directory structure in output")
		verbose        = fs.Bool("verbose", false, "Enable verbose output")
		single         = fs.String("file", "", "Transpile a single .gofa file")
		dryRun         = fs.Bool("dry-run", false, "Show what would be transpiled without actually doing it")
		force          = fs.Bool("force", false, "Overwrite existing .go files")
	)

	fs.Usage = func() {
		fmt.Println("Usage: gofasta transpile [options]")
		fmt.Println("\nTranspile .gofa files to .go files")
		fmt.Println("\nOptions:")
		fs.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Println("  gofasta transpile")
		fmt.Println("  gofasta transpile -input src -output dist")
		fmt.Println("  gofasta transpile -file user.controller.gofa")
		fmt.Println("  gofasta transpile -input . -workers 8 -verbose")
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Set default output directory if not specified
	if *outputDir == "" {
		*outputDir = *inputDir
	}

	// Validate input
	if *single != "" {
		return cli.transpileSingleFile(*single, *outputDir, *verbose, *dryRun, *force)
	}

	// Validate input directory
	if _, err := os.Stat(*inputDir); os.IsNotExist(err) {
		return fmt.Errorf("input directory does not exist: %s", *inputDir)
	}

	// Create output directory if it doesn't exist
	if !*dryRun {
		if err := os.MkdirAll(*outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Check for existing files
	if !*force && !*dryRun {
		if err := cli.checkExistingFiles(*inputDir, *outputDir, *preserveStruct); err != nil {
			return err
		}
	}

	// Setup transpiler options
	opts := TranspileOptions{
		MaxWorkers:     *maxWorkers,
		OutputDir:      *outputDir,
		FileExtension:  ".go",
		PreserveStruct: *preserveStruct,
		Verbose:        *verbose,
	}

	if *dryRun {
		return cli.dryRunTranspile(*inputDir, opts)
	}

	// Create and run batch transpiler
	batchTranspiler := cli.deps.NewBatchTranspiler(opts)
	return batchTranspiler.TranspileProject(*inputDir)
}

// watchCommand handles the watch command
func (cli *cliWithDeps) watchCommand(args []string) error {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)

	var (
		inputDir       = fs.String("input", ".", "Input directory to watch for .gofa files")
		outputDir      = fs.String("output", "", "Output directory for .go files (default: same as input)")
		maxWorkers     = fs.Int("workers", 0, "Maximum number of worker goroutines")
		preserveStruct = fs.Bool("preserve", true, "Preserve directory structure in output")
		debounce       = fs.Duration("debounce", 500*time.Millisecond, "Debounce delay for file changes")
		verbose        = fs.Bool("verbose", false, "Enable verbose output")
	)

	fs.Usage = func() {
		fmt.Println("Usage: gofasta watch [options]")
		fmt.Println("\nWatch .gofa files and transpile automatically on changes")
		fmt.Println("\nOptions:")
		fs.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Println("  gofasta watch")
		fmt.Println("  gofasta watch -input src -output dist")
		fmt.Println("  gofasta watch -debounce 1s -verbose")
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Set default output directory if not specified
	if *outputDir == "" {
		*outputDir = *inputDir
	}

	// Validate input directory
	if _, err := os.Stat(*inputDir); os.IsNotExist(err) {
		return fmt.Errorf("input directory does not exist: %s", *inputDir)
	}

	// Setup transpiler options
	opts := TranspileOptions{
		MaxWorkers:     *maxWorkers,
		OutputDir:      *outputDir,
		FileExtension:  ".go",
		PreserveStruct: *preserveStruct,
		Verbose:        *verbose,
	}

	// Create and start watch mode
	watchMode := cli.deps.NewWatchMode(opts, *inputDir, *debounce)

	fmt.Println("Press Ctrl+C to stop watching...")

	// Handle interrupt signal
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start watching
	if err := watchMode.Start(); err != nil {
		return fmt.Errorf("failed to start watch mode: %w", err)
	}

	// Wait for interrupt
	<-ctx.Done()
	watchMode.Stop()

	return nil
}

// versionCommand handles the version command
func (cli *cliWithDeps) versionCommand(args []string) error {
	fmt.Printf("Gofasta Transpiler v%s\n", cli.version)
	fmt.Println("Transform .gofa files with decorators to Go code")
	fmt.Println("")
	fmt.Println("Features:")
	fmt.Println("  ✅ Advanced decorators (@Controller, @Get, @Post, etc.)")
	fmt.Println("  ✅ Dependency injection with @Injectable")
	fmt.Println("  ✅ Parameter decorators (@Body, @Param, @Query)")
	fmt.Println("  ✅ Module system with @Module")
	fmt.Println("  ✅ Parallel transpilation for fast builds")
	fmt.Println("  ✅ File watching for development mode")
	return nil
}

// helpCommand handles the help command
func (cli *cliWithDeps) helpCommand(args []string) error {
	if len(args) > 0 {
		// Show help for specific command
		switch args[0] {
		case "transpile", "t":
			cli.transpileCommand([]string{"-h"})
		case "watch", "w":
			cli.watchCommand([]string{"-h"})
		case "version", "v":
			cli.versionCommand([]string{})
		default:
			fmt.Printf("No help available for command: %s\n", args[0])
		}
		return nil
	}

	cli.printUsage()
	return nil
}

// printUsage prints general usage information
func (cli *cliWithDeps) printUsage() {
	fmt.Printf("Gofasta Transpiler v%s\n", cli.version)
	fmt.Println("Transform .gofa files with decorators to Go code")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  gofasta <command> [options]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  transpile, t    Transpile .gofa files to .go files")
	fmt.Println("  watch, w        Watch .gofa files and transpile on changes")
	fmt.Println("  version, v      Show version information")
	fmt.Println("  help, h         Show help information")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  gofasta transpile")
	fmt.Println("  gofasta transpile -input src -output dist -verbose")
	fmt.Println("  gofasta watch -input . -debounce 1s")
	fmt.Println("  gofasta help transpile")
	fmt.Println("")
	fmt.Println("For more information about a command, run:")
	fmt.Println("  gofasta help <command>")
}

// transpileSingleFile transpiles a single .gofa file
func (cli *cliWithDeps) transpileSingleFile(inputFile, outputDir string, verbose, dryRun, force bool) error {
	if !strings.HasSuffix(inputFile, ".gofa") {
		return fmt.Errorf("input file must have .gofa extension: %s", inputFile)
	}

	// Check if input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", inputFile)
	}

	// Generate output file path
	baseName := filepath.Base(inputFile)
	baseName = strings.TrimSuffix(baseName, ".gofa") + ".go"
	outputFile := filepath.Join(outputDir, baseName)

	// Check if output file exists
	if !force && !dryRun {
		if _, err := os.Stat(outputFile); err == nil {
			return fmt.Errorf("output file already exists: %s (use -force to overwrite)", outputFile)
		}
	}

	if verbose {
		fmt.Printf("Transpiling: %s -> %s\n", inputFile, outputFile)
	}

	if dryRun {
		fmt.Printf("Would transpile: %s -> %s\n", inputFile, outputFile)
		return nil
	}

	// Read input file
	content, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Transpile
	start := time.Now()
	goCode, err := cli.deps.TranspileFile(inputFile, string(content))
	duration := time.Since(start)

	if err != nil {
		return fmt.Errorf("transpilation failed: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write output file
	if err := os.WriteFile(outputFile, []byte(goCode), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	if verbose {
		fmt.Printf("✅ Successfully transpiled in %v\n", duration)
	}

	return nil
}

// dryRunTranspile performs a dry run showing what would be transpiled
func (cli *cliWithDeps) dryRunTranspile(inputDir string, opts TranspileOptions) error {
	fmt.Printf("🔍 Dry run: scanning %s for .gofa files...\n", inputDir)

	transpiler := cli.deps.NewParallelTranspiler(opts)
	gofaFiles, err := transpiler.FindGofaFiles(inputDir)
	if err != nil {
		return fmt.Errorf("failed to find .gofa files: %w", err)
	}

	if len(gofaFiles) == 0 {
		fmt.Println("No .gofa files found")
		return nil
	}

	fmt.Printf("\nFound %d .gofa files:\n", len(gofaFiles))
	for i, file := range gofaFiles {
		outputPath := transpiler.GetOutputPath(inputDir, file)
		fmt.Printf("  %d. %s -> %s\n", i+1, file, outputPath)
	}

	fmt.Printf("\nTranspilation would use unlimited workers\n")
	fmt.Println("Run without -dry-run to perform actual transpilation")

	return nil
}

// checkExistingFiles checks if output files already exist
func (cli *cliWithDeps) checkExistingFiles(inputDir, outputDir string, preserveStruct bool) error {
	transpiler := cli.deps.NewParallelTranspiler(TranspileOptions{
		OutputDir:      outputDir,
		PreserveStruct: preserveStruct,
	})

	gofaFiles, err := transpiler.FindGofaFiles(inputDir)
	if err != nil {
		return err
	}

	var existingFiles []string
	for _, gofaFile := range gofaFiles {
		outputPath := transpiler.GetOutputPath(inputDir, gofaFile)
		if _, err := os.Stat(outputPath); err == nil {
			existingFiles = append(existingFiles, outputPath)
		}
	}

	if len(existingFiles) > 0 {
		fmt.Printf("The following output files already exist:\n")
		for _, file := range existingFiles {
			fmt.Printf("  %s\n", file)
		}
		fmt.Println("\nUse -force to overwrite existing files")
		return fmt.Errorf("output files already exist")
	}

	return nil
}