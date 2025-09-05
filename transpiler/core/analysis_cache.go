// Package core provides golang.org/x/tools/go/analysis with analysis cache.
// This implements Phase 1.2f: golang.org/x/tools/go/analysis with analysis cache.
package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	atomicpass "golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
)

// AnalysisCache manages analysis results with caching
type AnalysisCache struct {
	config        *AnalysisCacheConfig
	resultCache   map[string]*CachedResult
	diagnostics   map[string][]*analysis.Diagnostic
	factCache     map[string]map[analysis.Fact]bool
	analyzers     map[string]*analysis.Analyzer
	mu            sync.RWMutex
	
	// Metrics
	hits          int64
	misses        int64
	analyses      int64
	cacheSize     int64
	totalDiags    int64
	totalFacts    int64
}

// CachedResult represents a cached analysis result
type CachedResult struct {
	Pass        *analysis.Pass
	Result      interface{}
	Diagnostics []*analysis.Diagnostic
	Facts       map[analysis.Fact]bool
	AnalyzedAt  time.Time
	AccessCount int64
	Hash        string
	Duration    time.Duration
}

// AnalysisCacheConfig contains configuration for analysis cache
type AnalysisCacheConfig struct {
	// Cache settings
	MaxResults      int
	MaxCacheSizeMB  int
	TTL             time.Duration
	EnableMetrics   bool
	
	// Analysis settings
	EnableAllPasses bool
	CustomAnalyzers []*analysis.Analyzer
	SkipPasses      []string
	
	// Performance settings
	ConcurrentAnalysis bool
	AnalysisWorkers    int
	FactCaching        bool
	DiagnosticCaching  bool
}

// DefaultAnalysisCacheConfig returns default configuration
func DefaultAnalysisCacheConfig() *AnalysisCacheConfig {
	return &AnalysisCacheConfig{
		MaxResults:         1000,
		MaxCacheSizeMB:     100,
		TTL:                30 * time.Minute,
		EnableMetrics:      true,
		EnableAllPasses:    true,
		ConcurrentAnalysis: true,
		AnalysisWorkers:    4,
		FactCaching:        true,
		DiagnosticCaching:  true,
	}
}

// NewAnalysisCache creates a new analysis cache
func NewAnalysisCache(config *AnalysisCacheConfig) *AnalysisCache {
	if config == nil {
		config = DefaultAnalysisCacheConfig()
	}
	
	ac := &AnalysisCache{
		config:      config,
		resultCache: make(map[string]*CachedResult),
		diagnostics: make(map[string][]*analysis.Diagnostic),
		factCache:   make(map[string]map[analysis.Fact]bool),
		analyzers:   make(map[string]*analysis.Analyzer),
	}
	
	// Register standard analyzers
	if config.EnableAllPasses {
		ac.registerStandardAnalyzers()
	}
	
	// Register custom analyzers
	for _, analyzer := range config.CustomAnalyzers {
		ac.RegisterAnalyzer(analyzer)
	}
	
	return ac
}

// registerStandardAnalyzers registers all standard analysis passes
func (ac *AnalysisCache) registerStandardAnalyzers() {
	standardAnalyzers := []*analysis.Analyzer{
		asmdecl.Analyzer,
		assign.Analyzer,
		atomicpass.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		errorsas.Analyzer,
		httpresponse.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		printf.Analyzer,
		shift.Analyzer,
		stdmethods.Analyzer,
		structtag.Analyzer,
		tests.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
	}
	
	for _, analyzer := range standardAnalyzers {
		// Skip if in skip list
		skip := false
		for _, skipName := range ac.config.SkipPasses {
			if analyzer.Name == skipName {
				skip = true
				break
			}
		}
		if !skip {
			ac.RegisterAnalyzer(analyzer)
		}
	}
}

// RegisterAnalyzer registers a custom analyzer
func (ac *AnalysisCache) RegisterAnalyzer(analyzer *analysis.Analyzer) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.analyzers[analyzer.Name] = analyzer
}

