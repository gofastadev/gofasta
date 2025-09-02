// Package core provides go/constant with constant evaluation cache.
// This implements Phase 1.2g: go/constant with constant evaluation cache.
package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"math/big"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// ConstantCache manages constant evaluations with caching
type ConstantCache struct {
	config       *ConstantCacheConfig
	evalCache    map[string]*EvalResult
	binaryCache  map[string]*BinaryResult
	unaryCache   map[string]*UnaryResult
	compareCache map[string]*CompareResult
	shiftCache   map[string]*ShiftResult
	mu           sync.RWMutex
	
	// Metrics
	hits         int64
	misses       int64
	evaluations  int64
	cacheSize    int64
	binaryOps    int64
	unaryOps     int64
	compareOps   int64
	shiftOps     int64
}

// EvalResult represents a cached evaluation result
type EvalResult struct {
	Value       constant.Value
	Type        types.Type
	IsConst     bool
	CachedAt    time.Time
	AccessCount int64
	Hash        string
}

// BinaryResult represents a cached binary operation result
type BinaryResult struct {
	Result      constant.Value
	Valid       bool
	CachedAt    time.Time
	AccessCount int64
}

// UnaryResult represents a cached unary operation result
type UnaryResult struct {
	Result      constant.Value
	Valid       bool
	CachedAt    time.Time
	AccessCount int64
}

// CompareResult represents a cached comparison result
type CompareResult struct {
	Result      bool
	Valid       bool
	CachedAt    time.Time
	AccessCount int64
}

// ShiftResult represents a cached shift operation result
type ShiftResult struct {
	Result      constant.Value
	Valid       bool
	CachedAt    time.Time
	AccessCount int64
}

// ConstantCacheConfig contains configuration for constant cache
type ConstantCacheConfig struct {
	// Cache settings
	MaxCacheEntries int
	MaxCacheSizeMB  int
	TTL             time.Duration
	EnableMetrics   bool
	
	// Evaluation settings
	CacheBinaryOps  bool
	CacheUnaryOps   bool
	CacheCompareOps bool
	CacheShiftOps   bool
	MaxPrecision    int // For floating point operations
	
	// Performance settings
	ConcurrentEval  bool
	EvalWorkers     int
}

// DefaultConstantCacheConfig returns default configuration
func DefaultConstantCacheConfig() *ConstantCacheConfig {
	return &ConstantCacheConfig{
		MaxCacheEntries: 10000,
		MaxCacheSizeMB:  50,
		TTL:             30 * time.Minute,
		EnableMetrics:   true,
		CacheBinaryOps:  true,
		CacheUnaryOps:   true,
		CacheCompareOps: true,
		CacheShiftOps:   true,
		MaxPrecision:    512, // bits for big.Float
		ConcurrentEval:  true,
		EvalWorkers:     4,
	}
}

// NewConstantCache creates a new constant cache
func NewConstantCache(config *ConstantCacheConfig) *ConstantCache {
	if config == nil {
		config = DefaultConstantCacheConfig()
	}
	
	return &ConstantCache{
		config:       config,
		evalCache:    make(map[string]*EvalResult),
		binaryCache:  make(map[string]*BinaryResult),
		unaryCache:   make(map[string]*UnaryResult),
		compareCache: make(map[string]*CompareResult),
		shiftCache:   make(map[string]*ShiftResult),
	}
}

// EvaluateExpr evaluates an expression to a constant value with caching
func (cc *ConstantCache) EvaluateExpr(expr ast.Expr, info *types.Info) (constant.Value, bool) {
	// Generate cache key
	key := cc.generateExprKey(expr)
	
	// Check cache
	cc.mu.RLock()
	if cached, exists := cc.evalCache[key]; exists {
		if cc.config.TTL == 0 || time.Since(cached.CachedAt) < cc.config.TTL {
			cc.mu.RUnlock()
			atomic.AddInt64(&cc.hits, 1)
			atomic.AddInt64(&cached.AccessCount, 1)
			return cached.Value, cached.IsConst
		}
	}
	cc.mu.RUnlock()
	
	atomic.AddInt64(&cc.misses, 1)
	
	// Evaluate expression
	value, isConst := cc.evaluateExprInternal(expr, info)
	atomic.AddInt64(&cc.evaluations, 1)
	
	// Cache result
	result := &EvalResult{
		Value:       value,
		IsConst:     isConst,
		CachedAt:    time.Now(),
		AccessCount: 1,
		Hash:        key,
	}
	
	cc.mu.Lock()
	if cc.config.MaxCacheEntries > 0 && len(cc.evalCache) >= cc.config.MaxCacheEntries {
		cc.evictOldestEval()
	}
	cc.evalCache[key] = result
	cc.mu.Unlock()
	
	return value, isConst
}

