// Package core provides golang.org/x/tools/go/packages with package cache.
// This implements Phase 1.2d: golang.org/x/tools/go/packages with package cache.
package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/tools/go/packages"
)

// PackageCache manages package loading and caching using golang.org/x/tools/go/packages
type PackageCache struct {
	config      *PackageCacheConfig
	cache       map[string]*CachedPackage
	patterns    map[string]*PatternResult
	typeCache   map[string]*types.Package
	mu          sync.RWMutex
	fset        *token.FileSet
	
	// Dependency tracking
	depGraph    map[string][]string
	reverseDeps map[string][]string
	
	// Metrics
	hits         int64
	misses       int64
	loads        int64
	typeChecks   int64
	cacheSize    int64
	totalLoadTime time.Duration
}

// CachedPackage represents a cached package with all its information
type CachedPackage struct {
	*packages.Package
	LoadedAt    time.Time
	LastAccessed time.Time
	AccessCount  int64
	Hash         string
	Size         int64
	Dependencies []string
	Dependents   []string
}

// PatternResult represents cached results for a pattern query
type PatternResult struct {
	Pattern     string
	Packages    []*CachedPackage
	LoadedAt    time.Time
	AccessCount int64
}

// PackageCacheConfig contains configuration for package cache
type PackageCacheConfig struct {
	// Load configuration
	Mode            packages.LoadMode
	Dir             string
	Env             []string
	BuildFlags      []string
	Tests           bool
	
	// Cache configuration
	MaxPackages     int
	MaxCacheSizeMB  int
	TTL             time.Duration
	EnableMetrics   bool
	
	// Performance configuration
	ConcurrentLoads bool
	LoadWorkers     int
	PreloadImports  bool
	WarmupPatterns  []string
	
	// Analysis configuration
	NeedTypes       bool
	NeedSyntax      bool
	NeedTypesInfo   bool
	NeedTypesSizes  bool
	NeedImports     bool
	NeedDeps        bool
}

// DefaultPackageCacheConfig returns default configuration
func DefaultPackageCacheConfig() *PackageCacheConfig {
	return &PackageCacheConfig{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedTypesSizes,
		Dir:             ".",
		Tests:           false,
		MaxPackages:     1000,
		MaxCacheSizeMB:  200,
		TTL:             30 * time.Minute,
		EnableMetrics:   true,
		ConcurrentLoads: true,
		LoadWorkers:     4,
		PreloadImports:  true,
		NeedTypes:       true,
		NeedSyntax:      true,
		NeedTypesInfo:   true,
		NeedTypesSizes:  true,
		NeedImports:     true,
		NeedDeps:        true,
	}
}

// NewPackageCache creates a new package cache
func NewPackageCache(config *PackageCacheConfig) *PackageCache {
	if config == nil {
		config = DefaultPackageCacheConfig()
	}
	
	return &PackageCache{
		config:      config,
		cache:       make(map[string]*CachedPackage),
		patterns:    make(map[string]*PatternResult),
		typeCache:   make(map[string]*types.Package),
		fset:        token.NewFileSet(),
		depGraph:    make(map[string][]string),
		reverseDeps: make(map[string][]string),
	}
}

// Load loads packages matching the given patterns
func (pc *PackageCache) Load(patterns ...string) ([]*CachedPackage, error) {
	return pc.LoadWithContext(context.Background(), patterns...)
}

// LoadWithContext loads packages with context
func (pc *PackageCache) LoadWithContext(ctx context.Context, patterns ...string) ([]*CachedPackage, error) {
	// Check pattern cache first
	patternKey := strings.Join(patterns, ",")
	
	pc.mu.RLock()
	if cached, exists := pc.patterns[patternKey]; exists {
		if pc.config.TTL == 0 || time.Since(cached.LoadedAt) < pc.config.TTL {
			pc.mu.RUnlock()
			atomic.AddInt64(&pc.hits, 1)
			atomic.AddInt64(&cached.AccessCount, 1)
			return cached.Packages, nil
		}
	}
	pc.mu.RUnlock()
	
	atomic.AddInt64(&pc.misses, 1)
	
	// Load packages
	start := time.Now()
	
	cfg := &packages.Config{
		Mode:       pc.config.Mode,
		Context:    ctx,
		Dir:        pc.config.Dir,
		Env:        pc.config.Env,
		BuildFlags: pc.config.BuildFlags,
		Tests:      pc.config.Tests,
		Fset:       pc.fset,
	}
	
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %w", err)
	}
	
	loadTime := time.Since(start)
	pc.totalLoadTime += loadTime
	atomic.AddInt64(&pc.loads, 1)
	
	// Process and cache packages
	var cachedPkgs []*CachedPackage
	for _, pkg := range pkgs {
		cached := pc.cachePackage(pkg)
		cachedPkgs = append(cachedPkgs, cached)
		
		// Preload imports if configured
		if pc.config.PreloadImports {
			pc.preloadImports(pkg)
		}
	}
	
	// Cache pattern result
	pc.mu.Lock()
	pc.patterns[patternKey] = &PatternResult{
		Pattern:     patternKey,
		Packages:    cachedPkgs,
		LoadedAt:    time.Now(),
		AccessCount: 1,
	}
	pc.mu.Unlock()
	
	// Build dependency graph
	pc.buildDependencyGraph(cachedPkgs)
	
	return cachedPkgs, nil
}

