// Package core provides golang.org/x/tools/go/ast/astutil with AST operation cache.
// This implements Phase 1.2e: golang.org/x/tools/go/ast/astutil with AST operation cache.
package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/tools/go/ast/astutil"
)

// ASTUtilCache manages AST operations with caching using astutil
type ASTUtilCache struct {
	config        *ASTUtilCacheConfig
	pathCache     map[string]*PathResult
	applyCache    map[string]*ApplyResult
	importCache   map[string]*ImportInfo
	cursorCache   map[string]*CursorResult
	mu            sync.RWMutex
	fset          *token.FileSet
	
	// Metrics
	hits          int64
	misses        int64
	operations    int64
	cacheSize     int64
	applyOps      int64
	pathOps       int64
	importOps     int64
}

// PathResult represents cached path operation result
type PathResult struct {
	Path        []ast.Node
	Exact       bool
	CachedAt    time.Time
	AccessCount int64
	Hash        string
}

// ApplyResult represents cached apply operation result
type ApplyResult struct {
	Result      *ast.File
	Modified    bool
	CachedAt    time.Time
	AccessCount int64
	Hash        string
}

// ImportInfo represents cached import information
type ImportInfo struct {
	Imports     []*ast.ImportSpec
	Added       bool
	Deleted     bool
	CachedAt    time.Time
	AccessCount int64
}

// CursorResult represents cached cursor operation result
type CursorResult struct {
	Node        ast.Node
	Parent      ast.Node
	Index       int
	CachedAt    time.Time
	AccessCount int64
}

// ASTUtilCacheConfig contains configuration for AST util cache
type ASTUtilCacheConfig struct {
	// Cache settings
	MaxCacheEntries int
	MaxCacheSizeMB  int
	TTL             time.Duration
	EnableMetrics   bool
	
	// Operation settings
	CachePathOps    bool
	CacheApplyOps   bool
	CacheImportOps  bool
	CacheCursorOps  bool
	
	// Performance settings
	ConcurrentOps   bool
	OperationWorkers int
}

// DefaultASTUtilCacheConfig returns default configuration
func DefaultASTUtilCacheConfig() *ASTUtilCacheConfig {
	return &ASTUtilCacheConfig{
		MaxCacheEntries:  5000,
		MaxCacheSizeMB:   100,
		TTL:              15 * time.Minute,
		EnableMetrics:    true,
		CachePathOps:     true,
		CacheApplyOps:    true,
		CacheImportOps:   true,
		CacheCursorOps:   true,
		ConcurrentOps:    true,
		OperationWorkers: 4,
	}
}

// NewASTUtilCache creates a new AST util cache
func NewASTUtilCache(config *ASTUtilCacheConfig) *ASTUtilCache {
	if config == nil {
		config = DefaultASTUtilCacheConfig()
	}
	
	return &ASTUtilCache{
		config:      config,
		pathCache:   make(map[string]*PathResult),
		applyCache:  make(map[string]*ApplyResult),
		importCache: make(map[string]*ImportInfo),
		cursorCache: make(map[string]*CursorResult),
		fset:        token.NewFileSet(),
	}
}

