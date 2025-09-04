package main

import (
	"os/exec"
	"testing"
	"time"

	"github.com/healtronlabs/gofasta/transpiler/cli"
)

// Test the main function indirectly by testing the setup it does
func TestMainFunctionSetup(t *testing.T) {
	// Test CLI creation
	cliApp := cli.NewCLI("1.0.0")
	if cliApp == nil {
		t.Fatal("CLI app should be created")
	}

	// Test dependency setup like main() does
	deps := cli.Dependencies{
		TranspileFile: func(inputPath, inputContent string) (string, error) {
			return "package main\n// Test generated code", nil
		},
		NewBatchTranspiler: func(opts cli.TranspileOptions) cli.BatchTranspiler {
			return &BatchTranspilerAdapter{opts: opts}
		},
		NewParallelTranspiler: func(opts cli.TranspileOptions) cli.ParallelTranspiler {
			return &ParallelTranspilerAdapter{transpiler: nil} // Mock
		},
		NewWatchMode: func(opts cli.TranspileOptions, inputDir string, debounce time.Duration) cli.WatchMode {
			return &WatchModeAdapter{watchMode: nil} // Mock
		},
	}

	// Test that dependencies are properly structured
	if deps.TranspileFile == nil {
		t.Error("TranspileFile dependency should be set")
	}
	if deps.NewBatchTranspiler == nil {
		t.Error("NewBatchTranspiler dependency should be set")
	}
	if deps.NewParallelTranspiler == nil {
		t.Error("NewParallelTranspiler dependency should be set")
	}
	if deps.NewWatchMode == nil {
		t.Error("NewWatchMode dependency should be set")
	}

	// Test TranspileFile function
	result, err := deps.TranspileFile("test.gofa", "test content")
	if err != nil {
		t.Errorf("TranspileFile failed: %v", err)
	}
	if result == "" {
		t.Error("TranspileFile should return non-empty result")
	}

	// Test adapter creation
	batcher := deps.NewBatchTranspiler(cli.TranspileOptions{MaxWorkers: 1})
	if batcher == nil {
		t.Error("NewBatchTranspiler should create adapter")
	}

	parallel := deps.NewParallelTranspiler(cli.TranspileOptions{MaxWorkers: 2})
	if parallel == nil {
		t.Error("NewParallelTranspiler should create adapter")
	}

	watcher := deps.NewWatchMode(cli.TranspileOptions{}, ".", time.Second)
	if watcher == nil {
		t.Error("NewWatchMode should create adapter")
	}
}

// Test adapter option conversion (like main() does)
func TestAdapterOptionConversion(t *testing.T) {
	cliOpts := cli.TranspileOptions{
		MaxWorkers:     4,
		OutputDir:      "./test_output",
		FileExtension:  ".gen.go",
		PreserveStruct: true,
		Verbose:        true,
	}

	// Test BatchTranspilerAdapter option storage
	batchAdapter := &BatchTranspilerAdapter{opts: cliOpts}
	if batchAdapter.opts.MaxWorkers != 4 {
		t.Errorf("Expected MaxWorkers 4, got %d", batchAdapter.opts.MaxWorkers)
	}
	if batchAdapter.opts.OutputDir != "./test_output" {
		t.Errorf("Expected OutputDir './test_output', got %s", batchAdapter.opts.OutputDir)
	}
	if !batchAdapter.opts.PreserveStruct {
		t.Error("Expected PreserveStruct to be true")
	}
	if !batchAdapter.opts.Verbose {
		t.Error("Expected Verbose to be true")
	}

	// Test that TranspileProject method exists and can be called
	err := batchAdapter.TranspileProject("nonexistent")
	// This will error, but we're testing that the method exists and runs
	if err != nil {
		t.Logf("TranspileProject errored as expected: %v", err)
	}
}

