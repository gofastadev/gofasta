// Package core provides go/doc with documentation extraction cache.
// This implements Phase 1.2i: go/doc with documentation extraction cache.
package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/doc"
	"go/format"
	"go/parser"
	"go/token"
	"html/template"
	"io"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// GoDocCache manages documentation extraction with caching
type GoDocCache struct {
	config      *GoDocCacheConfig
	packageDocs map[string]*CachedPackageDoc
	typeDocs    map[string]*TypeDoc
	funcDocs    map[string]*FuncDoc
	examples    map[string][]*doc.Example
	templates   map[string]*template.Template
	mu          sync.RWMutex
	fset        *token.FileSet
	
	// Metrics
	hits         int64
	misses       int64
	extractions  int64
	cacheSize    int64
	packageCount int64
	typeCount    int64
	funcCount    int64
}

// CachedPackageDoc represents cached package documentation
type CachedPackageDoc struct {
	*doc.Package
	Examples     []*doc.Example
	Synopsis     string
	ImportPath   string
	CachedAt     time.Time
	AccessCount  int64
	Hash         string
	FormattedDoc string // Pre-formatted documentation
}

// TypeDoc represents cached type documentation
type TypeDoc struct {
	Name         string
	Doc          string
	Declaration  string
	Methods      []*MethodDoc
	Fields       []*FieldDoc
	Examples     []*doc.Example
	CachedAt     time.Time
	AccessCount  int64
}

// MethodDoc represents method documentation
type MethodDoc struct {
	Name        string
	Doc         string
	Declaration string
	Receiver    string
	Examples    []*doc.Example
}

// FieldDoc represents field documentation
type FieldDoc struct {
	Name string
	Doc  string
	Type string
	Tag  string
}

// FuncDoc represents function documentation
type FuncDoc struct {
	Name        string
	Doc         string
	Declaration string
	Examples    []*doc.Example
	CachedAt    time.Time
	AccessCount int64
}

// GoDocCacheConfig contains configuration for documentation cache
type GoDocCacheConfig struct {
	// Cache settings
	MaxCacheEntries int
	MaxCacheSizeMB  int
	TTL             time.Duration
	EnableMetrics   bool
	
	// Documentation settings
	AllMode         bool // doc.AllMode - include unexported
	AllMethods      bool // doc.AllMethods - show all methods
	PreserveAST     bool // doc.PreserveAST - preserve AST after extraction
	
	// Export settings
	GenerateHTML    bool
	GenerateJSON    bool
	TemplateDir     string
	
	// Performance settings
	ConcurrentExtraction bool
	ExtractionWorkers    int
	PrecomputeFormats    bool
}

// DefaultGoDocCacheConfig returns default configuration
func DefaultGoDocCacheConfig() *GoDocCacheConfig {
	return &GoDocCacheConfig{
		MaxCacheEntries:      1000,
		MaxCacheSizeMB:       50,
		TTL:                  30 * time.Minute,
		EnableMetrics:        true,
		AllMode:              false,
		AllMethods:           false,
		PreserveAST:          false,
		GenerateHTML:         true,
		GenerateJSON:         true,
		ConcurrentExtraction: true,
		ExtractionWorkers:    4,
		PrecomputeFormats:    true,
	}
}

// NewGoDocCache creates a new documentation cache
func NewGoDocCache(config *GoDocCacheConfig) *GoDocCache {
	if config == nil {
		config = DefaultGoDocCacheConfig()
	}
	
	return &GoDocCache{
		config:      config,
		packageDocs: make(map[string]*CachedPackageDoc),
		typeDocs:    make(map[string]*TypeDoc),
		funcDocs:    make(map[string]*FuncDoc),
		examples:    make(map[string][]*doc.Example),
		templates:   make(map[string]*template.Template),
		fset:        token.NewFileSet(),
	}
}

