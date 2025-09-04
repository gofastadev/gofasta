package main

import (
	"context"
	"flag"
	"fmt"
	"go/ast"
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
	config      *Config
	extractor   *core.DecoratorExtractor
	generator   *core.CodeGenerator
	registry    *core.DecoratorRegistry
	fileSet     *token.FileSet
	errorHandler *core.ErrorHandler
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
	// Initialize sophisticated error handler
	errorConfig := core.DefaultErrorHandlerConfig()
	errorConfig.ColorOutput = !strings.Contains(cli.config.LogLevel, "no-color")
	errorConfig.ShowSuggestions = true
	errorConfig.IncludeContext = true
	errorConfig.ContextLines = 3
	cli.errorHandler = core.NewErrorHandler(errorConfig)
	
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
		location := &core.ErrorLocation{File: inputFile}
		result.Error = cli.errorHandler.ReportError("E004", fmt.Sprintf("Failed to read source file: %v", err), location)
		return result, result.Error
	}
	result.InputSize = int64(len(sourceCode))

	// Step 1: Extract decorators from source code FIRST (before Go parsing)
	extractionResult, err := cli.extractor.Extract(sourceCode)
	if err != nil {
		location := &core.ErrorLocation{File: inputFile}
		result.Error = cli.errorHandler.ReportError("E003", fmt.Sprintf("Failed to extract decorators from source: %v", err), location)
		return result, result.Error
	}

	decorators := extractionResult.Decorators

	// Step 2: Transform .gofa to .go by processing decorators and generating code
	transformedCode, err := cli.transformGofaToGo(string(sourceCode), decorators, inputFile)
	if err != nil {
		result.Error = err // Error already handled by transformGofaToGo with sophisticated error handler
		return result, result.Error
	}

	// Step 3: Parse the transformed Go code - validation only
	_, err = parser.ParseFile(cli.fileSet, inputFile, transformedCode, parser.ParseComments)
	if err != nil {
		result.Error = cli.errorHandler.ReportSyntaxError(err, inputFile)
		return result, result.Error
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

	// Use the transformed code as the generated output
	generatedCode := transformedCode

	// Check if output file exists and force flag
	if !cli.config.Force {
		if _, err := os.Stat(outputFile); err == nil {
			location := &core.ErrorLocation{File: outputFile}
			result.Error = cli.errorHandler.ReportError("W001", "Output file already exists - use --force flag to overwrite", location)
			return result, result.Error
		}
	}

	// Write output file
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		location := &core.ErrorLocation{File: filepath.Dir(outputFile)}
		result.Error = cli.errorHandler.ReportError("E004", fmt.Sprintf("Failed to create output directory: %v", err), location)
		return result, result.Error
	}

	if err := os.WriteFile(outputFile, []byte(generatedCode), 0644); err != nil {
		location := &core.ErrorLocation{File: outputFile}
		result.Error = cli.errorHandler.ReportError("E004", fmt.Sprintf("Failed to write generated code to output file: %v", err), location)
		return result, result.Error
	}

	result.OutputSize = int64(len(generatedCode))
	result.Duration = time.Since(start)
	result.Success = true
	result.GeneratedCode = generatedCode
	result.Decorators = decorators

	return result, nil
}

// transformGofaToGo transforms .gofa source code with decorators using sophisticated core tooling
func (cli *CLI) transformGofaToGo(sourceCode string, decorators []core.Decorator, inputFile string) (string, error) {
	// DYNAMIC APPROACH: Use core tooling for intelligent AST-based transformation
	
	// Step 1: Parse source to AST using core.Parser (decorators already extracted, will be cleaned inside parseSourceToAST)
	parser := core.NewParallelParser(nil)
	parseResult, err := cli.parseSourceToAST(sourceCode, parser, inputFile)
	if err != nil {
		return "", err // Error already handled by parseSourceToAST with sophisticated error handler
	}
	
	// Step 2: Build generation context from AST and decorators
	generationContext, err := cli.buildGenerationContext(parseResult, decorators)
	if err != nil {
		return "", err // Error already handled by buildGenerationContext with sophisticated error handler
	}
	
	// Step 3: Generate code using core.CodeGenerator with templates
	generator := core.NewCodeGenerator(nil)
	generatedCode, err := cli.generateCodeFromContext(generationContext, generator, inputFile)
	if err != nil {
		return "", err // Error already handled by generateCodeFromContext with sophisticated error handler
	}
	
	// Step 4: Format using core.Formatter
	formatter := core.NewBatchFormatter(nil)
	formattedCode, err := cli.formatGeneratedCode(generatedCode, formatter, inputFile)
	if err != nil {
		return "", err // Error already handled by formatGeneratedCode with sophisticated error handler
	}
	
	return formattedCode, nil
}

