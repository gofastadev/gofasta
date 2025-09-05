// End-to-End Transpilation Integration Tests - Tests complete .gofa to .go transpilation pipeline
package core

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	e2eTestTimeout = 60 * time.Second
	binaryPath     = "../../transpiler/dist/gofasta"
)

// TestE2EBasicTranspilation tests basic end-to-end transpilation functionality
func TestE2EBasicTranspilation(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name           string
		inputContent   string
		expectedOutput bool
		description    string
	}{
		{
			name: "simple_go_file",
			inputContent: `package main

import "fmt"

func main() {
	fmt.Println("Hello, Gofasta!")
}`,
			expectedOutput: true,
			description:    "Simple Go file without decorators should transpile successfully",
		},
		{
			name: "go_file_with_imports",
			inputContent: `package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	fmt.Println("Current time:", time.Now())
	fmt.Println("Args:", os.Args)
}`,
			expectedOutput: true,
			description:    "Go file with multiple imports should transpile successfully",
		},
		{
			name: "go_file_with_structs",
			inputContent: `package main

import "fmt"

type User struct {
	Name string
	Age  int
}

func (u User) String() string {
	return fmt.Sprintf("User: %s (%d)", u.Name, u.Age)
}

func main() {
	user := User{Name: "Alice", Age: 30}
	fmt.Println(user)
}`,
			expectedOutput: true,
			description:    "Go file with structs and methods should transpile successfully",
		},
		{
			name: "go_file_with_interfaces",
			inputContent: `package main

import "fmt"

type Writer interface {
	Write(data string) error
}

type ConsoleWriter struct{}

func (c ConsoleWriter) Write(data string) error {
	fmt.Println(data)
	return nil
}

func main() {
	var w Writer = ConsoleWriter{}
	w.Write("Hello from interface!")
}`,
			expectedOutput: true,
			description:    "Go file with interfaces should transpile successfully",
		},
		{
			name: "gofa_file_with_decorators",
			inputContent: `package main

import "fmt"

@Controller("/api/users")
type UserController struct {}

@Get("/")
func (uc *UserController) GetUsers() string {
	return "users"
}

func main() {
	fmt.Println("Controller example")
}`,
			expectedOutput: false,
			description:    "Gofasta file with decorators should fail parsing (Phase 1 limitation)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create input file
			inputFile := filepath.Join(tempDir, tt.name+".gofa")
			if err := os.WriteFile(inputFile, []byte(tt.inputContent), 0644); err != nil {
				t.Fatalf("Failed to create input file: %v", err)
			}

			// Create output directory
			outputDir := filepath.Join(tempDir, "output", tt.name)
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				t.Fatalf("Failed to create output directory: %v", err)
			}

			// Run transpilation on specific file only
			cmd := exec.Command(binaryPath, "-input", tempDir, "-output", outputDir, "-pattern", tt.name+".gofa", "-verbose")
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			if tt.expectedOutput {
				// Should succeed
				if err != nil {
					t.Errorf("%s: Expected successful transpilation but got error: %v", tt.description, err)
					t.Logf("Stdout: %s", stdout.String())
					t.Logf("Stderr: %s", stderr.String())
					return
				}

				// Check output file exists
				outputFile := filepath.Join(outputDir, strings.Replace(tt.name, ".gofa", ".go", 1)+".go")
				if _, err := os.Stat(outputFile); os.IsNotExist(err) {
					t.Errorf("%s: Output file %s was not created", tt.description, outputFile)
					return
				}

				// Verify output file is valid Go
				content, err := os.ReadFile(outputFile)
				if err != nil {
					t.Errorf("%s: Failed to read output file: %v", tt.description, err)
					return
				}

				if len(content) == 0 {
					t.Errorf("%s: Output file is empty", tt.description)
					return
				}

				// Basic validation - should contain package declaration
				contentStr := string(content)
				if !strings.Contains(contentStr, "package main") {
					t.Errorf("%s: Output file does not contain package declaration", tt.description)
				}
			} else {
				// Should fail - check if error contains expected message or exit code indicates failure
				allOutput := stdout.String() + stderr.String()
				if err != nil {
					// Good - process failed with error
					t.Logf("%s: Correctly failed with error: %v", tt.description, err)
				} else if !strings.Contains(allOutput, "❌") && !strings.Contains(allOutput, "errors") {
					// Process succeeded but should have failed
					t.Errorf("%s: Expected transpilation to fail but it succeeded without errors\nOutput: %s", tt.description, allOutput)
				} else {
					// Process succeeded but reported errors in output - this is acceptable
					t.Logf("%s: Process completed but correctly reported errors in output", tt.description)
				}
			}
		})
	}
}