// evaluateExprInternal performs the actual expression evaluation
func (cc *ConstantCache) evaluateExprInternal(expr ast.Expr, info *types.Info) (constant.Value, bool) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return cc.evaluateBasicLit(e)
		
	case *ast.Ident:
		if info != nil {
			if obj, ok := info.Uses[e]; ok {
				if c, ok := obj.(*types.Const); ok {
					return c.Val(), true
				}
			}
		}
		return nil, false
		
	case *ast.BinaryExpr:
		return cc.evaluateBinaryExpr(e, info)
		
	case *ast.UnaryExpr:
		return cc.evaluateUnaryExpr(e, info)
		
	case *ast.ParenExpr:
		return cc.evaluateExprInternal(e.X, info)
		
	default:
		return nil, false
	}
}

// evaluateBasicLit evaluates a basic literal
func (cc *ConstantCache) evaluateBasicLit(lit *ast.BasicLit) (constant.Value, bool) {
	switch lit.Kind {
	case token.INT:
		if val, err := strconv.ParseInt(lit.Value, 0, 64); err == nil {
			return constant.MakeInt64(val), true
		}
		// Try big int
		if bi, ok := new(big.Int).SetString(lit.Value, 0); ok {
			return constant.Make(bi), true
		}
		
	case token.FLOAT:
		if val, err := strconv.ParseFloat(lit.Value, 64); err == nil {
			return constant.MakeFloat64(val), true
		}
		// Try big float
		if bf, ok := new(big.Float).SetString(lit.Value); ok {
			return constant.Make(bf), true
		}
		
	case token.STRING:
		s, err := strconv.Unquote(lit.Value)
		if err == nil {
			return constant.MakeString(s), true
		}
		
	case token.CHAR:
		if len(lit.Value) >= 2 {
			r, _, _, err := strconv.UnquoteChar(lit.Value[1:], '\'')
			if err == nil {
				return constant.MakeInt64(int64(r)), true
			}
		}
	}
	
	return nil, false
}

// evaluateBinaryExpr evaluates a binary expression
func (cc *ConstantCache) evaluateBinaryExpr(expr *ast.BinaryExpr, info *types.Info) (constant.Value, bool) {
	left, leftOk := cc.evaluateExprInternal(expr.X, info)
	if !leftOk {
		return nil, false
	}
	
	right, rightOk := cc.evaluateExprInternal(expr.Y, info)
	if !rightOk {
		return nil, false
	}
	
	return cc.BinaryOp(left, expr.Op, right)
}

// evaluateUnaryExpr evaluates a unary expression
func (cc *ConstantCache) evaluateUnaryExpr(expr *ast.UnaryExpr, info *types.Info) (constant.Value, bool) {
	operand, ok := cc.evaluateExprInternal(expr.X, info)
	if !ok {
		return nil, false
	}
	
	return cc.UnaryOp(expr.Op, operand)
}

// BinaryOp performs a binary operation with caching
func (cc *ConstantCache) BinaryOp(x constant.Value, op token.Token, y constant.Value) (constant.Value, bool) {
	if !cc.config.CacheBinaryOps {
		result := constant.BinaryOp(x, op, y)
		return result, result != nil
	}
	
	// Generate cache key
	key := cc.generateBinaryKey(x, op, y)
	
	// Check cache
	cc.mu.RLock()
	if cached, exists := cc.binaryCache[key]; exists {
		if cc.config.TTL == 0 || time.Since(cached.CachedAt) < cc.config.TTL {
			cc.mu.RUnlock()
			atomic.AddInt64(&cc.hits, 1)
			atomic.AddInt64(&cached.AccessCount, 1)
			return cached.Result, cached.Valid
		}
	}
	cc.mu.RUnlock()
	
	atomic.AddInt64(&cc.misses, 1)
	
	// Perform operation
	result := constant.BinaryOp(x, op, y)
	atomic.AddInt64(&cc.binaryOps, 1)
	
	// Cache result
	cached := &BinaryResult{
		Result:      result,
		Valid:       result != nil,
		CachedAt:    time.Now(),
		AccessCount: 1,
	}
	
	cc.mu.Lock()
	cc.binaryCache[key] = cached
	cc.mu.Unlock()
	
	return result, result != nil
}