// PathEnclosingInterval finds the path enclosing the given interval with caching
func (auc *ASTUtilCache) PathEnclosingInterval(root *ast.File, start, end token.Pos) ([]ast.Node, bool) {
	if !auc.config.CachePathOps {
		return astutil.PathEnclosingInterval(root, start, end)
	}
	
	// Generate cache key
	key := auc.generatePathKey(root, start, end)
	
	// Check cache
	auc.mu.RLock()
	if cached, exists := auc.pathCache[key]; exists {
		if auc.config.TTL == 0 || time.Since(cached.CachedAt) < auc.config.TTL {
			auc.mu.RUnlock()
			atomic.AddInt64(&auc.hits, 1)
			atomic.AddInt64(&cached.AccessCount, 1)
			return cached.Path, cached.Exact
		}
	}
	auc.mu.RUnlock()
	
	atomic.AddInt64(&auc.misses, 1)
	
	// Perform operation
	path, exact := astutil.PathEnclosingInterval(root, start, end)
	atomic.AddInt64(&auc.pathOps, 1)
	
	// Cache result
	result := &PathResult{
		Path:        path,
		Exact:       exact,
		CachedAt:    time.Now(),
		AccessCount: 1,
		Hash:        key,
	}
	
	auc.mu.Lock()
	if auc.config.MaxCacheEntries > 0 && len(auc.pathCache) >= auc.config.MaxCacheEntries {
		auc.evictOldestPath()
	}
	auc.pathCache[key] = result
	auc.mu.Unlock()
	
	atomic.AddInt64(&auc.operations, 1)
	return path, exact
}

// Apply applies a function to an AST with caching
func (auc *ASTUtilCache) Apply(root *ast.File, pre, post astutil.ApplyFunc) *ast.File {
	if !auc.config.CacheApplyOps {
		return astutil.Apply(root, pre, post).(*ast.File)
	}
	
	// Generate cache key
	key := auc.generateApplyKey(root, pre, post)
	
	// Check cache
	auc.mu.RLock()
	if cached, exists := auc.applyCache[key]; exists {
		if auc.config.TTL == 0 || time.Since(cached.CachedAt) < auc.config.TTL {
			auc.mu.RUnlock()
			atomic.AddInt64(&auc.hits, 1)
			atomic.AddInt64(&cached.AccessCount, 1)
			return cached.Result
		}
	}
	auc.mu.RUnlock()
	
	atomic.AddInt64(&auc.misses, 1)
	
	// Perform operation
	result := astutil.Apply(root, pre, post).(*ast.File)
	atomic.AddInt64(&auc.applyOps, 1)
	
	// Cache result
	cached := &ApplyResult{
		Result:      result,
		Modified:    !reflect.DeepEqual(root, result),
		CachedAt:    time.Now(),
		AccessCount: 1,
		Hash:        key,
	}
	
	auc.mu.Lock()
	if auc.config.MaxCacheEntries > 0 && len(auc.applyCache) >= auc.config.MaxCacheEntries {
		auc.evictOldestApply()
	}
	auc.applyCache[key] = cached
	auc.mu.Unlock()
	
	atomic.AddInt64(&auc.operations, 1)
	return result
}

// AddImport adds an import to a file with caching
func (auc *ASTUtilCache) AddImport(fset *token.FileSet, f *ast.File, path string) bool {
	if !auc.config.CacheImportOps {
		return astutil.AddImport(fset, f, path)
	}
	
	// Generate cache key
	key := auc.generateImportKey(f, "add", path)
	
	// Perform operation (imports are usually modified in place)
	added := astutil.AddImport(fset, f, path)
	atomic.AddInt64(&auc.importOps, 1)
	
	// Cache import info
	auc.mu.Lock()
	auc.importCache[key] = &ImportInfo{
		Imports:     f.Imports,
		Added:       added,
		CachedAt:    time.Now(),
		AccessCount: 1,
	}
	auc.mu.Unlock()
	
	atomic.AddInt64(&auc.operations, 1)
	return added
}

// AddNamedImport adds a named import with caching
func (auc *ASTUtilCache) AddNamedImport(fset *token.FileSet, f *ast.File, name, path string) bool {
	if !auc.config.CacheImportOps {
		return astutil.AddNamedImport(fset, f, name, path)
	}
	
	// Generate cache key
	key := auc.generateImportKey(f, "add_named", name+":"+path)
	
	// Perform operation
	added := astutil.AddNamedImport(fset, f, name, path)
	atomic.AddInt64(&auc.importOps, 1)
	
	// Cache import info
	auc.mu.Lock()
	auc.importCache[key] = &ImportInfo{
		Imports:     f.Imports,
		Added:       added,
		CachedAt:    time.Now(),
		AccessCount: 1,
	}
	auc.mu.Unlock()
	
	atomic.AddInt64(&auc.operations, 1)
	return added
}