// parseSourceToAST parses source code to AST using core.Parser
func (cli *CLI) parseSourceToAST(sourceCode string, parser *core.ParallelParser, originalFile string) (*core.ParseResult, error) {
	// CRITICAL: Remove decorators from source before AST parsing since Go parser doesn't understand @Decorator
	// This needs to be done AFTER decorator extraction but BEFORE AST parsing
	cleanedSource := cli.removeDecoratorLines(sourceCode)
	
	// Check if cleaning removed @ characters that might still be causing issues
	// But ignore @ characters in comments
	if strings.Contains(cleanedSource, "@") {
		lines := strings.Split(cleanedSource, "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			// Skip comments (both // and /* styles)
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
				continue
			}
			
			// Check for @ outside of comments
			if strings.Contains(line, "@") {
				// Make sure it's not in a comment within the line
				commentIndex := strings.Index(line, "//")
				atIndex := strings.Index(line, "@")
				
				// If @ appears before // comment, it's a real error
				if commentIndex == -1 || atIndex < commentIndex {
					location := &core.ErrorLocation{
						File:   originalFile,
						Line:   i + 1,
						Column: atIndex + 1,
					}
					return nil, cli.errorHandler.ReportError("E001", 
						fmt.Sprintf("Decorator syntax error: '@' character found in line %d at column %d. Make sure all decorators use proper syntax: @DecoratorName(param: value)", i+1, atIndex+1), 
						location)
				}
			}
		}
	}
	
	// Create temporary file for parsing with .go extension so Go parser recognizes it
	tmpFile, err := os.CreateTemp("", "gofa_parse_*.go")
	if err != nil {
		location := &core.ErrorLocation{File: originalFile}
		return nil, cli.errorHandler.ReportError("E004", 
			fmt.Sprintf("Internal error: Failed to create temporary file for parsing '%s': %v", originalFile, err), 
			location)
	}
	defer os.Remove(tmpFile.Name())
	
	if _, err := tmpFile.WriteString(cleanedSource); err != nil {
		location := &core.ErrorLocation{File: originalFile}
		return nil, cli.errorHandler.ReportError("E004", 
			fmt.Sprintf("Internal error: Failed to write cleaned source for '%s': %v", originalFile, err), 
			location)
	}
	tmpFile.Close()
	
	// Parse using core.Parser with single file
	ctx := context.Background()
	results, err := parser.ParseFiles(ctx, []string{tmpFile.Name()})
	if err != nil {
		// Create meaningful error with original file path
		location := &core.ErrorLocation{File: originalFile}
		return nil, cli.errorHandler.ReportError("E001", 
			fmt.Sprintf("Go syntax error in '%s': %v", originalFile, err), 
			location)
	}
	
	if len(results) == 0 {
		location := &core.ErrorLocation{File: originalFile}
		return nil, cli.errorHandler.ReportError("E001", 
			fmt.Sprintf("No parse results for '%s' - file appears to be empty after decorator removal", originalFile), 
			location)
	}
	
	if results[0].Error != nil {
		location := cli.extractLocationFromError(results[0].Error.Error(), originalFile)
		return nil, cli.errorHandler.ReportError("E001", 
			fmt.Sprintf("Go syntax error in '%s': %v", originalFile, results[0].Error), 
			location)
	}
	
	return results[0], nil
}

// extractLocationFromError extracts line and column information from Go parser error messages
func (cli *CLI) extractLocationFromError(errorMsg string, originalFile string) *core.ErrorLocation {
	location := &core.ErrorLocation{File: originalFile}
	
	// Try to parse location from error message
	// Examples: "/tmp/file.go:5:10: expected ';', found 'EOF'"
	if strings.Contains(errorMsg, ":") {
		parts := strings.Split(errorMsg, ":")
		if len(parts) >= 3 {
			// Try to parse line and column numbers
			var line, col int
			if n, _ := fmt.Sscanf(parts[len(parts)-3]+":"+parts[len(parts)-2], "%d:%d", &line, &col); n >= 1 {
				location.Line = line
				location.Column = col
			}
		}
	}
	
	return location
}

