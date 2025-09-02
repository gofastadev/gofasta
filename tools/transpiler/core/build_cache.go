// Package core provides go/build with build constraint cache.
// This implements Phase 1.2h: go/build with build constraint cache.
package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// BuildCache manages build constraints and package information with caching
type BuildCache struct {
	config          *BuildCacheConfig
	constraintCache map[string]*ConstraintResult
	packageCache    map[string]*BuildPackage
	importCache     map[string]*ImportResult
	contextCache    map[string]*build.Context
	fileCache       map[string]*FileInfo
	fset            *token.FileSet
	mu              sync.RWMutex
	
	// Metrics
	hits            int64
	misses          int64
	evaluations     int64
	cacheSize       int64
	constraintEvals int64
	packageLoads    int64
	importResolves  int64
}

// ConstraintResult represents cached build constraint evaluation
type ConstraintResult struct {
	Satisfied   bool
	Constraints []string
	Tags        map[string]bool
	OS          string
	Arch        string
	CachedAt    time.Time
	AccessCount int64
	Hash        string
}

// BuildPackage represents cached package build information
type BuildPackage struct {
	*build.Package
	Dependencies []string
	TestImports  []string
	Constraints  *ConstraintResult
	CachedAt     time.Time
	AccessCount  int64
}

// ImportResult represents cached import resolution
type ImportResult struct {
	Path        string
	Dir         string
	ImportMode  build.ImportMode
	Package     *build.Package
	Error       error
	CachedAt    time.Time
	AccessCount int64
}

// FileInfo represents cached file build information
type FileInfo struct {
	Path        string
	Package     string
	Imports     []string
	TestImports []string
	Constraints []string
	IsTest      bool
	IsGenerated bool
	CachedAt    time.Time
	AccessCount int64
}

// BuildCacheConfig contains configuration for build cache
type BuildCacheConfig struct {
	// Cache settings
	MaxCacheEntries int
	MaxCacheSizeMB  int
	TTL             time.Duration
	EnableMetrics   bool
	
	// Build settings
	DefaultContext  *build.Context
	CustomTags      []string
	IgnoreVendor    bool
	AllowBinary     bool
	IncludeTests    bool
	
	// Performance settings
	ConcurrentLoads bool
	LoadWorkers     int
	PreloadStdlib   bool
	CacheStdlib     bool
}

// DefaultBuildCacheConfig returns default configuration
func DefaultBuildCacheConfig() *BuildCacheConfig {
	return &BuildCacheConfig{
		MaxCacheEntries: 5000,
		MaxCacheSizeMB:  100,
		TTL:             30 * time.Minute,
		EnableMetrics:   true,
		DefaultContext:  &build.Default,
		IgnoreVendor:    true,
		AllowBinary:     false,
		ConcurrentLoads: true,
		LoadWorkers:     4,
		PreloadStdlib:   false,
		CacheStdlib:     true,
	}
}

// NewBuildCache creates a new build cache
func NewBuildCache(config *BuildCacheConfig) *BuildCache {
	if config == nil {
		config = DefaultBuildCacheConfig()
	}
	
	bc := &BuildCache{
		config:          config,
		constraintCache: make(map[string]*ConstraintResult),
		packageCache:    make(map[string]*BuildPackage),
		importCache:     make(map[string]*ImportResult),
		contextCache:    make(map[string]*build.Context),
		fileCache:       make(map[string]*FileInfo),
		fset:            token.NewFileSet(),
	}
	
	// Set default context if not provided
	if config.DefaultContext == nil {
		config.DefaultContext = &build.Default
	}
	
	// Preload standard library if requested
	if config.PreloadStdlib {
		go bc.preloadStandardLibrary()
	}
	
	return bc
}