// cachePackage caches a single package
func (pc *PackageCache) cachePackage(pkg *packages.Package) *CachedPackage {
	hash := pc.calculatePackageHash(pkg)
	
	cached := &CachedPackage{
		Package:      pkg,
		LoadedAt:     time.Now(),
		LastAccessed: time.Now(),
		AccessCount:  1,
		Hash:         hash,
		Size:         pc.calculatePackageSize(pkg),
		Dependencies: pc.extractDependencies(pkg),
	}
	
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	// Check cache size
	if pc.config.MaxPackages > 0 && len(pc.cache) >= pc.config.MaxPackages {
		pc.evictLRU()
	}
	
	pc.cache[pkg.ID] = cached
	atomic.AddInt64(&pc.cacheSize, cached.Size)
	
	// Cache types package if available
	if pkg.Types != nil {
		pc.typeCache[pkg.ID] = pkg.Types
	}
	
	return cached
}

// GetPackage retrieves a package from cache
func (pc *PackageCache) GetPackage(id string) (*CachedPackage, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	pkg, exists := pc.cache[id]
	if exists {
		atomic.AddInt64(&pkg.AccessCount, 1)
		pkg.LastAccessed = time.Now()
		atomic.AddInt64(&pc.hits, 1)
		return pkg, true
	}
	
	atomic.AddInt64(&pc.misses, 1)
	return nil, false
}

// GetTypePackage retrieves a types.Package from cache
func (pc *PackageCache) GetTypePackage(id string) (*types.Package, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	pkg, exists := pc.typeCache[id]
	if exists {
		atomic.AddInt64(&pc.hits, 1)
		return pkg, true
	}
	
	atomic.AddInt64(&pc.misses, 1)
	return nil, false
}

// LoadOne loads a single package by import path
func (pc *PackageCache) LoadOne(importPath string) (*CachedPackage, error) {
	// Check cache first
	if cached, exists := pc.GetPackage(importPath); exists {
		return cached, nil
	}
	
	// Load the package
	pkgs, err := pc.Load(importPath)
	if err != nil {
		return nil, err
	}
	
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("package not found: %s", importPath)
	}
	
	return pkgs[0], nil
}

// LoadRecursive loads a package and all its dependencies
func (pc *PackageCache) LoadRecursive(importPath string) ([]*CachedPackage, error) {
	visited := make(map[string]bool)
	var result []*CachedPackage
	
	var loadRec func(string) error
	loadRec = func(path string) error {
		if visited[path] {
			return nil
		}
		visited[path] = true
		
		pkg, err := pc.LoadOne(path)
		if err != nil {
			return err
		}
		
		result = append(result, pkg)
		
		// Load dependencies
		for _, dep := range pkg.Dependencies {
			if err := loadRec(dep); err != nil {
				return err
			}
		}
		
		return nil
	}
	
	if err := loadRec(importPath); err != nil {
		return nil, err
	}
	
	return result, nil
}

// preloadImports preloads imports for a package
func (pc *PackageCache) preloadImports(pkg *packages.Package) {
	if !pc.config.ConcurrentLoads {
		// Load sequentially
		for path := range pkg.Imports {
			_, _ = pc.LoadOne(path)
		}
		return
	}
	
	// Load concurrently
	var wg sync.WaitGroup
	sem := make(chan struct{}, pc.config.LoadWorkers)
	
	for path := range pkg.Imports {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			
			_, _ = pc.LoadOne(p)
		}(path)
	}
	
	wg.Wait()
}

// buildDependencyGraph builds the dependency graph
func (pc *PackageCache) buildDependencyGraph(pkgs []*CachedPackage) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	for _, pkg := range pkgs {
		pc.depGraph[pkg.ID] = pkg.Dependencies
		
		// Build reverse dependencies
		for _, dep := range pkg.Dependencies {
			pc.reverseDeps[dep] = append(pc.reverseDeps[dep], pkg.ID)
		}
	}
}

// GetDependencies returns all dependencies of a package
func (pc *PackageCache) GetDependencies(id string) []string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	return pc.depGraph[id]
}

// GetDependents returns all packages that depend on the given package
func (pc *PackageCache) GetDependents(id string) []string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	return pc.reverseDeps[id]
}