// buildGenerationContext builds generation context from AST and decorators
func (cli *CLI) buildGenerationContext(parseResult *core.ParseResult, decorators []core.Decorator) (*core.GenerationContext, error) {
	// Extract package name from AST
	packageName := "main"
	if parseResult.File.Name != nil {
		packageName = parseResult.File.Name.Name
	}
	
	// Build decorator map for efficient lookup
	decoratorMap := cli.buildDecoratorMap(parseResult.File, decorators)
	
	// Extract functions from AST with decorator information
	functions := cli.extractFunctionsFromAST(parseResult.File, decoratorMap)
	
	// Build generation context
	context := &core.GenerationContext{
		PackageName: packageName,
		Imports:     cli.buildDynamicImports(decorators),
		Decorators:  decorators,
		Functions:   functions,
		Metadata:    cli.buildGenerationMetadata(decorators),
	}
	
	return context, nil
}

// buildDecoratorMap dynamically maps decorators to their target functions
func (cli *CLI) buildDecoratorMap(file *ast.File, decorators []core.Decorator) map[string][]core.Decorator {
	decoratorMap := make(map[string][]core.Decorator)
	
	// Walk AST to find functions and match with decorators by line numbers
	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			if funcDecl.Name != nil {
				funcName := funcDecl.Name.Name
				funcLine := cli.fileSet.Position(funcDecl.Pos()).Line
				
				// Find decorators that belong to this function (preceding lines)
				var funcDecorators []core.Decorator
				for _, decorator := range decorators {
					if decorator.Line < funcLine && (funcLine - decorator.Line) <= 5 {
						funcDecorators = append(funcDecorators, decorator)
					}
				}
				
				if len(funcDecorators) > 0 {
					decoratorMap[funcName] = funcDecorators
				}
			}
		}
		return true
	})
	
	return decoratorMap
}

// extractFunctionsFromAST extracts functions from AST with decorator information
func (cli *CLI) extractFunctionsFromAST(file *ast.File, decoratorMap map[string][]core.Decorator) []core.FunctionDefinition {
	var functions []core.FunctionDefinition
	
	// Walk AST to extract all functions
	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			if funcDecl.Name != nil {
				funcName := funcDecl.Name.Name
				function := core.FunctionDefinition{
					Name:       funcName,
					Parameters: cli.extractParameters(funcDecl),
					Returns:    cli.extractReturns(funcDecl),
					Body:       "// Dynamic runtime decorator call will be generated\n",
				}
				
				// Add decorator information if this function has decorators
				if funcDecorators, hasDecorators := decoratorMap[funcName]; hasDecorators {
					function.Decorators = funcDecorators
				}
				
				functions = append(functions, function)
			}
		}
		return true
	})
	
	return functions
}

// extractParameters extracts function parameters from AST
func (cli *CLI) extractParameters(funcDecl *ast.FuncDecl) []core.ParameterDefinition {
	var parameters []core.ParameterDefinition
	
	if funcDecl.Type != nil && funcDecl.Type.Params != nil {
		for _, param := range funcDecl.Type.Params.List {
			paramType := cli.extractTypeString(param.Type)
			for _, name := range param.Names {
				parameters = append(parameters, core.ParameterDefinition{
					Name: name.Name,
					Type: paramType,
				})
			}
		}
	}
	
	return parameters
}

// extractReturns extracts function return types from AST
func (cli *CLI) extractReturns(funcDecl *ast.FuncDecl) []string {
	var returns []string
	
	if funcDecl.Type != nil && funcDecl.Type.Results != nil {
		for _, result := range funcDecl.Type.Results.List {
			returnType := cli.extractTypeString(result.Type)
			returns = append(returns, returnType)
		}
	}
	
	return returns
}

// extractTypeString extracts type string from AST expression
func (cli *CLI) extractTypeString(expr ast.Expr) string {
	// Simple type extraction - could be enhanced for complex types
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return "interface{}"
}

