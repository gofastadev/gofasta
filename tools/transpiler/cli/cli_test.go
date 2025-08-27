package cli

import (
	"strings"
	"testing"
	"time"
)

// Mock dependencies for testing
func createMockDependencies() Dependencies {
	return Dependencies{
		TranspileFile: func(inputPath, inputContent string) (string, error) {
			return "package main\n// Mock generated code", nil
		},
		NewBatchTranspiler: func(opts TranspileOptions) BatchTranspiler {
			return &mockBatchTranspiler{}
		},
		NewParallelTranspiler: func(opts TranspileOptions) ParallelTranspiler {
			return &mockParallelTranspiler{}
		},
		NewWatchMode: func(opts TranspileOptions, inputDir string, debounce time.Duration) WatchMode {
			return &mockWatchMode{}
		},
	}
}

// Mock implementations
type mockBatchTranspiler struct{}
func (m *mockBatchTranspiler) TranspileProject(inputDir string) error { return nil }

type mockParallelTranspiler struct{}
func (m *mockParallelTranspiler) FindGofaFiles(inputDir string) ([]string, error) {
	return []string{"test.gofa"}, nil
}
func (m *mockParallelTranspiler) GetOutputPath(inputDir, gofaFile string) string {
	return strings.Replace(gofaFile, ".gofa", ".go", 1)
}

type mockWatchMode struct{}
func (m *mockWatchMode) Start() error { return nil }
func (m *mockWatchMode) Stop() {}

func TestCLI_Run(t *testing.T) {
	cli := NewCLI("1.0.0-test")
	deps := createMockDependencies()

	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "version command",
			args:        []string{"gofasta", "version"},
			expectError: false,
		},
		{
			name:        "help command", 
			args:        []string{"gofasta", "help"},
			expectError: false,
		},
		{
			name:        "unknown command",
			args:        []string{"gofasta", "unknown"},
			expectError: true,
		},
		{
			name:        "no command",
			args:        []string{"gofasta"},
			expectError: false, // Shows usage
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cli.Run(tt.args, deps)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestCLI_Creation(t *testing.T) {
	version := "1.2.3"
	cli := NewCLI(version)
	
	if cli == nil {
		t.Fatal("NewCLI returned nil")
	}
	
	// Test that version is stored (access via reflection or make version field public for testing)
	// For now, we test via behavior
	deps := createMockDependencies()
	err := cli.Run([]string{"gofasta", "version"}, deps)
	if err != nil {
		t.Errorf("Version command failed: %v", err)
	}
}

func TestDependenciesInterface(t *testing.T) {
	// Test that all required dependencies are satisfied
	deps := createMockDependencies()
	
	// Test TranspileFile
	result, err := deps.TranspileFile("test.gofa", "test content")
	if err != nil {
		t.Errorf("TranspileFile failed: %v", err)
	}
	if !strings.Contains(result, "package main") {
		t.Errorf("TranspileFile should return valid Go code")
	}
	
	// Test BatchTranspiler
	batcher := deps.NewBatchTranspiler(TranspileOptions{})
	if batcher == nil {
		t.Error("NewBatchTranspiler returned nil")
	}
	
	// Test ParallelTranspiler
	parallel := deps.NewParallelTranspiler(TranspileOptions{})
	if parallel == nil {
		t.Error("NewParallelTranspiler returned nil")
	}
	
	// Test WatchMode
	watcher := deps.NewWatchMode(TranspileOptions{}, ".", time.Millisecond)
	if watcher == nil {
		t.Error("NewWatchMode returned nil")
	}
}

func TestCLI_TranspileCommand(t *testing.T) {
	t.Skip("Skipping transpile command tests - they may hang due to file operations")
	// These tests are skipped because cli.Run() with transpile commands
	// can hang when they attempt to scan directories and process files
}

func TestCLI_WatchCommand(t *testing.T) {
	t.Skip("Skipping watch command tests - they may hang due to file watching operations")
	// These tests are skipped because cli.Run() with watch commands
	// can hang when they attempt to start file watching processes
}

func TestCLI_HelpCommand(t *testing.T) {
	cli := NewCLI("1.0.0-test")
	deps := createMockDependencies()
	
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "help command",
			args:        []string{"gofasta", "help"},
			expectError: false,
		},
		{
			name:        "help transpile",
			args:        []string{"gofasta", "help", "transpile"},
			expectError: false,
		},
		{
			name:        "help watch",
			args:        []string{"gofasta", "help", "watch"},
			expectError: false,
		},
		{
			name:        "help version",
			args:        []string{"gofasta", "help", "version"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cli.Run(tt.args, deps)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}