// Package core provides high-performance parsing capabilities for GoFasta transpiler.
// This implements Phase 1.1a: Set up go/parser with parallel file processing.
package core

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// ParseResult represents the result of parsing a single file
type ParseResult struct {
	File     *ast.File
	FileSet  *token.FileSet
	FilePath string
	Size     int64
	Duration time.Duration
	Error    error
}

// ParserConfig contains configuration options for the parallel parser
type ParserConfig struct {
	// MaxWorkers sets the maximum number of parallel workers.
	// If 0, defaults to runtime.NumCPU()
	MaxWorkers int
	
	// ParseComments enables comment parsing
	ParseComments bool
	
	// AllowErrors continues parsing even when individual files have errors
	AllowErrors bool
	
	// FileExtensions specifies which file extensions to parse
	// Defaults to [".gofa", ".go"] if empty
	FileExtensions []string
	
	// ExcludeDirs specifies directory patterns to exclude
	ExcludeDirs []string
}

// DefaultConfig returns a default parser configuration optimized for performance
func DefaultConfig() *ParserConfig {
	return &ParserConfig{
		MaxWorkers:     runtime.NumCPU(),
		ParseComments:  true,
		AllowErrors:    true,
		FileExtensions: []string{".gofa", ".go"},
		ExcludeDirs:    []string{"vendor", "node_modules", ".git"},
	}
}

// ParallelParser provides high-performance parallel parsing capabilities
type ParallelParser struct {
	config   *ParserConfig
	fileSet  *token.FileSet
	results  []*ParseResult
	mu       sync.RWMutex
	
	// Performance metrics
	startTime     time.Time
	totalFiles    int
	totalBytes    int64
	successCount  int
	errorCount    int
}

// NewParallelParser creates a new high-performance parallel parser
func NewParallelParser(config *ParserConfig) *ParallelParser {
	if config == nil {
		config = DefaultConfig()
	}
	
	if config.MaxWorkers <= 0 {
		config.MaxWorkers = runtime.NumCPU()
	}
	
	if len(config.FileExtensions) == 0 {
		config.FileExtensions = []string{".gofa", ".go"}
	}
	
	if config.ExcludeDirs == nil {
		config.ExcludeDirs = []string{}
	}
	
	return &ParallelParser{
		config:  config,
		fileSet: token.NewFileSet(),
		results: make([]*ParseResult, 0),
	}
}

// ParseDirectory parses all eligible files in a directory tree using parallel processing
func (p *ParallelParser) ParseDirectory(ctx context.Context, rootPath string) ([]*ParseResult, error) {
	p.startTime = time.Now()
	p.reset()
	
	files, err := p.discoverFiles(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to discover files: %w", err)
	}
	
	p.totalFiles = len(files)
	if p.totalFiles == 0 {
		return p.results, nil
	}
	
	// Create error group with context for parallel processing
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(p.config.MaxWorkers)
	
	// Channel for collecting results
	resultChan := make(chan *ParseResult, p.totalFiles)
	
	// Start result collector goroutine
	go p.collectResults(resultChan)
	
	// Process files in parallel
	for _, filePath := range files {
		filePath := filePath // capture loop variable
		eg.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				result := p.parseFile(filePath)
				resultChan <- result
				return nil
			}
		})
	}
	
	// Wait for all parsing to complete
	if err := eg.Wait(); err != nil {
		close(resultChan)
		return nil, fmt.Errorf("parallel parsing failed: %w", err)
	}
	
	close(resultChan)
	
	// Wait for result collection to complete
	p.waitForCollection()
	
	return p.GetResults(), nil
}

// ParseFiles parses a specific list of files using parallel processing
func (p *ParallelParser) ParseFiles(ctx context.Context, filePaths []string) ([]*ParseResult, error) {
	p.startTime = time.Now()
	p.reset()
	
	p.totalFiles = len(filePaths)
	if p.totalFiles == 0 {
		return p.results, nil
	}
	
	// Create error group with context for parallel processing
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(p.config.MaxWorkers)
	
	// Channel for collecting results
	resultChan := make(chan *ParseResult, p.totalFiles)
	
	// Start result collector goroutine
	go p.collectResults(resultChan)
	
	// Process files in parallel
	for _, filePath := range filePaths {
		filePath := filePath // capture loop variable
		eg.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				result := p.parseFile(filePath)
				resultChan <- result
				return nil
			}
		})
	}
	
	// Wait for all parsing to complete
	if err := eg.Wait(); err != nil {
		close(resultChan)
		return nil, fmt.Errorf("parallel parsing failed: %w", err)
	}
	
	close(resultChan)
	
	// Wait for result collection to complete
	p.waitForCollection()
	
	return p.GetResults(), nil
}