// buildGenerationMetadata builds metadata for code generation
func (cli *CLI) buildGenerationMetadata(decorators []core.Decorator) map[string]interface{} {
	metadata := make(map[string]interface{})
	metadata["generatedAt"] = time.Now().Format(time.RFC3339)
	metadata["decoratorCount"] = len(decorators)
	metadata["HeaderTemplate"] = "// Code generated by Gofasta. DO NOT EDIT."
	return metadata
}

// removeDecoratorLines removes @Decorator lines from the source code
func (cli *CLI) removeDecoratorLines(sourceCode string) string {
	lines := strings.Split(sourceCode, "\n")
	var cleanedLines []string
	inDecorator := false
	parenCount := 0
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Check if starting a decorator
		if strings.HasPrefix(trimmed, "@") {
			inDecorator = true
			parenCount = strings.Count(line, "(") - strings.Count(line, ")")
			
			// If decorator is complete on one line, skip it and continue
			if parenCount <= 0 {
				inDecorator = false
				continue
			}
			// Skip the line and continue processing multi-line decorator
			continue
		}
		
		// If we're inside a multi-line decorator, continue tracking
		if inDecorator {
			parenCount += strings.Count(line, "(") - strings.Count(line, ")")
			
			// If we've closed all parentheses, we're done with this decorator
			if parenCount <= 0 {
				inDecorator = false
			}
			// Skip this line as it's part of the decorator
			continue
		}
		
		// Only add lines that are not part of decorators
		cleanedLines = append(cleanedLines, line)
	}
	
	return strings.Join(cleanedLines, "\n")
}

// buildDynamicImports dynamically builds imports based on used decorators
func (cli *CLI) buildDynamicImports(decorators []core.Decorator) []string {
	importSet := make(map[string]bool)
	
	// Always add core runtime imports
	importSet["context"] = true
	importSet["log"] = true
	
	// Add decorator-specific imports dynamically
	for _, decorator := range decorators {
		switch decorator.Name {
		case "Actor":
			importSet["github.com/healtronlabs/gofasta/transpiler/decorators/faulttolerance/actor"] = true
		case "Supervisor":
			importSet["github.com/healtronlabs/gofasta/transpiler/decorators/faulttolerance/supervisor"] = true
		case "ActorRef":
			importSet["github.com/healtronlabs/gofasta/transpiler/decorators/faulttolerance/actorref"] = true
		case "ActorSystem":
			importSet["github.com/healtronlabs/gofasta/transpiler/decorators/faulttolerance/actorsystem"] = true
		}
	}
	
	// Add core decorator registry
	importSet["github.com/healtronlabs/gofasta/transpiler/core"] = true
	
	// Convert to slice
	var imports []string
	for imp := range importSet {
		imports = append(imports, imp)
	}
	
	return imports
}

// generateCodeFromContext generates Go code using generation context
func (cli *CLI) generateCodeFromContext(context *core.GenerationContext, generator *core.CodeGenerator, inputFile string) (string, error) {
	// Add custom templates for dynamic runtime calls
	if err := cli.addDecoratorTemplates(generator); err != nil {
		location := &core.ErrorLocation{File: inputFile}
		return "", cli.errorHandler.ReportError("E004", 
			fmt.Sprintf("Internal error: Failed to initialize code templates for '%s': %v", inputFile, err), 
			location)
	}
	
	// Select template based on decorator types dynamically
	templateNamecandidate := cli.selectTemplateForDecorators(context.Decorators)
	
	// Generate code using dynamic template selection
	result, err := generator.Generate(templateNamecandidate, context)
	if err != nil {
		location := &core.ErrorLocation{File: inputFile}
		// Make the error message more helpful
		errorMsg := cli.makeTemplateErrorFriendly(err, templateNamecandidate, inputFile)
		return "", cli.errorHandler.ReportError("E004", errorMsg, location)
	}
	
	if result == nil || result.Code == "" {
		location := &core.ErrorLocation{File: inputFile}
		return "", cli.errorHandler.ReportError("E004", 
			fmt.Sprintf("Code generation failed for '%s': template '%s' produced no output. Check your decorator syntax and parameters.", 
				inputFile, templateNamecandidate), 
			location)
	}
	
	return result.Code, nil
}