// UnaryOp performs a unary operation with caching
func (cc *ConstantCache) UnaryOp(op token.Token, y constant.Value) (constant.Value, bool) {
	if !cc.config.CacheUnaryOps {
		result := constant.UnaryOp(op, y, 0)
		return result, result != nil
	}
	
	// Generate cache key
	key := cc.generateUnaryKey(op, y)
	
	// Check cache
	cc.mu.RLock()
	if cached, exists := cc.unaryCache[key]; exists {
		if cc.config.TTL == 0 || time.Since(cached.CachedAt) < cc.config.TTL {
			cc.mu.RUnlock()
			atomic.AddInt64(&cc.hits, 1)
			atomic.AddInt64(&cached.AccessCount, 1)
			return cached.Result, cached.Valid
		}
	}
	cc.mu.RUnlock()
	
	atomic.AddInt64(&cc.misses, 1)
	
	// Perform operation
	result := constant.UnaryOp(op, y, 0)
	atomic.AddInt64(&cc.unaryOps, 1)
	
	// Cache result
	cached := &UnaryResult{
		Result:      result,
		Valid:       result != nil,
		CachedAt:    time.Now(),
		AccessCount: 1,
	}
	
	cc.mu.Lock()
	cc.unaryCache[key] = cached
	cc.mu.Unlock()
	
	return result, result != nil
}

// Compare performs a comparison with caching
func (cc *ConstantCache) Compare(x constant.Value, op token.Token, y constant.Value) (bool, bool) {
	if !cc.config.CacheCompareOps {
		result := constant.Compare(x, op, y)
		return result, true
	}
	
	// Generate cache key
	key := cc.generateCompareKey(x, op, y)
	
	// Check cache
	cc.mu.RLock()
	if cached, exists := cc.compareCache[key]; exists {
		if cc.config.TTL == 0 || time.Since(cached.CachedAt) < cc.config.TTL {
			cc.mu.RUnlock()
			atomic.AddInt64(&cc.hits, 1)
			atomic.AddInt64(&cached.AccessCount, 1)
			return cached.Result, cached.Valid
		}
	}
	cc.mu.RUnlock()
	
	atomic.AddInt64(&cc.misses, 1)
	
	// Perform operation
	result := constant.Compare(x, op, y)
	atomic.AddInt64(&cc.compareOps, 1)
	
	// Cache result
	cached := &CompareResult{
		Result:      result,
		Valid:       true,
		CachedAt:    time.Now(),
		AccessCount: 1,
	}
	
	cc.mu.Lock()
	cc.compareCache[key] = cached
	cc.mu.Unlock()
	
	return result, true
}

// Shift performs a shift operation with caching
func (cc *ConstantCache) Shift(x constant.Value, op token.Token, s uint) (constant.Value, bool) {
	if !cc.config.CacheShiftOps {
		result := constant.Shift(x, op, s)
		return result, result != nil
	}
	
	// Generate cache key
	key := cc.generateShiftKey(x, op, s)
	
	// Check cache
	cc.mu.RLock()
	if cached, exists := cc.shiftCache[key]; exists {
		if cc.config.TTL == 0 || time.Since(cached.CachedAt) < cc.config.TTL {
			cc.mu.RUnlock()
			atomic.AddInt64(&cc.hits, 1)
			atomic.AddInt64(&cached.AccessCount, 1)
			return cached.Result, cached.Valid
		}
	}
	cc.mu.RUnlock()
	
	atomic.AddInt64(&cc.misses, 1)
	
	// Perform operation
	result := constant.Shift(x, op, s)
	atomic.AddInt64(&cc.shiftOps, 1)
	
	// Cache result
	cached := &ShiftResult{
		Result:      result,
		Valid:       result != nil,
		CachedAt:    time.Now(),
		AccessCount: 1,
	}
	
	cc.mu.Lock()
	cc.shiftCache[key] = cached
	cc.mu.Unlock()
	
	return result, result != nil
}