// RunAnalysis runs an analysis pass with caching
func (ac *AnalysisCache) RunAnalysis(analyzer *analysis.Analyzer, pass *analysis.Pass) (interface{}, error) {
	// Generate cache key
	key := ac.generateCacheKey(analyzer.Name, pass)
	
	// Check cache
	ac.mu.RLock()
	if cached, exists := ac.resultCache[key]; exists {
		if ac.config.TTL == 0 || time.Since(cached.AnalyzedAt) < ac.config.TTL {
			ac.mu.RUnlock()
			atomic.AddInt64(&ac.hits, 1)
			atomic.AddInt64(&cached.AccessCount, 1)
			
			// Restore diagnostics
			if ac.config.DiagnosticCaching {
				pass.Report = ac.createReportFunc(key)
			}
			
			return cached.Result, nil
		}
	}
	ac.mu.RUnlock()
	
	atomic.AddInt64(&ac.misses, 1)
	
	// Run analysis
	start := time.Now()
	
	// Capture diagnostics
	var capturedDiags []*analysis.Diagnostic
	originalReport := pass.Report
	if ac.config.DiagnosticCaching {
		pass.Report = func(diag analysis.Diagnostic) {
			capturedDiags = append(capturedDiags, &diag)
			originalReport(diag)
		}
	}
	
	result, err := analyzer.Run(pass)
	if err != nil {
		return nil, err
	}
	
	duration := time.Since(start)
	atomic.AddInt64(&ac.analyses, 1)
	
	// Cache result
	cached := &CachedResult{
		Pass:        pass,
		Result:      result,
		Diagnostics: capturedDiags,
		Facts:       ac.extractFacts(pass),
		AnalyzedAt:  time.Now(),
		AccessCount: 1,
		Hash:        key,
		Duration:    duration,
	}
	
	ac.mu.Lock()
	if ac.config.MaxResults > 0 && len(ac.resultCache) >= ac.config.MaxResults {
		ac.evictOldestResult()
	}
	ac.resultCache[key] = cached
	
	if ac.config.DiagnosticCaching {
		ac.diagnostics[key] = capturedDiags
		atomic.AddInt64(&ac.totalDiags, int64(len(capturedDiags)))
	}
	
	if ac.config.FactCaching {
		ac.factCache[key] = cached.Facts
		atomic.AddInt64(&ac.totalFacts, int64(len(cached.Facts)))
	}
	ac.mu.Unlock()
	
	return result, nil
}

// RunAllAnalyzers runs all registered analyzers on a package
func (ac *AnalysisCache) RunAllAnalyzers(pkg *Package) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	errors := make(map[string]error)
	
	if !ac.config.ConcurrentAnalysis {
		// Run sequentially
		for name, analyzer := range ac.analyzers {
			pass := ac.createPass(analyzer, pkg)
			result, err := ac.RunAnalysis(analyzer, pass)
			if err != nil {
				errors[name] = err
			} else {
				results[name] = result
			}
		}
	} else {
		// Run concurrently
		var mu sync.Mutex
		var wg sync.WaitGroup
		sem := make(chan struct{}, ac.config.AnalysisWorkers)
		
		for name, analyzer := range ac.analyzers {
			wg.Add(1)
			go func(n string, a *analysis.Analyzer) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				
				pass := ac.createPass(a, pkg)
				result, err := ac.RunAnalysis(a, pass)
				
				mu.Lock()
				if err != nil {
					errors[n] = err
				} else {
					results[n] = result
				}
				mu.Unlock()
			}(name, analyzer)
		}
		
		wg.Wait()
	}
	
	// Return error if any analysis failed
	if len(errors) > 0 {
		var errStrs []string
		for name, err := range errors {
			errStrs = append(errStrs, fmt.Sprintf("%s: %v", name, err))
		}
		return results, fmt.Errorf("analysis errors: %s", strings.Join(errStrs, "; "))
	}
	
	return results, nil
}

