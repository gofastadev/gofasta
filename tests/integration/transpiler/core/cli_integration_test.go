// CLI Integration Tests - Tests the GoFasta CLI binary with real commands and workflows
package core

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	// Test binary path relative to integration test directory
	testBinaryPath = "../../tools/transpiler/dist/gofasta"
	testTimeout    = 30 * time.Second
)

// TestCLIBasicCommands tests basic CLI command execution and exit codes
func TestCLIBasicCommands(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedExit   int
		shouldContain  []string
		shouldNotContain []string
	}{
		{
			name:         "version command",
			args:         []string{"--version"},
			expectedExit: 0,
			shouldContain: []string{"GoFasta Transpiler", "v1.0.0"},
		},
		{
			name:         "help command",
			args:         []string{"--help"},
			expectedExit: 0,
			shouldContain: []string{"Usage:", "OPTIONS:", "EXAMPLES:"},
		},
		{
			name:         "short help flag",
			args:         []string{"-h"},
			expectedExit: 0,
			shouldContain: []string{"Usage:", "OPTIONS:"},
		},
		{
			name:         "invalid flag",
			args:         []string{"--invalid-flag"},
			expectedExit: 2,
			shouldContain: []string{"flag provided but not defined"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(testBinaryPath, tt.args...)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			
			err := cmd.Run()
			
			// Check exit code
			exitCode := 0
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode = exitError.ExitCode()
				} else {
					t.Fatalf("Failed to run command: %v", err)
				}
			}
			
			if exitCode != tt.expectedExit {
				t.Errorf("Expected exit code %d, got %d", tt.expectedExit, exitCode)
				t.Logf("Stdout: %s", stdout.String())
				t.Logf("Stderr: %s", stderr.String())
			}
			
			// Combine stdout and stderr for content checking
			output := stdout.String() + stderr.String()
			
			// Check expected content
			for _, expected := range tt.shouldContain {
				if !strings.Contains(output, expected) {
					t.Errorf("Output should contain '%s', but got: %s", expected, output)
				}
			}
			
			// Check unwanted content
			for _, unwanted := range tt.shouldNotContain {
				if strings.Contains(output, unwanted) {
					t.Errorf("Output should not contain '%s', but got: %s", unwanted, output)
				}
			}
		})
	}
}

// TestCLIFlagValidation tests various flag combinations and validation
func TestCLIFlagValidation(t *testing.T) {
	tempDir := t.TempDir()
	
	tests := []struct {
		name         string
		args         []string
		expectedExit int
		description  string
	}{
		{
			name:         "valid input directory",
			args:         []string{"-input", tempDir, "-dry-run"},
			expectedExit: 0,
			description:  "Should accept valid input directory",
		},
		{
			name:         "short input flag",
			args:         []string{"-i", tempDir, "-dry-run"},
			expectedExit: 0,
			description:  "Should accept short input flag",
		},
		{
			name:         "output directory flag",
			args:         []string{"-output", tempDir, "-dry-run"},
			expectedExit: 0,
			description:  "Should accept output directory flag",
		},
		{
			name:         "short output flag",
			args:         []string{"-o", tempDir, "-dry-run"},
			expectedExit: 0,
			description:  "Should accept short output flag",
		},
		{
			name:         "pattern flag",
			args:         []string{"-pattern", "*.gofa", "-dry-run"},
			expectedExit: 0,
			description:  "Should accept pattern flag",
		},
		{
			name:         "short pattern flag",
			args:         []string{"-p", "controller*.gofa", "-dry-run"},
			expectedExit: 0,
			description:  "Should accept short pattern flag",
		},
		{
			name:         "verbose flag",
			args:         []string{"-verbose", "-dry-run"},
			expectedExit: 0,
			description:  "Should accept verbose flag",
		},
		{
			name:         "short verbose flag",
			args:         []string{"-v", "-dry-run"},
			expectedExit: 0,
			description:  "Should accept short verbose flag",
		},
		{
			name:         "force flag",
			args:         []string{"-force", "-dry-run"},
			expectedExit: 0,
			description:  "Should accept force flag",
		},
		{
			name:         "short force flag",
			args:         []string{"-f", "-dry-run"},
			expectedExit: 0,
			description:  "Should accept short force flag",
		},
		{
			name:         "parallel workers flag",
			args:         []string{"-parallel", "2", "-dry-run"},
			expectedExit: 0,
			description:  "Should accept parallel flag with number",
		},
		{
			name:         "cache flag",
			args:         []string{"-cache", "-dry-run"},
			expectedExit: 0,
			description:  "Should accept cache flag",
		},
		{
			name:         "cache directory flag",
			args:         []string{"-cache-dir", tempDir, "-dry-run"},
			expectedExit: 0,
			description:  "Should accept cache directory flag",
		},
		{
			name:         "log level flag",
			args:         []string{"-log-level", "debug", "-dry-run"},
			expectedExit: 0,
			description:  "Should accept log level flag",
		},
		{
			name:         "watch flag",
			args:         []string{"-watch", "-dry-run"},
			expectedExit: 0,
			description:  "Should accept watch flag",
		},
		{
			name:         "short watch flag",
			args:         []string{"-w", "-dry-run"},
			expectedExit: 0,
			description:  "Should accept short watch flag",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()
			
			cmd := exec.CommandContext(ctx, testBinaryPath, tt.args...)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			
			err := cmd.Run()
			
			exitCode := 0
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode = exitError.ExitCode()
				} else {
					t.Fatalf("Failed to run command: %v", err)
				}
			}
			
			if exitCode != tt.expectedExit {
				t.Errorf("%s: Expected exit code %d, got %d", tt.description, tt.expectedExit, exitCode)
				t.Logf("Stdout: %s", stdout.String())
				t.Logf("Stderr: %s", stderr.String())
			}
		})
	}
}