// ExtractPackageDoc extracts package documentation with caching
func (gdc *GoDocCache) ExtractPackageDoc(path string, src map[string][]byte) (*doc.Package, error) {
	// Generate cache key
	key := gdc.generatePackageKey(path, src)
	
	// Check cache
	gdc.mu.RLock()
	if cached, exists := gdc.packageDocs[key]; exists {
		if gdc.config.TTL == 0 || time.Since(cached.CachedAt) < gdc.config.TTL {
			gdc.mu.RUnlock()
			atomic.AddInt64(&gdc.hits, 1)
			atomic.AddInt64(&cached.AccessCount, 1)
			return cached.Package, nil
		}
	}
	gdc.mu.RUnlock()
	
	atomic.AddInt64(&gdc.misses, 1)
	
	// Parse files
	files := make(map[string]*ast.File)
	for name, content := range src {
		file, err := parser.ParseFile(gdc.fset, name, content, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", name, err)
		}
		files[name] = file
	}
	
	// Convert map to slice of files
	fileList := make([]*ast.File, 0, len(files))
	for _, file := range files {
		fileList = append(fileList, file)
	}
	
	// Create package
	pkg, err := doc.NewFromFiles(gdc.fset, fileList, path, gdc.getMode())
	if err != nil {
		return nil, err
	}
	
	atomic.AddInt64(&gdc.extractions, 1)
	atomic.AddInt64(&gdc.packageCount, 1)
	
	// Extract examples
	var examples []*doc.Example
	for _, file := range files {
		examples = append(examples, doc.Examples(file)...)
	}
	
	// Create cached version
	cached := &CachedPackageDoc{
		Package:     pkg,
		Examples:    examples,
		Synopsis:    doc.Synopsis(pkg.Doc),
		ImportPath:  path,
		CachedAt:    time.Now(),
		AccessCount: 1,
		Hash:        key,
	}
	
	// Pre-format if configured
	if gdc.config.PrecomputeFormats {
		cached.FormattedDoc = gdc.formatPackageDoc(pkg)
	}
	
	// Cache result
	gdc.mu.Lock()
	if gdc.config.MaxCacheEntries > 0 && len(gdc.packageDocs) >= gdc.config.MaxCacheEntries {
		gdc.evictOldestPackage()
	}
	gdc.packageDocs[key] = cached
	
	// Cache individual types and functions
	gdc.cacheTypes(pkg)
	gdc.cacheFuncs(pkg)
	gdc.mu.Unlock()
	
	return pkg, nil
}

// ExtractPackageDocFromDir extracts documentation from a directory
func (gdc *GoDocCache) ExtractPackageDocFromDir(dir string) (*doc.Package, error) {
	// Find all Go files
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, err
	}
	
	// Read files
	src := make(map[string][]byte)
	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, err
		}
		src[filepath.Base(file)] = content
	}
	
	return gdc.ExtractPackageDoc(filepath.Base(dir), src)
}

// GetTypeDoc returns documentation for a type
func (gdc *GoDocCache) GetTypeDoc(pkg *doc.Package, typeName string) *TypeDoc {
	key := fmt.Sprintf("%s.%s", pkg.ImportPath, typeName)
	
	gdc.mu.RLock()
	if cached, exists := gdc.typeDocs[key]; exists {
		gdc.mu.RUnlock()
		atomic.AddInt64(&gdc.hits, 1)
		atomic.AddInt64(&cached.AccessCount, 1)
		return cached
	}
	gdc.mu.RUnlock()
	
	// Extract type documentation
	for _, t := range pkg.Types {
		if t.Name == typeName {
			typeDoc := gdc.extractTypeDoc(t)
			
			gdc.mu.Lock()
			gdc.typeDocs[key] = typeDoc
			gdc.mu.Unlock()
			
			atomic.AddInt64(&gdc.typeCount, 1)
			return typeDoc
		}
	}
	
	return nil
}

// GetFuncDoc returns documentation for a function
func (gdc *GoDocCache) GetFuncDoc(pkg *doc.Package, funcName string) *FuncDoc {
	key := fmt.Sprintf("%s.%s", pkg.ImportPath, funcName)
	
	gdc.mu.RLock()
	if cached, exists := gdc.funcDocs[key]; exists {
		gdc.mu.RUnlock()
		atomic.AddInt64(&gdc.hits, 1)
		atomic.AddInt64(&cached.AccessCount, 1)
		return cached
	}
	gdc.mu.RUnlock()
	
	// Extract function documentation
	for _, f := range pkg.Funcs {
		if f.Name == funcName {
			funcDoc := gdc.extractFuncDoc(f)
			
			gdc.mu.Lock()
			gdc.funcDocs[key] = funcDoc
			gdc.mu.Unlock()
			
			atomic.AddInt64(&gdc.funcCount, 1)
			return funcDoc
		}
	}
	
	return nil
}