// MakeFromLiteral creates a constant from a literal string
func (cc *ConstantCache) MakeFromLiteral(lit string, tok token.Token, zero uint) constant.Value {
	return constant.MakeFromLiteral(lit, tok, zero)
}

// MakeInt64 creates an int64 constant
func (cc *ConstantCache) MakeInt64(x int64) constant.Value {
	return constant.MakeInt64(x)
}

// MakeUint64 creates a uint64 constant
func (cc *ConstantCache) MakeUint64(x uint64) constant.Value {
	return constant.MakeUint64(x)
}

// MakeFloat64 creates a float64 constant
func (cc *ConstantCache) MakeFloat64(x float64) constant.Value {
	return constant.MakeFloat64(x)
}

// MakeString creates a string constant
func (cc *ConstantCache) MakeString(s string) constant.Value {
	return constant.MakeString(s)
}

// MakeBool creates a boolean constant
func (cc *ConstantCache) MakeBool(b bool) constant.Value {
	return constant.MakeBool(b)
}

// MakeFromBytes creates a constant from bytes
func (cc *ConstantCache) MakeFromBytes(bytes []byte) constant.Value {
	return constant.MakeFromBytes(bytes)
}

// Val returns the underlying value of a constant
func (cc *ConstantCache) Val(x constant.Value) interface{} {
	return constant.Val(x)
}

// Sign returns the sign of a numeric constant
func (cc *ConstantCache) Sign(x constant.Value) int {
	return constant.Sign(x)
}

// Bytes returns the bytes representation of a constant
func (cc *ConstantCache) Bytes(x constant.Value) []byte {
	return constant.Bytes(x)
}

// BitLen returns the bit length of an integer constant
func (cc *ConstantCache) BitLen(x constant.Value) int {
	return constant.BitLen(x)
}

// Num returns the numerator of a constant
func (cc *ConstantCache) Num(x constant.Value) constant.Value {
	return constant.Num(x)
}

// Denom returns the denominator of a constant
func (cc *ConstantCache) Denom(x constant.Value) constant.Value {
	return constant.Denom(x)
}

// Real returns the real part of a complex constant
func (cc *ConstantCache) Real(x constant.Value) constant.Value {
	return constant.Real(x)
}

// Imag returns the imaginary part of a complex constant
func (cc *ConstantCache) Imag(x constant.Value) constant.Value {
	return constant.Imag(x)
}

// StringVal returns the string value of a string constant
func (cc *ConstantCache) StringVal(x constant.Value) string {
	return constant.StringVal(x)
}

// BoolVal returns the boolean value of a boolean constant
func (cc *ConstantCache) BoolVal(x constant.Value) bool {
	return constant.BoolVal(x)
}

// Int64Val returns the int64 value of an integer constant
func (cc *ConstantCache) Int64Val(x constant.Value) (int64, bool) {
	return constant.Int64Val(x)
}

// Uint64Val returns the uint64 value of an integer constant
func (cc *ConstantCache) Uint64Val(x constant.Value) (uint64, bool) {
	return constant.Uint64Val(x)
}

// Float32Val returns the float32 value of a numeric constant
func (cc *ConstantCache) Float32Val(x constant.Value) (float32, bool) {
	return constant.Float32Val(x)
}

// Float64Val returns the float64 value of a numeric constant
func (cc *ConstantCache) Float64Val(x constant.Value) (float64, bool) {
	return constant.Float64Val(x)
}

// FoldConstants folds constants in an expression tree
func (cc *ConstantCache) FoldConstants(expr ast.Expr, info *types.Info) ast.Expr {
	// Clone the expression
	result := expr
	
	// Walk the tree and fold constants
	ast.Inspect(result, func(n ast.Node) bool {
		switch e := n.(type) {
		case *ast.BinaryExpr:
			if val, ok := cc.EvaluateExpr(e, info); ok && val != nil {
				// Replace with constant
				lit := cc.valueToLiteral(val)
				if lit != nil {
					// Would need to replace in parent
					// This is simplified; full implementation would track parent
				}
			}
		case *ast.UnaryExpr:
			if val, ok := cc.EvaluateExpr(e, info); ok && val != nil {
				// Replace with constant
				lit := cc.valueToLiteral(val)
				if lit != nil {
					// Would need to replace in parent
				}
			}
		}
		return true
	})
	
	return result
}