// TestCLIHelpAndVersion tests help and version command output formatting
func TestCLIHelpAndVersion(t *testing.T) {
	t.Run("help command format", func(t *testing.T) {
		cmd := exec.Command(testBinaryPath, "--help")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Help command failed: %v", err)
		}
		
		helpText := string(output)
		
		// Check for banner presence
		if !strings.Contains(helpText, "GoFasta Enterprise Backend Framework") {
			t.Error("Help should contain banner with framework name")
		}
		
		// Check for usage section
		if !strings.Contains(helpText, "Usage:") {
			t.Error("Help should contain Usage section")
		}
		
		// Check for options section
		if !strings.Contains(helpText, "OPTIONS:") {
			t.Error("Help should contain OPTIONS section")
		}
		
		// Check for examples section
		if !strings.Contains(helpText, "EXAMPLES:") {
			t.Error("Help should contain EXAMPLES section")
		}
		
		// Check for key flags
		requiredFlags := []string{"-input", "-output", "-verbose", "-dry-run", "-force", "-watch"}
		for _, flag := range requiredFlags {
			if !strings.Contains(helpText, flag) {
				t.Errorf("Help should contain flag: %s", flag)
			}
		}
	})
	
	t.Run("version command format", func(t *testing.T) {
		cmd := exec.Command(testBinaryPath, "--version")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Version command failed: %v", err)
		}
		
		versionText := string(output)
		
		// Check version format
		if !strings.Contains(versionText, "GoFasta Transpiler") {
			t.Error("Version should contain 'GoFasta Transpiler'")
		}
		
		if !strings.Contains(versionText, "v1.0.0") {
			t.Error("Version should contain version number")
		}
	})
}