// ExportHTML exports documentation as HTML
func (gdc *GoDocCache) ExportHTML(pkg *doc.Package, w io.Writer) error {
	tmpl := gdc.getHTMLTemplate()
	
	data := struct {
		Package   *doc.Package
		Timestamp time.Time
	}{
		Package:   pkg,
		Timestamp: time.Now(),
	}
	
	return tmpl.Execute(w, data)
}

// ExportJSON exports documentation as JSON
func (gdc *GoDocCache) ExportJSON(pkg *doc.Package, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	
	// Create exportable structure
	export := struct {
		Name       string                 `json:"name"`
		ImportPath string                 `json:"import_path"`
		Doc        string                 `json:"doc"`
		Synopsis   string                 `json:"synopsis"`
		Consts     []*doc.Value           `json:"constants,omitempty"`
		Vars       []*doc.Value           `json:"variables,omitempty"`
		Funcs      []SimplifiedFunc       `json:"functions,omitempty"`
		Types      []SimplifiedType       `json:"types,omitempty"`
		Examples   []*doc.Example         `json:"examples,omitempty"`
		Timestamp  time.Time              `json:"timestamp"`
	}{
		Name:       pkg.Name,
		ImportPath: pkg.ImportPath,
		Doc:        pkg.Doc,
		Synopsis:   doc.Synopsis(pkg.Doc),
		Consts:     pkg.Consts,
		Vars:       pkg.Vars,
		Timestamp:  time.Now(),
	}
	
	// Simplify functions
	for _, f := range pkg.Funcs {
		export.Funcs = append(export.Funcs, SimplifiedFunc{
			Name: f.Name,
			Doc:  f.Doc,
		})
	}
	
	// Simplify types
	for _, t := range pkg.Types {
		st := SimplifiedType{
			Name: t.Name,
			Doc:  t.Doc,
		}
		
		for _, m := range t.Methods {
			st.Methods = append(st.Methods, m.Name)
		}
		
		export.Types = append(export.Types, st)
	}
	
	// Add examples
	if cached, exists := gdc.packageDocs[pkg.ImportPath]; exists {
		export.Examples = cached.Examples
	}
	
	return encoder.Encode(export)
}

// SimplifiedFunc is a simplified function representation for JSON export
type SimplifiedFunc struct {
	Name string `json:"name"`
	Doc  string `json:"doc"`
}

// SimplifiedType is a simplified type representation for JSON export
type SimplifiedType struct {
	Name    string   `json:"name"`
	Doc     string   `json:"doc"`
	Methods []string `json:"methods,omitempty"`
}

// ExportMarkdown exports documentation as Markdown
func (gdc *GoDocCache) ExportMarkdown(pkg *doc.Package, w io.Writer) error {
	var buf bytes.Buffer
	
	// Package header
	fmt.Fprintf(&buf, "# Package %s\n\n", pkg.Name)
	
	if pkg.ImportPath != "" {
		fmt.Fprintf(&buf, "`import \"%s\"`\n\n", pkg.ImportPath)
	}
	
	// Synopsis
	if synopsis := doc.Synopsis(pkg.Doc); synopsis != "" {
		fmt.Fprintf(&buf, "> %s\n\n", synopsis)
	}
	
	// Package documentation
	if pkg.Doc != "" {
		fmt.Fprintf(&buf, "## Overview\n\n%s\n\n", pkg.Doc)
	}
	
	// Constants
	if len(pkg.Consts) > 0 {
		fmt.Fprintf(&buf, "## Constants\n\n")
		for _, c := range pkg.Consts {
			gdc.writeValueMarkdown(&buf, c)
		}
	}
	
	// Variables
	if len(pkg.Vars) > 0 {
		fmt.Fprintf(&buf, "## Variables\n\n")
		for _, v := range pkg.Vars {
			gdc.writeValueMarkdown(&buf, v)
		}
	}
	
	// Functions
	if len(pkg.Funcs) > 0 {
		fmt.Fprintf(&buf, "## Functions\n\n")
		for _, f := range pkg.Funcs {
			gdc.writeFuncMarkdown(&buf, f)
		}
	}
	
	// Types
	if len(pkg.Types) > 0 {
		fmt.Fprintf(&buf, "## Types\n\n")
		for _, t := range pkg.Types {
			gdc.writeTypeMarkdown(&buf, t)
		}
	}
	
	_, err := w.Write(buf.Bytes())
	return err
}

