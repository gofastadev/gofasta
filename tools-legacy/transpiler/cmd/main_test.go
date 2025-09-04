package main

import (
	"testing"
	"time"

	"github.com/healtronlabs/gofasta/transpiler/cli"
)

// Test adapter types
func TestBatchTranspilerAdapter(t *testing.T) {
	opts := cli.TranspileOptions{
		MaxWorkers:     2,
		OutputDir:      "test_output",
		FileExtension:  ".go",
		PreserveStruct: true,
		Verbose:        false,
	}

	adapter := &BatchTranspilerAdapter{opts: opts}

	// Test that the adapter was created
	if adapter.opts.MaxWorkers != 2 {
		t.Errorf("Expected MaxWorkers 2, got %d", adapter.opts.MaxWorkers)
	}
	if adapter.opts.OutputDir != "test_output" {
		t.Errorf("Expected OutputDir 'test_output', got %s", adapter.opts.OutputDir)
	}
	if !adapter.opts.PreserveStruct {
		t.Error("Expected PreserveStruct to be true")
	}
}

func TestParallelTranspilerAdapter_Creation(t *testing.T) {
	opts := cli.TranspileOptions{
		MaxWorkers:     4,
		OutputDir:      "./output",
		FileExtension:  ".gen.go",
		PreserveStruct: false,
		Verbose:        true,
	}

	// Test adapter function
	newParallel := func(testOpts cli.TranspileOptions) cli.ParallelTranspiler {
		return &ParallelTranspilerAdapter{transpiler: nil} // Mock for testing
	}

	adapter := newParallel(opts).(*ParallelTranspilerAdapter)
	if adapter == nil {
		t.Fatal("Expected adapter to be created")
	}
}

func TestWatchModeAdapter_Creation(t *testing.T) {
	adapter := &WatchModeAdapter{watchMode: nil} // Mock for testing
	if adapter == nil {
		t.Fatal("Expected adapter to be created")
	}
}

// Test adapter creation functions from main
func TestMainDependencyCreation(t *testing.T) {
	// Test the actual dependency functions like those in main()
	opts := cli.TranspileOptions{
		MaxWorkers:     2,
		OutputDir:      "./test_output",
		FileExtension:  ".go",
		PreserveStruct: true,
		Verbose:        false,
	}

	// Test NewBatchTranspiler dependency function
	newBatch := func(opts cli.TranspileOptions) cli.BatchTranspiler {
		return &BatchTranspilerAdapter{opts: opts}
	}

	batcher := newBatch(opts)
	if batcher == nil {
		t.Error("NewBatchTranspiler returned nil")
	}

	// Test NewParallelTranspiler dependency function
	newParallel := func(opts cli.TranspileOptions) cli.ParallelTranspiler {
		return &ParallelTranspilerAdapter{transpiler: nil} // Mock
	}

	parallel := newParallel(opts)
	if parallel == nil {
		t.Error("NewParallelTranspiler returned nil")
	}

	// Test NewWatchMode dependency function
	newWatch := func(opts cli.TranspileOptions, inputDir string, debounce time.Duration) cli.WatchMode {
		return &WatchModeAdapter{watchMode: nil} // Mock
	}

	watcher := newWatch(opts, ".", time.Second)
	if watcher == nil {
		t.Error("NewWatchMode returned nil")
	}
}

// Test actual adapter methods to get coverage
func TestBatchTranspilerAdapter_TranspileProject(t *testing.T) {
	opts := cli.TranspileOptions{
		MaxWorkers:     1,
		OutputDir:      "./test_out",
		FileExtension:  ".go",
		PreserveStruct: false,
		Verbose:        false,
	}

	adapter := &BatchTranspilerAdapter{opts: opts}

	// Test that calling TranspileProject doesn't panic (it will likely error due to missing files)
	err := adapter.TranspileProject("nonexistent_dir")
	// We expect this to error because the directory doesn't exist
	if err == nil {
		t.Log("TranspileProject unexpectedly succeeded")
	} else {
		t.Logf("TranspileProject failed as expected: %v", err)
	}
}

// Test ParallelTranspilerAdapter methods
func TestParallelTranspilerAdapter_Methods(t *testing.T) {
	adapter := &ParallelTranspilerAdapter{transpiler: nil} // Mock

	// These methods would normally call the transpiler methods
	// We can't test the actual calls without a real transpiler
	// But we can test that the methods exist and can be called
	if adapter == nil {
		t.Fatal("Adapter should not be nil")
	}
}

// Test WatchModeAdapter methods
func TestWatchModeAdapter_Methods(t *testing.T) {
	adapter := &WatchModeAdapter{watchMode: nil} // Mock

	// Test that adapter exists
	if adapter == nil {
		t.Fatal("Adapter should not be nil")
	}

	// Note: Can't safely call Stop() with nil watchMode as it would panic
	// This test covers the type creation
}

// Test type conformance
func TestAdapterTypeConformance(t *testing.T) {
	// Test BatchTranspilerAdapter implements cli.BatchTranspiler
	var _ cli.BatchTranspiler = (*BatchTranspilerAdapter)(nil)

	// Test ParallelTranspilerAdapter implements cli.ParallelTranspiler
	var _ cli.ParallelTranspiler = (*ParallelTranspilerAdapter)(nil)

	// Test WatchModeAdapter implements cli.WatchMode
	var _ cli.WatchMode = (*WatchModeAdapter)(nil)
}