// EvaluateConstraints evaluates build constraints with caching
func (bc *BuildCache) EvaluateConstraints(filename string, src []byte) (bool, error) {
	// Generate cache key
	key := bc.generateConstraintKey(filename, src)
	
	// Check cache
	bc.mu.RLock()
	if cached, exists := bc.constraintCache[key]; exists {
		if bc.config.TTL == 0 || time.Since(cached.CachedAt) < bc.config.TTL {
			bc.mu.RUnlock()
			atomic.AddInt64(&bc.hits, 1)
			atomic.AddInt64(&cached.AccessCount, 1)
			return cached.Satisfied, nil
		}
	}
	bc.mu.RUnlock()
	
	atomic.AddInt64(&bc.misses, 1)
	
	// Parse build constraints
	constraints, err := bc.parseBuildConstraints(filename, src)
	if err != nil {
		return false, err
	}
	
	// Evaluate constraints
	satisfied := bc.evaluateConstraintList(constraints)
	atomic.AddInt64(&bc.constraintEvals, 1)
	
	// Cache result
	result := &ConstraintResult{
		Satisfied:   satisfied,
		Constraints: constraints,
		Tags:        bc.getCurrentTags(),
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		CachedAt:    time.Now(),
		AccessCount: 1,
		Hash:        key,
	}
	
	bc.mu.Lock()
	if bc.config.MaxCacheEntries > 0 && len(bc.constraintCache) >= bc.config.MaxCacheEntries {
		bc.evictOldestConstraint()
	}
	bc.constraintCache[key] = result
	bc.mu.Unlock()
	
	atomic.AddInt64(&bc.evaluations, 1)
	return satisfied, nil
}

// ImportPackage imports a package with caching
func (bc *BuildCache) ImportPackage(path string, srcDir string, mode build.ImportMode) (*build.Package, error) {
	// Generate cache key
	key := bc.generateImportKey(path, srcDir, mode)
	
	// Check cache
	bc.mu.RLock()
	if cached, exists := bc.importCache[key]; exists {
		if bc.config.TTL == 0 || time.Since(cached.CachedAt) < bc.config.TTL {
			bc.mu.RUnlock()
			atomic.AddInt64(&bc.hits, 1)
			atomic.AddInt64(&cached.AccessCount, 1)
			return cached.Package, cached.Error
		}
	}
	bc.mu.RUnlock()
	
	atomic.AddInt64(&bc.misses, 1)
	
	// Import package
	ctx := bc.getContext()
	pkg, err := ctx.Import(path, srcDir, mode)
	atomic.AddInt64(&bc.importResolves, 1)
	
	// Cache result
	result := &ImportResult{
		Path:        path,
		Dir:         srcDir,
		ImportMode:  mode,
		Package:     pkg,
		Error:       err,
		CachedAt:    time.Now(),
		AccessCount: 1,
	}
	
	bc.mu.Lock()
	bc.importCache[key] = result
	bc.mu.Unlock()
	
	return pkg, err
}

// LoadPackage loads package information with caching
func (bc *BuildCache) LoadPackage(dir string) (*BuildPackage, error) {
	// Check cache
	bc.mu.RLock()
	if cached, exists := bc.packageCache[dir]; exists {
		if bc.config.TTL == 0 || time.Since(cached.CachedAt) < bc.config.TTL {
			bc.mu.RUnlock()
			atomic.AddInt64(&bc.hits, 1)
			atomic.AddInt64(&cached.AccessCount, 1)
			return cached, nil
		}
	}
	bc.mu.RUnlock()
	
	atomic.AddInt64(&bc.misses, 1)
	
	// Load package
	ctx := bc.getContext()
	pkg, err := ctx.ImportDir(dir, 0)
	if err != nil {
		return nil, err
	}
	
	atomic.AddInt64(&bc.packageLoads, 1)
	
	// Evaluate constraints for the package
	constraints := bc.evaluatePackageConstraints(pkg)
	
	// Create cached package
	buildPkg := &BuildPackage{
		Package:      pkg,
		Dependencies: bc.extractDependencies(pkg),
		TestImports:  bc.extractTestImports(pkg),
		Constraints:  constraints,
		CachedAt:     time.Now(),
		AccessCount:  1,
	}
	
	// Cache result
	bc.mu.Lock()
	if bc.config.MaxCacheEntries > 0 && len(bc.packageCache) >= bc.config.MaxCacheEntries {
		bc.evictOldestPackage()
	}
	bc.packageCache[dir] = buildPkg
	bc.mu.Unlock()
	
	return buildPkg, nil
}

// GetContext returns the build context
func (bc *BuildCache) getContext() *build.Context {
	if bc.config.DefaultContext != nil {
		ctx := *bc.config.DefaultContext
		
		// Apply custom tags
		if len(bc.config.CustomTags) > 0 {
			ctx.BuildTags = append(ctx.BuildTags, bc.config.CustomTags...)
		}
		
		// Apply other settings
		if bc.config.IgnoreVendor {
			ctx.UseAllFiles = false
		}
		
		return &ctx
	}
	
	return &build.Default
}

