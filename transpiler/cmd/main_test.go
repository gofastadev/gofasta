package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseFlags(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name     string
		args     []string
		expected *Config
	}{
		{
			name: "default values",
			args: []string{"gofasta"},
			expected: &Config{
				InputDir:    ".",
				OutputDir:   ".",
				Pattern:     "*.gofa",
				Verbose:     false,
				Force:       false,
				DryRun:      false,
				Watch:       false,
				ShowVersion: false,
				ShowHelp:    false,
				Parallel:    4,
				CacheDir:    ".gofasta-cache",
				EnableCache: true,
				LogLevel:    "info",
			},
		},
		{
			name: "custom values",
			args: []string{"gofasta", "-input", "./src", "-output", "./dist", "-verbose", "-force", "-parallel", "8"},
			expected: &Config{
				InputDir:    "./src",
				OutputDir:   "./dist",
				Pattern:     "*.gofa",
				Verbose:     true,
				Force:       true,
				DryRun:      false,
				Watch:       false,
				ShowVersion: false,
				ShowHelp:    false,
				Parallel:    8,
				CacheDir:    ".gofasta-cache",
				EnableCache: true,
				LogLevel:    "info",
			},
		},
		{
			name: "short flags",
			args: []string{"gofasta", "-i", "./input", "-o", "./output", "-v", "-f", "-n"},
			expected: &Config{
				InputDir:    "./input",
				OutputDir:   "./output",
				Pattern:     "*.gofa",
				Verbose:     true,
				Force:       true,
				DryRun:      true,
				Watch:       false,
				ShowVersion: false,
				ShowHelp:    false,
				Parallel:    4,
				CacheDir:    ".gofasta-cache",
				EnableCache: true,
				LogLevel:    "info",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := parseFlagsFromArgs(tt.args)

			if config.InputDir != tt.expected.InputDir {
				t.Errorf("InputDir = %v, want %v", config.InputDir, tt.expected.InputDir)
			}
			if config.OutputDir != tt.expected.OutputDir {
				t.Errorf("OutputDir = %v, want %v", config.OutputDir, tt.expected.OutputDir)
			}
			if config.Verbose != tt.expected.Verbose {
				t.Errorf("Verbose = %v, want %v", config.Verbose, tt.expected.Verbose)
			}
			if config.Force != tt.expected.Force {
				t.Errorf("Force = %v, want %v", config.Force, tt.expected.Force)
			}
			if config.DryRun != tt.expected.DryRun {
				t.Errorf("DryRun = %v, want %v", config.DryRun, tt.expected.DryRun)
			}
			if config.Parallel != tt.expected.Parallel {
				t.Errorf("Parallel = %v, want %v", config.Parallel, tt.expected.Parallel)
			}
		})
	}
}

func TestCLI_getOutputPath(t *testing.T) {
	cli := &CLI{
		config: &Config{
			InputDir:  "./src",
			OutputDir: "./dist",
		},
	}

	tests := []struct {
		name      string
		inputFile string
		expected  string
	}{
		{
			name:      "simple gofa file",
			inputFile: "./src/hello.gofa",
			expected:  "dist/hello.go",
		},
		{
			name:      "nested gofa file",
			inputFile: "./src/controllers/user.gofa",
			expected:  "dist/controllers/user.go",
		},
		{
			name:      "same directory output",
			inputFile: "./test.gofa",
			expected:  "test.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cli.getOutputPath(tt.inputFile)
			if result != tt.expected {
				t.Errorf("getOutputPath(%s) = %s, want %s", tt.inputFile, result, tt.expected)
			}
		})
	}
}

func TestCLI_findGofastaFiles(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	
	// Create test files
	testFiles := []string{
		"hello.gofa",
		"controllers/user.gofa",
		"services/auth.gofa",
		"utils/helpers.go", // Should be ignored
		"README.md",        // Should be ignored
	}

	for _, file := range testFiles {
		dir := filepath.Dir(filepath.Join(tmpDir, file))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, file), []byte("test content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cli := &CLI{
		config: &Config{
			InputDir: tmpDir,
			Pattern:  "*.gofa",
		},
	}

	files, err := cli.findGofastaFiles()
	if err != nil {
		t.Fatalf("findGofastaFiles() error = %v", err)
	}

	expectedCount := 3 // Only .gofa files
	if len(files) != expectedCount {
		t.Errorf("findGofastaFiles() found %d files, want %d", len(files), expectedCount)
	}

	// Check that all found files end with .gofa
	for _, file := range files {
		if !strings.HasSuffix(file, ".gofa") {
			t.Errorf("findGofastaFiles() returned non-.gofa file: %s", file)
		}
	}
}

func TestCLI_initializeComponents(t *testing.T) {
	cli := &CLI{
		config: &Config{
			EnableCache: true,
			CacheDir:    t.TempDir(),
			Parallel:    2,
			Verbose:     true,
		},
	}

	err := cli.initializeComponents()
	if err != nil {
		t.Fatalf("initializeComponents() error = %v", err)
	}

	if cli.extractor == nil {
		t.Error("extractor not initialized")
	}
	if cli.generator == nil {
		t.Error("generator not initialized")
	}
	if cli.registry == nil {
		t.Error("registry not initialized")
	}
}

func TestTranspileResult(t *testing.T) {
	result := &TranspileResult{
		InputFile:  "test.gofa",
		OutputFile: "test.go",
		Success:    true,
		StartTime:  time.Now(),
		Duration:   time.Millisecond * 100,
		InputSize:  100,
		OutputSize: 150,
	}

	if result.InputFile != "test.gofa" {
		t.Errorf("InputFile = %v, want %v", result.InputFile, "test.gofa")
	}
	if result.OutputFile != "test.go" {
		t.Errorf("OutputFile = %v, want %v", result.OutputFile, "test.go")
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.InputSize != 100 {
		t.Errorf("InputSize = %v, want %v", result.InputSize, 100)
	}
	if result.OutputSize != 150 {
		t.Errorf("OutputSize = %v, want %v", result.OutputSize, 150)
	}
}

func TestShowHelp(t *testing.T) {
	// This test just ensures showHelp() doesn't panic
	// In a real test, you might capture stdout and verify the output
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("showHelp() panicked: %v", r)
		}
	}()
	
	// Redirect output to prevent cluttering test output
	oldStdout := os.Stdout
	os.Stdout = nil
	defer func() { os.Stdout = oldStdout }()
	
	showHelp()
}

func TestCLI_dryRun(t *testing.T) {
	cli := &CLI{
		config: &Config{
			InputDir:  "./src",
			OutputDir: "./dist",
		},
	}

	files := []string{"./src/test1.gofa", "./src/test2.gofa"}
	
	// Capture output to prevent cluttering test output
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()
	
	err := cli.dryRun(files)
	if err != nil {
		t.Errorf("dryRun() error = %v", err)
	}
}

// Benchmark tests for performance validation
func BenchmarkParseFlags(b *testing.B) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	
	os.Args = []string{"gofasta", "-input", "./src", "-output", "./dist", "-verbose"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parseFlags()
	}
}

func BenchmarkGetOutputPath(b *testing.B) {
	cli := &CLI{
		config: &Config{
			InputDir:  "./src",
			OutputDir: "./dist",
		},
	}
	
	inputFile := "./src/controllers/user.controller.gofa"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cli.getOutputPath(inputFile)
	}
}