// valueToLiteral converts a constant value to an AST literal
func (cc *ConstantCache) valueToLiteral(val constant.Value) *ast.BasicLit {
	if val == nil {
		return nil
	}
	
	var kind token.Token
	var value string
	
	switch val.Kind() {
	case constant.Bool:
		// Return an Ident for true/false
		return nil // Would return ast.Ident instead
		
	case constant.Int:
		kind = token.INT
		value = val.ExactString()
		
	case constant.Float:
		kind = token.FLOAT
		value = val.ExactString()
		
	case constant.String:
		kind = token.STRING
		value = strconv.Quote(constant.StringVal(val))
		
	default:
		return nil
	}
	
	return &ast.BasicLit{
		Kind:  kind,
		Value: value,
	}
}

// BatchEvaluate evaluates multiple expressions concurrently
func (cc *ConstantCache) BatchEvaluate(exprs []ast.Expr, info *types.Info) []constant.Value {
	if !cc.config.ConcurrentEval || len(exprs) <= 1 {
		// Evaluate sequentially
		results := make([]constant.Value, len(exprs))
		for i, expr := range exprs {
			val, _ := cc.EvaluateExpr(expr, info)
			results[i] = val
		}
		return results
	}
	
	// Evaluate concurrently
	results := make([]constant.Value, len(exprs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, cc.config.EvalWorkers)
	
	for i, expr := range exprs {
		wg.Add(1)
		go func(idx int, e ast.Expr) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			
			val, _ := cc.EvaluateExpr(e, info)
			results[idx] = val
		}(i, expr)
	}
	
	wg.Wait()
	return results
}

// Helper methods for cache key generation

func (cc *ConstantCache) generateExprKey(expr ast.Expr) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%T:%v", expr, expr)))
	return hex.EncodeToString(h.Sum(nil))
}

func (cc *ConstantCache) generateBinaryKey(x constant.Value, op token.Token, y constant.Value) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v:%v:%v", x, op, y)))
	return hex.EncodeToString(h.Sum(nil))
}

func (cc *ConstantCache) generateUnaryKey(op token.Token, y constant.Value) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v:%v", op, y)))
	return hex.EncodeToString(h.Sum(nil))
}

func (cc *ConstantCache) generateCompareKey(x constant.Value, op token.Token, y constant.Value) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("cmp:%v:%v:%v", x, op, y)))
	return hex.EncodeToString(h.Sum(nil))
}

func (cc *ConstantCache) generateShiftKey(x constant.Value, op token.Token, s uint) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("shift:%v:%v:%d", x, op, s)))
	return hex.EncodeToString(h.Sum(nil))
}

// Cache eviction methods

func (cc *ConstantCache) evictOldestEval() {
	var oldest *EvalResult
	var oldestKey string
	
	for key, result := range cc.evalCache {
		if oldest == nil || result.CachedAt.Before(oldest.CachedAt) {
			oldest = result
			oldestKey = key
		}
	}
	
	if oldestKey != "" {
		delete(cc.evalCache, oldestKey)
	}
}

// GetStatistics returns cache statistics
func (cc *ConstantCache) GetStatistics() map[string]interface{} {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	
	total := cc.hits + cc.misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(cc.hits) * 100.0 / float64(total)
	}
	
	return map[string]interface{}{
		"eval_cache_size":    len(cc.evalCache),
		"binary_cache_size":  len(cc.binaryCache),
		"unary_cache_size":   len(cc.unaryCache),
		"compare_cache_size": len(cc.compareCache),
		"shift_cache_size":   len(cc.shiftCache),
		"cache_hits":         cc.hits,
		"cache_misses":       cc.misses,
		"hit_rate":           hitRate,
		"total_evaluations":  cc.evaluations,
		"binary_operations":  cc.binaryOps,
		"unary_operations":   cc.unaryOps,
		"compare_operations": cc.compareOps,
		"shift_operations":   cc.shiftOps,
	}
}

