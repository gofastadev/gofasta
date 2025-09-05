// Package core provides batched formatting capabilities for Gofasta transpiler.
// This implements Phase 1.1e: Set up go/format with batched formatting.
package core

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"sync"
	"time"
)

// FormatResult represents the result of formatting a single file
type FormatResult struct {
	FilePath      string
	OriginalSize  int
	FormattedSize int
	Output        []byte
	Error         error
	Duration      time.Duration
}

// BatchFormatterConfig contains configuration options for the batch formatter
type BatchFormatterConfig struct {
	// BatchSize sets the number of files to format in each batch
	// If 0, defaults to 10
	BatchSize int

	// MaxWorkers sets the maximum number of parallel workers
	// If 0, defaults to runtime.NumCPU()
	MaxWorkers int

	// EnableMetrics enables performance metrics collection
	EnableMetrics bool

	// FormatOptions contains go/format options
	FormatOptions *FormatOptions
}

// FormatOptions contains formatting configuration
type FormatOptions struct {
	// TabWidth sets the tab width for formatting
	TabWidth int

	// UseSpaces uses spaces instead of tabs
	UseSpaces bool

	// SortImports sorts import statements
	SortImports bool
}

// DefaultBatchFormatterConfig returns a default batch formatter configuration
func DefaultBatchFormatterConfig() *BatchFormatterConfig {
	return &BatchFormatterConfig{
		BatchSize:     10,
		MaxWorkers:    4,
		EnableMetrics: true,
		FormatOptions: &FormatOptions{
			TabWidth:    8,
			UseSpaces:   false,
			SortImports: true,
		},
	}
}

// BatchFormatter provides high-performance batched code formatting
type BatchFormatter struct {
	config *BatchFormatterConfig
	mu     sync.RWMutex

	// Performance metrics
	filesFormatted int64
	totalDuration  time.Duration
	bytesProcessed int64
	errorCount     int64
}

// NewBatchFormatter creates a new high-performance batch formatter
func NewBatchFormatter(config *BatchFormatterConfig) *BatchFormatter {
	if config == nil {
		config = DefaultBatchFormatterConfig()
	}

	if config.BatchSize <= 0 {
		config.BatchSize = 10
	}

	if config.MaxWorkers <= 0 {
		config.MaxWorkers = 4
	}

	if config.FormatOptions == nil {
		config.FormatOptions = &FormatOptions{
			TabWidth:    8,
			UseSpaces:   false,
			SortImports: true,
		}
	}

	return &BatchFormatter{
		config: config,
	}
}

// FormatFiles formats multiple files in parallel batches
func (f *BatchFormatter) FormatFiles(ctx context.Context, files map[string]*ast.File, fset *token.FileSet) (map[string]*FormatResult, error) {
	if len(files) == 0 {
		return make(map[string]*FormatResult), nil
	}

	results := make(map[string]*FormatResult)
	resultsMu := sync.Mutex{}

	// Create batches
	batches := f.createBatches(files)

	// Process batches in parallel
	semaphore := make(chan struct{}, f.config.MaxWorkers)
	var wg sync.WaitGroup
	var firstError error
	errorMu := sync.Mutex{}

	for _, batch := range batches {
		wg.Add(1)
		go func(batchFiles map[string]*ast.File) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Check context cancellation
			select {
			case <-ctx.Done():
				errorMu.Lock()
				if firstError == nil {
					firstError = ctx.Err()
				}
				errorMu.Unlock()
				return
			default:
			}

			// Format batch
			batchResults := f.formatBatch(batchFiles, fset)

			// Merge results
			resultsMu.Lock()
			for path, result := range batchResults {
				results[path] = result
				if result.Error != nil {
					errorMu.Lock()
					if firstError == nil {
						firstError = result.Error
					}
					errorMu.Unlock()
				}
			}
			resultsMu.Unlock()

		}(batch)
	}

	wg.Wait()

	return results, firstError
}