// DeleteImport deletes an import with caching
func (auc *ASTUtilCache) DeleteImport(fset *token.FileSet, f *ast.File, path string) bool {
	if !auc.config.CacheImportOps {
		return astutil.DeleteImport(fset, f, path)
	}
	
	// Generate cache key
	key := auc.generateImportKey(f, "delete", path)
	
	// Perform operation
	deleted := astutil.DeleteImport(fset, f, path)
	atomic.AddInt64(&auc.importOps, 1)
	
	// Cache import info
	auc.mu.Lock()
	auc.importCache[key] = &ImportInfo{
		Imports:     f.Imports,
		Deleted:     deleted,
		CachedAt:    time.Now(),
		AccessCount: 1,
	}
	auc.mu.Unlock()
	
	atomic.AddInt64(&auc.operations, 1)
	return deleted
}

// DeleteNamedImport deletes a named import with caching
func (auc *ASTUtilCache) DeleteNamedImport(fset *token.FileSet, f *ast.File, name, path string) bool {
	if !auc.config.CacheImportOps {
		return astutil.DeleteNamedImport(fset, f, name, path)
	}
	
	// Generate cache key
	key := auc.generateImportKey(f, "delete_named", name+":"+path)
	
	// Perform operation
	deleted := astutil.DeleteNamedImport(fset, f, name, path)
	atomic.AddInt64(&auc.importOps, 1)
	
	// Cache import info
	auc.mu.Lock()
	auc.importCache[key] = &ImportInfo{
		Imports:     f.Imports,
		Deleted:     deleted,
		CachedAt:    time.Now(),
		AccessCount: 1,
	}
	auc.mu.Unlock()
	
	atomic.AddInt64(&auc.operations, 1)
	return deleted
}

// RewriteImport rewrites imports in a file
func (auc *ASTUtilCache) RewriteImport(fset *token.FileSet, f *ast.File, oldPath, newPath string) bool {
	// This doesn't benefit much from caching as it modifies the AST
	rewrote := astutil.RewriteImport(fset, f, oldPath, newPath)
	atomic.AddInt64(&auc.importOps, 1)
	atomic.AddInt64(&auc.operations, 1)
	return rewrote
}

// UsesImport checks if a file uses an import
func (auc *ASTUtilCache) UsesImport(f *ast.File, path string) bool {
	// Generate cache key
	key := auc.generateImportKey(f, "uses", path)
	
	// Check cache
	auc.mu.RLock()
	if cached, exists := auc.importCache[key]; exists {
		if auc.config.TTL == 0 || time.Since(cached.CachedAt) < auc.config.TTL {
			auc.mu.RUnlock()
			atomic.AddInt64(&auc.hits, 1)
			atomic.AddInt64(&cached.AccessCount, 1)
			// Check if import exists
			for _, imp := range cached.Imports {
				if imp.Path.Value == `"`+path+`"` {
					return true
				}
			}
			return false
		}
	}
	auc.mu.RUnlock()
	
	// Perform operation
	uses := astutil.UsesImport(f, path)
	atomic.AddInt64(&auc.operations, 1)
	return uses
}

// Imports returns all imports in a file
func (auc *ASTUtilCache) Imports(fset *token.FileSet, f *ast.File) [][]*ast.ImportSpec {
	return astutil.Imports(fset, f)
}

// NodeDescription returns a description of a node
func (auc *ASTUtilCache) NodeDescription(n ast.Node) string {
	return astutil.NodeDescription(n)
}

// Unparen removes parentheses from an expression
func (auc *ASTUtilCache) Unparen(e ast.Expr) ast.Expr {
	return astutil.Unparen(e)
}