// CreateContext creates a new build context with specified parameters
func (bc *BuildCache) CreateContext(goos, goarch string, tags []string) *build.Context {
	key := bc.generateContextKey(goos, goarch, tags)
	
	// Check cache
	bc.mu.RLock()
	if cached, exists := bc.contextCache[key]; exists {
		bc.mu.RUnlock()
		return cached
	}
	bc.mu.RUnlock()
	
	// Create new context
	ctx := build.Default
	ctx.GOOS = goos
	ctx.GOARCH = goarch
	ctx.BuildTags = tags
	
	// Cache context
	bc.mu.Lock()
	bc.contextCache[key] = &ctx
	bc.mu.Unlock()
	
	return &ctx
}

// MatchFile checks if a file should be built
func (bc *BuildCache) MatchFile(dir, name string) (bool, error) {
	// Exclude test files by default unless explicitly included
	if !bc.config.IncludeTests && strings.HasSuffix(name, "_test.go") {
		return false, nil
	}
	
	ctx := bc.getContext()
	match, err := ctx.MatchFile(dir, name)
	
	if err == nil && match {
		// Cache file info
		bc.cacheFileInfo(filepath.Join(dir, name))
	}
	
	return match, err
}

// GoodOSArchFile checks if a file matches OS/Arch constraints
func (bc *BuildCache) GoodOSArchFile(name string, allTags map[string]bool) bool {
	// Extract base name without directory
	base := filepath.Base(name)
	
	// Remove .go extension
	if !strings.HasSuffix(base, ".go") {
		return false
	}
	base = base[:len(base)-3]
	
	// Check for test files
	if strings.HasSuffix(base, "_test") {
		return false
	}
	
	// If no underscore, it's a regular file that should match
	if !strings.Contains(base, "_") {
		return true
	}
	
	// Parse OS/Arch constraints from filename
	idx := strings.LastIndex(base, "_")
	if idx < 0 {
		return true
	}
	
	tag := base[idx+1:]
	
	// Check against known OS and arch values
	ctx := bc.getContext()
	knownOS := []string{"aix", "android", "darwin", "dragonfly", "freebsd", "illumos", "ios", "js", "linux", "netbsd", "openbsd", "plan9", "solaris", "windows"}
	knownArch := []string{"386", "amd64", "arm", "arm64", "mips", "mips64", "mips64le", "mipsle", "ppc64", "ppc64le", "riscv64", "s390x", "wasm"}
	
	for _, os := range knownOS {
		if tag == os {
			return tag == ctx.GOOS || allTags[tag]
		}
	}
	
	for _, arch := range knownArch {
		if tag == arch {
			return tag == ctx.GOARCH || allTags[tag]
		}
	}
	
	// If it's not a known OS or arch, it might be a build tag
	return allTags[tag]
}

// ShouldBuild determines if a file should be built based on constraints
func (bc *BuildCache) ShouldBuild(filename string) (bool, error) {
	// Read file
	src, err := ioutil.ReadFile(filename)
	if err != nil {
		return false, err
	}
	
	return bc.EvaluateConstraints(filename, src)
}

// ListBuildTags returns all build tags in use
func (bc *BuildCache) ListBuildTags() []string {
	ctx := bc.getContext()
	tags := make([]string, len(ctx.BuildTags))
	copy(tags, ctx.BuildTags)
	
	// Add custom tags
	for _, tag := range bc.config.CustomTags {
		found := false
		for _, existing := range tags {
			if existing == tag {
				found = true
				break
			}
		}
		if !found {
			tags = append(tags, tag)
		}
	}
	
	sort.Strings(tags)
	return tags
}

// AddBuildTag adds a build tag
func (bc *BuildCache) AddBuildTag(tag string) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	// Add to custom tags
	for _, existing := range bc.config.CustomTags {
		if existing == tag {
			return
		}
	}
	
	bc.config.CustomTags = append(bc.config.CustomTags, tag)
	
	// Clear caches as constraints may have changed
	bc.clearConstraintCache()
}

// RemoveBuildTag removes a build tag
func (bc *BuildCache) RemoveBuildTag(tag string) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	// Remove from custom tags
	newTags := make([]string, 0, len(bc.config.CustomTags))
	for _, existing := range bc.config.CustomTags {
		if existing != tag {
			newTags = append(newTags, existing)
		}
	}
	
	bc.config.CustomTags = newTags
	
	// Clear caches as constraints may have changed
	bc.clearConstraintCache()
}