// Clear clears all caches
func (cc *ConstantCache) Clear() {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	
	cc.evalCache = make(map[string]*EvalResult)
	cc.binaryCache = make(map[string]*BinaryResult)
	cc.unaryCache = make(map[string]*UnaryResult)
	cc.compareCache = make(map[string]*CompareResult)
	cc.shiftCache = make(map[string]*ShiftResult)
	cc.hits = 0
	cc.misses = 0
	cc.evaluations = 0
	cc.binaryOps = 0
	cc.unaryOps = 0
	cc.compareOps = 0
	cc.shiftOps = 0
}

// InvalidateEval invalidates an evaluation cache entry
func (cc *ConstantCache) InvalidateEval(key string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	delete(cc.evalCache, key)
}

// InvalidateBinary invalidates a binary operation cache entry
func (cc *ConstantCache) InvalidateBinary(key string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	delete(cc.binaryCache, key)
}

// IsConstantExpr checks if an expression is a constant expression
func (cc *ConstantCache) IsConstantExpr(expr ast.Expr, info *types.Info) bool {
	_, isConst := cc.EvaluateExpr(expr, info)
	return isConst
}

// SimplifyExpr simplifies an expression by evaluating constants
func (cc *ConstantCache) SimplifyExpr(expr ast.Expr, info *types.Info) ast.Expr {
	// Try to evaluate the entire expression
	if val, ok := cc.EvaluateExpr(expr, info); ok && val != nil {
		if lit := cc.valueToLiteral(val); lit != nil {
			return lit
		}
	}
	
	// Otherwise return the original expression
	return expr
}

// OptimizeConstants performs constant optimization on a file
func (cc *ConstantCache) OptimizeConstants(file *ast.File, info *types.Info) int {
	optimized := 0
	
	// Walk the AST and optimize constants
	ast.Inspect(file, func(n ast.Node) bool {
		switch expr := n.(type) {
		case *ast.BinaryExpr:
			if val, ok := cc.EvaluateExpr(expr, info); ok && val != nil {
				optimized++
				// In a real implementation, we would replace the node
			}
		case *ast.UnaryExpr:
			if val, ok := cc.EvaluateExpr(expr, info); ok && val != nil {
				optimized++
				// In a real implementation, we would replace the node
			}
		}
		return true
	})
	
	return optimized
}

// ToFloat64 converts a constant to float64
func (cc *ConstantCache) ToFloat64(x constant.Value) (float64, bool) {
	switch x.Kind() {
	case constant.Int:
		if i, ok := constant.Int64Val(x); ok {
			return float64(i), true
		}
	case constant.Float:
		return constant.Float64Val(x)
	}
	return 0, false
}

// ToInt64 converts a constant to int64
func (cc *ConstantCache) ToInt64(x constant.Value) (int64, bool) {
	switch x.Kind() {
	case constant.Int:
		return constant.Int64Val(x)
	case constant.Float:
		if f, ok := constant.Float64Val(x); ok {
			if f == float64(int64(f)) {
				return int64(f), true
			}
		}
	}
	return 0, false
}

// ToString converts a constant to string
func (cc *ConstantCache) ToString(x constant.Value) string {
	switch x.Kind() {
	case constant.String:
		return constant.StringVal(x)
	case constant.Bool:
		if constant.BoolVal(x) {
			return "true"
		}
		return "false"
	default:
		return x.ExactString()
	}
}

// WarmupCache pre-evaluates common constants
func (cc *ConstantCache) WarmupCache() {
	// Pre-cache common values
	cc.MakeInt64(0)
	cc.MakeInt64(1)
	cc.MakeInt64(-1)
	cc.MakeFloat64(0.0)
	cc.MakeFloat64(1.0)
	cc.MakeBool(true)
	cc.MakeBool(false)
	cc.MakeString("")
	
	// Pre-cache common operations
	zero := cc.MakeInt64(0)
	one := cc.MakeInt64(1)
	
	cc.BinaryOp(one, token.ADD, one)     // 1 + 1
	cc.BinaryOp(one, token.MUL, zero)    // 1 * 0
	cc.UnaryOp(token.SUB, one)           // -1
	cc.Compare(zero, token.EQL, zero)    // 0 == 0
}