// makeTemplateErrorFriendly converts cryptic template errors into developer-friendly messages
func (cli *CLI) makeTemplateErrorFriendly(err error, templateName string, inputFile string) string {
	errStr := err.Error()
	
	// Check for common template errors and provide helpful messages
	if strings.Contains(errStr, "HeaderTemplate") {
		return fmt.Sprintf("Template error in '%s': Missing HeaderTemplate field. This is likely an internal error with the code generator setup.", inputFile)
	}
	
	if strings.Contains(errStr, "can't evaluate field") {
		fieldName := cli.extractFieldName(errStr)
		return fmt.Sprintf("Template error in '%s': Field '%s' is missing from the generation context. Check your decorator parameters and syntax.", inputFile, fieldName)
	}
	
	if strings.Contains(errStr, "template not found") || strings.Contains(errStr, "no such template") {
		return fmt.Sprintf("Template error in '%s': Template '%s' not found. This indicates an issue with decorator type '%s' - check that the decorator is properly supported.", inputFile, templateName, templateName)
	}
	
	if strings.Contains(errStr, "executing") {
		return fmt.Sprintf("Template execution error in '%s': %v. Check your decorator parameters - one or more values may be invalid.", inputFile, err)
	}
	
	// Generic fallback for other template errors
	return fmt.Sprintf("Template error in '%s' using template '%s': %v", inputFile, templateName, err)
}

// extractFieldName extracts field name from template error messages
func (cli *CLI) extractFieldName(errStr string) string {
	// Look for patterns like "can't evaluate field FieldName in type"
	if idx := strings.Index(errStr, "can't evaluate field "); idx != -1 {
		start := idx + len("can't evaluate field ")
		if end := strings.Index(errStr[start:], " in type"); end != -1 {
			return errStr[start : start+end]
		}
	}
	return "unknown"
}

// addDecoratorTemplates adds custom templates for decorator runtime calls
func (cli *CLI) addDecoratorTemplates(generator *core.CodeGenerator) error {
	// Actor runtime template
	actorTemplate := `package {{.PackageName}}

{{if .Imports}}import (
{{range .Imports}}	"{{.}}"
{{end}}
){{end}}

{{range .Functions}}{{if .Decorators}}
// Actor runtime initialization for {{.Name}}
var {{.Name}}_runtime *actor.ActorTarget

func init() {
	// Apply Actor decorator using the real runtime implementation
	args := core.DecoratorArgs{
		Properties: {{range .Decorators}}{{.Properties | printf "%#v"}}{{end}},
	}
	
	result, err := actor.ActorDecorator(context.Background(), args)
	if err != nil {
		log.Printf("Failed to initialize actor: %v", err)
		return
	}
	
	if actorTarget, ok := result.Modified.(*actor.ActorTarget); ok {
		{{.Name}}_runtime = actorTarget
		log.Printf("Actor runtime initialized successfully")
	}
}

// Helper function to get the actor runtime
func get{{.Name}}Runtime() *actor.ActorTarget {
	return {{.Name}}_runtime
}
{{end}}

func {{.Name}}({{range $i, $p := .Parameters}}{{if $i}}, {{end}}{{$p.Name}} {{$p.Type}}{{end}}){{if .Returns}} ({{range $i, $r := .Returns}}{{if $i}}, {{end}}{{$r}}{{end}}){{end}} {
	// Original function logic here
	{{.Body}}
}
{{end}}`

	if err := generator.AddTemplate("actor_runtime", actorTemplate, nil); err != nil {
		return err
	}

	// Supervisor runtime template
	supervisorTemplate := `package {{.PackageName}}

{{if .Imports}}import (
{{range .Imports}}	"{{.}}"
{{end}}
){{end}}

{{range .Functions}}{{if .Decorators}}
// Supervisor runtime initialization for {{.Name}}
var {{.Name}}_runtime *supervisor.SupervisorTarget

func init() {
	// Apply Supervisor decorator using the real runtime implementation
	args := core.DecoratorArgs{
		Properties: {{range .Decorators}}{{.Properties | printf "%#v"}}{{end}},
	}
	
	result, err := supervisor.SupervisorDecorator(context.Background(), args)
	if err != nil {
		log.Printf("Failed to initialize supervisor: %v", err)
		return
	}
	
	if supervisorTarget, ok := result.Modified.(*supervisor.SupervisorTarget); ok {
		{{.Name}}_runtime = supervisorTarget
		log.Printf("Supervisor runtime initialized successfully")
	}
}

// Helper function to get the supervisor runtime
func get{{.Name}}Runtime() *supervisor.SupervisorTarget {
	return {{.Name}}_runtime
}
{{end}}

func {{.Name}}({{range $i, $p := .Parameters}}{{if $i}}, {{end}}{{$p.Name}} {{$p.Type}}{{end}}){{if .Returns}} ({{range $i, $r := .Returns}}{{if $i}}, {{end}}{{$r}}{{end}}){{end}} {
	// Original function logic here
	{{.Body}}
}
{{end}}`

	return generator.AddTemplate("supervisor_runtime", supervisorTemplate, nil)
}