// TestE2EMultiFileProcessing tests processing multiple files in a single operation
func TestE2EMultiFileProcessing(t *testing.T) {
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	outputDir := filepath.Join(tempDir, "output")

	// Create input directory
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatalf("Failed to create input directory: %v", err)
	}

	// Create multiple test files
	testFiles := map[string]string{
		"file1.gofa": `package main
import "fmt"
func main() { fmt.Println("File 1") }`,
		"file2.gofa": `package utils
func Helper() string { return "helper" }`,
		"file3.gofa": `package models
type User struct { Name string }`,
		"subdir/file4.gofa": `package subpackage
func SubFunction() int { return 42 }`,
	}

	// Write test files
	for filename, content := range testFiles {
		fullPath := filepath.Join(inputDir, filename)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", filename, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", filename, err)
		}
	}

	// Run transpilation on all files
	cmd := exec.Command(binaryPath, "-input", inputDir, "-output", outputDir, "-verbose")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Multi-file transpilation failed: %v\nStdout: %s\nStderr: %s",
			err, stdout.String(), stderr.String())
	}

	// Verify all output files were created
	expectedOutputs := []string{
		"file1.go",
		"file2.go",
		"file3.go",
		"subdir/file4.go",
	}

	for _, expectedFile := range expectedOutputs {
		outputFile := filepath.Join(outputDir, expectedFile)
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Errorf("Expected output file %s was not created", outputFile)
		} else {
			// Verify file has content
			content, err := os.ReadFile(outputFile)
			if err != nil {
				t.Errorf("Failed to read output file %s: %v", outputFile, err)
			} else if len(content) == 0 {
				t.Errorf("Output file %s is empty", outputFile)
			}
		}
	}

	// Verify output contains information about all files
	outputStr := stdout.String()
	if !strings.Contains(outputStr, "Found 4 .gofa files") {
		t.Errorf("Output should indicate 4 files were found, got: %s", outputStr)
	}

	// Check that at least some files were processed successfully
	if !strings.Contains(outputStr, "✅") && !strings.Contains(outputStr, "Generated") {
		t.Errorf("Output should show successful file processing, got: %s", outputStr)
	}
}

// TestE2EDirectoryStructure tests that directory structure is preserved
func TestE2EDirectoryStructure(t *testing.T) {
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "src")
	outputDir := filepath.Join(tempDir, "dist")

	// Create complex directory structure
	structure := map[string]string{
		"main.gofa":             "package main\nfunc main() {}",
		"utils/helper.gofa":     "package utils\nfunc Help() {}",
		"models/user.gofa":      "package models\ntype User struct{}",
		"controllers/base.gofa": "package controllers\ntype Base struct{}",
		"api/v1/routes.gofa":    "package v1\nfunc Routes() {}",
		"api/v2/routes.gofa":    "package v2\nfunc Routes() {}",
		"internal/config.gofa":  "package internal\nvar Config string",
	}

	// Create input files
	for relPath, content := range structure {
		fullPath := filepath.Join(inputDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", relPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", relPath, err)
		}
	}

	// Run transpilation
	cmd := exec.Command(binaryPath, "-input", inputDir, "-output", outputDir, "-verbose")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Directory structure transpilation failed: %v\nOutput: %s", err, string(output))
	}

	// Debug: show what was actually created
	t.Logf("Transpilation output: %s", string(output))
	t.Logf("Checking output directory: %s", outputDir)
	if dirContents, err := os.ReadDir(outputDir); err == nil {
		t.Logf("Output directory contents:")
		for _, entry := range dirContents {
			t.Logf("  %s (isDir: %v)", entry.Name(), entry.IsDir())
		}
	}

	// Verify output directory structure matches input
	for relPath := range structure {
		// Convert .gofa to .go
		outputPath := strings.Replace(relPath, ".gofa", ".go", 1)
		fullOutputPath := filepath.Join(outputDir, outputPath)

		if _, err := os.Stat(fullOutputPath); os.IsNotExist(err) {
			t.Errorf("Expected output file %s was not created", fullOutputPath)
		} else {
			// Verify the file has content
			content, err := os.ReadFile(fullOutputPath)
			if err != nil {
				t.Errorf("Failed to read output file %s: %v", fullOutputPath, err)
			} else if len(content) == 0 {
				t.Errorf("Output file %s is empty", fullOutputPath)
			}
		}
	}

	// Verify subdirectories were created
	expectedDirs := []string{
		"utils", "models", "controllers", "api", "api/v1", "api/v2", "internal",
	}

	for _, dir := range expectedDirs {
		dirPath := filepath.Join(outputDir, dir)
		if stat, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf("Expected directory %s was not created", dirPath)
		} else if !stat.IsDir() {
			t.Errorf("Path %s exists but is not a directory", dirPath)
		}
	}
}

