// Package core provides text/template with compiled template cache.
// This implements Phase 1.2b: text/template with compiled template cache.
package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"
	"time"
)

// CompiledTemplate represents a pre-compiled text template
type CompiledTemplate struct {
	Name         string
	Template     *template.Template
	Source       string
	Hash         string
	CompiledAt   time.Time
	LastUsed     time.Time
	Uses         int64
	Dependencies []string // Other templates this one depends on
	Metadata     map[string]interface{}
}

// TemplateCache manages compiled text/template cache
type TemplateCache struct {
	config    *TemplateCacheConfig
	templates map[string]*CompiledTemplate
	hashIndex map[string]string // hash -> name mapping for deduplication
	mu        sync.RWMutex
	
	// Template sets for grouping related templates
	sets map[string]*template.Template
	
	// Metrics
	hits          int64
	misses        int64
	compilations  int64
	executions    int64
	cacheSize     int64 // Approximate memory usage in bytes
	evictions     int64
}

// TemplateCacheConfig contains configuration for template cache
type TemplateCacheConfig struct {
	// Cache settings
	MaxTemplates      int
	MaxCacheSizeMB    int
	EnableDedup       bool // Deduplicate identical templates
	EnableMetrics     bool
	
	// Template settings
	Delims           [2]string // Custom delimiters
	FuncMap          template.FuncMap
	StrictMode       bool // Fail on missing keys
	EnableAutoReload bool // Auto-reload modified templates
	
	// Performance settings
	PrecompileOnAdd  bool
	LazyCompilation  bool
	ConcurrentCompile bool
	CompileWorkers   int
}

// DefaultTemplateCacheConfig returns default configuration
func DefaultTemplateCacheConfig() *TemplateCacheConfig {
	return &TemplateCacheConfig{
		MaxTemplates:      500,
		MaxCacheSizeMB:    100,
		EnableDedup:       true,
		EnableMetrics:     true,
		Delims:            [2]string{"{{", "}}"},
		FuncMap:           DefaultTemplateFuncMap(),
		StrictMode:        false,
		EnableAutoReload:  false,
		PrecompileOnAdd:   true,
		LazyCompilation:   false,
		ConcurrentCompile: true,
		CompileWorkers:    4,
	}
}

// DefaultTemplateFuncMap returns default template functions
func DefaultTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		// String functions
		"upper":      strings.ToUpper,
		"lower":      strings.ToLower,
		"title":      strings.Title,
		"trim":       strings.TrimSpace,
		"replace":    strings.ReplaceAll,
		"contains":   strings.Contains,
		"hasPrefix":  strings.HasPrefix,
		"hasSuffix":  strings.HasSuffix,
		"split":      strings.Split,
		"join":       strings.Join,
		
		// Formatting functions
		"sprintf":    fmt.Sprintf,
		"quote":      quote,
		"unquote":    unquote,
		"indent":     indent,
		"dedent":     dedent,
		"wrap":       wrap,
		
		// Path functions
		"basename":   filepath.Base,
		"dirname":    filepath.Dir,
		"ext":        filepath.Ext,
		"clean":      filepath.Clean,
		"abs":        filepath.Abs,
		
		// Logic functions
		"default":    defaultValue,
		"empty":      isEmpty,
		"coalesce":   coalesce,
		"ternary":    ternary,
		
		// Collection functions
		"first":      first,
		"last":       last,
		"reverse":    reverse,
		"uniq":       unique,
		"dict":       dict,
		"list":       list,
		
		// Date/time functions
		"now":        time.Now,
		"date":       formatDate,
		"timestamp":  timestamp,
	}
}

// NewTemplateCache creates a new template cache
func NewTemplateCache(config *TemplateCacheConfig) *TemplateCache {
	if config == nil {
		config = DefaultTemplateCacheConfig()
	}
	
	return &TemplateCache{
		config:    config,
		templates: make(map[string]*CompiledTemplate),
		hashIndex: make(map[string]string),
		sets:      make(map[string]*template.Template),
	}
}

// AddTemplate adds a new template to the cache
func (tc *TemplateCache) AddTemplate(name, source string) error {
	return tc.AddTemplateWithMetadata(name, source, nil)
}

