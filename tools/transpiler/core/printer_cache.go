// Package core provides go/printer integration with template pre-compilation.
// This implements Phase 1.2a: go/printer with template pre-compilation.
package core

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/printer"
	"go/token"
	"io"
	"sync"
	"sync/atomic"
	"text/template"
	"time"
)

// PrinterTemplate represents a pre-compiled template for code generation
type PrinterTemplate struct {
	Name     string
	Template *template.Template
	Source   string
	Compiled time.Time
	Uses     int64
}

// PrinterCache manages go/printer with template pre-compilation
type PrinterCache struct {
	config    *PrinterCacheConfig
	templates map[string]*PrinterTemplate
	printer   *printer.Config
	fset      *token.FileSet
	mu        sync.RWMutex
	
	// Metrics
	hits         int64
	misses       int64
	compilations int64
	printOps     int64
	totalBytes   int64
}

// PrinterCacheConfig contains configuration for printer cache
type PrinterCacheConfig struct {
	// Printer configuration
	Mode     printer.Mode
	Tabwidth int
	
	// Template options
	PrecompileTemplates bool
	MaxTemplates        int
	
	// Performance options
	BufferPoolSize int
	EnableMetrics  bool
	
	// Custom functions for templates
	TemplateFuncs template.FuncMap
}

// DefaultPrinterCacheConfig returns default configuration
func DefaultPrinterCacheConfig() *PrinterCacheConfig {
	return &PrinterCacheConfig{
		Mode:                printer.UseSpaces | printer.TabIndent,
		Tabwidth:            8,
		PrecompileTemplates: true,
		MaxTemplates:        100,
		BufferPoolSize:      50,
		EnableMetrics:       true,
		TemplateFuncs:       DefaultTemplateFuncs(),
	}
}

// DefaultTemplateFuncs returns default template functions
func DefaultTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"title":      stringTitle,
		"lower":      stringLower,
		"upper":      stringUpper,
		"capitalize": stringCapitalize,
		"indent":     stringIndent,
		"dedent":     stringDedent,
		"comment":    makeComment,
		"multiline":  makeMultiline,
	}
}

// Buffer pool for efficient memory usage
var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// NewPrinterCache creates a new printer cache
func NewPrinterCache(config *PrinterCacheConfig) *PrinterCache {
	if config == nil {
		config = DefaultPrinterCacheConfig()
	}
	
	printerConfig := &printer.Config{
		Mode:     config.Mode,
		Tabwidth: config.Tabwidth,
	}
	
	return &PrinterCache{
		config:    config,
		templates: make(map[string]*PrinterTemplate),
		printer:   printerConfig,
		fset:      token.NewFileSet(),
		mu:        sync.RWMutex{},
	}
}

// CompileTemplate pre-compiles a template for reuse
func (pc *PrinterCache) CompileTemplate(name, source string) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	// Check template limit
	if pc.config.MaxTemplates > 0 && len(pc.templates) >= pc.config.MaxTemplates {
		// Evict least used template
		pc.evictLeastUsedTemplate()
	}
	
	// Parse and compile template
	tmpl, err := template.New(name).
		Funcs(pc.config.TemplateFuncs).
		Parse(source)
	if err != nil {
		return fmt.Errorf("failed to compile template %s: %w", name, err)
	}
	
	// Store compiled template
	pc.templates[name] = &PrinterTemplate{
		Name:     name,
		Template: tmpl,
		Source:   source,
		Compiled: time.Now(),
		Uses:     0,
	}
	
	atomic.AddInt64(&pc.compilations, 1)
	return nil
}

// evictLeastUsedTemplate removes the least used template
func (pc *PrinterCache) evictLeastUsedTemplate() {
	var leastUsed *PrinterTemplate
	var leastUsedName string
	
	for name, tmpl := range pc.templates {
		if leastUsed == nil || tmpl.Uses < leastUsed.Uses {
			leastUsed = tmpl
			leastUsedName = name
		}
	}
	
	if leastUsedName != "" {
		delete(pc.templates, leastUsedName)
	}
}

// GetTemplate retrieves a compiled template
func (pc *PrinterCache) GetTemplate(name string) (*PrinterTemplate, error) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	tmpl, exists := pc.templates[name]
	if !exists {
		atomic.AddInt64(&pc.misses, 1)
		return nil, fmt.Errorf("template %s not found", name)
	}
	
	atomic.AddInt64(&tmpl.Uses, 1)
	atomic.AddInt64(&pc.hits, 1)
	return tmpl, nil
}

// ExecuteTemplate executes a pre-compiled template with data
func (pc *PrinterCache) ExecuteTemplate(name string, data interface{}) (string, error) {
	tmpl, err := pc.GetTemplate(name)
	if err != nil {
		return "", err
	}
	
	// Get buffer from pool
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)
	
	// Execute template
	if err := tmpl.Template.Execute(buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}
	
	result := buf.String()
	atomic.AddInt64(&pc.totalBytes, int64(len(result)))
	return result, nil
}

// PrintNode prints an AST node using the configured printer
func (pc *PrinterCache) PrintNode(node ast.Node) (string, error) {
	// Get buffer from pool
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)
	
	// Print node
	if err := pc.printer.Fprint(buf, pc.fset, node); err != nil {
		return "", fmt.Errorf("failed to print node: %w", err)
	}
	
	atomic.AddInt64(&pc.printOps, 1)
	result := buf.String()
	atomic.AddInt64(&pc.totalBytes, int64(len(result)))
	return result, nil
}

