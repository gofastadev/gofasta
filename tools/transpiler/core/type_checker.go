// Package core provides incremental type checking capabilities for GoFasta transpiler.
// This implements Phase 1.1d: Implement go/types with incremental type checking.
package core

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sync"
	"time"
)

// TypeCheckResult represents the result of type checking a package
type TypeCheckResult struct {
	Package    *types.Package
	Info       *types.Info
	Error      error
	Duration   time.Duration
	CheckTime  time.Time
	FilePaths  []string
}

// TypeCheckerConfig contains configuration options for the incremental type checker
type TypeCheckerConfig struct {
	// EnableCaching enables type checking result caching
	EnableCaching bool
	
	// CacheTTL sets the time-to-live for cached results
	CacheTTL time.Duration
	
	// MaxCacheEntries sets the maximum number of cached results
	MaxCacheEntries int
	
	// ParallelChecking enables parallel type checking
	ParallelChecking bool
	
	// MaxWorkers sets the maximum number of parallel workers
	MaxWorkers int
	
	// EnableMetrics enables performance metrics collection
	EnableMetrics bool
}

// DefaultTypeCheckerConfig returns a default type checker configuration
func DefaultTypeCheckerConfig() *TypeCheckerConfig {
	return &TypeCheckerConfig{
		EnableCaching:    true,
		CacheTTL:         30 * time.Minute,
		MaxCacheEntries:  500,
		ParallelChecking: true,
		MaxWorkers:       4,
		EnableMetrics:    true,
	}
}

// IncrementalTypeChecker provides high-performance incremental type checking
type IncrementalTypeChecker struct {
	config        *TypeCheckerConfig
	cache         map[string]*TypeCheckResult
	dependencies  map[string][]string
	mu            sync.RWMutex
	
	// Performance metrics
	cacheHits     int64
	cacheMisses   int64
	checksRun     int64
	totalDuration time.Duration
}

// NewIncrementalTypeChecker creates a new incremental type checker
func NewIncrementalTypeChecker(config *TypeCheckerConfig) *IncrementalTypeChecker {
	if config == nil {
		config = DefaultTypeCheckerConfig()
	}
	
	if config.CacheTTL <= 0 {
		config.CacheTTL = 30 * time.Minute
	}
	
	if config.MaxCacheEntries <= 0 {
		config.MaxCacheEntries = 500
	}
	
	if config.MaxWorkers <= 0 {
		config.MaxWorkers = 4
	}
	
	return &IncrementalTypeChecker{
		config:       config,
		cache:        make(map[string]*TypeCheckResult),
		dependencies: make(map[string][]string),
	}
}

// CheckPackage performs incremental type checking on a package
func (tc *IncrementalTypeChecker) CheckPackage(ctx context.Context, pkgPath string, files []*ast.File, fset *token.FileSet) (*TypeCheckResult, error) {
	start := time.Now()
	
	// Check cache first
	if tc.config.EnableCaching {
		if result := tc.getCachedResult(pkgPath, files); result != nil {
			if tc.config.EnableMetrics {
				tc.mu.Lock()
				tc.cacheHits++
				tc.mu.Unlock()
			}
			return result, nil
		}
		
		if tc.config.EnableMetrics {
			tc.mu.Lock()
			tc.cacheMisses++
			tc.mu.Unlock()
		}
	}
	
	// Perform type checking
	result := tc.performTypeCheck(ctx, pkgPath, files, fset)
	result.Duration = time.Since(start)
	result.CheckTime = time.Now()
	
	// Cache result
	if tc.config.EnableCaching && result.Error == nil {
		tc.cacheResult(pkgPath, result)
	}
	
	// Update metrics
	if tc.config.EnableMetrics {
		tc.mu.Lock()
		tc.checksRun++
		tc.totalDuration += result.Duration
		tc.mu.Unlock()
	}
	
	return result, result.Error
}

// CheckPackages performs parallel type checking on multiple packages
func (tc *IncrementalTypeChecker) CheckPackages(ctx context.Context, packages map[string][]*ast.File, fset *token.FileSet) (map[string]*TypeCheckResult, error) {
	if !tc.config.ParallelChecking {
		return tc.checkPackagesSequential(ctx, packages, fset)
	}
	
	// Check context cancellation early
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	results := make(map[string]*TypeCheckResult)
	resultsMu := sync.Mutex{}
	
	// Create worker pool
	semaphore := make(chan struct{}, tc.config.MaxWorkers)
	var wg sync.WaitGroup
	var firstError error
	errorMu := sync.Mutex{}
	
	for pkgPath, files := range packages {
		wg.Add(1)
		go func(path string, pkgFiles []*ast.File) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			result, err := tc.CheckPackage(ctx, path, pkgFiles, fset)
			
			resultsMu.Lock()
			results[path] = result
			resultsMu.Unlock()
			
			if err != nil {
				errorMu.Lock()
				if firstError == nil {
					firstError = err
				}
				errorMu.Unlock()
			}
		}(pkgPath, files)
	}
	
	wg.Wait()
	
	return results, firstError
}