// TestE2EOutputValidation tests that output files are valid and properly formatted
func TestE2EOutputValidation(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name         string
		input        string
		validateFunc func(t *testing.T, outputContent string)
	}{
		{
			name: "formatting_preserved",
			input: `package main

import (
	"fmt"
	"os"
)

// Main function comment
func main() {
	// Print greeting
	fmt.Println("Hello, World!")
	
	// Check arguments
	if len(os.Args) > 1 {
		fmt.Println("Arguments:", os.Args[1:])
	}
}`,
			validateFunc: func(t *testing.T, output string) {
				// Check that comments are preserved
				if !strings.Contains(output, "Main function comment") {
					t.Error("Output should preserve function comments")
				}
				if !strings.Contains(output, "Print greeting") {
					t.Error("Output should preserve inline comments")
				}

				// Check that imports are formatted properly
				if !strings.Contains(output, "import (") {
					t.Error("Output should preserve import grouping")
				}

				// Check that the code structure is maintained
				if !strings.Contains(output, "func main()") {
					t.Error("Output should contain main function")
				}
			},
		},
		{
			name: "package_declaration",
			input: `package mypackage

const Version = "1.0.0"

var GlobalVar = "test"

type MyStruct struct {
	Field1 string
	Field2 int
}

func (m MyStruct) Method() string {
	return m.Field1
}`,
			validateFunc: func(t *testing.T, output string) {
				// Check package name is preserved
				if !strings.Contains(output, "package mypackage") {
					t.Error("Output should preserve package name")
				}

				// Check that constants, variables, types, and methods are preserved
				elements := []string{"const Version", "var GlobalVar", "type MyStruct", "func (m MyStruct) Method()"}
				for _, element := range elements {
					if !strings.Contains(output, element) {
						t.Errorf("Output should contain: %s", element)
					}
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create input file
			inputFile := filepath.Join(tempDir, tc.name+".gofa")
			if err := os.WriteFile(inputFile, []byte(tc.input), 0644); err != nil {
				t.Fatalf("Failed to create input file: %v", err)
			}

			// Run transpilation
			outputDir := filepath.Join(tempDir, "output_"+tc.name)
			cmd := exec.Command(binaryPath, "-input", tempDir, "-output", outputDir, "-pattern", tc.name+".gofa")
			if err := cmd.Run(); err != nil {
				t.Fatalf("Transpilation failed: %v", err)
			}

			// Read output file
			outputFile := filepath.Join(outputDir, tc.name+".go")
			content, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			// Run validation
			tc.validateFunc(t, string(content))
		})
	}
}

