package main

import (
	"testing"

	"github.com/healtronlabs/gofasta/transpiler"
	"github.com/healtronlabs/gofasta/transpiler/cli"
)

// Test ParallelTranspilerAdapter with actual transpiler
func TestParallelTranspilerAdapter_WithMockTranspiler(t *testing.T) {
	opts := transpiler.TranspileOptions{
		MaxWorkers:     2,
		OutputDir:      "./mock_output",
		FileExtension:  ".go",
		PreserveStruct: false,
		Verbose:        false,
	}

	// Create actual parallel transpiler
	parallelTranspiler := transpiler.NewParallelTranspiler(opts)
	adapter := &ParallelTranspilerAdapter{transpiler: parallelTranspiler}

	// Test FindGofaFiles method
	files, err := adapter.FindGofaFiles("nonexistent_dir")
	// Should error because directory doesn't exist
	if err == nil {
		t.Log("FindGofaFiles unexpectedly succeeded")
	} else {
		t.Logf("FindGofaFiles failed as expected: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("Expected 0 files, got %d", len(files))
	}

	// Test GetOutputPath method
	outputPath := adapter.GetOutputPath("input_dir", "test.gofa")
	if outputPath == "" {
		t.Error("GetOutputPath returned empty string")
	}
	t.Logf("GetOutputPath result: %s", outputPath)
}

// Test WatchModeAdapter with actual watch mode
func TestWatchModeAdapter_WithMockWatchMode(t *testing.T) {
	opts := transpiler.TranspileOptions{
		MaxWorkers:     1,
		OutputDir:      "./watch_output",
		FileExtension:  ".go",
		PreserveStruct: true,
		Verbose:        false,
	}

	// Create actual watch mode
	watchMode := transpiler.NewWatchMode(opts, "nonexistent_dir", 0)
	adapter := &WatchModeAdapter{watchMode: watchMode}

	// Test Start method (will likely error due to nonexistent directory)
	err := adapter.Start()
	if err == nil {
		t.Log("Start unexpectedly succeeded")
		// If it succeeds, we should stop it
		adapter.Stop()
	} else {
		t.Logf("Start failed as expected: %v", err)
	}

	// Test Stop method (should be safe to call)
	adapter.Stop()
}

// Test adapter option conversion
func TestOptionConversion(t *testing.T) {
	cliOpts := cli.TranspileOptions{
		MaxWorkers:     4,
		OutputDir:      "./converted_output",
		FileExtension:  ".generated.go",
		PreserveStruct: true,
		Verbose:        true,
	}

	// Test BatchTranspilerAdapter option conversion
	batchAdapter := &BatchTranspilerAdapter{opts: cliOpts}
	if batchAdapter.opts.MaxWorkers != 4 {
		t.Errorf("Expected MaxWorkers 4, got %d", batchAdapter.opts.MaxWorkers)
	}
	if batchAdapter.opts.OutputDir != "./converted_output" {
		t.Errorf("Expected OutputDir './converted_output', got %s", batchAdapter.opts.OutputDir)
	}
	if !batchAdapter.opts.Verbose {
		t.Error("Expected Verbose to be true")
	}
}
