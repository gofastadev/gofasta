package transpiler

import (
	"fmt"
	"path/filepath"
)

// TranspileFile is the main entry point for transpiling a .gofa file to .go
func TranspileFile(inputPath string, inputContent string) (string, error) {
	// Parse the .gofa file
	file, err := ParseGofaFile(inputContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse .gofa file: %w", err)
	}

	// Use package name from parsed file, fallback to directory name
	var packageName string
	if file.Package != nil && file.Package.Name != "" {
		packageName = file.Package.Name
	} else {
		// Fallback to directory-based package name
		packageName = filepath.Base(filepath.Dir(inputPath))
		if packageName == "." {
			packageName = "main"
		}
	}

	// Generate Go code
	generator := NewCodeGenerator(packageName)
	goCode, err := generator.GenerateGoCode(file)
	if err != nil {
		return "", fmt.Errorf("failed to generate Go code: %w", err)
	}

	return goCode, nil
}

