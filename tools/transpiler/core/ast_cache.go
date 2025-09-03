// Package core provides AST caching capabilities for GoFasta transpiler.
// This implements Phase 1.1b: Integrate go/ast with AST caching system.
package core

import (
	"go/ast"
	"go/token"
	"sync"
	"time"
)

// ASTCacheEntry represents a cached AST with metadata
type ASTCacheEntry struct {
	File         *ast.File
	FileSet      *token.FileSet
	ModTime      time.Time
	Size         int64
	CacheTime    time.Time
	AccessTime   time.Time
	AccessCount  int64
}

// ASTCacheConfig contains configuration options for the AST cache
type ASTCacheConfig struct {
	// MaxEntries sets the maximum number of cached ASTs
	// If 0, defaults to 1000
	MaxEntries int
	
	// TTL sets the time-to-live for cached entries
	// If 0, defaults to 1 hour
	TTL time.Duration
	
	// MaxMemoryMB sets the maximum memory usage in MB
	// If 0, defaults to 512MB
	MaxMemoryMB int64
	
	// EnableMetrics enables cache metrics collection
	EnableMetrics bool
}

// DefaultASTCacheConfig returns a default AST cache configuration
func DefaultASTCacheConfig() *ASTCacheConfig {
	return &ASTCacheConfig{
		MaxEntries:    1000,
		TTL:           time.Hour,
		MaxMemoryMB:   512,
		EnableMetrics: true,
	}
}

// ASTCache provides high-performance AST caching with LRU eviction
type ASTCache struct {
	config    *ASTCacheConfig
	cache     map[string]*ASTCacheEntry
	lruList   []string
	mu        sync.RWMutex
	
	// Metrics
	hits      int64
	misses    int64
	evictions int64
	memoryMB  int64
}

// NewASTCache creates a new high-performance AST cache
func NewASTCache(config *ASTCacheConfig) *ASTCache {
	if config == nil {
		config = DefaultASTCacheConfig()
	}
	
	if config.MaxEntries <= 0 {
		config.MaxEntries = 1000
	}
	
	if config.TTL <= 0 {
		config.TTL = time.Hour
	}
	
	if config.MaxMemoryMB <= 0 {
		config.MaxMemoryMB = 512
	}
	
	return &ASTCache{
		config:  config,
		cache:   make(map[string]*ASTCacheEntry),
		lruList: make([]string, 0, config.MaxEntries),
	}
}

// Get retrieves an AST from cache if available and valid
func (c *ASTCache) Get(key string, modTime time.Time, size int64) (*ast.File, *token.FileSet, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	entry, exists := c.cache[key]
	if !exists {
		if c.config.EnableMetrics {
			c.misses++
		}
		return nil, nil, false
	}
	
	// Check if entry is still valid
	if !c.isValidEntry(entry, modTime, size) {
		delete(c.cache, key)
		c.removeFromLRU(key)
		if c.config.EnableMetrics {
			c.misses++
		}
		return nil, nil, false
	}
	
	// Update access metadata
	entry.AccessTime = time.Now()
	entry.AccessCount++
	
	// Move to front of LRU
	c.moveToFront(key)
	
	if c.config.EnableMetrics {
		c.hits++
	}
	
	return entry.File, entry.FileSet, true
}

// Put stores an AST in cache with automatic eviction
func (c *ASTCache) Put(key string, file *ast.File, fileSet *token.FileSet, modTime time.Time, size int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	entry := &ASTCacheEntry{
		File:        file,
		FileSet:     fileSet,
		ModTime:     modTime,
		Size:        size,
		CacheTime:   now,
		AccessTime:  now,
		AccessCount: 1,
	}
	
	// Check if we need to evict entries
	c.evictIfNeeded()
	
	// Add new entry
	c.cache[key] = entry
	c.addToLRU(key)
	
	// Update memory usage estimate
	if c.config.EnableMetrics {
		c.updateMemoryUsage()
	}
}