// PrintNodeToWriter prints an AST node directly to a writer
func (pc *PrinterCache) PrintNodeToWriter(w io.Writer, node ast.Node) error {
	if err := pc.printer.Fprint(w, pc.fset, node); err != nil {
		return fmt.Errorf("failed to print node to writer: %w", err)
	}
	
	atomic.AddInt64(&pc.printOps, 1)
	return nil
}

// FormatNode formats an AST node using go/format
func (pc *PrinterCache) FormatNode(node ast.Node) (string, error) {
	// Get buffer from pool
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)
	
	// Format node
	if err := format.Node(buf, pc.fset, node); err != nil {
		return "", fmt.Errorf("failed to format node: %w", err)
	}
	
	atomic.AddInt64(&pc.printOps, 1)
	result := buf.String()
	atomic.AddInt64(&pc.totalBytes, int64(len(result)))
	return result, nil
}

// FormatSource formats source code
func (pc *PrinterCache) FormatSource(src []byte) ([]byte, error) {
	formatted, err := format.Source(src)
	if err != nil {
		return nil, fmt.Errorf("failed to format source: %w", err)
	}
	
	atomic.AddInt64(&pc.printOps, 1)
	atomic.AddInt64(&pc.totalBytes, int64(len(formatted)))
	return formatted, nil
}

// GenerateFromTemplate generates code using a template and AST nodes
func (pc *PrinterCache) GenerateFromTemplate(templateName string, data interface{}) (ast.Node, error) {
	// Execute template to get source code
	source, err := pc.ExecuteTemplate(templateName, data)
	if err != nil {
		return nil, err
	}
	
	// Parse the generated source
	node, err := ParseString(source)
	if err != nil {
		return nil, fmt.Errorf("failed to parse generated code: %w", err)
	}
	
	return node, nil
}

// BatchPrint prints multiple nodes efficiently
func (pc *PrinterCache) BatchPrint(nodes []ast.Node) ([]string, error) {
	results := make([]string, len(nodes))
	errors := make([]error, len(nodes))
	
	// Process in parallel for large batches
	if len(nodes) > 10 {
		var wg sync.WaitGroup
		wg.Add(len(nodes))
		
		for i, node := range nodes {
			go func(idx int, n ast.Node) {
				defer wg.Done()
				result, err := pc.PrintNode(n)
				results[idx] = result
				errors[idx] = err
			}(i, node)
		}
		
		wg.Wait()
		
		// Check for errors
		for _, err := range errors {
			if err != nil {
				return results, err
			}
		}
	} else {
		// Process sequentially for small batches
		for i, node := range nodes {
			result, err := pc.PrintNode(node)
			if err != nil {
				return results, err
			}
			results[i] = result
		}
	}
	
	return results, nil
}

// GetStatistics returns cache statistics
func (pc *PrinterCache) GetStatistics() map[string]interface{} {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	// Calculate hit rate
	total := pc.hits + pc.misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(pc.hits) * 100.0 / float64(total)
	}
	
	// Get template statistics
	totalUses := int64(0)
	for _, tmpl := range pc.templates {
		totalUses += tmpl.Uses
	}
	
	avgBytesPerOp := int64(0)
	if pc.printOps > 0 {
		avgBytesPerOp = pc.totalBytes / pc.printOps
	}
	
	return map[string]interface{}{
		"template_count":     len(pc.templates),
		"template_hits":      pc.hits,
		"template_misses":    pc.misses,
		"hit_rate":           hitRate,
		"compilations":       pc.compilations,
		"print_operations":   pc.printOps,
		"total_bytes":        pc.totalBytes,
		"avg_bytes_per_op":   avgBytesPerOp,
		"total_template_uses": totalUses,
	}
}

// Clear clears all cached templates
func (pc *PrinterCache) Clear() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	pc.templates = make(map[string]*PrinterTemplate)
	pc.hits = 0
	pc.misses = 0
	pc.compilations = 0
	pc.printOps = 0
	pc.totalBytes = 0
}

// Template helper functions

func stringTitle(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(bytes.ToUpper([]byte{s[0]})) + s[1:]
}

func stringLower(s string) string {
	return string(bytes.ToLower([]byte(s)))
}

func stringUpper(s string) string {
	return string(bytes.ToUpper([]byte(s)))
}

func stringCapitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	result := bytes.ToUpper([]byte{s[0]})
	if len(s) > 1 {
		result = append(result, bytes.ToLower([]byte(s[1:]))...)
	}
	return string(result)
}

func stringIndent(n int, s string) string {
	indent := bytes.Repeat([]byte("\t"), n)
	lines := bytes.Split([]byte(s), []byte("\n"))
	for i := range lines {
		if len(lines[i]) > 0 {
			lines[i] = append(indent, lines[i]...)
		}
	}
	return string(bytes.Join(lines, []byte("\n")))
}

func stringDedent(s string) string {
	lines := bytes.Split([]byte(s), []byte("\n"))
	minIndent := -1
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		indent := 0
		for _, ch := range line {
			if ch == '\t' || ch == ' ' {
				indent++
			} else {
				break
			}
		}
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
	
	return string(bytes.Join(lines, []byte("\n")))
}

func makeComment(s string) string {
	return "// " + s
}

func makeMultiline(s string) string {
	lines := bytes.Split([]byte(s), []byte("\n"))
	result := []byte("/*\n")
	for _, line := range lines {
		result = append(result, []byte(" * ")...)
		result = append(result, line...)
		result = append(result, '\n')
	}
	result = append(result, []byte(" */")...)
	return string(result)
}

// ParseString is a helper to parse a string into an AST node
func ParseString(source string) (ast.Node, error) {
	// This would use the parser from Phase 1.1a
	// For now, return a simple implementation
	return nil, fmt.Errorf("ParseString not yet implemented - requires parser from Phase 1.1a")
}