// AddTemplateWithMetadata adds a template with metadata
func (tc *TemplateCache) AddTemplateWithMetadata(name, source string, metadata map[string]interface{}) error {
	// Calculate hash for deduplication
	hash := tc.calculateHash(source)
	
	tc.mu.Lock()
	defer tc.mu.Unlock()
	
	// Check for deduplication
	if tc.config.EnableDedup {
		if existingName, exists := tc.hashIndex[hash]; exists {
			// Template already exists with different name, create alias
			tc.templates[name] = tc.templates[existingName]
			return nil
		}
	}
	
	// Check cache limits
	if tc.config.MaxTemplates > 0 && len(tc.templates) >= tc.config.MaxTemplates {
		tc.evictLRU()
	}
	
	// Create compiled template
	compiled := &CompiledTemplate{
		Name:       name,
		Source:     source,
		Hash:       hash,
		CompiledAt: time.Now(),
		LastUsed:   time.Now(),
		Uses:       0,
		Metadata:   metadata,
	}
	
	// Compile if configured
	if tc.config.PrecompileOnAdd && !tc.config.LazyCompilation {
		if err := tc.compileTemplate(compiled); err != nil {
			return err
		}
	}
	
	// Store template
	tc.templates[name] = compiled
	if tc.config.EnableDedup {
		tc.hashIndex[hash] = name
	}
	
	// Update metrics
	atomic.AddInt64(&tc.compilations, 1)
	tc.updateCacheSize(int64(len(source)))
	
	return nil
}

// compileTemplate compiles a template
func (tc *TemplateCache) compileTemplate(ct *CompiledTemplate) error {
	// Create new template with custom delimiters
	tmpl := template.New(ct.Name)
	
	if tc.config.Delims[0] != "" && tc.config.Delims[1] != "" {
		tmpl = tmpl.Delims(tc.config.Delims[0], tc.config.Delims[1])
	}
	
	// Add functions
	if tc.config.FuncMap != nil {
		tmpl = tmpl.Funcs(tc.config.FuncMap)
	}
	
	// Set options
	if tc.config.StrictMode {
		tmpl = tmpl.Option("missingkey=error")
	} else {
		tmpl = tmpl.Option("missingkey=default")
	}
	
	// Parse template
	parsed, err := tmpl.Parse(ct.Source)
	if err != nil {
		return fmt.Errorf("failed to compile template %s: %w", ct.Name, err)
	}
	
	ct.Template = parsed
	ct.CompiledAt = time.Now()
	
	// Extract dependencies (templates referenced)
	ct.Dependencies = tc.extractDependencies(ct.Source)
	
	return nil
}

// GetTemplate retrieves a compiled template
func (tc *TemplateCache) GetTemplate(name string) (*CompiledTemplate, error) {
	tc.mu.RLock()
	tmpl, exists := tc.templates[name]
	tc.mu.RUnlock()
	
	if !exists {
		atomic.AddInt64(&tc.misses, 1)
		return nil, fmt.Errorf("template %s not found", name)
	}
	
	// Compile if needed (lazy compilation or not compiled yet)
	if tmpl.Template == nil {
		tc.mu.Lock()
		if tmpl.Template == nil {
			if err := tc.compileTemplate(tmpl); err != nil {
				tc.mu.Unlock()
				return nil, err
			}
		}
		tc.mu.Unlock()
	}
	
	// Update usage statistics
	atomic.AddInt64(&tmpl.Uses, 1)
	atomic.AddInt64(&tc.hits, 1)
	tmpl.LastUsed = time.Now()
	
	return tmpl, nil
}

// Execute executes a template with data
func (tc *TemplateCache) Execute(name string, data interface{}) (string, error) {
	return tc.ExecuteToWriter(name, data, nil)
}

// ExecuteToWriter executes a template to a writer
func (tc *TemplateCache) ExecuteToWriter(name string, data interface{}, w io.Writer) (string, error) {
	tmpl, err := tc.GetTemplate(name)
	if err != nil {
		return "", err
	}
	
	if tmpl.Template == nil {
		return "", fmt.Errorf("template %s not compiled", name)
	}
	
	// Use provided writer or create buffer
	var buf *bytes.Buffer
	if w == nil {
		buf = bufferPool.Get().(*bytes.Buffer)
		buf.Reset()
		defer bufferPool.Put(buf)
		w = buf
	}
	
	// Execute template
	if err := tmpl.Template.Execute(w, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}
	
	atomic.AddInt64(&tc.executions, 1)
	
	// Return result if using buffer
	if buf != nil {
		return buf.String(), nil
	}
	return "", nil
}

