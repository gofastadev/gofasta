// Package core provides import caching capabilities for Gofasta transpiler.
// This implements Phase 1.1f: Configure go/importer with import caching.
package core

import (
	"fmt"
	"go/importer"
	"go/types"
	"sync"
	"time"
)

// ImportCacheEntry represents a cached import with metadata
type ImportCacheEntry struct {
	Package     *types.Package
	CacheTime   time.Time
	AccessTime  time.Time
	AccessCount int64
	Size        int64
}

// ImportCacheConfig contains configuration options for the import cache
type ImportCacheConfig struct {
	// MaxEntries sets the maximum number of cached imports
	// If 0, defaults to 1000
	MaxEntries int

	// TTL sets the time-to-live for cached entries
	// If 0, defaults to 1 hour
	TTL time.Duration

	// EnableMetrics enables cache metrics collection
	EnableMetrics bool

	// MaxMemoryMB sets the maximum memory usage in MB
	// If 0, defaults to 256MB
	MaxMemoryMB int64
}

// DefaultImportCacheConfig returns a default import cache configuration
func DefaultImportCacheConfig() *ImportCacheConfig {
	return &ImportCacheConfig{
		MaxEntries:    1000,
		TTL:           time.Hour,
		EnableMetrics: true,
		MaxMemoryMB:   256,
	}
}

// CachedImporter provides high-performance import caching with LRU eviction
type CachedImporter struct {
	config   *ImportCacheConfig
	cache    map[string]*ImportCacheEntry
	lruList  []string
	importer types.Importer
	mu       sync.RWMutex

	// Metrics
	hits      int64
	misses    int64
	evictions int64
	errors    int64
}

// NewCachedImporter creates a new high-performance cached importer
func NewCachedImporter(config *ImportCacheConfig) *CachedImporter {
	if config == nil {
		config = DefaultImportCacheConfig()
	}

	if config.MaxEntries <= 0 {
		config.MaxEntries = 1000
	}

	if config.TTL <= 0 {
		config.TTL = time.Hour
	}

	if config.MaxMemoryMB <= 0 {
		config.MaxMemoryMB = 256
	}

	return &CachedImporter{
		config:   config,
		cache:    make(map[string]*ImportCacheEntry),
		lruList:  make([]string, 0, config.MaxEntries),
		importer: importer.Default(),
	}
}

// Import implements the types.Importer interface with caching
func (ci *CachedImporter) Import(path string) (*types.Package, error) {
	// Check cache first
	if pkg := ci.getFromCache(path); pkg != nil {
		if ci.config.EnableMetrics {
			ci.mu.Lock()
			ci.hits++
			ci.mu.Unlock()
		}
		return pkg, nil
	}

	// Cache miss - import from source
	if ci.config.EnableMetrics {
		ci.mu.Lock()
		ci.misses++
		ci.mu.Unlock()
	}

	pkg, err := ci.importer.Import(path)
	if err != nil {
		if ci.config.EnableMetrics {
			ci.mu.Lock()
			ci.errors++
			ci.mu.Unlock()
		}
		return nil, err
	}

	// Cache successful import
	ci.putInCache(path, pkg)

	return pkg, nil
}

// ImportWithFallback imports with fallback to different importers
func (ci *CachedImporter) ImportWithFallback(path string, fallbackImporters ...types.Importer) (*types.Package, error) {
	// Try cached importer first
	pkg, err := ci.Import(path)
	if err == nil {
		return pkg, nil
	}

	// Try fallback importers
	for _, fallback := range fallbackImporters {
		if pkg, fallbackErr := fallback.Import(path); fallbackErr == nil {
			// Cache successful fallback import
			ci.putInCache(path, pkg)
			return pkg, nil
		}
	}

	// Return original error if all fallbacks fail
	return nil, err
}

// Preload preloads common packages into cache
func (ci *CachedImporter) Preload(packages []string) map[string]error {
	errors := make(map[string]error)

	for _, pkgPath := range packages {
		if _, err := ci.Import(pkgPath); err != nil {
			errors[pkgPath] = err
		}
	}

	return errors
}

// GetStatistics returns cache performance statistics
func (ci *CachedImporter) GetStatistics() map[string]interface{} {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	totalRequests := ci.hits + ci.misses
	hitRatio := 0.0
	if totalRequests > 0 {
		hitRatio = float64(ci.hits) / float64(totalRequests) * 100.0
	}

	errorRate := 0.0
	if totalRequests > 0 {
		errorRate = float64(ci.errors) / float64(totalRequests) * 100.0
	}

	memoryUsage := ci.estimateMemoryUsage()

	return map[string]interface{}{
		"entries":        len(ci.cache),
		"max_entries":    ci.config.MaxEntries,
		"hits":           ci.hits,
		"misses":         ci.misses,
		"evictions":      ci.evictions,
		"errors":         ci.errors,
		"hit_ratio":      hitRatio,
		"error_rate":     errorRate,
		"memory_mb":      memoryUsage,
		"max_memory":     ci.config.MaxMemoryMB,
		"total_requests": totalRequests,
	}
}