// Package represents a package for analysis
type Package struct {
	Fset      *token.FileSet
	Files     []*ast.File
	Pkg       *types.Package
	TypesInfo *types.Info
	Path      string
}

// SetupPackageFacts configures package-level fact import/export if supported
// This is a separate method to handle version compatibility issues
func (ac *AnalysisCache) SetupPackageFacts(pass *analysis.Pass, pkg *Package) {
	// Note: Package fact handling varies between golang.org/x/tools versions
	// This can be extended when the exact signature is determined
	// For now, object-level facts are sufficient for most analysis needs
}

// createPass creates an analysis pass for a package
func (ac *AnalysisCache) createPass(analyzer *analysis.Analyzer, pkg *Package) *analysis.Pass {
	pass := &analysis.Pass{
		Analyzer:  analyzer,
		Fset:      pkg.Fset,
		Files:     pkg.Files,
		Pkg:       pkg.Pkg,
		TypesInfo: pkg.TypesInfo,
		Report: func(diag analysis.Diagnostic) {
			// Default report function
			fmt.Printf("%v: %s\n", diag.Pos, diag.Message)
		},
		ResultOf:  make(map[*analysis.Analyzer]interface{}),
		ImportObjectFact: func(obj types.Object, fact analysis.Fact) bool {
			// Implement fact importing
			return ac.importFact(obj, fact)
		},
		ExportObjectFact: func(obj types.Object, fact analysis.Fact) {
			// Implement fact exporting
			ac.exportFact(obj, fact)
		},
		ImportPackageFact: func(pkg *types.Package, fact analysis.Fact) bool {
			// Implement package fact importing
			if pkg != nil {
				return ac.importPackageFact(pkg, fact)
			}
			return false
		},
		ExportPackageFact: func(fact analysis.Fact) {
			// Implement package fact exporting
			if pkg.Pkg != nil {
				ac.exportPackageFact(pkg.Pkg, fact)
			}
		},
	}
	
	return pass
}

// GetDiagnostics retrieves cached diagnostics for an analysis
func (ac *AnalysisCache) GetDiagnostics(analyzerName string, pkg *Package) []*analysis.Diagnostic {
	key := ac.generateCacheKey(analyzerName, ac.createPass(ac.analyzers[analyzerName], pkg))
	
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	
	if diags, exists := ac.diagnostics[key]; exists {
		return diags
	}
	
	return nil
}

// GetAllDiagnostics retrieves all cached diagnostics
func (ac *AnalysisCache) GetAllDiagnostics() map[string][]*analysis.Diagnostic {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	
	// Return a copy to prevent mutations
	result := make(map[string][]*analysis.Diagnostic)
	for k, v := range ac.diagnostics {
		result[k] = append([]*analysis.Diagnostic{}, v...)
	}
	
	return result
}

// FindDiagnosticsByType finds diagnostics by type
func (ac *AnalysisCache) FindDiagnosticsByType(diagType string) []*analysis.Diagnostic {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	
	var result []*analysis.Diagnostic
	
	for _, diags := range ac.diagnostics {
		for _, diag := range diags {
			if strings.Contains(diag.Message, diagType) {
				result = append(result, diag)
			}
		}
	}
	
	return result
}

// createReportFunc creates a report function that captures diagnostics
func (ac *AnalysisCache) createReportFunc(key string) func(analysis.Diagnostic) {
	return func(diag analysis.Diagnostic) {
		ac.mu.Lock()
		ac.diagnostics[key] = append(ac.diagnostics[key], &diag)
		ac.mu.Unlock()
		atomic.AddInt64(&ac.totalDiags, 1)
	}
}

// extractFacts extracts facts from a pass
func (ac *AnalysisCache) extractFacts(pass *analysis.Pass) map[analysis.Fact]bool {
	facts := make(map[analysis.Fact]bool)
	// This would extract facts from the pass
	// Implementation depends on specific fact types
	return facts
}

// importFact imports a fact for an object
func (ac *AnalysisCache) importFact(obj types.Object, fact analysis.Fact) bool {
	key := ac.generateFactKey(obj, fact)
	
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	
	if facts, exists := ac.factCache[key]; exists {
		return facts[fact]
	}
	
	return false
}