// InvalidateCache removes cached results for packages that depend on the given package
func (tc *IncrementalTypeChecker) InvalidateCache(changedPackage string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	
	// Remove direct cache entry
	delete(tc.cache, changedPackage)
	
	// Remove dependent packages
	for pkgPath, deps := range tc.dependencies {
		for _, dep := range deps {
			if dep == changedPackage {
				delete(tc.cache, pkgPath)
				break
			}
		}
	}
}

// GetStatistics returns type checker performance statistics
func (tc *IncrementalTypeChecker) GetStatistics() map[string]interface{} {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	
	totalRequests := tc.cacheHits + tc.cacheMisses
	hitRatio := 0.0
	if totalRequests > 0 {
		hitRatio = float64(tc.cacheHits) / float64(totalRequests) * 100.0
	}
	
	avgDuration := 0.0
	if tc.checksRun > 0 {
		avgDuration = float64(tc.totalDuration.Milliseconds()) / float64(tc.checksRun)
	}
	
	return map[string]interface{}{
		"cached_packages":     len(tc.cache),
		"cache_hits":          tc.cacheHits,
		"cache_misses":        tc.cacheMisses,
		"hit_ratio":          hitRatio,
		"checks_run":         tc.checksRun,
		"avg_duration_ms":    avgDuration,
		"total_duration_ms":  tc.totalDuration.Milliseconds(),
	}
}

// Clear removes all cached results
func (tc *IncrementalTypeChecker) Clear() {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	
	tc.cache = make(map[string]*TypeCheckResult)
	tc.dependencies = make(map[string][]string)
}

// getCachedResult retrieves a cached type check result if valid
func (tc *IncrementalTypeChecker) getCachedResult(pkgPath string, files []*ast.File) *TypeCheckResult {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	
	result, exists := tc.cache[pkgPath]
	if !exists {
		return nil
	}
	
	// Check TTL
	if time.Since(result.CheckTime) > tc.config.CacheTTL {
		return nil
	}
	
	// Check if files match (simple check - could be more sophisticated)
	if len(result.FilePaths) != len(files) {
		return nil
	}
	
	return result
}

// cacheResult stores a type check result
func (tc *IncrementalTypeChecker) cacheResult(pkgPath string, result *TypeCheckResult) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	
	// Evict if cache is full
	if len(tc.cache) >= tc.config.MaxCacheEntries {
		tc.evictOldestEntry()
	}
	
	tc.cache[pkgPath] = result
}

// evictOldestEntry removes the oldest cached entry
func (tc *IncrementalTypeChecker) evictOldestEntry() {
	var oldestKey string
	var oldestTime time.Time
	
	for key, result := range tc.cache {
		if oldestKey == "" || result.CheckTime.Before(oldestTime) {
			oldestKey = key
			oldestTime = result.CheckTime
		}
	}
	
	if oldestKey != "" {
		delete(tc.cache, oldestKey)
		delete(tc.dependencies, oldestKey)
	}
}

// performTypeCheck performs the actual type checking
func (tc *IncrementalTypeChecker) performTypeCheck(ctx context.Context, pkgPath string, files []*ast.File, fset *token.FileSet) *TypeCheckResult {
	result := &TypeCheckResult{
		FilePaths: make([]string, len(files)),
	}
	
	// Extract file paths and filter out nil files
	validFiles := make([]*ast.File, 0, len(files))
	for i, file := range files {
		if file != nil && fset != nil {
			pos := fset.Position(file.Pos())
			result.FilePaths[i] = pos.Filename
			validFiles = append(validFiles, file)
		} else if file == nil {
			result.FilePaths[i] = "nil_file"
		}
	}
	
	// If no valid files, return early with error
	if len(validFiles) == 0 {
		result.Error = fmt.Errorf("no valid files to type check")
		return result
	}
	
	// Create type checker config
	config := &types.Config{
		Error: func(err error) {
			if result.Error == nil {
				result.Error = err
			}
		},
		Importer: NewCachedImporter(DefaultImportCacheConfig()),
	}
	
	// Create type info
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
	}
	
	// Perform type checking
	pkg, err := config.Check(pkgPath, fset, validFiles, info)
	if err != nil {
		result.Error = err
	} else {
		result.Package = pkg
		result.Info = info
	}
	
	return result
}

// checkPackagesSequential performs sequential type checking
func (tc *IncrementalTypeChecker) checkPackagesSequential(ctx context.Context, packages map[string][]*ast.File, fset *token.FileSet) (map[string]*TypeCheckResult, error) {
	results := make(map[string]*TypeCheckResult)
	
	for pkgPath, files := range packages {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			result, err := tc.CheckPackage(ctx, pkgPath, files, fset)
			results[pkgPath] = result
			
			if err != nil {
				return results, err
			}
		}
	}
	
	return results, nil
}