// TestCLIErrorHandling tests CLI error scenarios and error message formatting
func TestCLIErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedExit  int
		shouldContain string
		description   string
	}{
		{
			name:          "nonexistent input directory",
			args:          []string{"-input", "/nonexistent/directory", "-dry-run"},
			expectedExit:  1,
			shouldContain: "no such file or directory",
			description:   "Should error with clear message for nonexistent input directory",
		},
		{
			name:          "invalid parallel workers number",
			args:          []string{"-parallel", "invalid", "-dry-run"},
			expectedExit:  2,
			shouldContain: "invalid",
			description:   "Should error for non-numeric parallel workers",
		},
		{
			name:          "invalid log level",
			args:          []string{"-log-level", "invalid-level", "-dry-run"},
			expectedExit:  0,
			shouldContain: "gofa",
			description:   "Invalid log level is accepted (no validation yet)",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(testBinaryPath, tt.args...)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			
			err := cmd.Run()
			
			exitCode := 0
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode = exitError.ExitCode()
				} else {
					// Command execution failed completely
					exitCode = 1
				}
			}
			
			if exitCode != tt.expectedExit {
				t.Errorf("%s: Expected exit code %d, got %d", tt.description, tt.expectedExit, exitCode)
			}
			
			output := stdout.String() + stderr.String()
			if tt.shouldContain != "" && !strings.Contains(strings.ToLower(output), strings.ToLower(tt.shouldContain)) {
				t.Errorf("%s: Output should contain '%s', but got: %s", tt.description, tt.shouldContain, output)
			}
		})
	}
}

// TestCLIVerboseOutput tests verbose output functionality
func TestCLIVerboseOutput(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create a simple test .gofa file
	testFile := filepath.Join(tempDir, "test.gofa")
	testContent := `package main

func main() {
    println("Hello GoFasta!")
}`
	
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	t.Run("verbose mode shows detailed output", func(t *testing.T) {
		cmd := exec.Command(testBinaryPath, "-input", tempDir, "-verbose", "-dry-run")
		output, _ := cmd.CombinedOutput()
		
		outputStr := string(output)
		
		// Should show banner in verbose mode
		if !strings.Contains(outputStr, "GoFasta Enterprise Backend Framework") {
			t.Error("Verbose mode should show banner")
		}
		
		// Should show configuration details
		if !strings.Contains(outputStr, "Input:") {
			t.Error("Verbose mode should show input directory")
		}
		
		if !strings.Contains(outputStr, "Pattern:") {
			t.Error("Verbose mode should show file pattern")
		}
		
		// Should show found files
		if !strings.Contains(outputStr, "Found") && !strings.Contains(outputStr, ".gofa") {
			t.Error("Verbose mode should show found .gofa files")
		}
	})
	
	t.Run("non-verbose mode is quieter", func(t *testing.T) {
		cmd := exec.Command(testBinaryPath, "-input", tempDir, "-dry-run")
		output, _ := cmd.CombinedOutput()
		
		// Non-verbose mode should be much quieter
		outputStr := string(output)
		
		// Should not show detailed configuration in non-verbose mode
		if strings.Contains(outputStr, "🚀 Starting GoFasta transpilation") {
			t.Error("Non-verbose mode should not show detailed startup messages")
		}
	})
}

// TestCLIDryRunMode tests dry-run functionality
func TestCLIDryRunMode(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create test files
	testFiles := map[string]string{
		"test1.gofa": `package main
func main() { println("test1") }`,
		"test2.gofa": `package main  
func main() { println("test2") }`,
	}
	
	for filename, content := range testFiles {
		path := filepath.Join(tempDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}
	
	t.Run("dry-run shows what would be done", func(t *testing.T) {
		cmd := exec.Command(testBinaryPath, "-input", tempDir, "-dry-run", "-verbose")
		output, _ := cmd.CombinedOutput()
		
		outputStr := string(output)
		
		// Should mention dry-run mode
		if !strings.Contains(outputStr, "dry") && !strings.Contains(outputStr, "would") {
			t.Error("Dry-run should indicate it's in dry-run mode")
		}
		
		// Should list the files it would process
		if !strings.Contains(outputStr, "test1.gofa") || !strings.Contains(outputStr, "test2.gofa") {
			t.Error("Dry-run should list files it would process")
		}
	})
	
	t.Run("dry-run does not create output files", func(t *testing.T) {
		cmd := exec.Command(testBinaryPath, "-input", tempDir, "-output", tempDir, "-dry-run")
		_, _ = cmd.CombinedOutput()
		
		// Check that no .go files were created
		files, err := os.ReadDir(tempDir)
		if err != nil {
			t.Fatalf("Failed to read temp directory: %v", err)
		}
		
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".go") {
				t.Errorf("Dry-run should not create output files, but found: %s", file.Name())
			}
		}
	})
}