// exportFact exports a fact for an object
func (ac *AnalysisCache) exportFact(obj types.Object, fact analysis.Fact) {
	key := ac.generateFactKey(obj, fact)
	
	ac.mu.Lock()
	defer ac.mu.Unlock()
	
	if ac.factCache[key] == nil {
		ac.factCache[key] = make(map[analysis.Fact]bool)
	}
	ac.factCache[key][fact] = true
	atomic.AddInt64(&ac.totalFacts, 1)
}

// importPackageFact imports a fact for a package
func (ac *AnalysisCache) importPackageFact(pkg *types.Package, fact analysis.Fact) bool {
	key := ac.generatePackageFactKey(pkg, fact)
	
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	
	if facts, exists := ac.factCache[key]; exists {
		return facts[fact]
	}
	
	return false
}

// exportPackageFact exports a fact for a package
func (ac *AnalysisCache) exportPackageFact(pkg *types.Package, fact analysis.Fact) {
	key := ac.generatePackageFactKey(pkg, fact)
	
	ac.mu.Lock()
	defer ac.mu.Unlock()
	
	if ac.factCache[key] == nil {
		ac.factCache[key] = make(map[analysis.Fact]bool)
	}
	ac.factCache[key][fact] = true
	atomic.AddInt64(&ac.totalFacts, 1)
}

// generateCacheKey generates a cache key for an analysis
func (ac *AnalysisCache) generateCacheKey(analyzerName string, pass *analysis.Pass) string {
	h := sha256.New()
	
	// Include analyzer name
	h.Write([]byte(analyzerName))
	
	// Include file paths
	var paths []string
	for _, file := range pass.Files {
		if file.Package != 0 {
			paths = append(paths, pass.Fset.Position(file.Package).Filename)
		}
	}
	sort.Strings(paths)
	for _, path := range paths {
		h.Write([]byte(path))
	}
	
	// Include package path
	if pass.Pkg != nil {
		h.Write([]byte(pass.Pkg.Path()))
	}
	
	return hex.EncodeToString(h.Sum(nil))
}

// generateFactKey generates a cache key for an object fact
func (ac *AnalysisCache) generateFactKey(obj types.Object, fact analysis.Fact) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v:%T", obj, fact)))
	return hex.EncodeToString(h.Sum(nil))
}

// generatePackageFactKey generates a cache key for a package fact
func (ac *AnalysisCache) generatePackageFactKey(pkg *types.Package, fact analysis.Fact) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s:%T", pkg.Path(), fact)))
	return hex.EncodeToString(h.Sum(nil))
}

// evictOldestResult evicts the oldest cached result
func (ac *AnalysisCache) evictOldestResult() {
	var oldest *CachedResult
	var oldestKey string
	
	for key, result := range ac.resultCache {
		if oldest == nil || result.AnalyzedAt.Before(oldest.AnalyzedAt) {
			oldest = result
			oldestKey = key
		}
	}
	
	if oldestKey != "" {
		delete(ac.resultCache, oldestKey)
		delete(ac.diagnostics, oldestKey)
		delete(ac.factCache, oldestKey)
	}
}

// GetStatistics returns cache statistics
func (ac *AnalysisCache) GetStatistics() map[string]interface{} {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	
	total := ac.hits + ac.misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(ac.hits) * 100.0 / float64(total)
	}
	
	// Calculate average analysis duration
	totalDuration := time.Duration(0)
	for _, result := range ac.resultCache {
		totalDuration += result.Duration
	}
	avgDuration := time.Duration(0)
	if len(ac.resultCache) > 0 {
		avgDuration = totalDuration / time.Duration(len(ac.resultCache))
	}
	
	return map[string]interface{}{
		"cached_results":     len(ac.resultCache),
		"registered_analyzers": len(ac.analyzers),
		"cache_hits":         ac.hits,
		"cache_misses":       ac.misses,
		"hit_rate":           hitRate,
		"total_analyses":     ac.analyses,
		"total_diagnostics":  ac.totalDiags,
		"total_facts":        ac.totalFacts,
		"avg_duration":       avgDuration.String(),
		"diagnostic_count":   len(ac.diagnostics),
		"fact_count":         len(ac.factCache),
	}
}