// SearchDocumentation searches for documentation by keyword
func (gdc *GoDocCache) SearchDocumentation(keyword string) []SearchResult {
	var results []SearchResult
	keyword = strings.ToLower(keyword)
	
	gdc.mu.RLock()
	defer gdc.mu.RUnlock()
	
	// Search in packages
	for _, pkg := range gdc.packageDocs {
		if strings.Contains(strings.ToLower(pkg.Name), keyword) ||
			strings.Contains(strings.ToLower(pkg.Doc), keyword) {
			results = append(results, SearchResult{
				Type:    "package",
				Name:    pkg.Name,
				Path:    pkg.ImportPath,
				Doc:     doc.Synopsis(pkg.Doc),
				Matches: 1,
			})
		}
	}
	
	// Search in types
	for key, t := range gdc.typeDocs {
		if strings.Contains(strings.ToLower(t.Name), keyword) ||
			strings.Contains(strings.ToLower(t.Doc), keyword) {
			results = append(results, SearchResult{
				Type:    "type",
				Name:    t.Name,
				Path:    key,
				Doc:     doc.Synopsis(t.Doc),
				Matches: 1,
			})
		}
	}
	
	// Search in functions
	for key, f := range gdc.funcDocs {
		if strings.Contains(strings.ToLower(f.Name), keyword) ||
			strings.Contains(strings.ToLower(f.Doc), keyword) {
			results = append(results, SearchResult{
				Type:    "function",
				Name:    f.Name,
				Path:    key,
				Doc:     doc.Synopsis(f.Doc),
				Matches: 1,
			})
		}
	}
	
	// Sort by relevance (name matches first)
	sort.Slice(results, func(i, j int) bool {
		iNameMatch := strings.Contains(strings.ToLower(results[i].Name), keyword)
		jNameMatch := strings.Contains(strings.ToLower(results[j].Name), keyword)
		
		if iNameMatch != jNameMatch {
			return iNameMatch
		}
		
		return results[i].Name < results[j].Name
	})
	
	return results
}

// SearchResult represents a documentation search result
type SearchResult struct {
	Type    string // "package", "type", "function", etc.
	Name    string
	Path    string
	Doc     string
	Matches int
}

// GetExamples returns examples for a package or symbol
func (gdc *GoDocCache) GetExamples(pkg *doc.Package, symbol string) []*doc.Example {
	key := pkg.ImportPath
	if symbol != "" {
		key = fmt.Sprintf("%s.%s", pkg.ImportPath, symbol)
	}
	
	gdc.mu.RLock()
	defer gdc.mu.RUnlock()
	
	if examples, exists := gdc.examples[key]; exists {
		return examples
	}
	
	// Extract from cached package
	if cached, exists := gdc.packageDocs[pkg.ImportPath]; exists {
		var result []*doc.Example
		for _, ex := range cached.Examples {
			if symbol == "" || ex.Name == symbol {
				result = append(result, ex)
			}
		}
		return result
	}
	
	return nil
}

// BatchExtract extracts documentation from multiple packages concurrently
func (gdc *GoDocCache) BatchExtract(packages map[string]map[string][]byte) map[string]*doc.Package {
	if !gdc.config.ConcurrentExtraction || len(packages) <= 1 {
		// Extract sequentially
		results := make(map[string]*doc.Package)
		for path, src := range packages {
			if pkg, err := gdc.ExtractPackageDoc(path, src); err == nil {
				results[path] = pkg
			}
		}
		return results
	}
	
	// Extract concurrently
	results := make(map[string]*doc.Package)
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, gdc.config.ExtractionWorkers)
	
	for path, src := range packages {
		wg.Add(1)
		go func(p string, s map[string][]byte) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			
			if pkg, err := gdc.ExtractPackageDoc(p, s); err == nil {
				mu.Lock()
				results[p] = pkg
				mu.Unlock()
			}
		}(path, src)
	}
	
	wg.Wait()
	return results
}

// Helper methods

func (gdc *GoDocCache) getMode() doc.Mode {
	mode := doc.Mode(0)
	if gdc.config.AllMode {
		mode |= doc.AllDecls
	}
	if gdc.config.AllMethods {
		mode |= doc.AllMethods
	}
	if gdc.config.PreserveAST {
		mode |= doc.PreserveAST
	}
	return mode
}

