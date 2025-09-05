package transpiler

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/healtronlabs/gofasta/transpiler/cli"
	"github.com/healtronlabs/gofasta/transpiler/codegen"
	"github.com/healtronlabs/gofasta/transpiler/core"
	"github.com/healtronlabs/gofasta/transpiler/parsing"
)

// Re-export commonly used types for backward compatibility
type GofaASTNode = core.GofaASTNode
type Visitor = core.Visitor

// Re-export CLI functions
var NewCLI = cli.NewCLI

// TranspileFile transpiles a single .gofa file to Go code using the modular codegen
func TranspileFile(inputPath string, inputContent string) (string, error) {
	// Parse the .gofa file using the parsing package
	file, err := parsing.ParseGofaFile(inputContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse .gofa file: %w", err)
	}

	// Determine package name from file path or content
	packageName := determinePackageName(inputPath, file)

	// Use the comprehensive modular codegen package
	generator := codegen.NewCodeGenerator(packageName)
	goCode, err := generator.GenerateGoCode(file)
	if err != nil {
		return "", fmt.Errorf("failed to generate Go code: %w", err)
	}

	return goCode, nil
}

// determinePackageName determines the package name from file path or content
func determinePackageName(inputPath string, file *core.GofaFile) string {
	// If file has a package declaration, use it
	if file.Package != nil && file.Package.Name != "" {
		return file.Package.Name
	}

	// Extract package name from file path
	dir := filepath.Dir(inputPath)
	packageName := filepath.Base(dir)

	// Clean up package name to be valid Go identifier
	packageName = strings.ReplaceAll(packageName, "-", "")
	packageName = strings.ReplaceAll(packageName, ".", "")

	// Default to "main" if we can't determine a good package name
	if packageName == "" || packageName == "/" || packageName == "." {
		packageName = "main"
	}

	return packageName
}