// Clear removes all cached imports
func (ci *CachedImporter) Clear() {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	ci.cache = make(map[string]*ImportCacheEntry)
	ci.lruList = make([]string, 0, ci.config.MaxEntries)
}

// Cleanup removes expired entries
func (ci *CachedImporter) Cleanup() int {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	removed := 0
	now := time.Now()

	for path, entry := range ci.cache {
		if now.Sub(entry.CacheTime) > ci.config.TTL {
			delete(ci.cache, path)
			ci.removeFromLRU(path)
			removed++
		}
	}

	return removed
}

// GetCachedPackages returns a list of all cached package paths
func (ci *CachedImporter) GetCachedPackages() []string {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	packages := make([]string, 0, len(ci.cache))
	for path := range ci.cache {
		packages = append(packages, path)
	}

	return packages
}

// getFromCache retrieves a package from cache if available and valid
func (ci *CachedImporter) getFromCache(path string) *types.Package {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	entry, exists := ci.cache[path]
	if !exists {
		return nil
	}

	// Check TTL
	if time.Since(entry.CacheTime) > ci.config.TTL {
		delete(ci.cache, path)
		ci.removeFromLRU(path)
		return nil
	}

	// Update access metadata
	entry.AccessTime = time.Now()
	entry.AccessCount++

	// Move to front of LRU
	ci.moveToFront(path)

	return entry.Package
}

// putInCache stores a package in cache with automatic eviction
func (ci *CachedImporter) putInCache(path string, pkg *types.Package) {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	// Check if we need to evict entries
	ci.evictIfNeeded()

	now := time.Now()
	entry := &ImportCacheEntry{
		Package:     pkg,
		CacheTime:   now,
		AccessTime:  now,
		AccessCount: 1,
		Size:        ci.estimatePackageSize(pkg),
	}

	ci.cache[path] = entry
	ci.addToLRU(path)
}

// evictIfNeeded evicts entries if cache is full or memory limit exceeded
func (ci *CachedImporter) evictIfNeeded() {
	// Evict by count
	for len(ci.cache) >= ci.config.MaxEntries {
		ci.evictLRU()
	}

	// Evict by memory
	if ci.config.EnableMetrics {
		memUsage := ci.estimateMemoryUsage()
		for memUsage > ci.config.MaxMemoryMB && len(ci.cache) > 0 {
			ci.evictLRU()
			memUsage = ci.estimateMemoryUsage()
		}
	}
}

// evictLRU evicts the least recently used entry
func (ci *CachedImporter) evictLRU() {
	if len(ci.lruList) == 0 {
		return
	}

	path := ci.lruList[len(ci.lruList)-1]
	delete(ci.cache, path)
	ci.lruList = ci.lruList[:len(ci.lruList)-1]

	if ci.config.EnableMetrics {
		ci.evictions++
	}
}

// addToLRU adds a path to the front of the LRU list
func (ci *CachedImporter) addToLRU(path string) {
	// Remove if already exists
	ci.removeFromLRU(path)

	// Add to front
	ci.lruList = append([]string{path}, ci.lruList...)
}

// removeFromLRU removes a path from the LRU list
func (ci *CachedImporter) removeFromLRU(path string) {
	for i, p := range ci.lruList {
		if p == path {
			ci.lruList = append(ci.lruList[:i], ci.lruList[i+1:]...)
			break
		}
	}
}

// moveToFront moves a path to the front of the LRU list
func (ci *CachedImporter) moveToFront(path string) {
	ci.removeFromLRU(path)
	ci.lruList = append([]string{path}, ci.lruList...)
}

// estimateMemoryUsage estimates current memory usage in MB
func (ci *CachedImporter) estimateMemoryUsage() int64 {
	totalSize := int64(0)
	for _, entry := range ci.cache {
		totalSize += entry.Size
	}

	return totalSize / (1024 * 1024) // Convert to MB
}

// estimatePackageSize estimates the memory size of a package
func (ci *CachedImporter) estimatePackageSize(pkg *types.Package) int64 {
	if pkg == nil {
		return 0
	}

	// Rough estimate based on package name length and scope
	size := int64(len(pkg.Name()) + len(pkg.Path()))

	// Add estimate for scope contents
	if scope := pkg.Scope(); scope != nil {
		size += int64(scope.Len() * 100) // Rough estimate per scope entry
	}

	return size
}

// WarmUp preloads standard library packages
func (ci *CachedImporter) WarmUp() error {
	standardPackages := []string{
		"fmt", "os", "io", "time", "context", "sync",
		"net/http", "encoding/json", "strings", "strconv",
		"errors", "log", "path/filepath", "regexp",
	}

	errors := ci.Preload(standardPackages)
	if len(errors) > 0 {
		// Return first error encountered
		for pkg, err := range errors {
			return fmt.Errorf("failed to preload %s: %w", pkg, err)
		}
	}

	return nil
}

// GetLoadedImporter returns the underlying importer for direct access
func (ci *CachedImporter) GetLoadedImporter() types.Importer {
	return ci.importer
}