// FindImportPath finds the import path for a directory
func (bc *BuildCache) FindImportPath(dir string) (string, error) {
	pkg, err := bc.LoadPackage(dir)
	if err != nil {
		return "", err
	}
	
	return pkg.ImportPath, nil
}

// IsStandardPackage checks if a package is from the standard library
func (bc *BuildCache) IsStandardPackage(path string) bool {
	// Check common standard library prefixes
	stdPrefixes := []string{
		"archive/", "bufio", "builtin", "bytes", "compress/",
		"container/", "context", "crypto", "database/", "debug/",
		"embed", "encoding/", "errors", "expvar", "flag", "fmt",
		"go/", "hash", "html", "image/", "index/", "io", "log",
		"math", "mime", "net", "os", "path", "plugin", "reflect",
		"regexp", "runtime", "sort", "strconv", "strings", "sync",
		"syscall", "testing", "text/", "time", "unicode", "unsafe",
	}
	
	for _, prefix := range stdPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix) {
			return true
		}
	}
	
	// Special case for C pseudo-package
	if path == "C" {
		return true
	}
	
	// Standard packages don't contain dots (except for internal use)
	// But not all packages without dots are standard packages
	// We need to check if it's actually in the standard library
	if strings.Contains(path, ".") {
		return false
	}
	
	// Check if the package exists in GOROOT
	ctx := bc.getContext()
	if ctx.GOROOT != "" {
		_, err := ctx.Import(path, "", build.FindOnly)
		return err == nil
	}
	
	return false
}

// GetPackageDependencies returns all dependencies of a package
func (bc *BuildCache) GetPackageDependencies(dir string) ([]string, error) {
	pkg, err := bc.LoadPackage(dir)
	if err != nil {
		return nil, err
	}
	
	return pkg.Dependencies, nil
}

// BatchLoadPackages loads multiple packages concurrently
func (bc *BuildCache) BatchLoadPackages(dirs []string) map[string]*BuildPackage {
	if !bc.config.ConcurrentLoads || len(dirs) <= 1 {
		// Load sequentially
		results := make(map[string]*BuildPackage)
		for _, dir := range dirs {
			pkg, err := bc.LoadPackage(dir)
			if err == nil {
				results[dir] = pkg
			}
		}
		return results
	}
	
	// Load concurrently
	results := make(map[string]*BuildPackage)
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, bc.config.LoadWorkers)
	
	for _, dir := range dirs {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			
			pkg, err := bc.LoadPackage(d)
			if err == nil {
				mu.Lock()
				results[d] = pkg
				mu.Unlock()
			}
		}(dir)
	}
	
	wg.Wait()
	return results
}

// Helper methods

func (bc *BuildCache) parseBuildConstraints(filename string, src []byte) ([]string, error) {
	// Parse file for build constraints
	constraints := []string{}
	
	// Check filename constraints (e.g., _linux.go, _amd64.go)
	base := filepath.Base(filename)
	if idx := strings.Index(base, "_"); idx >= 0 {
		parts := strings.Split(base[idx+1:], "_")
		for _, part := range parts {
			if strings.HasSuffix(part, ".go") {
				part = part[:len(part)-3]
			}
			if part != "" && part != "test" {
				constraints = append(constraints, part)
			}
		}
	}
	
	// Parse build tags from source
	// Look for // +build lines
	lines := strings.Split(string(src), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "// +build ") {
			tags := strings.TrimPrefix(line, "// +build ")
			constraints = append(constraints, strings.Fields(tags)...)
		} else if strings.HasPrefix(line, "//go:build ") {
			// Handle go:build constraints (Go 1.17+)
			expr := strings.TrimPrefix(line, "//go:build ")
			constraints = append(constraints, expr)
		} else if line == "" || strings.HasPrefix(line, "//") {
			continue
		} else if strings.HasPrefix(line, "package ") {
			break // Stop at package declaration
		}
	}
	
	return constraints, nil
}

func (bc *BuildCache) evaluateConstraintList(constraints []string) bool {
	if len(constraints) == 0 {
		return true
	}
	
	ctx := bc.getContext()
	tags := make(map[string]bool)
	
	// Add build tags
	for _, tag := range ctx.BuildTags {
		tags[tag] = true
	}
	
	// Add OS and arch
	tags[ctx.GOOS] = true
	tags[ctx.GOARCH] = true
	
	// Add Go version tags
	// Parse current Go version
	goVersion := runtime.Version()
	if strings.HasPrefix(goVersion, "go") {
		goVersion = goVersion[2:]
	}
	parts := strings.Split(goVersion, ".")
	if len(parts) >= 2 {
		major, _ := strconv.Atoi(parts[0])
		minor, _ := strconv.Atoi(parts[1])
		// Add tags for all versions up to current
		if major == 1 {
			for i := 1; i <= minor; i++ {
				tags[fmt.Sprintf("go1.%d", i)] = true
			}
		}
	}
	
	// Evaluate each constraint
	for _, constraint := range constraints {
		if !bc.evaluateSingleConstraint(constraint, tags) {
			return false
		}
	}
	
	return true
}