// GetResults returns all parsing results (thread-safe)
func (p *ParallelParser) GetResults() []*ParseResult {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	// Return a copy to prevent external modifications
	results := make([]*ParseResult, len(p.results))
	copy(results, p.results)
	return results
}

// GetSuccessfulResults returns only successfully parsed files
func (p *ParallelParser) GetSuccessfulResults() []*ParseResult {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	var successful []*ParseResult
	for _, result := range p.results {
		if result.Error == nil && result.File != nil {
			successful = append(successful, result)
		}
	}
	return successful
}

// GetStatistics returns parsing performance statistics
func (p *ParallelParser) GetStatistics() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	totalDuration := time.Since(p.startTime)
	
	stats := map[string]interface{}{
		"total_files":       p.totalFiles,
		"successful_files":  p.successCount,
		"failed_files":      p.errorCount,
		"total_bytes":       p.totalBytes,
		"total_duration_ms": totalDuration.Milliseconds(),
		"max_workers":       p.config.MaxWorkers,
		"files_per_second":  float64(p.totalFiles) / totalDuration.Seconds(),
	}
	
	if p.totalBytes > 0 {
		stats["bytes_per_second"] = float64(p.totalBytes) / totalDuration.Seconds()
		stats["mb_per_second"] = float64(p.totalBytes) / (1024 * 1024) / totalDuration.Seconds()
	}
	
	return stats
}

// discoverFiles finds all eligible files in a directory tree
func (p *ParallelParser) discoverFiles(rootPath string) ([]string, error) {
	var files []string
	
	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		// Skip excluded directories
		if d.IsDir() {
			for _, excluded := range p.config.ExcludeDirs {
				if matched, _ := filepath.Match(excluded, d.Name()); matched {
					return filepath.SkipDir
				}
			}
			return nil
		}
		
		// Check file extension
		ext := filepath.Ext(path)
		for _, allowedExt := range p.config.FileExtensions {
			if ext == allowedExt {
				files = append(files, path)
				break
			}
		}
		
		return nil
	})
	
	return files, err
}

// parseFile parses a single file and returns the result
func (p *ParallelParser) parseFile(filePath string) *ParseResult {
	startTime := time.Now()
	result := &ParseResult{
		FilePath: filePath,
	}
	
	// Get file info for size
	if stat, err := os.Stat(filePath); err == nil {
		result.Size = stat.Size()
	}
	
	// Parse mode configuration
	mode := parser.ParseComments
	if !p.config.ParseComments {
		mode = 0
	}
	
	// Create file-specific token.FileSet for this parse operation
	fset := token.NewFileSet()
	
	// Parse the file
	file, err := parser.ParseFile(fset, filePath, nil, mode)
	if err != nil {
		result.Error = fmt.Errorf("parsing %s: %w", filePath, err)
	} else {
		result.File = file
		result.FileSet = fset
	}
	
	result.Duration = time.Since(startTime)
	return result
}

// collectResults collects parsing results from the result channel
func (p *ParallelParser) collectResults(resultChan <-chan *ParseResult) {
	for result := range resultChan {
		p.mu.Lock()
		p.results = append(p.results, result)
		p.totalBytes += result.Size
		
		if result.Error != nil {
			p.errorCount++
		} else {
			p.successCount++
		}
		p.mu.Unlock()
	}
}

// waitForCollection waits for result collection to complete
func (p *ParallelParser) waitForCollection() {
	// Simple spin wait - in production, this could use a WaitGroup
	for {
		p.mu.RLock()
		collected := len(p.results)
		p.mu.RUnlock()
		
		if collected == p.totalFiles {
			break
		}
		time.Sleep(1 * time.Millisecond)
	}
}

// reset clears previous results for a new parsing operation
func (p *ParallelParser) reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.results = make([]*ParseResult, 0)
	p.totalFiles = 0
	p.totalBytes = 0
	p.successCount = 0
	p.errorCount = 0
}

// FilterResultsByExtension returns results filtered by file extension
func (p *ParallelParser) FilterResultsByExtension(extension string) []*ParseResult {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	var filtered []*ParseResult
	for _, result := range p.results {
		if strings.HasSuffix(result.FilePath, extension) {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

// GetFileSet returns the main FileSet used by the parser
func (p *ParallelParser) GetFileSet() *token.FileSet {
	return p.fileSet
}