// GetTransitiveDependencies returns all transitive dependencies
func (pc *PackageCache) GetTransitiveDependencies(id string) []string {
	visited := make(map[string]bool)
	var result []string
	
	var visit func(string)
	visit = func(pkgID string) {
		if visited[pkgID] {
			return
		}
		visited[pkgID] = true
		
		deps := pc.GetDependencies(pkgID)
		for _, dep := range deps {
			if !visited[dep] {
				result = append(result, dep)
				visit(dep)
			}
		}
	}
	
	visit(id)
	return result
}

// FindPackagesByName finds packages by name pattern
func (pc *PackageCache) FindPackagesByName(pattern string) []*CachedPackage {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	var result []*CachedPackage
	for _, pkg := range pc.cache {
		if matched, _ := filepath.Match(pattern, pkg.Name); matched {
			result = append(result, pkg)
		}
	}
	
	return result
}

// FindPackagesByImportPath finds packages by import path pattern
func (pc *PackageCache) FindPackagesByImportPath(pattern string) []*CachedPackage {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	var result []*CachedPackage
	for _, pkg := range pc.cache {
		if matched, _ := filepath.Match(pattern, pkg.PkgPath); matched {
			result = append(result, pkg)
		}
	}
	
	return result
}

// AnalyzePackage performs analysis on a package
func (pc *PackageCache) AnalyzePackage(id string) (*PackageAnalysis, error) {
	pkg, exists := pc.GetPackage(id)
	if !exists {
		return nil, fmt.Errorf("package not found: %s", id)
	}
	
	analysis := &PackageAnalysis{
		ID:           pkg.ID,
		Name:         pkg.Name,
		Path:         pkg.PkgPath,
		FileCount:    len(pkg.GoFiles),
		LineCount:    pc.countLines(pkg),
		TypeCount:    pc.countTypes(pkg),
		FuncCount:    pc.countFunctions(pkg),
		Dependencies: len(pkg.Dependencies),
		Dependents:   len(pc.GetDependents(id)),
		Size:         pkg.Size,
		LoadedAt:     pkg.LoadedAt,
		LastAccessed: pkg.LastAccessed,
		AccessCount:  pkg.AccessCount,
	}
	
	atomic.AddInt64(&pc.typeChecks, 1)
	return analysis, nil
}

// PackageAnalysis represents analysis results for a package
type PackageAnalysis struct {
	ID           string
	Name         string
	Path         string
	FileCount    int
	LineCount    int
	TypeCount    int
	FuncCount    int
	Dependencies int
	Dependents   int
	Size         int64
	LoadedAt     time.Time
	LastAccessed time.Time
	AccessCount  int64
}

// countLines counts lines in a package
func (pc *PackageCache) countLines(pkg *CachedPackage) int {
	lines := 0
	for _, file := range pkg.Syntax {
		if file != nil {
			lines += pc.fset.Position(file.End()).Line
		}
	}
	return lines
}

// countTypes counts types in a package
func (pc *PackageCache) countTypes(pkg *CachedPackage) int {
	count := 0
	if pkg.Types != nil {
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			if obj := scope.Lookup(name); obj != nil {
				if _, ok := obj.(*types.TypeName); ok {
					count++
				}
			}
		}
	}
	return count
}

// countFunctions counts functions in a package
func (pc *PackageCache) countFunctions(pkg *CachedPackage) int {
	count := 0
	for _, file := range pkg.Syntax {
		if file != nil {
			ast.Inspect(file, func(n ast.Node) bool {
				if _, ok := n.(*ast.FuncDecl); ok {
					count++
				}
				return true
			})
		}
	}
	return count
}

// Warmup preloads commonly used packages
func (pc *PackageCache) Warmup(patterns []string) error {
	if len(patterns) == 0 {
		patterns = pc.config.WarmupPatterns
	}
	
	for _, pattern := range patterns {
		_, err := pc.Load(pattern)
		if err != nil {
			return fmt.Errorf("warmup failed for %s: %w", pattern, err)
		}
	}
	
	return nil
}

// evictLRU evicts least recently used package
func (pc *PackageCache) evictLRU() {
	var lru *CachedPackage
	var lruID string
	
	for id, pkg := range pc.cache {
		if lru == nil || pkg.LastAccessed.Before(lru.LastAccessed) {
			lru = pkg
			lruID = id
		}
	}
	
	if lruID != "" {
		delete(pc.cache, lruID)
		delete(pc.typeCache, lruID)
		atomic.AddInt64(&pc.cacheSize, -lru.Size)
	}
}

