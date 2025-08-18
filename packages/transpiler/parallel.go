package transpiler

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// TranspileResult represents the result of transpiling a single file
type TranspileResult struct {
	InputPath   string
	OutputPath  string
	Content     string
	Error       error
	Duration    time.Duration
}

// TranspileJob represents a single transpilation job
type TranspileJob struct {
	InputPath  string
	OutputPath string
	Content    string
}

// ParallelTranspiler handles parallel transpilation of multiple .gofa files
type ParallelTranspiler struct {
	maxWorkers     int
	outputDir      string
	fileExtension  string
	preserveStruct bool
	verbose        bool
}

// TranspileOptions configures the parallel transpiler
type TranspileOptions struct {
	MaxWorkers     int    // Maximum number of worker goroutines
	OutputDir      string // Output directory for .go files
	FileExtension  string // Output file extension (default: .go)
	PreserveStruct bool   // Preserve directory structure
	Verbose        bool   // Enable verbose logging
}

// NewParallelTranspiler creates a new parallel transpiler
func NewParallelTranspiler(opts TranspileOptions) *ParallelTranspiler {
	maxWorkers := opts.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}
	
	fileExt := opts.FileExtension
	if fileExt == "" {
		fileExt = ".go"
	}
	
	return &ParallelTranspiler{
		maxWorkers:     maxWorkers,
		outputDir:      opts.OutputDir,
		fileExtension:  fileExt,
		preserveStruct: opts.PreserveStruct,
		verbose:        opts.Verbose,
	}
}

// TranspileDirectory transpiles all .gofa files in a directory recursively
func (pt *ParallelTranspiler) TranspileDirectory(ctx context.Context, inputDir string) ([]TranspileResult, error) {
	// Find all .gofa files
	gofaFiles, err := pt.findGofaFiles(inputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find .gofa files: %w", err)
	}
	
	if len(gofaFiles) == 0 {
		return []TranspileResult{}, nil
	}
	
	if pt.verbose {
		fmt.Printf("Found %d .gofa files to transpile\n", len(gofaFiles))
	}
	
	// Create transpile jobs
	jobs := make([]TranspileJob, 0, len(gofaFiles))
	for _, gofaFile := range gofaFiles {
		outputPath := pt.getOutputPath(inputDir, gofaFile)
		
		// Read file content
		content, err := os.ReadFile(gofaFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", gofaFile, err)
		}
		
		jobs = append(jobs, TranspileJob{
			InputPath:  gofaFile,
			OutputPath: outputPath,
			Content:    string(content),
		})
	}
	
	// Transpile files in parallel
	return pt.transpileJobs(ctx, jobs)
}

// TranspileFiles transpiles a list of specific files
func (pt *ParallelTranspiler) TranspileFiles(ctx context.Context, filePaths []string) ([]TranspileResult, error) {
	jobs := make([]TranspileJob, 0, len(filePaths))
	
	for _, inputPath := range filePaths {
		if !strings.HasSuffix(inputPath, ".gofa") {
			continue
		}
		
		// Read file content
		content, err := os.ReadFile(inputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", inputPath, err)
		}
		
		outputPath := pt.getOutputPathForFile(inputPath)
		
		jobs = append(jobs, TranspileJob{
			InputPath:  inputPath,
			OutputPath: outputPath,
			Content:    string(content),
		})
	}
	
	return pt.transpileJobs(ctx, jobs)
}

// transpileJobs executes transpilation jobs in parallel
func (pt *ParallelTranspiler) transpileJobs(ctx context.Context, jobs []TranspileJob) ([]TranspileResult, error) {
	jobChan := make(chan TranspileJob, len(jobs))
	resultChan := make(chan TranspileResult, len(jobs))
	
	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < pt.maxWorkers; i++ {
		wg.Add(1)
		go pt.worker(ctx, &wg, jobChan, resultChan)
	}
	
	// Send jobs to workers
	go func() {
		defer close(jobChan)
		for _, job := range jobs {
			select {
			case jobChan <- job:
			case <-ctx.Done():
				return
			}
		}
	}()
	
	// Wait for workers to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	
	// Collect results
	var results []TranspileResult
	for result := range resultChan {
		results = append(results, result)
		
		if pt.verbose {
			if result.Error != nil {
				fmt.Printf("❌ %s: %v (took %v)\n", result.InputPath, result.Error, result.Duration)
			} else {
				fmt.Printf("✅ %s -> %s (took %v)\n", result.InputPath, result.OutputPath, result.Duration)
			}
		}
	}
	
	// Check for context cancellation
	if ctx.Err() != nil {
		return results, ctx.Err()
	}
	
	return results, nil
}