// Clear removes all entries from cache
func (c *ASTCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.cache = make(map[string]*ASTCacheEntry)
	c.lruList = make([]string, 0, c.config.MaxEntries)
	c.memoryMB = 0
}

// GetStatistics returns cache performance statistics
func (c *ASTCache) GetStatistics() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	totalRequests := c.hits + c.misses
	hitRatio := 0.0
	if totalRequests > 0 {
		hitRatio = float64(c.hits) / float64(totalRequests) * 100.0
	}
	
	return map[string]interface{}{
		"entries":      len(c.cache),
		"max_entries":  c.config.MaxEntries,
		"hits":         c.hits,
		"misses":       c.misses,
		"evictions":    c.evictions,
		"hit_ratio":    hitRatio,
		"memory_mb":    c.memoryMB,
		"max_memory":   c.config.MaxMemoryMB,
	}
}

// Cleanup removes expired entries
func (c *ASTCache) Cleanup() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	removed := 0
	now := time.Now()
	
	for key, entry := range c.cache {
		if now.Sub(entry.CacheTime) > c.config.TTL {
			delete(c.cache, key)
			c.removeFromLRU(key)
			removed++
		}
	}
	
	if c.config.EnableMetrics {
		c.updateMemoryUsage()
	}
	
	return removed
}

// isValidEntry checks if a cache entry is still valid
func (c *ASTCache) isValidEntry(entry *ASTCacheEntry, modTime time.Time, size int64) bool {
	// Check TTL
	if time.Since(entry.CacheTime) > c.config.TTL {
		return false
	}
	
	// Check if file has been modified
	if !entry.ModTime.Equal(modTime) || entry.Size != size {
		return false
	}
	
	return true
}

// evictIfNeeded evicts entries if cache is full or memory limit exceeded
func (c *ASTCache) evictIfNeeded() {
	// Evict by count
	for len(c.cache) >= c.config.MaxEntries {
		c.evictLRU()
	}
	
	// Evict by memory (rough estimate)
	if c.config.EnableMetrics {
		c.updateMemoryUsage()
		for c.memoryMB > c.config.MaxMemoryMB && len(c.cache) > 0 {
			c.evictLRU()
		}
	}
}

// evictLRU evicts the least recently used entry
func (c *ASTCache) evictLRU() {
	if len(c.lruList) == 0 {
		return
	}
	
	key := c.lruList[len(c.lruList)-1]
	delete(c.cache, key)
	c.lruList = c.lruList[:len(c.lruList)-1]
	
	if c.config.EnableMetrics {
		c.evictions++
	}
}

// addToLRU adds a key to the front of the LRU list
func (c *ASTCache) addToLRU(key string) {
	// Remove if already exists
	c.removeFromLRU(key)
	
	// Add to front
	c.lruList = append([]string{key}, c.lruList...)
}

// removeFromLRU removes a key from the LRU list
func (c *ASTCache) removeFromLRU(key string) {
	for i, k := range c.lruList {
		if k == key {
			c.lruList = append(c.lruList[:i], c.lruList[i+1:]...)
			break
		}
	}
}

// moveToFront moves a key to the front of the LRU list
func (c *ASTCache) moveToFront(key string) {
	c.removeFromLRU(key)
	c.lruList = append([]string{key}, c.lruList...)
}

// updateMemoryUsage estimates current memory usage
func (c *ASTCache) updateMemoryUsage() {
	// Rough estimate: 1KB per AST node average
	totalNodes := int64(0)
	for _, entry := range c.cache {
		if entry.File != nil {
			nodeCount := c.countASTNodes(entry.File)
			totalNodes += int64(nodeCount)
		}
	}
	
	c.memoryMB = totalNodes / 1024 // KB to MB conversion (rough)
}

// countASTNodes recursively counts AST nodes
func (c *ASTCache) countASTNodes(node ast.Node) int {
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