func (gdc *GoDocCache) extractTypeDoc(t *doc.Type) *TypeDoc {
	typeDoc := &TypeDoc{
		Name:        t.Name,
		Doc:         t.Doc,
		Declaration: gdc.getDeclaration(t.Decl),
		CachedAt:    time.Now(),
		AccessCount: 1,
	}
	
	// Extract methods
	for _, m := range t.Methods {
		typeDoc.Methods = append(typeDoc.Methods, &MethodDoc{
			Name:        m.Name,
			Doc:         m.Doc,
			Declaration: gdc.getDeclaration(m.Decl),
			Receiver:    m.Recv,
		})
	}
	
	// Extract fields (if struct)
	if t.Decl != nil {
		gdc.extractFields(t.Decl, typeDoc)
	}
	
	return typeDoc
}

func (gdc *GoDocCache) extractFuncDoc(f *doc.Func) *FuncDoc {
	return &FuncDoc{
		Name:        f.Name,
		Doc:         f.Doc,
		Declaration: gdc.getDeclaration(f.Decl),
		CachedAt:    time.Now(),
		AccessCount: 1,
	}
}

func (gdc *GoDocCache) extractFields(decl ast.Decl, typeDoc *TypeDoc) {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok {
		return
	}
	
	for _, spec := range genDecl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}
		
		for _, field := range structType.Fields.List {
			fieldDoc := &FieldDoc{}
			
			// Field names
			if len(field.Names) > 0 {
				fieldDoc.Name = field.Names[0].Name
			}
			
			// Field type
			fieldDoc.Type = gdc.formatType(field.Type)
			
			// Field tag
			if field.Tag != nil {
				fieldDoc.Tag = field.Tag.Value
			}
			
			// Field doc
			if field.Doc != nil {
				fieldDoc.Doc = field.Doc.Text()
			} else if field.Comment != nil {
				fieldDoc.Doc = field.Comment.Text()
			}
			
			typeDoc.Fields = append(typeDoc.Fields, fieldDoc)
		}
	}
}

func (gdc *GoDocCache) getDeclaration(node ast.Node) string {
	if node == nil {
		return ""
	}
	
	var buf bytes.Buffer
	if err := format.Node(&buf, gdc.fset, node); err != nil {
		return ""
	}
	return buf.String()
}

func (gdc *GoDocCache) formatType(expr ast.Expr) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, gdc.fset, expr); err != nil {
		return ""
	}
	return buf.String()
}

func (gdc *GoDocCache) formatPackageDoc(pkg *doc.Package) string {
	var buf bytes.Buffer
	
	fmt.Fprintf(&buf, "Package %s\n", pkg.Name)
	if pkg.ImportPath != "" {
		fmt.Fprintf(&buf, "Import: %s\n", pkg.ImportPath)
	}
	fmt.Fprintf(&buf, "\n%s\n", pkg.Doc)
	
	return buf.String()
}

func (gdc *GoDocCache) cacheTypes(pkg *doc.Package) {
	for _, t := range pkg.Types {
		key := fmt.Sprintf("%s.%s", pkg.ImportPath, t.Name)
		if _, exists := gdc.typeDocs[key]; !exists {
			gdc.typeDocs[key] = gdc.extractTypeDoc(t)
			atomic.AddInt64(&gdc.typeCount, 1)
		}
	}
}

func (gdc *GoDocCache) cacheFuncs(pkg *doc.Package) {
	for _, f := range pkg.Funcs {
		key := fmt.Sprintf("%s.%s", pkg.ImportPath, f.Name)
		if _, exists := gdc.funcDocs[key]; !exists {
			gdc.funcDocs[key] = gdc.extractFuncDoc(f)
			atomic.AddInt64(&gdc.funcCount, 1)
		}
	}
}

func (gdc *GoDocCache) getHTMLTemplate() *template.Template {
	if tmpl, exists := gdc.templates["html"]; exists {
		return tmpl
	}
	
	tmplText := `<!DOCTYPE html>
<html>
<head>
    <title>{{.Package.Name}} - GoDoc</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        h1 { color: #333; }
        h2 { color: #666; border-bottom: 1px solid #ccc; }
        pre { background: #f4f4f4; padding: 10px; overflow-x: auto; }
        .synopsis { font-style: italic; color: #666; }
    </style>
</head>
<body>
    <h1>Package {{.Package.Name}}</h1>
    {{if .Package.ImportPath}}
    <p><code>import "{{.Package.ImportPath}}"</code></p>
    {{end}}
    
    {{if .Package.Doc}}
    <div class="synopsis">{{.Package.Doc}}</div>
    {{end}}
    
    <p>Generated: {{.Timestamp.Format "2006-01-02 15:04:05"}}</p>
</body>
</html>`
	
	tmpl, _ := template.New("html").Parse(tmplText)
	gdc.templates["html"] = tmpl
	return tmpl
}