// BatchApply applies multiple operations concurrently
func (auc *ASTUtilCache) BatchApply(files []*ast.File, pre, post astutil.ApplyFunc) []*ast.File {
	if !auc.config.ConcurrentOps || len(files) <= 1 {
		// Apply sequentially
		results := make([]*ast.File, len(files))
		for i, file := range files {
			results[i] = auc.Apply(file, pre, post)
		}
		return results
	}
	
	// Apply concurrently
	results := make([]*ast.File, len(files))
	var wg sync.WaitGroup
	sem := make(chan struct{}, auc.config.OperationWorkers)
	
	for i, file := range files {
		wg.Add(1)
		go func(idx int, f *ast.File) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			
			results[idx] = auc.Apply(f, pre, post)
		}(i, file)
	}
	
	wg.Wait()
	return results
}

// FindIdentifiers finds all identifiers in a file
func (auc *ASTUtilCache) FindIdentifiers(f *ast.File) []*ast.Ident {
	var idents []*ast.Ident
	
	ast.Inspect(f, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok {
			idents = append(idents, ident)
		}
		return true
	})
	
	return idents
}

// FindFuncDecls finds all function declarations
func (auc *ASTUtilCache) FindFuncDecls(f *ast.File) []*ast.FuncDecl {
	var funcs []*ast.FuncDecl
	
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			funcs = append(funcs, fn)
		}
	}
	
	return funcs
}

// FindTypeSpecs finds all type specifications
func (auc *ASTUtilCache) FindTypeSpecs(f *ast.File) []*ast.TypeSpec {
	var types []*ast.TypeSpec
	
	for _, decl := range f.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.TYPE {
			for _, spec := range gen.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					types = append(types, ts)
				}
			}
		}
	}
	
	return types
}

// RenameIdent renames all occurrences of an identifier
func (auc *ASTUtilCache) RenameIdent(f *ast.File, oldName, newName string) int {
	count := 0
	
	ast.Inspect(f, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok && ident.Name == oldName {
			ident.Name = newName
			count++
		}
		return true
	})
	
	atomic.AddInt64(&auc.operations, 1)
	return count
}

// RemoveUnusedImports removes unused imports from a file
func (auc *ASTUtilCache) RemoveUnusedImports(fset *token.FileSet, f *ast.File) int {
	removed := 0
	
	// Get all imports
	imports := astutil.Imports(fset, f)
	
	for _, group := range imports {
		for _, imp := range group {
			path := strings.Trim(imp.Path.Value, `"`)
			if !astutil.UsesImport(f, path) {
				if astutil.DeleteImport(fset, f, path) {
					removed++
				}
			}
		}
	}
	
	atomic.AddInt64(&auc.operations, 1)
	return removed
}

// OptimizeImports sorts and groups imports
func (auc *ASTUtilCache) OptimizeImports(fset *token.FileSet, f *ast.File) {
	// Collect all imports
	var stdImports []string
	var extImports []string
	var localImports []string
	
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		
		if !strings.Contains(path, ".") {
			stdImports = append(stdImports, path)
		} else if strings.Contains(path, "github.com/healtronlabs") {
			localImports = append(localImports, path)
		} else {
			extImports = append(extImports, path)
		}
		
		// Remove the import
		astutil.DeleteImport(fset, f, path)
	}
	
	// Sort each group
	sort.Strings(stdImports)
	sort.Strings(extImports)
	sort.Strings(localImports)
	
	// Re-add imports in order
	for _, path := range stdImports {
		astutil.AddImport(fset, f, path)
	}
	for _, path := range extImports {
		astutil.AddImport(fset, f, path)
	}
	for _, path := range localImports {
		astutil.AddImport(fset, f, path)
	}
	
	atomic.AddInt64(&auc.operations, 1)
}

// Helper methods for cache key generation