func (bc *BuildCache) evaluateSingleConstraint(constraint string, tags map[string]bool) bool {
	// Handle negation
	if strings.HasPrefix(constraint, "!") {
		return !tags[constraint[1:]]
	}
	
	// Handle OR constraints (space-separated)
	if strings.Contains(constraint, " ") {
		parts := strings.Fields(constraint)
		for _, part := range parts {
			if bc.evaluateSingleConstraint(part, tags) {
				return true
			}
		}
		return false
	}
	
	// Handle AND constraints (comma-separated)
	if strings.Contains(constraint, ",") {
		parts := strings.Split(constraint, ",")
		for _, part := range parts {
			if !bc.evaluateSingleConstraint(strings.TrimSpace(part), tags) {
				return false
			}
		}
		return true
	}
	
	// Simple tag check
	return tags[constraint]
}

func (bc *BuildCache) getCurrentTags() map[string]bool {
	ctx := bc.getContext()
	tags := make(map[string]bool)
	
	for _, tag := range ctx.BuildTags {
		tags[tag] = true
	}
	
	tags[ctx.GOOS] = true
	tags[ctx.GOARCH] = true
	
	return tags
}

func (bc *BuildCache) evaluatePackageConstraints(pkg *build.Package) *ConstraintResult {
	satisfied := true
	var constraints []string
	
	// Collect constraints from all files
	for _, file := range pkg.GoFiles {
		path := filepath.Join(pkg.Dir, file)
		if src, err := ioutil.ReadFile(path); err == nil {
			if fileConstraints, err := bc.parseBuildConstraints(path, src); err == nil {
				constraints = append(constraints, fileConstraints...)
			}
		}
	}
	
	if len(constraints) > 0 {
		satisfied = bc.evaluateConstraintList(constraints)
	}
	
	return &ConstraintResult{
		Satisfied:   satisfied,
		Constraints: constraints,
		Tags:        bc.getCurrentTags(),
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		CachedAt:    time.Now(),
	}
}

func (bc *BuildCache) extractDependencies(pkg *build.Package) []string {
	deps := make([]string, 0, len(pkg.Imports))
	deps = append(deps, pkg.Imports...)
	return deps
}

func (bc *BuildCache) extractTestImports(pkg *build.Package) []string {
	imports := make([]string, 0, len(pkg.TestImports)+len(pkg.XTestImports))
	imports = append(imports, pkg.TestImports...)
	imports = append(imports, pkg.XTestImports...)
	return imports
}

func (bc *BuildCache) cacheFileInfo(path string) {
	src, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	
	// Parse file for basic info
	info := &FileInfo{
		Path:        path,
		IsTest:      strings.HasSuffix(path, "_test.go"),
		IsGenerated: strings.Contains(string(src), "Code generated"),
		CachedAt:    time.Now(),
		AccessCount: 1,
	}
	
	// Extract package name and imports
	if file, err := parser.ParseFile(bc.fset, path, src, parser.ImportsOnly); err == nil {
		if file.Name != nil {
			info.Package = file.Name.Name
		}
		
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if info.IsTest {
				info.TestImports = append(info.TestImports, importPath)
			} else {
				info.Imports = append(info.Imports, importPath)
			}
		}
	}
	
	// Parse constraints
	if constraints, err := bc.parseBuildConstraints(path, src); err == nil {
		info.Constraints = constraints
	}
	
	bc.mu.Lock()
	bc.fileCache[path] = info
	bc.mu.Unlock()
}

func (bc *BuildCache) preloadStandardLibrary() {
	if !bc.config.CacheStdlib {
		return
	}
	
	// Common standard library packages
	stdPackages := []string{
		"fmt", "io", "os", "strings", "bytes", "bufio",
		"encoding/json", "encoding/xml", "encoding/base64",
		"net/http", "net/url", "crypto/sha256", "crypto/md5",
		"time", "context", "sync", "errors", "log", "math",
		"regexp", "sort", "strconv", "path/filepath",
	}
	
	for _, pkg := range stdPackages {
		bc.ImportPackage(pkg, "", 0)
	}
}