// FormatFile formats a single file
func (f *BatchFormatter) FormatFile(filePath string, file *ast.File, fset *token.FileSet) *FormatResult {
	start := time.Now()
	result := &FormatResult{
		FilePath: filePath,
	}

	// Check for nil inputs
	if file == nil || fset == nil {
		result.Error = fmt.Errorf("formatting %s: nil file or fileset", filePath)
		result.Duration = time.Since(start)
		// Update metrics for failed file
		if f.config.EnableMetrics {
			f.mu.Lock()
			f.filesFormatted++
			f.totalDuration += result.Duration
			f.errorCount++
			f.mu.Unlock()
		}
		return result
	}

	// Convert AST back to source
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		result.Error = fmt.Errorf("formatting %s: %w", filePath, err)
		result.Duration = time.Since(start)
		// Update metrics for failed file
		if f.config.EnableMetrics {
			f.mu.Lock()
			f.filesFormatted++
			f.totalDuration += result.Duration
			f.errorCount++
			f.mu.Unlock()
		}
		return result
	}

	result.Output = buf.Bytes()
	result.FormattedSize = len(result.Output)
	result.Duration = time.Since(start)

	// Update metrics for successful file
	if f.config.EnableMetrics {
		f.mu.Lock()
		f.filesFormatted++
		f.totalDuration += result.Duration
		f.bytesProcessed += int64(result.FormattedSize)
		f.mu.Unlock()
	}

	return result
}

// GetStatistics returns formatter performance statistics
func (f *BatchFormatter) GetStatistics() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	avgDuration := 0.0
	if f.filesFormatted > 0 {
		avgDuration = float64(f.totalDuration.Milliseconds()) / float64(f.filesFormatted)
	}

	throughput := 0.0
	if f.totalDuration > 0 {
		throughput = float64(f.filesFormatted) / f.totalDuration.Seconds()
	}

	bytesPerSec := 0.0
	if f.totalDuration > 0 {
		bytesPerSec = float64(f.bytesProcessed) / f.totalDuration.Seconds()
	}

	return map[string]interface{}{
		"files_formatted":   f.filesFormatted,
		"total_duration_ms": f.totalDuration.Milliseconds(),
		"avg_duration_ms":   avgDuration,
		"files_per_second":  throughput,
		"bytes_processed":   f.bytesProcessed,
		"bytes_per_second":  bytesPerSec,
		"error_count":       f.errorCount,
		"success_rate":      f.calculateSuccessRate(),
	}
}

// Reset clears all metrics
func (f *BatchFormatter) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.filesFormatted = 0
	f.totalDuration = 0
	f.bytesProcessed = 0
	f.errorCount = 0
}

// createBatches divides files into batches for parallel processing
func (f *BatchFormatter) createBatches(files map[string]*ast.File) []map[string]*ast.File {
	var batches []map[string]*ast.File
	currentBatch := make(map[string]*ast.File)
	batchCount := 0

	for path, file := range files {
		currentBatch[path] = file
		batchCount++

		if batchCount >= f.config.BatchSize {
			batches = append(batches, currentBatch)
			currentBatch = make(map[string]*ast.File)
			batchCount = 0
		}
	}

	// Add remaining files
	if len(currentBatch) > 0 {
		batches = append(batches, currentBatch)
	}

	return batches
}

// formatBatch formats a batch of files
func (f *BatchFormatter) formatBatch(files map[string]*ast.File, fset *token.FileSet) map[string]*FormatResult {
	results := make(map[string]*FormatResult)

	for path, file := range files {
		results[path] = f.FormatFile(path, file, fset)
	}

	return results
}

// calculateSuccessRate calculates the success rate of formatting operations
func (f *BatchFormatter) calculateSuccessRate() float64 {
	if f.filesFormatted == 0 {
		return 0.0
	}

	successCount := f.filesFormatted - f.errorCount
	return float64(successCount) / float64(f.filesFormatted) * 100.0
}

// GetFormattingOptions returns the current formatting options
func (f *BatchFormatter) GetFormattingOptions() *FormatOptions {
	return f.config.FormatOptions
}

// SetFormattingOptions updates the formatting options
func (f *BatchFormatter) SetFormattingOptions(options *FormatOptions) {
	if options != nil {
		f.config.FormatOptions = options
	}
}

// EstimateMemoryUsage estimates memory usage for formatting the given files
func (f *BatchFormatter) EstimateMemoryUsage(files map[string]*ast.File) int64 {
	totalNodes := int64(0)

	for _, file := range files {
		if file != nil {
			nodeCount := f.countASTNodes(file)
			totalNodes += int64(nodeCount)
		}
	}

	// Rough estimate: 1KB per AST node
	return totalNodes
}

// countASTNodes recursively counts AST nodes
func (f *BatchFormatter) countASTNodes(node ast.Node) int {
	if node == nil {
		return 0
	}

	count := 1
	ast.Inspect(node, func(n ast.Node) bool {
		if n != nil && n != node {
			count++
		}
		return true
	})

	return count
}