func (auc *ASTUtilCache) generatePathKey(root *ast.File, start, end token.Pos) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%p:%d:%d", root, start, end)))
	return hex.EncodeToString(h.Sum(nil))
}

func (auc *ASTUtilCache) generateApplyKey(root *ast.File, pre, post astutil.ApplyFunc) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%p:%p:%p", root, pre, post)))
	return hex.EncodeToString(h.Sum(nil))
}

func (auc *ASTUtilCache) generateImportKey(f *ast.File, op, path string) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%p:%s:%s", f, op, path)))
	return hex.EncodeToString(h.Sum(nil))
}

// Cache eviction methods

func (auc *ASTUtilCache) evictOldestPath() {
	var oldest *PathResult
	var oldestKey string
	
	for key, result := range auc.pathCache {
		if oldest == nil || result.CachedAt.Before(oldest.CachedAt) {
			oldest = result
			oldestKey = key
		}
	}
	
	if oldestKey != "" {
		delete(auc.pathCache, oldestKey)
	}
}

func (auc *ASTUtilCache) evictOldestApply() {
	var oldest *ApplyResult
	var oldestKey string
	
	for key, result := range auc.applyCache {
		if oldest == nil || result.CachedAt.Before(oldest.CachedAt) {
			oldest = result
			oldestKey = key
		}
	}
	
	if oldestKey != "" {
		delete(auc.applyCache, oldestKey)
	}
}

// GetStatistics returns cache statistics
func (auc *ASTUtilCache) GetStatistics() map[string]interface{} {
	auc.mu.RLock()
	defer auc.mu.RUnlock()
	
	total := auc.hits + auc.misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(auc.hits) * 100.0 / float64(total)
	}
	
	return map[string]interface{}{
		"path_cache_size":   len(auc.pathCache),
		"apply_cache_size":  len(auc.applyCache),
		"import_cache_size": len(auc.importCache),
		"cursor_cache_size": len(auc.cursorCache),
		"cache_hits":        auc.hits,
		"cache_misses":      auc.misses,
		"hit_rate":          hitRate,
		"total_operations":  auc.operations,
		"apply_operations":  auc.applyOps,
		"path_operations":   auc.pathOps,
		"import_operations": auc.importOps,
	}
}

// Clear clears all caches
func (auc *ASTUtilCache) Clear() {
	auc.mu.Lock()
	defer auc.mu.Unlock()
	
	auc.pathCache = make(map[string]*PathResult)
	auc.applyCache = make(map[string]*ApplyResult)
	auc.importCache = make(map[string]*ImportInfo)
	auc.cursorCache = make(map[string]*CursorResult)
	auc.hits = 0
	auc.misses = 0
	auc.operations = 0
	auc.applyOps = 0
	auc.pathOps = 0
	auc.importOps = 0
}

// InvalidatePath invalidates a path cache entry
func (auc *ASTUtilCache) InvalidatePath(key string) {
	auc.mu.Lock()
	defer auc.mu.Unlock()
	delete(auc.pathCache, key)
}

// InvalidateApply invalidates an apply cache entry
func (auc *ASTUtilCache) InvalidateApply(key string) {
	auc.mu.Lock()
	defer auc.mu.Unlock()
	delete(auc.applyCache, key)
}

// InvalidateImport invalidates an import cache entry
func (auc *ASTUtilCache) InvalidateImport(key string) {
	auc.mu.Lock()
	defer auc.mu.Unlock()
	delete(auc.importCache, key)
}

// ASTWalker provides a convenient way to walk AST nodes
type ASTWalker struct {
	cache *ASTUtilCache
}

// NewASTWalker creates a new AST walker
func (auc *ASTUtilCache) NewASTWalker() *ASTWalker {
	return &ASTWalker{cache: auc}
}

// Walk walks the AST and calls the visitor function for each node
func (w *ASTWalker) Walk(node ast.Node, visitor func(ast.Node) bool) {
	ast.Inspect(node, visitor)
}