func (bc *BuildCache) clearConstraintCache() {
	bc.constraintCache = make(map[string]*ConstraintResult)
	bc.packageCache = make(map[string]*BuildPackage)
	bc.importCache = make(map[string]*ImportResult)
}

// Cache key generation

func (bc *BuildCache) generateConstraintKey(filename string, src []byte) string {
	h := sha256.New()
	h.Write([]byte(filename))
	h.Write(src)
	
	// Include current tags in key
	tags := bc.ListBuildTags()
	for _, tag := range tags {
		h.Write([]byte(tag))
	}
	
	ctx := bc.getContext()
	h.Write([]byte(ctx.GOOS))
	h.Write([]byte(ctx.GOARCH))
	
	return hex.EncodeToString(h.Sum(nil))
}

func (bc *BuildCache) generateImportKey(path, srcDir string, mode build.ImportMode) string {
	h := sha256.New()
	h.Write([]byte(path))
	h.Write([]byte(srcDir))
	h.Write([]byte(fmt.Sprintf("%d", mode)))
	
	ctx := bc.getContext()
	h.Write([]byte(ctx.GOPATH))
	h.Write([]byte(ctx.GOROOT))
	
	return hex.EncodeToString(h.Sum(nil))
}

func (bc *BuildCache) generateContextKey(goos, goarch string, tags []string) string {
	h := sha256.New()
	h.Write([]byte(goos))
	h.Write([]byte(goarch))
	
	sort.Strings(tags)
	for _, tag := range tags {
		h.Write([]byte(tag))
	}
	
	return hex.EncodeToString(h.Sum(nil))
}

// Cache eviction

func (bc *BuildCache) evictOldestConstraint() {
	var oldest *ConstraintResult
	var oldestKey string
	
	for key, result := range bc.constraintCache {
		if oldest == nil || result.CachedAt.Before(oldest.CachedAt) {
			oldest = result
			oldestKey = key
		}
	}
	
	if oldestKey != "" {
		delete(bc.constraintCache, oldestKey)
	}
}

func (bc *BuildCache) evictOldestPackage() {
	var oldest *BuildPackage
	var oldestKey string
	
	for key, pkg := range bc.packageCache {
		if oldest == nil || pkg.CachedAt.Before(oldest.CachedAt) {
			oldest = pkg
			oldestKey = key
		}
	}
	
	if oldestKey != "" {
		delete(bc.packageCache, oldestKey)
	}
}

// GetStatistics returns cache statistics
func (bc *BuildCache) GetStatistics() map[string]interface{} {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	total := bc.hits + bc.misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(bc.hits) * 100.0 / float64(total)
	}
	
	return map[string]interface{}{
		"constraint_cache_size": len(bc.constraintCache),
		"package_cache_size":    len(bc.packageCache),
		"import_cache_size":     len(bc.importCache),
		"context_cache_size":    len(bc.contextCache),
		"file_cache_size":       len(bc.fileCache),
		"cache_hits":            bc.hits,
		"cache_misses":          bc.misses,
		"hit_rate":              hitRate,
		"total_evaluations":     bc.evaluations,
		"constraint_evals":      bc.constraintEvals,
		"package_loads":         bc.packageLoads,
		"import_resolves":       bc.importResolves,
		"custom_tags":           bc.config.CustomTags,
	}
}

// Clear clears all caches
func (bc *BuildCache) Clear() {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	bc.constraintCache = make(map[string]*ConstraintResult)
	bc.packageCache = make(map[string]*BuildPackage)
	bc.importCache = make(map[string]*ImportResult)
	bc.contextCache = make(map[string]*build.Context)
	bc.fileCache = make(map[string]*FileInfo)
	bc.hits = 0
	bc.misses = 0
	bc.evaluations = 0
	bc.constraintEvals = 0
	bc.packageLoads = 0
	bc.importResolves = 0
}

// GetFileInfo returns cached file information
func (bc *BuildCache) GetFileInfo(path string) *FileInfo {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.fileCache[path]
}

// WarmupCache preloads commonly used packages
func (bc *BuildCache) WarmupCache(dirs []string) {
	// Preload packages
	bc.BatchLoadPackages(dirs)
	
	// Preload standard library if configured
	if bc.config.PreloadStdlib {
		bc.preloadStandardLibrary()
	}
}