// TestE2EErrorPropagation tests how errors are handled throughout the pipeline
func TestE2EErrorPropagation(t *testing.T) {
	tempDir := t.TempDir()

	errorCases := []struct {
		name             string
		input            string
		expectError      bool
		expectedInOutput string
		description      string
	}{
		{
			name: "syntax_error",
			input: `package main

func main() {
	fmt.Println("Missing import"
}`,
			expectError:      true,
			expectedInOutput: "missing",
			description:      "File with syntax errors should report parsing errors",
		},
		{
			name: "decorator_syntax",
			input: `package main

@Controller("/api")
type MyController struct {}

func main() {}`,
			expectError:      true,
			expectedInOutput: "illegal character U+0040 '@'",
			description:      "Decorator syntax should report specific parsing error",
		},
		{
			name: "malformed_package",
			input: `packag main

func main() {}`,
			expectError:      true,
			expectedInOutput: "expected",
			description:      "Malformed package declaration should be caught",
		},
	}

	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create input file
			inputFile := filepath.Join(tempDir, tc.name+".gofa")
			if err := os.WriteFile(inputFile, []byte(tc.input), 0644); err != nil {
				t.Fatalf("Failed to create input file: %v", err)
			}

			// Run transpilation
			outputDir := filepath.Join(tempDir, "error_output_"+tc.name)
			cmd := exec.Command(binaryPath, "-input", tempDir, "-output", outputDir, "-pattern", tc.name+".gofa", "-verbose")
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			if tc.expectError {
				// Should have error or warning in output
				allOutput := stdout.String() + stderr.String()
				if !strings.Contains(strings.ToLower(allOutput), strings.ToLower(tc.expectedInOutput)) {
					t.Errorf("%s: Expected error message containing '%s', got: %s",
						tc.description, tc.expectedInOutput, allOutput)
				}
			} else {
				if err != nil {
					t.Errorf("%s: Unexpected error: %v\nOutput: %s", tc.description, err, stdout.String()+stderr.String())
				}
			}
		})
	}
}

// TestE2EPerformanceValidation tests transpilation performance
func TestE2EPerformanceValidation(t *testing.T) {
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "perf_input")
	outputDir := filepath.Join(tempDir, "perf_output")

	// Create input directory
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatalf("Failed to create input directory: %v", err)
	}

	// Generate multiple files for performance testing
	const numFiles = 10
	for i := 0; i < numFiles; i++ {
		filename := fmt.Sprintf("file_%d.gofa", i)
		content := fmt.Sprintf(`package main

import (
	"fmt"
	"time"
	"math/rand"
)

// File %d generated for performance testing
func main() {
	fmt.Println("Processing file %d")
	start := time.Now()
	
	// Some work
	sum := 0
	for j := 0; j < 1000; j++ {
		sum += rand.Intn(100)
	}
	
	fmt.Printf("File %d processed in %%v with sum %%d\n", time.Since(start), sum)
}

func helper%d() string {
	return "helper for file %d"
}

type Data%d struct {
	ID    int
	Value string
}

func (d Data%d) Process() {
	fmt.Printf("Processing data %%d: %%s\n", d.ID, d.Value)
}`, i, i, i, i, i, i, i)

		filePath := filepath.Join(inputDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create performance test file %s: %v", filename, err)
		}
	}

	// Run transpilation with timing
	start := time.Now()
	cmd := exec.Command(binaryPath, "-input", inputDir, "-output", outputDir, "-verbose")
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Performance test transpilation failed: %v\nOutput: %s", err, string(output))
	}

	// Validate performance metrics
	outputStr := string(output)

	// Check that all files were processed
	if !strings.Contains(outputStr, fmt.Sprintf("Found %d .gofa files", numFiles)) {
		t.Errorf("Expected to find %d files, output: %s", numFiles, outputStr)
	}

	// Performance expectations (adjust based on your requirements)
	maxExpectedDuration := 10 * time.Second
	if duration > maxExpectedDuration {
		t.Errorf("Transpilation took too long: %v (max expected: %v)", duration, maxExpectedDuration)
	}

	// Verify all output files exist and are non-empty
	for i := 0; i < numFiles; i++ {
		outputFile := filepath.Join(outputDir, fmt.Sprintf("file_%d.go", i))
		if stat, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Errorf("Output file %s was not created", outputFile)
		} else if stat.Size() == 0 {
			t.Errorf("Output file %s is empty", outputFile)
		}
	}

	t.Logf("Performance test completed: %d files processed in %v", numFiles, duration)
}

