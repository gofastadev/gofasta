// Package core provides token pooling capabilities for GoFasta transpiler.
// This implements Phase 1.1c: Configure go/token with memory pooling.
package core

import (
	"go/token"
	"sync"
)

// TokenPoolConfig contains configuration options for the token pool
type TokenPoolConfig struct {
	// InitialSize sets the initial pool size
	// If 0, defaults to 10
	InitialSize int
	
	// MaxSize sets the maximum pool size
	// If 0, defaults to 100
	MaxSize int
	
	// EnableMetrics enables pool metrics collection
	EnableMetrics bool
}

// DefaultTokenPoolConfig returns a default token pool configuration
func DefaultTokenPoolConfig() *TokenPoolConfig {
	return &TokenPoolConfig{
		InitialSize:   10,
		MaxSize:       100,
		EnableMetrics: true,
	}
}

// TokenPool provides memory-pooled token.FileSet instances for high performance
type TokenPool struct {
	config *TokenPoolConfig
	pool   chan *token.FileSet
	mu     sync.RWMutex
	
	// Metrics
	created   int64
	reused    int64
	discarded int64
}

// NewTokenPool creates a new high-performance token pool
func NewTokenPool(config *TokenPoolConfig) *TokenPool {
	if config == nil {
		config = DefaultTokenPoolConfig()
	}
	
	if config.InitialSize <= 0 {
		config.InitialSize = 10
	}
	
	if config.MaxSize <= 0 {
		config.MaxSize = 100
	}
	
	if config.MaxSize < config.InitialSize {
		config.MaxSize = config.InitialSize
	}
	
	pool := &TokenPool{
		config: config,
		pool:   make(chan *token.FileSet, config.MaxSize),
	}
	
	// Pre-populate pool with initial FileSets
	for i := 0; i < config.InitialSize; i++ {
		select {
		case pool.pool <- token.NewFileSet():
			if config.EnableMetrics {
				pool.created++
			}
		default:
			// Pool is full, stop trying to add more
			goto donePopulating
		}
	}
donePopulating:
	
	return pool
}

// Get retrieves a FileSet from the pool or creates a new one
func (p *TokenPool) Get() *token.FileSet {
	select {
	case fset := <-p.pool:
		if p.config.EnableMetrics {
			p.mu.Lock()
			p.reused++
			p.mu.Unlock()
		}
		return fset
	default:
		// Pool is empty, create new FileSet
		if p.config.EnableMetrics {
			p.mu.Lock()
			p.created++
			p.mu.Unlock()
		}
		return token.NewFileSet()
	}
}

// Put returns a FileSet to the pool for reuse
func (p *TokenPool) Put(fset *token.FileSet) {
	if fset == nil {
		return
	}
	
	// Reset the FileSet by creating a new one
	// (go/token.FileSet doesn't have a Reset method)
	newFset := token.NewFileSet()
	
	select {
	case p.pool <- newFset:
		// Successfully returned to pool
	default:
		// Pool is full, discard
		if p.config.EnableMetrics {
			p.mu.Lock()
			p.discarded++
			p.mu.Unlock()
		}
	}
}

// GetStatistics returns pool performance statistics
func (p *TokenPool) GetStatistics() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	poolSize := len(p.pool)
	maxSize := cap(p.pool)
	utilization := 0.0
	if maxSize > 0 {
		utilization = float64(maxSize-poolSize) / float64(maxSize) * 100.0
	}
	
	reuseRate := 0.0
	total := p.created + p.reused
	if total > 0 {
		reuseRate = float64(p.reused) / float64(total) * 100.0
	}
	
	return map[string]interface{}{
		"pool_size":      poolSize,
		"max_size":       maxSize,
		"utilization":    utilization,
		"created":        p.created,
		"reused":         p.reused,
		"discarded":      p.discarded,
		"reuse_rate":     reuseRate,
		"total_requests": total,
	}
}

// Drain removes all FileSets from the pool
func (p *TokenPool) Drain() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	drained := 0
	for {
		select {
		case <-p.pool:
			drained++
		default:
			return drained
		}
	}
}

// Resize changes the pool size (creates new pool with different capacity)
func (p *TokenPool) Resize(newMaxSize int) {
	if newMaxSize <= 0 {
		return
	}
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Create new pool
	newPool := make(chan *token.FileSet, newMaxSize)
	
	// Transfer existing FileSets up to new capacity
	transferred := 0
	oldPoolSize := len(p.pool)
	maxTransfer := newMaxSize
	if oldPoolSize < maxTransfer {
		maxTransfer = oldPoolSize
	}
	
	for transferred < maxTransfer {
		select {
		case fset := <-p.pool:
			select {
			case newPool <- fset:
				transferred++
			default:
				// New pool is full
				goto doneTransferring
			}
		default:
			// Old pool is empty
			goto doneTransferring
		}
	}
doneTransferring:
	
	// Update pool and config
	p.pool = newPool
	p.config.MaxSize = newMaxSize
}

// WarmUp pre-fills the pool to the specified size
func (p *TokenPool) WarmUp(targetSize int) {
	if targetSize <= 0 || targetSize > p.config.MaxSize {
		return // Do nothing for invalid sizes
	}
	
	currentSize := len(p.pool)
	needed := targetSize - currentSize
	
	for i := 0; i < needed; i++ {
		select {
		case p.pool <- token.NewFileSet():
			if p.config.EnableMetrics {
				p.mu.Lock()
				p.created++
				p.mu.Unlock()
			}
		default:
			// Pool is full
			return
		}
	}
}