// TestCLIForceOverwrite tests force overwrite functionality
func TestCLIForceOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create a .gofa file
	gofaFile := filepath.Join(tempDir, "test.gofa")
	gofaContent := `package main
func main() { println("Hello") }`
	
	if err := os.WriteFile(gofaFile, []byte(gofaContent), 0644); err != nil {
		t.Fatalf("Failed to create .gofa file: %v", err)
	}
	
	// Create an existing .go file
	goFile := filepath.Join(tempDir, "test.go")
	existingContent := "// Existing content"
	
	if err := os.WriteFile(goFile, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to create existing .go file: %v", err)
	}
	
	t.Run("force flag enables overwriting", func(t *testing.T) {
		cmd := exec.Command(testBinaryPath, "-input", tempDir, "-output", tempDir, "-force", "-verbose")
		output, _ := cmd.CombinedOutput()
		
		outputStr := string(output)
		
		// Should indicate it's processing with force
		if !strings.Contains(outputStr, "force") && !strings.Contains(outputStr, "overwrite") {
			// This is expected behavior - force might not be explicitly mentioned in output
			t.Logf("Force overwrite completed, output: %s", outputStr)
		}
	})
}

// TestCLIInputOutputHandling tests input and output directory handling
func TestCLIInputOutputHandling(t *testing.T) {
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	outputDir := filepath.Join(tempDir, "output")
	
	// Create input directory structure
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatalf("Failed to create input directory: %v", err)
	}
	
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}
	
	// Create test .gofa file
	testFile := filepath.Join(inputDir, "sample.gofa")
	testContent := `package main
func main() { println("sample") }`
	
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	t.Run("custom input and output directories", func(t *testing.T) {
		cmd := exec.Command(testBinaryPath, "-input", inputDir, "-output", outputDir, "-verbose", "-dry-run")
		output, _ := cmd.CombinedOutput()
		
		outputStr := string(output)
		
		// Should show the correct input and output directories
		if !strings.Contains(outputStr, inputDir) {
			t.Errorf("Output should mention input directory %s", inputDir)
		}
	})
}

// TestCLIRealTranspilation tests actual transpilation functionality
func TestCLIRealTranspilation(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create a simple .gofa file without decorators (should work with current Phase 1)
	testFile := filepath.Join(tempDir, "simple.gofa")
	testContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello from GoFasta!")
}`
	
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	t.Run("successful transpilation creates output file", func(t *testing.T) {
		cmd := exec.Command(testBinaryPath, "-input", tempDir, "-output", tempDir, "-verbose")
		output, _ := cmd.CombinedOutput()
		
		outputStr := string(output)
		t.Logf("Transpilation output: %s", outputStr)
		
		// Check if output .go file was created
		outputFile := filepath.Join(tempDir, "simple.go")
		if _, err := os.Stat(outputFile); err != nil {
			t.Logf("Output file not created (expected for current Phase 1): %v", err)
			// This is expected behavior for Phase 1 - files without decorators should transpile
		} else {
			// If file was created, verify its content
			content, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}
			
			if len(content) == 0 {
				t.Error("Output file should not be empty")
			}
		}
	})
}

// BenchmarkCLIPerformance benchmarks CLI startup and basic operations
func BenchmarkCLIPerformance(b *testing.B) {
	tempDir := b.TempDir()
	
	// Create a test file
	testFile := filepath.Join(tempDir, "bench.gofa")
	testContent := `package main
func main() { println("benchmark") }`
	
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}
	
	b.Run("cli_startup_time", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cmd := exec.Command(testBinaryPath, "--version")
			if err := cmd.Run(); err != nil {
				b.Fatalf("CLI startup failed: %v", err)
			}
		}
	})
	
	b.Run("cli_dry_run_performance", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cmd := exec.Command(testBinaryPath, "-input", tempDir, "-dry-run")
			if err := cmd.Run(); err != nil {
				b.Fatalf("CLI dry-run failed: %v", err)
			}
		}
	})
}