// worker processes transpilation jobs
func (pt *ParallelTranspiler) worker(ctx context.Context, wg *sync.WaitGroup, jobs <-chan TranspileJob, results chan<- TranspileResult) {
	defer wg.Done()
	
	for {
		select {
		case job, ok := <-jobs:
			if !ok {
				return
			}
			
			start := time.Now()
			result := pt.processJob(job)
			result.Duration = time.Since(start)
			
			select {
			case results <- result:
			case <-ctx.Done():
				return
			}
			
		case <-ctx.Done():
			return
		}
	}
}

// processJob processes a single transpilation job
func (pt *ParallelTranspiler) processJob(job TranspileJob) TranspileResult {
	result := TranspileResult{
		InputPath:  job.InputPath,
		OutputPath: job.OutputPath,
	}
	
	// Transpile the file
	goCode, err := TranspileFile(job.InputPath, job.Content)
	if err != nil {
		result.Error = fmt.Errorf("transpilation failed: %w", err)
		return result
	}
	
	result.Content = goCode
	
	// Ensure output directory exists
	outputDir := filepath.Dir(job.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create output directory: %w", err)
		return result
	}
	
	// Write the generated Go code
	if err := os.WriteFile(job.OutputPath, []byte(goCode), 0644); err != nil {
		result.Error = fmt.Errorf("failed to write output file: %w", err)
		return result
	}
	
	return result
}