// ExecuteSet executes a set of related templates
func (tc *TemplateCache) ExecuteSet(setName string, templateName string, data interface{}) (string, error) {
	tc.mu.RLock()
	set, exists := tc.sets[setName]
	tc.mu.RUnlock()
	
	if !exists {
		return "", fmt.Errorf("template set %s not found", setName)
	}
	
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)
	
	if err := set.ExecuteTemplate(buf, templateName, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s in set %s: %w", templateName, setName, err)
	}
	
	atomic.AddInt64(&tc.executions, 1)
	return buf.String(), nil
}

// CreateSet creates a new template set
func (tc *TemplateCache) CreateSet(setName string, templates map[string]string) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	
	// Create root template for set
	set := template.New(setName)
	
	// Apply configuration
	if tc.config.Delims[0] != "" && tc.config.Delims[1] != "" {
		set = set.Delims(tc.config.Delims[0], tc.config.Delims[1])
	}
	if tc.config.FuncMap != nil {
		set = set.Funcs(tc.config.FuncMap)
	}
	
	// Parse all templates in set
	for name, source := range templates {
		tmpl := set.New(name)
		if _, err := tmpl.Parse(source); err != nil {
			return fmt.Errorf("failed to parse template %s in set %s: %w", name, setName, err)
		}
		
		// Also add to individual cache
		tc.templates[fmt.Sprintf("%s.%s", setName, name)] = &CompiledTemplate{
			Name:       name,
			Template:   tmpl,
			Source:     source,
			Hash:       tc.calculateHash(source),
			CompiledAt: time.Now(),
			LastUsed:   time.Now(),
		}
	}
	
	tc.sets[setName] = set
	return nil
}

// BatchCompile compiles multiple templates concurrently
func (tc *TemplateCache) BatchCompile(templates map[string]string) error {
	if !tc.config.ConcurrentCompile {
		// Compile sequentially
		for name, source := range templates {
			if err := tc.AddTemplate(name, source); err != nil {
				return err
			}
		}
		return nil
	}
	
	// Compile concurrently
	type result struct {
		name string
		err  error
	}
	
	results := make(chan result, len(templates))
	sem := make(chan struct{}, tc.config.CompileWorkers)
	
	var wg sync.WaitGroup
	for name, source := range templates {
		wg.Add(1)
		go func(n, s string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			
			err := tc.AddTemplate(n, s)
			results <- result{name: n, err: err}
		}(name, source)
	}
	
	go func() {
		wg.Wait()
		close(results)
	}()
	
	// Collect results
	var errors []string
	for r := range results {
		if r.err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", r.name, r.err))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("batch compile errors: %s", strings.Join(errors, "; "))
	}
	
	return nil
}

// evictLRU evicts least recently used template
func (tc *TemplateCache) evictLRU() {
	var lru *CompiledTemplate
	var lruName string
	
	for name, tmpl := range tc.templates {
		if lru == nil || tmpl.LastUsed.Before(lru.LastUsed) {
			lru = tmpl
			lruName = name
		}
	}
	
	if lruName != "" {
		delete(tc.templates, lruName)
		if tc.config.EnableDedup {
			delete(tc.hashIndex, lru.Hash)
		}
		atomic.AddInt64(&tc.evictions, 1)
	}
}

// calculateHash calculates SHA256 hash of template source
func (tc *TemplateCache) calculateHash(source string) string {
	h := sha256.New()
	h.Write([]byte(source))
	return hex.EncodeToString(h.Sum(nil))
}

// extractDependencies extracts template dependencies
func (tc *TemplateCache) extractDependencies(source string) []string {
	var deps []string
	
	// Look for {{template "name"}} patterns
	// This is a simplified extraction - real implementation would use proper parsing
	lines := strings.Split(source, "\n")
	for _, line := range lines {
		if strings.Contains(line, "{{template") || strings.Contains(line, "{{ template") {
			// Extract template name
			start := strings.Index(line, `"`)
			if start != -1 {
				end := strings.Index(line[start+1:], `"`)
				if end != -1 {
					deps = append(deps, line[start+1:start+1+end])
				}
			}
		}
	}
	
	return deps
}

// updateCacheSize updates approximate cache size
func (tc *TemplateCache) updateCacheSize(delta int64) {
	atomic.AddInt64(&tc.cacheSize, delta)
}