func (gdc *GoDocCache) writeValueMarkdown(w io.Writer, v *doc.Value) {
	if len(v.Names) > 0 {
		fmt.Fprintf(w, "### %s\n\n", strings.Join(v.Names, ", "))
	}
	if v.Doc != "" {
		fmt.Fprintf(w, "%s\n\n", v.Doc)
	}
	fmt.Fprintf(w, "```go\n%s\n```\n\n", gdc.getDeclaration(v.Decl))
}

func (gdc *GoDocCache) writeFuncMarkdown(w io.Writer, f *doc.Func) {
	fmt.Fprintf(w, "### func %s\n\n", f.Name)
	if f.Doc != "" {
		fmt.Fprintf(w, "%s\n\n", f.Doc)
	}
	fmt.Fprintf(w, "```go\n%s\n```\n\n", gdc.getDeclaration(f.Decl))
}

func (gdc *GoDocCache) writeTypeMarkdown(w io.Writer, t *doc.Type) {
	fmt.Fprintf(w, "### type %s\n\n", t.Name)
	if t.Doc != "" {
		fmt.Fprintf(w, "%s\n\n", t.Doc)
	}
	fmt.Fprintf(w, "```go\n%s\n```\n\n", gdc.getDeclaration(t.Decl))
	
	// Methods
	if len(t.Methods) > 0 {
		fmt.Fprintf(w, "#### Methods\n\n")
		for _, m := range t.Methods {
			fmt.Fprintf(w, "##### %s\n\n", m.Name)
			if m.Doc != "" {
				fmt.Fprintf(w, "%s\n\n", m.Doc)
			}
		}
	}
}

// Cache key generation

func (gdc *GoDocCache) generatePackageKey(path string, src map[string][]byte) string {
	h := sha256.New()
	h.Write([]byte(path))
	
	// Sort files for consistent hashing
	var files []string
	for name := range src {
		files = append(files, name)
	}
	sort.Strings(files)
	
	for _, name := range files {
		h.Write([]byte(name))
		h.Write(src[name])
	}
	
	return hex.EncodeToString(h.Sum(nil))
}

// Cache eviction

func (gdc *GoDocCache) evictOldestPackage() {
	var oldest *CachedPackageDoc
	var oldestKey string
	
	for key, pkg := range gdc.packageDocs {
		if oldest == nil || pkg.CachedAt.Before(oldest.CachedAt) {
			oldest = pkg
			oldestKey = key
		}
	}
	
	if oldestKey != "" {
		delete(gdc.packageDocs, oldestKey)
	}
}

// GetStatistics returns cache statistics
func (gdc *GoDocCache) GetStatistics() map[string]interface{} {
	gdc.mu.RLock()
	defer gdc.mu.RUnlock()
	
	total := gdc.hits + gdc.misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(gdc.hits) * 100.0 / float64(total)
	}
	
	return map[string]interface{}{
		"package_cache_size": len(gdc.packageDocs),
		"type_cache_size":    len(gdc.typeDocs),
		"func_cache_size":    len(gdc.funcDocs),
		"example_cache_size": len(gdc.examples),
		"cache_hits":         gdc.hits,
		"cache_misses":       gdc.misses,
		"hit_rate":           hitRate,
		"total_extractions":  gdc.extractions,
		"package_count":      gdc.packageCount,
		"type_count":         gdc.typeCount,
		"func_count":         gdc.funcCount,
	}
}

// Clear clears all caches
func (gdc *GoDocCache) Clear() {
	gdc.mu.Lock()
	defer gdc.mu.Unlock()
	
	gdc.packageDocs = make(map[string]*CachedPackageDoc)
	gdc.typeDocs = make(map[string]*TypeDoc)
	gdc.funcDocs = make(map[string]*FuncDoc)
	gdc.examples = make(map[string][]*doc.Example)
	gdc.hits = 0
	gdc.misses = 0
	gdc.extractions = 0
	gdc.packageCount = 0
	gdc.typeCount = 0
	gdc.funcCount = 0
}

// WarmupCache preloads documentation for common packages
func (gdc *GoDocCache) WarmupCache(packages map[string]map[string][]byte) {
	gdc.BatchExtract(packages)
}