// findGofaFiles finds all .gofa files in a directory recursively
func (pt *ParallelTranspiler) findGofaFiles(rootDir string) ([]string, error) {
	var gofaFiles []string
	
	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if !d.IsDir() && strings.HasSuffix(path, ".gofa") {
			gofaFiles = append(gofaFiles, path)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return gofaFiles, nil
}

// getOutputPath generates output path for a .gofa file
func (pt *ParallelTranspiler) getOutputPath(inputDir, gofaFile string) string {
	if pt.preserveStruct {
		// Preserve directory structure
		relPath, _ := filepath.Rel(inputDir, gofaFile)
		relPath = strings.TrimSuffix(relPath, ".gofa") + pt.fileExtension
		return filepath.Join(pt.outputDir, relPath)
	}
	
	// Flatten structure
	baseName := filepath.Base(gofaFile)
	baseName = strings.TrimSuffix(baseName, ".gofa") + pt.fileExtension
	return filepath.Join(pt.outputDir, baseName)
}

// getOutputPathForFile generates output path for a single file
func (pt *ParallelTranspiler) getOutputPathForFile(inputPath string) string {
	if pt.outputDir != "" {
		baseName := filepath.Base(inputPath)
		baseName = strings.TrimSuffix(baseName, ".gofa") + pt.fileExtension
		return filepath.Join(pt.outputDir, baseName)
	}
	
	// Same directory as input file
	return strings.TrimSuffix(inputPath, ".gofa") + pt.fileExtension
}

// TranspileStats provides statistics about transpilation
type TranspileStats struct {
	TotalFiles      int
	SuccessfulFiles int
	FailedFiles     int
	TotalDuration   time.Duration
	AverageDuration time.Duration
	ErrorSummary    map[string]int
}

// GetStats calculates statistics from transpile results
func GetStats(results []TranspileResult) TranspileStats {
	stats := TranspileStats{
		TotalFiles:   len(results),
		ErrorSummary: make(map[string]int),
	}
	
	var totalDuration time.Duration
	
	for _, result := range results {
		totalDuration += result.Duration
		
		if result.Error != nil {
			stats.FailedFiles++
			errorType := getErrorType(result.Error)
			stats.ErrorSummary[errorType]++
		} else {
			stats.SuccessfulFiles++
		}
	}
	
	stats.TotalDuration = totalDuration
	if stats.TotalFiles > 0 {
		stats.AverageDuration = totalDuration / time.Duration(stats.TotalFiles)
	}
	
	return stats
}

// getErrorType categorizes error types for statistics
func getErrorType(err error) string {
	errStr := err.Error()
	
	if strings.Contains(errStr, "parsing") {
		return "parsing_error"
	} else if strings.Contains(errStr, "lexer") || strings.Contains(errStr, "token") {
		return "lexer_error"
	} else if strings.Contains(errStr, "generation") || strings.Contains(errStr, "codegen") {
		return "codegen_error"
	} else if strings.Contains(errStr, "file") || strings.Contains(errStr, "directory") {
		return "io_error"
	}
	
	return "other_error"
}

// PrintStats prints transpilation statistics
func PrintStats(stats TranspileStats) {
	fmt.Println("\n📊 Transpilation Statistics:")
	fmt.Printf("Total files: %d\n", stats.TotalFiles)
	fmt.Printf("✅ Successful: %d\n", stats.SuccessfulFiles)
	fmt.Printf("❌ Failed: %d\n", stats.FailedFiles)
	fmt.Printf("⏱️ Total duration: %v\n", stats.TotalDuration)
	fmt.Printf("⚡ Average duration: %v\n", stats.AverageDuration)
	
	if len(stats.ErrorSummary) > 0 {
		fmt.Println("\nError Summary:")
		for errorType, count := range stats.ErrorSummary {
			fmt.Printf("  %s: %d\n", errorType, count)
		}
	}
	
	if stats.TotalFiles > 0 {
		successRate := float64(stats.SuccessfulFiles) / float64(stats.TotalFiles) * 100
		fmt.Printf("\n🎯 Success rate: %.1f%%\n", successRate)
	}
}

// BatchTranspiler provides a high-level interface for batch transpilation
type BatchTranspiler struct {
	parallelTranspiler *ParallelTranspiler
	ctx                context.Context
	cancel             context.CancelFunc
}

// NewBatchTranspiler creates a new batch transpiler
func NewBatchTranspiler(opts TranspileOptions) *BatchTranspiler {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &BatchTranspiler{
		parallelTranspiler: NewParallelTranspiler(opts),
		ctx:                ctx,
		cancel:             cancel,
	}
}

// TranspileProject transpiles an entire project with .gofa files
func (bt *BatchTranspiler) TranspileProject(projectDir string) error {
	fmt.Printf("🚀 Starting GoFasta transpilation for project: %s\n", projectDir)
	
	start := time.Now()
	results, err := bt.parallelTranspiler.TranspileDirectory(bt.ctx, projectDir)
	totalDuration := time.Since(start)
	
	if err != nil {
		return fmt.Errorf("transpilation failed: %w", err)
	}
	
	// Print statistics
	stats := GetStats(results)
	stats.TotalDuration = totalDuration
	PrintStats(stats)
	
	if stats.FailedFiles > 0 {
		fmt.Println("\n❌ Some files failed to transpile. Check the errors above.")
		return fmt.Errorf("transpilation completed with %d errors", stats.FailedFiles)
	}
	
	fmt.Println("\n✅ All files transpiled successfully!")
	return nil
}

// Stop cancels ongoing transpilation
func (bt *BatchTranspiler) Stop() {
	bt.cancel()
}

// WatchMode provides file watching for automatic transpilation
type WatchMode struct {
	transpiler    *ParallelTranspiler
	watchedDir    string
	debounceDelay time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewWatchMode creates a new watch mode transpiler
func NewWatchMode(opts TranspileOptions, watchedDir string, debounceDelay time.Duration) *WatchMode {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &WatchMode{
		transpiler:    NewParallelTranspiler(opts),
		watchedDir:    watchedDir,
		debounceDelay: debounceDelay,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start starts watching for .gofa file changes
func (wm *WatchMode) Start() error {
	// This is a simplified implementation
	// In a real implementation, you'd use a file watcher library like fsnotify
	fmt.Printf("👀 Watching directory: %s\n", wm.watchedDir)
	fmt.Println("Note: File watching requires additional implementation with fsnotify")
	
	// For now, just perform initial transpilation
	results, err := wm.transpiler.TranspileDirectory(wm.ctx, wm.watchedDir)
	if err != nil {
		return err
	}
	
	stats := GetStats(results)
	PrintStats(stats)
	
	return nil
}

// Stop stops the file watcher
func (wm *WatchMode) Stop() {
	wm.cancel()
}