// GetStatistics returns cache statistics
func (tc *TemplateCache) GetStatistics() map[string]interface{} {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	
	total := tc.hits + tc.misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(tc.hits) * 100.0 / float64(total)
	}
	
	totalUses := int64(0)
	avgAge := time.Duration(0)
	count := 0
	
	for _, tmpl := range tc.templates {
		totalUses += tmpl.Uses
		age := time.Since(tmpl.CompiledAt)
		avgAge += age
		count++
	}
	
	if count > 0 {
		avgAge = avgAge / time.Duration(count)
	}
	
	return map[string]interface{}{
		"template_count":      len(tc.templates),
		"set_count":           len(tc.sets),
		"cache_hits":          tc.hits,
		"cache_misses":        tc.misses,
		"hit_rate":            hitRate,
		"compilations":        tc.compilations,
		"executions":          tc.executions,
		"evictions":           tc.evictions,
		"cache_size_bytes":    tc.cacheSize,
		"cache_size_mb":       float64(tc.cacheSize) / (1024 * 1024),
		"total_uses":          totalUses,
		"avg_template_age":    avgAge.String(),
		"dedup_enabled":       tc.config.EnableDedup,
		"unique_templates":    len(tc.hashIndex),
	}
}

// Clear clears the cache
func (tc *TemplateCache) Clear() {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	
	tc.templates = make(map[string]*CompiledTemplate)
	tc.hashIndex = make(map[string]string)
	tc.sets = make(map[string]*template.Template)
	tc.hits = 0
	tc.misses = 0
	tc.compilations = 0
	tc.executions = 0
	tc.cacheSize = 0
	tc.evictions = 0
}

// Template helper functions

func quote(s string) string {
	return fmt.Sprintf("%q", s)
}

func unquote(s string) (string, error) {
	return strings.Trim(s, `"`), nil
}

func indent(spaces int, s string) string {
	pad := strings.Repeat(" ", spaces)
	lines := strings.Split(s, "\n")
	for i := range lines {
		if lines[i] != "" {
			lines[i] = pad + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}

func dedent(s string) string {
	lines := strings.Split(s, "\n")
	minIndent := -1
	
	for _, line := range lines {
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}
	
	if minIndent > 0 {
		for i := range lines {
			if len(lines[i]) >= minIndent {
				lines[i] = lines[i][minIndent:]
			}
		}
	}
	
	return strings.Join(lines, "\n")
}

func wrap(width int, s string) string {
	if width <= 0 {
		return s
	}
	
	var result []string
	words := strings.Fields(s)
	line := ""
	
	for _, word := range words {
		if len(line)+len(word)+1 > width {
			if line != "" {
				result = append(result, line)
				line = word
			} else {
				result = append(result, word)
			}
		} else {
			if line == "" {
				line = word
			} else {
				line += " " + word
			}
		}
	}
	
	if line != "" {
		result = append(result, line)
	}
	
	return strings.Join(result, "\n")
}

func defaultValue(def, val interface{}) interface{} {
	if isEmpty(val) {
		return def
	}
	return val
}

func isEmpty(val interface{}) bool {
	if val == nil {
		return true
	}
	
	switch v := val.(type) {
	case string:
		return v == ""
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	default:
		return false
	}
}

func coalesce(vals ...interface{}) interface{} {
	for _, val := range vals {
		if !isEmpty(val) {
			return val
		}
	}
	return nil
}

func ternary(cond bool, t, f interface{}) interface{} {
	if cond {
		return t
	}
	return f
}

func first(list []interface{}) interface{} {
	if len(list) > 0 {
		return list[0]
	}
	return nil
}

func last(list []interface{}) interface{} {
	if len(list) > 0 {
		return list[len(list)-1]
	}
	return nil
}

func reverse(list []interface{}) []interface{} {
	result := make([]interface{}, len(list))
	for i, j := 0, len(list)-1; i <= j; i, j = i+1, j-1 {
		result[i], result[j] = list[j], list[i]
	}
	return result
}

func unique(list []interface{}) []interface{} {
	seen := make(map[interface{}]bool)
	result := []interface{}{}
	
	for _, item := range list {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

func dict(pairs ...interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for i := 0; i < len(pairs)-1; i += 2 {
		key, ok := pairs[i].(string)
		if ok {
			result[key] = pairs[i+1]
		}
	}
	return result
}

func list(items ...interface{}) []interface{} {
	return items
}

func formatDate(format string, t time.Time) string {
	return t.Format(format)
}

func timestamp() int64 {
	return time.Now().Unix()
}