// Clear clears all caches
func (ac *AnalysisCache) Clear() {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	
	ac.resultCache = make(map[string]*CachedResult)
	ac.diagnostics = make(map[string][]*analysis.Diagnostic)
	ac.factCache = make(map[string]map[analysis.Fact]bool)
	ac.hits = 0
	ac.misses = 0
	ac.analyses = 0
	ac.totalDiags = 0
	ac.totalFacts = 0
}

// InvalidateResult invalidates a cached result
func (ac *AnalysisCache) InvalidateResult(key string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	
	delete(ac.resultCache, key)
	delete(ac.diagnostics, key)
	delete(ac.factCache, key)
}

// GetAnalyzer returns a registered analyzer by name
func (ac *AnalysisCache) GetAnalyzer(name string) (*analysis.Analyzer, bool) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	
	analyzer, exists := ac.analyzers[name]
	return analyzer, exists
}

// ListAnalyzers returns all registered analyzer names
func (ac *AnalysisCache) ListAnalyzers() []string {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	
	var names []string
	for name := range ac.analyzers {
		names = append(names, name)
	}
	sort.Strings(names)
	
	return names
}

// AnalysisReport represents a complete analysis report
type AnalysisReport struct {
	Package     string
	Analyzers   []string
	Results     map[string]interface{}
	Diagnostics map[string][]*analysis.Diagnostic
	Facts       map[string]int
	Duration    time.Duration
	Timestamp   time.Time
}

// GenerateReport generates a comprehensive analysis report
func (ac *AnalysisCache) GenerateReport(pkg *Package) (*AnalysisReport, error) {
	start := time.Now()
	
	results, err := ac.RunAllAnalyzers(pkg)
	if err != nil {
		return nil, err
	}
	
	report := &AnalysisReport{
		Package:     pkg.Path,
		Analyzers:   ac.ListAnalyzers(),
		Results:     results,
		Diagnostics: make(map[string][]*analysis.Diagnostic),
		Facts:       make(map[string]int),
		Duration:    time.Since(start),
		Timestamp:   time.Now(),
	}
	
	// Collect diagnostics
	for name := range ac.analyzers {
		diags := ac.GetDiagnostics(name, pkg)
		if len(diags) > 0 {
			report.Diagnostics[name] = diags
		}
	}
	
	// Count facts
	ac.mu.RLock()
	for key, facts := range ac.factCache {
		if strings.Contains(key, pkg.Path) {
			report.Facts[key] = len(facts)
		}
	}
	ac.mu.RUnlock()
	
	return report, nil
}

// FilterDiagnostics filters diagnostics by severity
func (ac *AnalysisCache) FilterDiagnostics(severity string) []*analysis.Diagnostic {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	
	var result []*analysis.Diagnostic
	
	for _, diags := range ac.diagnostics {
		for _, diag := range diags {
			// Simple severity filtering based on message content
			// In a real implementation, you might have a severity field
			if strings.Contains(strings.ToLower(diag.Message), strings.ToLower(severity)) {
				result = append(result, diag)
			}
		}
	}
	
	return result
}

// GetCachedResult retrieves a cached analysis result
func (ac *AnalysisCache) GetCachedResult(key string) (*CachedResult, bool) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	
	result, exists := ac.resultCache[key]
	return result, exists
}

// WarmupCache pre-runs analyses on common packages
func (ac *AnalysisCache) WarmupCache(packages []*Package) error {
	for _, pkg := range packages {
		_, err := ac.RunAllAnalyzers(pkg)
		if err != nil {
			return fmt.Errorf("warmup failed for %s: %w", pkg.Path, err)
		}
	}
	
	return nil
}