// TestE2EPipelineIntegration tests integration with core transpiler components
func TestE2EPipelineIntegration(t *testing.T) {
	tempDir := t.TempDir()

	// Test file that exercises various core components
	testContent := `package integration

import (
	"context"
	"fmt"
	"time"
)

// TestStruct exercises type checking
type TestStruct struct {
	Name      string
	Value     int
	Timestamp time.Time
}

// TestInterface exercises interface handling
type TestInterface interface {
	Process(ctx context.Context) error
	Validate() bool
}

// Implementation exercises method sets
func (ts TestStruct) Process(ctx context.Context) error {
	fmt.Printf("Processing %s with value %d\n", ts.Name, ts.Value)
	return nil
}

func (ts TestStruct) Validate() bool {
	return ts.Name != "" && ts.Value > 0
}

// TestFunction exercises function analysis
func TestFunction(data []TestStruct) map[string]int {
	result := make(map[string]int)
	
	for _, item := range data {
		if item.Validate() {
			result[item.Name] = item.Value
		}
	}
	
	return result
}

// TestGeneric exercises type parameter handling (Go 1.18+)
func ProcessItems[T comparable](items []T, processor func(T) string) []string {
	var results []string
	for _, item := range items {
		results = append(results, processor(item))
	}
	return results
}

func init() {
	fmt.Println("Package integration initialized")
}`

	// Create test file
	inputFile := filepath.Join(tempDir, "pipeline_test.gofa")
	if err := os.WriteFile(inputFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test with different transpiler configurations
	configurations := []struct {
		name  string
		flags []string
	}{
		{
			name:  "default_config",
			flags: []string{"-verbose"},
		},
		{
			name:  "parallel_processing",
			flags: []string{"-verbose", "-parallel", "4"},
		},
		{
			name:  "with_caching",
			flags: []string{"-verbose", "-cache"},
		},
		{
			name:  "force_overwrite",
			flags: []string{"-verbose", "-force"},
		},
	}

	for _, config := range configurations {
		t.Run(config.name, func(t *testing.T) {
			outputDir := filepath.Join(tempDir, "output_"+config.name)

			// Build command with configuration
			args := append([]string{"-input", tempDir, "-output", outputDir}, config.flags...)
			args = append(args, "-pattern", "pipeline_test.gofa")

			cmd := exec.Command(binaryPath, args...)
			output, err := cmd.CombinedOutput()

			if err != nil {
				t.Errorf("Configuration %s failed: %v\nOutput: %s", config.name, err, string(output))
				return
			}

			// Verify output file exists and is valid
			outputFile := filepath.Join(outputDir, "pipeline_test.go")
			content, err := os.ReadFile(outputFile)
			if err != nil {
				t.Errorf("Failed to read output for config %s: %v", config.name, err)
				return
			}

			// Validate key elements are present
			contentStr := string(content)
			requiredElements := []string{
				"package integration",
				"type TestStruct struct",
				"type TestInterface interface",
				"func (ts TestStruct) Process",
				"func TestFunction",
				"func ProcessItems",
				"func init()",
			}

			for _, element := range requiredElements {
				if !strings.Contains(contentStr, element) {
					t.Errorf("Config %s: Output missing required element: %s", config.name, element)
				}
			}

			t.Logf("Configuration %s completed successfully", config.name)
		})
	}
}

// BenchmarkE2ETranspilation benchmarks end-to-end transpilation performance
func BenchmarkE2ETranspilation(b *testing.B) {
	tempDir := b.TempDir()

	// Create a representative test file
	testContent := `package benchmark

import (
	"fmt"
	"time"
)

type BenchmarkData struct {
	ID        int
	Name      string
	Value     float64
	Timestamp time.Time
}

func ProcessBenchmarkData(data []BenchmarkData) map[int]float64 {
	result := make(map[int]float64)
	for _, item := range data {
		result[item.ID] = item.Value * 1.1
	}
	return result
}

func main() {
	data := []BenchmarkData{
		{ID: 1, Name: "test1", Value: 10.5, Timestamp: time.Now()},
		{ID: 2, Name: "test2", Value: 20.5, Timestamp: time.Now()},
	}
	
	result := ProcessBenchmarkData(data)
	fmt.Printf("Processed %d items\n", len(result))
}`

	inputFile := filepath.Join(tempDir, "benchmark.gofa")
	if err := os.WriteFile(inputFile, []byte(testContent), 0644); err != nil {
		b.Fatalf("Failed to create benchmark file: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		outputDir := filepath.Join(tempDir, fmt.Sprintf("output_%d", i))

		cmd := exec.Command(binaryPath, "-input", tempDir, "-output", outputDir, "-pattern", "benchmark.gofa")
		if err := cmd.Run(); err != nil {
			b.Fatalf("Benchmark transpilation failed: %v", err)
		}

		// Clean up for next iteration
		os.RemoveAll(outputDir)
	}
}