// calculatePackageHash calculates hash of package content
func (pc *PackageCache) calculatePackageHash(pkg *packages.Package) string {
	h := sha256.New()
	
	// Hash package files
	files := append([]string{}, pkg.GoFiles...)
	sort.Strings(files)
	for _, file := range files {
		h.Write([]byte(file))
	}
	
	// Hash imports
	imports := make([]string, 0, len(pkg.Imports))
	for path := range pkg.Imports {
		imports = append(imports, path)
	}
	sort.Strings(imports)
	for _, imp := range imports {
		h.Write([]byte(imp))
	}
	
	return hex.EncodeToString(h.Sum(nil))
}

// calculatePackageSize estimates package size
func (pc *PackageCache) calculatePackageSize(pkg *packages.Package) int64 {
	size := int64(0)
	
	// Estimate based on syntax trees
	for _, file := range pkg.Syntax {
		if file != nil {
			size += int64(pc.fset.Position(file.End()).Offset)
		}
	}
	
	// Add overhead for type information
	if pkg.Types != nil {
		size += int64(len(pkg.TypesInfo.Types) * 100) // Rough estimate
	}
	
	return size
}

// extractDependencies extracts package dependencies
func (pc *PackageCache) extractDependencies(pkg *packages.Package) []string {
	deps := make([]string, 0, len(pkg.Imports))
	for path := range pkg.Imports {
		deps = append(deps, path)
	}
	sort.Strings(deps)
	return deps
}

// GetStatistics returns cache statistics
func (pc *PackageCache) GetStatistics() map[string]interface{} {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	total := pc.hits + pc.misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(pc.hits) * 100.0 / float64(total)
	}
	
	avgLoadTime := time.Duration(0)
	if pc.loads > 0 {
		avgLoadTime = pc.totalLoadTime / time.Duration(pc.loads)
	}
	
	// Count total dependencies
	totalDeps := 0
	for _, deps := range pc.depGraph {
		totalDeps += len(deps)
	}
	
	return map[string]interface{}{
		"cached_packages":    len(pc.cache),
		"cached_patterns":    len(pc.patterns),
		"cached_types":       len(pc.typeCache),
		"cache_hits":         pc.hits,
		"cache_misses":       pc.misses,
		"hit_rate":           hitRate,
		"total_loads":        pc.loads,
		"type_checks":        pc.typeChecks,
		"cache_size_bytes":   pc.cacheSize,
		"cache_size_mb":      float64(pc.cacheSize) / (1024 * 1024),
		"avg_load_time":      avgLoadTime.String(),
		"total_dependencies": totalDeps,
		"dependency_graph":   len(pc.depGraph),
	}
}

// Clear clears the cache
func (pc *PackageCache) Clear() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	pc.cache = make(map[string]*CachedPackage)
	pc.patterns = make(map[string]*PatternResult)
	pc.typeCache = make(map[string]*types.Package)
	pc.depGraph = make(map[string][]string)
	pc.reverseDeps = make(map[string][]string)
	pc.hits = 0
	pc.misses = 0
	pc.loads = 0
	pc.typeChecks = 0
	pc.cacheSize = 0
	pc.totalLoadTime = 0
}

// InvalidatePackage invalidates a specific package
func (pc *PackageCache) InvalidatePackage(id string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	if pkg, exists := pc.cache[id]; exists {
		delete(pc.cache, id)
		delete(pc.typeCache, id)
		atomic.AddInt64(&pc.cacheSize, -pkg.Size)
		
		// Remove from dependency graph
		delete(pc.depGraph, id)
		
		// Remove from reverse dependencies
		for _, deps := range pc.reverseDeps {
			for i, dep := range deps {
				if dep == id {
					deps = append(deps[:i], deps[i+1:]...)
					break
				}
			}
		}
	}
	
	// Invalidate pattern results containing this package
	for key, result := range pc.patterns {
		for _, pkg := range result.Packages {
			if pkg.ID == id {
				delete(pc.patterns, key)
				break
			}
		}
	}
}

// InvalidatePattern invalidates cached pattern results
func (pc *PackageCache) InvalidatePattern(pattern string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	delete(pc.patterns, pattern)
}

// GetCyclicDependencies finds cyclic dependencies
func (pc *PackageCache) GetCyclicDependencies() [][]string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	
	var detectCycle func(string, []string) bool
	detectCycle = func(node string, path []string) bool {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)
		
		for _, dep := range pc.depGraph[node] {
			if !visited[dep] {
				if detectCycle(dep, path) {
					return true
				}
			} else if recStack[dep] {
				// Found a cycle
				cycleStart := -1
				for i, n := range path {
					if n == dep {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cycle := append([]string{}, path[cycleStart:]...)
					cycles = append(cycles, cycle)
				}
				return true
			}
		}
		
		recStack[node] = false
		return false
	}
	
	for node := range pc.depGraph {
		if !visited[node] {
			detectCycle(node, []string{})
		}
	}
	
	return cycles
}