// selectTemplateForDecorators dynamically selects template based on decorator types
func (cli *CLI) selectTemplateForDecorators(decorators []core.Decorator) string {
	hasActor := false
	hasSupervisor := false
	
	for _, decorator := range decorators {
		switch decorator.Name {
		case "Actor":
			hasActor = true
		case "Supervisor":
			hasSupervisor = true
		}
	}
	
	// Dynamic template selection based on decorator combination
	if hasActor {
		return "actor_runtime"
	} else if hasSupervisor {
		return "supervisor_runtime"
	}
	
	// Default to basic package template
	return "package"
}

// formatGeneratedCode formats code using core.Formatter
func (cli *CLI) formatGeneratedCode(code string, formatter *core.BatchFormatter, inputFile string) (string, error) {
	// Parse the generated code to AST for formatting
	parser := core.NewParallelParser(nil)
	tmpFile, err := os.CreateTemp("", "generated_*.go")
	if err != nil {
		location := &core.ErrorLocation{File: inputFile}
		cli.errorHandler.ReportWarning("W001", fmt.Sprintf("Code formatting skipped for '%s': Failed to create temporary file: %v", inputFile, err), location)
		return code, nil // Return original if temp file creation fails
	}
	defer os.Remove(tmpFile.Name())
	
	if _, err := tmpFile.WriteString(code); err != nil {
		location := &core.ErrorLocation{File: inputFile}
		cli.errorHandler.ReportWarning("W001", fmt.Sprintf("Code formatting skipped for '%s': Failed to write to temporary file: %v", inputFile, err), location)
		return code, nil // Return original if write fails
	}
	tmpFile.Close()
	
	ctx := context.Background()
	results, err := parser.ParseFiles(ctx, []string{tmpFile.Name()})
	if err != nil {
		// If parsing fails for formatting, log the error but return unformatted code
		location := &core.ErrorLocation{File: inputFile}
		cli.errorHandler.ReportWarning("W001", fmt.Sprintf("Code formatting skipped for '%s': Generated code has syntax issues: %v", inputFile, err), location)
		return code, nil // Return original if parse fails
	}
	
	if len(results) == 0 {
		location := &core.ErrorLocation{File: inputFile}
		cli.errorHandler.ReportWarning("W001", fmt.Sprintf("Code formatting skipped for '%s': No parsing results returned", inputFile), location)
		return code, nil // Return original if no results
	}
	
	if results[0].Error != nil {
		location := &core.ErrorLocation{File: inputFile}
		cli.errorHandler.ReportWarning("W001", fmt.Sprintf("Code formatting skipped for '%s': Generated code syntax error: %v", inputFile, results[0].Error), location)
		return code, nil // Return original if parse error
	}
	
	// Format using BatchFormatter
	files := map[string]*ast.File{
		tmpFile.Name(): results[0].File,
	}
	
	formatResults, err := formatter.FormatFiles(ctx, files, results[0].FileSet)
	if err != nil {
		location := &core.ErrorLocation{File: inputFile}
		cli.errorHandler.ReportWarning("W001", fmt.Sprintf("Code formatting skipped for '%s': Formatting failed: %v", inputFile, err), location)
		return code, nil // Return original if formatting fails
	}
	
	if formatResult, ok := formatResults[tmpFile.Name()]; ok && formatResult.Error == nil {
		return string(formatResult.Output), nil
	}
	
	// Log formatting issues but don't fail
	if formatResult, ok := formatResults[tmpFile.Name()]; ok && formatResult.Error != nil {
		location := &core.ErrorLocation{File: inputFile}
		cli.errorHandler.ReportWarning("W001", fmt.Sprintf("Code formatting skipped for '%s': %v", inputFile, formatResult.Error), location)
	}
	
	return code, nil // Return original if formatting fails
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