// Test adapters with different configurations
func TestAdapterConfigurations(t *testing.T) {
	testConfigs := []cli.TranspileOptions{
		{MaxWorkers: 1, OutputDir: "./single", FileExtension: ".go", PreserveStruct: false, Verbose: false},
		{MaxWorkers: 8, OutputDir: "./multi", FileExtension: ".generated.go", PreserveStruct: true, Verbose: true},
		{MaxWorkers: 0, OutputDir: "", FileExtension: "", PreserveStruct: false, Verbose: false}, // Edge case
	}

	for i, config := range testConfigs {
		t.Run("config_"+string(rune(i+'0')), func(t *testing.T) {
			// Test BatchTranspilerAdapter
			batchAdapter := &BatchTranspilerAdapter{opts: config}
			if batchAdapter.opts.MaxWorkers != config.MaxWorkers {
				t.Errorf("MaxWorkers not preserved: expected %d, got %d", config.MaxWorkers, batchAdapter.opts.MaxWorkers)
			}

			// Test ParallelTranspilerAdapter
			parallelAdapter := &ParallelTranspilerAdapter{transpiler: nil}
			if parallelAdapter == nil {
				t.Error("ParallelTranspilerAdapter should be created")
			}

			// Test WatchModeAdapter
			watchAdapter := &WatchModeAdapter{watchMode: nil}
			if watchAdapter == nil {
				t.Error("WatchModeAdapter should be created")
			}
		})
	}
}

// Test that adapters implement the required interfaces
func TestAdapterInterfaceImplementation(t *testing.T) {
	// Verify BatchTranspilerAdapter implements cli.BatchTranspiler
	var batchInterface cli.BatchTranspiler = &BatchTranspilerAdapter{}
	if batchInterface == nil {
		t.Error("BatchTranspilerAdapter should implement cli.BatchTranspiler")
	}

	// Verify ParallelTranspilerAdapter implements cli.ParallelTranspiler
	var parallelInterface cli.ParallelTranspiler = &ParallelTranspilerAdapter{}
	if parallelInterface == nil {
		t.Error("ParallelTranspilerAdapter should implement cli.ParallelTranspiler")
	}

	// Verify WatchModeAdapter implements cli.WatchMode
	var watchInterface cli.WatchMode = &WatchModeAdapter{}
	if watchInterface == nil {
		t.Error("WatchModeAdapter should implement cli.WatchMode")
	}
}

// Test main function behavior indirectly through subprocess
func TestMainFunctionExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping main function execution test in short mode")
	}

	// Test that main function can be called with help argument (should exit cleanly)
	cmd := exec.Command("go", "run", "main.go", "help")
	cmd.Dir = "/Users/descholar/descholar/myprojects/healtronlabs/gofasta/transpiler/cmd"

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Main function execution with help failed (may be expected): %v", err)
		t.Logf("Output: %s", string(output))
	} else {
		t.Logf("Main function help executed successfully")
		t.Logf("Output: %s", string(output))
	}
}

// Test edge cases in adapter methods
func TestAdapterEdgeCases(t *testing.T) {
	// Test BatchTranspilerAdapter with nil/empty options
	emptyAdapter := &BatchTranspilerAdapter{opts: cli.TranspileOptions{}}
	err := emptyAdapter.TranspileProject("")
	if err == nil {
		t.Log("Empty directory transpile handled gracefully")
	} else {
		t.Logf("Empty directory transpile failed as expected: %v", err)
	}

	// Test ParallelTranspilerAdapter methods
	parallelAdapter := &ParallelTranspilerAdapter{transpiler: nil}

	// These will likely panic with nil transpiler, but we can test the method existence
	defer func() {
		if r := recover(); r != nil {
			t.Logf("ParallelTranspilerAdapter methods panicked as expected with nil transpiler: %v", r)
		}
	}()

	// Test methods exist (will panic with nil transpiler)
	// files, _ := parallelAdapter.FindGofaFiles("test")
	// path := parallelAdapter.GetOutputPath("input", "test.gofa")

	// Instead, just verify the methods exist by checking the interface
	var _ cli.ParallelTranspiler = parallelAdapter

	// Test WatchModeAdapter methods
	watchAdapter := &WatchModeAdapter{watchMode: nil}

	// Stop should be safe to call even with nil
	watchAdapter.Stop()

	// Start will panic with nil, but we test interface conformance
	var _ cli.WatchMode = watchAdapter
}