// FindNodes finds all nodes of a specific type
func (w *ASTWalker) FindNodes(root ast.Node, nodeType reflect.Type) []ast.Node {
	var nodes []ast.Node
	
	w.Walk(root, func(n ast.Node) bool {
		if n != nil && reflect.TypeOf(n) == nodeType {
			nodes = append(nodes, n)
		}
		return true
	})
	
	return nodes
}

// FindNodesByName finds nodes with a specific name
func (w *ASTWalker) FindNodesByName(root ast.Node, name string) []ast.Node {
	var nodes []ast.Node
	
	w.Walk(root, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Ident:
			if x.Name == name {
				nodes = append(nodes, n)
			}
		case *ast.FuncDecl:
			if x.Name.Name == name {
				nodes = append(nodes, n)
			}
		case *ast.TypeSpec:
			if x.Name.Name == name {
				nodes = append(nodes, n)
			}
		}
		return true
	})
	
	return nodes
}

// GetParent finds the parent of a node
func (w *ASTWalker) GetParent(root, target ast.Node) ast.Node {
	var parent ast.Node
	
	w.Walk(root, func(n ast.Node) bool {
		if n == target {
			return false
		}
		
		// Check if any child is the target
		childFound := false
		ast.Inspect(n, func(child ast.Node) bool {
			if child == target {
				childFound = true
				return false
			}
			if child == n {
				return true
			}
			return false
		})
		
		if childFound {
			parent = n
			return false
		}
		
		return true
	})
	
	return parent
}

// ExtractComments extracts all comments from a file
func (auc *ASTUtilCache) ExtractComments(f *ast.File) []*ast.CommentGroup {
	var comments []*ast.CommentGroup
	
	// File-level comments
	if f.Doc != nil {
		comments = append(comments, f.Doc)
	}
	
	// Comments in declarations
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Doc != nil {
				comments = append(comments, d.Doc)
			}
		case *ast.GenDecl:
			if d.Doc != nil {
				comments = append(comments, d.Doc)
			}
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.ValueSpec:
					if s.Doc != nil {
						comments = append(comments, s.Doc)
					}
				case *ast.TypeSpec:
					if s.Doc != nil {
						comments = append(comments, s.Doc)
					}
				}
			}
		}
	}
	
	// All comments in the file
	comments = append(comments, f.Comments...)
	
	return comments
}

// HasSideEffects checks if an expression has side effects
func (auc *ASTUtilCache) HasSideEffects(expr ast.Expr) bool {
	hasSideEffects := false
	
	ast.Inspect(expr, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.CallExpr, *ast.UnaryExpr, *ast.IncDecStmt, *ast.AssignStmt:
			hasSideEffects = true
			return false
		}
		return true
	})
	
	return hasSideEffects
}

// SimplifyExpr attempts to simplify an expression
func (auc *ASTUtilCache) SimplifyExpr(expr ast.Expr) ast.Expr {
	// Remove unnecessary parentheses
	expr = astutil.Unparen(expr)
	
	// Further simplification could be added here
	// For example, constant folding, dead code elimination, etc.
	
	return expr
}

// GetEnclosingFunction finds the enclosing function for a node
func (auc *ASTUtilCache) GetEnclosingFunction(root *ast.File, node ast.Node) *ast.FuncDecl {
	var enclosingFunc *ast.FuncDecl
	
	ast.Inspect(root, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			// Check if node is within this function
			if auc.containsNode(fn, node) {
				enclosingFunc = fn
			}
		}
		return true
	})
	
	return enclosingFunc
}

// containsNode checks if a parent node contains a child node
func (auc *ASTUtilCache) containsNode(parent, child ast.Node) bool {
	found := false
	
	ast.Inspect(parent, func(n ast.Node) bool {
		if n == child {
			found = true
			return false
		}
		return true
	})
	
	return found
}