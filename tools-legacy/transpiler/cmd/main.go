package main

import (
	"fmt"
	"os"
	"time"

	"github.com/healtronlabs/gofasta/tools/transpiler"
	"github.com/healtronlabs/gofasta/tools/transpiler/cli"
)

func main() {
	cliApp := cli.NewCLI("1.0.0")

	// Inject dependencies that bridge to our modular packages
	deps := cli.Dependencies{
		TranspileFile: transpiler.TranspileFile,
		NewBatchTranspiler: func(opts cli.TranspileOptions) cli.BatchTranspiler {
			return &BatchTranspilerAdapter{opts: opts}
		},
		NewParallelTranspiler: func(opts cli.TranspileOptions) cli.ParallelTranspiler {
			// Convert CLI options to transpiler options
			transpilerOpts := transpiler.TranspileOptions{
				MaxWorkers:     opts.MaxWorkers,
				OutputDir:      opts.OutputDir,
				FileExtension:  opts.FileExtension,
				PreserveStruct: opts.PreserveStruct,
				Verbose:        opts.Verbose,
			}
			return &ParallelTranspilerAdapter{
				transpiler: transpiler.NewParallelTranspiler(transpilerOpts),
			}
		},
		NewWatchMode: func(opts cli.TranspileOptions, inputDir string, debounce time.Duration) cli.WatchMode {
			// Convert CLI options to transpiler options
			transpilerOpts := transpiler.TranspileOptions{
				MaxWorkers:     opts.MaxWorkers,
				OutputDir:      opts.OutputDir,
				FileExtension:  opts.FileExtension,
				PreserveStruct: opts.PreserveStruct,
				Verbose:        opts.Verbose,
			}
			return &WatchModeAdapter{
				watchMode: transpiler.NewWatchMode(transpilerOpts, inputDir, debounce),
			}
		},
	}

	if err := cliApp.Run(os.Args, deps); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// BatchTranspilerAdapter adapts the existing batch transpiler  
type BatchTranspilerAdapter struct {
	opts cli.TranspileOptions
}

func (b *BatchTranspilerAdapter) TranspileProject(inputDir string) error {
	// Convert CLI options to transpiler options
	transpilerOpts := transpiler.TranspileOptions{
		MaxWorkers:     b.opts.MaxWorkers,
		OutputDir:      b.opts.OutputDir,
		FileExtension:  b.opts.FileExtension,
		PreserveStruct: b.opts.PreserveStruct,
		Verbose:        b.opts.Verbose,
	}
	
	batchTranspiler := transpiler.NewBatchTranspiler(transpilerOpts)
	return batchTranspiler.TranspileProject(inputDir)
}

// ParallelTranspilerAdapter adapts the existing parallel transpiler
type ParallelTranspilerAdapter struct {
	transpiler *transpiler.ParallelTranspiler
}

func (p *ParallelTranspilerAdapter) FindGofaFiles(inputDir string) ([]string, error) {
	return p.transpiler.FindGofaFiles(inputDir)
}

func (p *ParallelTranspilerAdapter) GetOutputPath(inputDir, gofaFile string) string {
	return p.transpiler.GetOutputPath(inputDir, gofaFile)
}

// WatchModeAdapter adapts the existing watch mode
type WatchModeAdapter struct {
	watchMode *transpiler.WatchMode
}

func (w *WatchModeAdapter) Start() error {
	return w.watchMode.Start()
}

func (w *WatchModeAdapter) Stop() {
	w.watchMode.Stop()
}
