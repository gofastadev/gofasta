// Package core provides html/template with documentation generation cache.
// This implements Phase 1.2c: html/template with documentation generation cache.
package core

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/doc"
	"go/token"
	"html/template"
	"io"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// DocTemplate represents a compiled documentation template
type DocTemplate struct {
	Name       string
	Template   *template.Template
	Source     string
	Hash       string
	CompiledAt time.Time
	LastUsed   time.Time
	Uses       int64
	Type       string // "api", "package", "type", "function", etc.
}

// DocCache manages html/template with documentation generation cache
type DocCache struct {
	config      *DocCacheConfig
	templates   map[string]*DocTemplate
	docCache    map[string]*CachedDoc
	packageDocs map[string]*doc.Package
	mu          sync.RWMutex
	fset        *token.FileSet
	
	// Metrics
	hits            int64
	misses          int64
	compilations    int64
	generations     int64
	cacheSize       int64
	docExtractions  int64
}

// CachedDoc represents cached documentation
type CachedDoc struct {
	HTML       string
	Markdown   string
	JSON       string
	Package    string
	Type       string
	Generated  time.Time
	Hash       string
	References []string // Other docs this references
}

// DocCacheConfig contains configuration for documentation cache
type DocCacheConfig struct {
	// Cache settings
	MaxTemplates   int
	MaxDocs        int
	MaxCacheSizeMB int
	EnableMetrics  bool
	
	// Template settings
	TemplateDir     string
	CustomFuncs     template.FuncMap
	EnableMarkdown  bool
	EnableJSON      bool
	
	// Documentation settings
	IncludePrivate  bool
	IncludeExamples bool
	IncludeTests    bool
	GenerateIndex   bool
	
	// Performance settings
	PrecompileTemplates bool
	ConcurrentGeneration bool
	GenerationWorkers   int
}

// DefaultDocCacheConfig returns default configuration
func DefaultDocCacheConfig() *DocCacheConfig {
	return &DocCacheConfig{
		MaxTemplates:         100,
		MaxDocs:              500,
		MaxCacheSizeMB:       50,
		EnableMetrics:        true,
		TemplateDir:          "templates/doc",
		CustomFuncs:          DefaultDocFuncMap(),
		EnableMarkdown:       true,
		EnableJSON:           true,
		IncludePrivate:       false,
		IncludeExamples:      true,
		IncludeTests:         false,
		GenerateIndex:        true,
		PrecompileTemplates:  true,
		ConcurrentGeneration: true,
		GenerationWorkers:    4,
	}
}

// DefaultDocFuncMap returns default template functions for documentation
func DefaultDocFuncMap() template.FuncMap {
	return template.FuncMap{
		// String functions
		"escape":     template.HTMLEscapeString,
		"unescape":   template.HTMLEscapeString,
		"trim":       strings.TrimSpace,
		"lower":      strings.ToLower,
		"upper":      strings.ToUpper,
		"title":      strings.Title,
		"replace":    strings.ReplaceAll,
		
		// Documentation functions
		"anchor":     makeAnchor,
		"link":       makeLink,
		"codeblock":  makeCodeBlock,
		"highlight":  syntaxHighlight,
		"toc":        generateTOC,
		"breadcrumb": makeBreadcrumb,
		"signature":  formatSignature,
		"example":    formatExample,
		
		// Type functions
		"typeLink":   makeTypeLink,
		"pkgLink":    makePackageLink,
		"methodLink": makeMethodLink,
		"funcLink":   makeFunctionLink,
		
		// Formatting functions
		"markdown":   renderMarkdown,
		"json":       renderJSON,
		"indent":     indentCode,
		"dedent":     dedentCode,
		"wrap":       wrapText,
	}
}

// NewDocCache creates a new documentation cache
func NewDocCache(config *DocCacheConfig) *DocCache {
	if config == nil {
		config = DefaultDocCacheConfig()
	}
	
	return &DocCache{
		config:      config,
		templates:   make(map[string]*DocTemplate),
		docCache:    make(map[string]*CachedDoc),
		packageDocs: make(map[string]*doc.Package),
		fset:        token.NewFileSet(),
	}
}

// LoadTemplates loads documentation templates from directory
func (dc *DocCache) LoadTemplates(dir string) error {
	if dir == "" {
		dir = dc.config.TemplateDir
	}
	
	// Define standard documentation templates
	standardTemplates := map[string]string{
		"package": packageTemplate,
		"type":    typeTemplate,
		"func":    functionTemplate,
		"method":  methodTemplate,
		"const":   constantTemplate,
		"var":     variableTemplate,
		"index":   indexTemplate,
		"api":     apiTemplate,
	}
	
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	// Compile standard templates
	for name, source := range standardTemplates {
		if err := dc.compileTemplate(name, source, name); err != nil {
			return fmt.Errorf("failed to compile template %s: %w", name, err)
		}
	}
	
	// Load custom templates from directory if specified
	// This would load from filesystem in a real implementation
	
	return nil
}

// compileTemplate compiles a documentation template
func (dc *DocCache) compileTemplate(name, source, templateType string) error {
	// Create template with custom functions
	tmpl := template.New(name).Funcs(dc.config.CustomFuncs)
	
	// Parse template
	parsed, err := tmpl.Parse(source)
	if err != nil {
		return err
	}
	
	// Calculate hash
	h := md5.New()
	h.Write([]byte(source))
	hash := hex.EncodeToString(h.Sum(nil))
	
	// Store compiled template
	dc.templates[name] = &DocTemplate{
		Name:       name,
		Template:   parsed,
		Source:     source,
		Hash:       hash,
		CompiledAt: time.Now(),
		LastUsed:   time.Now(),
		Uses:       0,
		Type:       templateType,
	}
	
	atomic.AddInt64(&dc.compilations, 1)
	return nil
}

// ExtractPackageDoc extracts documentation from a package
func (dc *DocCache) ExtractPackageDoc(pkg *ast.Package, importPath string) (*doc.Package, error) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	// Check cache
	if cached, exists := dc.packageDocs[importPath]; exists {
		return cached, nil
	}
	
	// Extract documentation
	mode := doc.Mode(0)
	if dc.config.IncludePrivate {
		mode = doc.AllDecls
	}
	
	docPkg := doc.New(pkg, importPath, mode)
	
	// Sort documentation elements
	sortDocumentation(docPkg)
	
	// Cache the package documentation
	dc.packageDocs[importPath] = docPkg
	atomic.AddInt64(&dc.docExtractions, 1)
	
	return docPkg, nil
}

// GeneratePackageDoc generates HTML documentation for a package
func (dc *DocCache) GeneratePackageDoc(pkg *doc.Package) (string, error) {
	return dc.generateDoc("package", pkg, pkg.ImportPath)
}

// GenerateTypeDoc generates HTML documentation for a type
func (dc *DocCache) GenerateTypeDoc(t *doc.Type, pkg string) (string, error) {
	data := map[string]interface{}{
		"Type":    t,
		"Package": pkg,
	}
	return dc.generateDoc("type", data, fmt.Sprintf("%s.%s", pkg, t.Name))
}

// GenerateFunctionDoc generates HTML documentation for a function
func (dc *DocCache) GenerateFunctionDoc(f *doc.Func, pkg string) (string, error) {
	data := map[string]interface{}{
		"Function": f,
		"Package":  pkg,
	}
	return dc.generateDoc("func", data, fmt.Sprintf("%s.%s", pkg, f.Name))
}

// generateDoc generates documentation using a template
func (dc *DocCache) generateDoc(templateName string, data interface{}, cacheKey string) (string, error) {
	// Check cache
	dc.mu.RLock()
	if cached, exists := dc.docCache[cacheKey]; exists {
		dc.mu.RUnlock()
		atomic.AddInt64(&dc.hits, 1)
		return cached.HTML, nil
	}
	dc.mu.RUnlock()
	
	atomic.AddInt64(&dc.misses, 1)
	
	// Get template
	dc.mu.RLock()
	tmpl, exists := dc.templates[templateName]
	dc.mu.RUnlock()
	
	if !exists {
		return "", fmt.Errorf("template %s not found", templateName)
	}
	
	// Generate documentation
	var buf bytes.Buffer
	if err := tmpl.Template.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	
	html := buf.String()
	
	// Generate other formats if enabled
	var markdown, jsonStr string
	if dc.config.EnableMarkdown {
		markdown = dc.htmlToMarkdown(html)
	}
	if dc.config.EnableJSON {
		jsonStr = dc.dataToJSON(data)
	}
	
	// Cache the generated documentation
	cached := &CachedDoc{
		HTML:      html,
		Markdown:  markdown,
		JSON:      jsonStr,
		Package:   extractPackageName(cacheKey),
		Type:      templateName,
		Generated: time.Now(),
		Hash:      dc.calculateHash(html),
	}
	
	dc.mu.Lock()
	// Check cache size and evict if necessary
	if dc.config.MaxDocs > 0 && len(dc.docCache) >= dc.config.MaxDocs {
		dc.evictOldestDoc()
	}
	dc.docCache[cacheKey] = cached
	dc.mu.Unlock()
	
	atomic.AddInt64(&dc.generations, 1)
	atomic.AddInt64(&dc.cacheSize, int64(len(html)))
	atomic.AddInt64(&tmpl.Uses, 1)
	tmpl.LastUsed = time.Now()
	
	return html, nil
}

// GenerateAPIDoc generates API documentation for multiple packages
func (dc *DocCache) GenerateAPIDoc(packages []*doc.Package) (string, error) {
	data := map[string]interface{}{
		"Packages": packages,
		"Title":    "API Documentation",
		"Generated": time.Now(),
	}
	
	return dc.generateDoc("api", data, "api_index")
}

// GenerateIndex generates an index of all documentation
func (dc *DocCache) GenerateIndex() (string, error) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	
	// Collect all packages
	var packages []string
	for key := range dc.packageDocs {
		packages = append(packages, key)
	}
	sort.Strings(packages)
	
	data := map[string]interface{}{
		"Packages":  packages,
		"Generated": time.Now(),
		"Stats":     dc.GetStatistics(),
	}
	
	return dc.generateDoc("index", data, "doc_index")
}

// BatchGenerate generates documentation for multiple items concurrently
func (dc *DocCache) BatchGenerate(items []DocItem) (map[string]string, error) {
	if !dc.config.ConcurrentGeneration {
		// Generate sequentially
		results := make(map[string]string)
		for _, item := range items {
			doc, err := dc.generateForItem(item)
			if err != nil {
				return results, err
			}
			results[item.Key] = doc
		}
		return results, nil
	}
	
	// Generate concurrently
	type result struct {
		key string
		doc string
		err error
	}
	
	results := make(chan result, len(items))
	sem := make(chan struct{}, dc.config.GenerationWorkers)
	
	var wg sync.WaitGroup
	for _, item := range items {
		wg.Add(1)
		go func(it DocItem) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			
			doc, err := dc.generateForItem(it)
			results <- result{key: it.Key, doc: doc, err: err}
		}(item)
	}
	
	go func() {
		wg.Wait()
		close(results)
	}()
	
	// Collect results
	docs := make(map[string]string)
	for r := range results {
		if r.err != nil {
			return docs, r.err
		}
		docs[r.key] = r.doc
	}
	
	return docs, nil
}

// DocItem represents an item to generate documentation for
type DocItem struct {
	Key  string
	Type string
	Data interface{}
}

// generateForItem generates documentation for a single item
func (dc *DocCache) generateForItem(item DocItem) (string, error) {
	return dc.generateDoc(item.Type, item.Data, item.Key)
}

// evictOldestDoc evicts the oldest documentation from cache
func (dc *DocCache) evictOldestDoc() {
	var oldest *CachedDoc
	var oldestKey string
	
	for key, doc := range dc.docCache {
		if oldest == nil || doc.Generated.Before(oldest.Generated) {
			oldest = doc
			oldestKey = key
		}
	}
	
	if oldestKey != "" {
		delete(dc.docCache, oldestKey)
		atomic.AddInt64(&dc.cacheSize, -int64(len(oldest.HTML)))
	}
}

// calculateHash calculates hash of content
func (dc *DocCache) calculateHash(content string) string {
	h := md5.New()
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))
}

// htmlToMarkdown converts HTML to Markdown (simplified)
func (dc *DocCache) htmlToMarkdown(html string) string {
	// This is a simplified conversion
	md := html
	md = strings.ReplaceAll(md, "<h1>", "# ")
	md = strings.ReplaceAll(md, "</h1>", "\n")
	md = strings.ReplaceAll(md, "<h2>", "## ")
	md = strings.ReplaceAll(md, "</h2>", "\n")
	md = strings.ReplaceAll(md, "<h3>", "### ")
	md = strings.ReplaceAll(md, "</h3>", "\n")
	md = strings.ReplaceAll(md, "<code>", "`")
	md = strings.ReplaceAll(md, "</code>", "`")
	md = strings.ReplaceAll(md, "<pre>", "```\n")
	md = strings.ReplaceAll(md, "</pre>", "\n```")
	md = strings.ReplaceAll(md, "<br>", "\n")
	md = strings.ReplaceAll(md, "<p>", "\n")
	md = strings.ReplaceAll(md, "</p>", "\n")
	return md
}

// dataToJSON converts data to JSON (simplified)
func (dc *DocCache) dataToJSON(data interface{}) string {
	// This would use json.Marshal in a real implementation
	return fmt.Sprintf("%+v", data)
}

// GetStatistics returns cache statistics
func (dc *DocCache) GetStatistics() map[string]interface{} {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	
	total := dc.hits + dc.misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(dc.hits) * 100.0 / float64(total)
	}
	
	return map[string]interface{}{
		"template_count":     len(dc.templates),
		"cached_docs":        len(dc.docCache),
		"package_docs":       len(dc.packageDocs),
		"cache_hits":         dc.hits,
		"cache_misses":       dc.misses,
		"hit_rate":           hitRate,
		"compilations":       dc.compilations,
		"generations":        dc.generations,
		"doc_extractions":    dc.docExtractions,
		"cache_size_bytes":   dc.cacheSize,
		"cache_size_mb":      float64(dc.cacheSize) / (1024 * 1024),
	}
}

// Clear clears the cache
func (dc *DocCache) Clear() {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	dc.templates = make(map[string]*DocTemplate)
	dc.docCache = make(map[string]*CachedDoc)
	dc.packageDocs = make(map[string]*doc.Package)
	dc.hits = 0
	dc.misses = 0
	dc.compilations = 0
	dc.generations = 0
	dc.cacheSize = 0
	dc.docExtractions = 0
}

// GetCachedDoc retrieves cached documentation
func (dc *DocCache) GetCachedDoc(key string) (*CachedDoc, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	doc, exists := dc.docCache[key]
	return doc, exists
}

// InvalidateDoc invalidates cached documentation
func (dc *DocCache) InvalidateDoc(key string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	if doc, exists := dc.docCache[key]; exists {
		delete(dc.docCache, key)
		atomic.AddInt64(&dc.cacheSize, -int64(len(doc.HTML)))
	}
}

// Helper functions for templates

func makeAnchor(text string) template.HTML {
	id := strings.ToLower(strings.ReplaceAll(text, " ", "-"))
	return template.HTML(fmt.Sprintf(`<a id="%s"></a>`, id))
}

func makeLink(text, href string) template.HTML {
	return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, href, template.HTMLEscapeString(text)))
}

func makeCodeBlock(code string, lang string) template.HTML {
	return template.HTML(fmt.Sprintf(`<pre><code class="language-%s">%s</code></pre>`, 
		lang, template.HTMLEscapeString(code)))
}

func syntaxHighlight(code string, lang string) template.HTML {
	// Simplified syntax highlighting
	highlighted := template.HTMLEscapeString(code)
	// In a real implementation, this would use a proper syntax highlighter
	return template.HTML(fmt.Sprintf(`<pre class="highlight-%s">%s</pre>`, lang, highlighted))
}

func generateTOC(headings []string) template.HTML {
	var toc strings.Builder
	toc.WriteString("<ul class=\"toc\">\n")
	for _, h := range headings {
		id := strings.ToLower(strings.ReplaceAll(h, " ", "-"))
		toc.WriteString(fmt.Sprintf(`<li><a href="#%s">%s</a></li>`, id, template.HTMLEscapeString(h)))
		toc.WriteString("\n")
	}
	toc.WriteString("</ul>")
	return template.HTML(toc.String())
}

func makeBreadcrumb(parts []string) template.HTML {
	var crumb strings.Builder
	crumb.WriteString(`<nav class="breadcrumb">`)
	for i, part := range parts {
		if i > 0 {
			crumb.WriteString(" › ")
		}
		crumb.WriteString(template.HTMLEscapeString(part))
	}
	crumb.WriteString("</nav>")
	return template.HTML(crumb.String())
}

func formatSignature(sig string) template.HTML {
	return template.HTML(fmt.Sprintf(`<code class="signature">%s</code>`, template.HTMLEscapeString(sig)))
}

func formatExample(example string) template.HTML {
	return makeCodeBlock(example, "go")
}

func makeTypeLink(typeName, pkg string) template.HTML {
	href := fmt.Sprintf("/pkg/%s#%s", pkg, typeName)
	return makeLink(typeName, href)
}

func makePackageLink(pkg string) template.HTML {
	return makeLink(pkg, fmt.Sprintf("/pkg/%s", pkg))
}

func makeMethodLink(method, typeName, pkg string) template.HTML {
	href := fmt.Sprintf("/pkg/%s#%s.%s", pkg, typeName, method)
	return makeLink(fmt.Sprintf("%s.%s", typeName, method), href)
}

func makeFunctionLink(fn, pkg string) template.HTML {
	href := fmt.Sprintf("/pkg/%s#%s", pkg, fn)
	return makeLink(fn, href)
}

func renderMarkdown(text string) template.HTML {
	// Simplified markdown rendering
	return template.HTML(text)
}

func renderJSON(data interface{}) template.HTML {
	return template.HTML(fmt.Sprintf("<pre>%+v</pre>", data))
}

func indentCode(spaces int, code string) string {
	indent := strings.Repeat(" ", spaces)
	lines := strings.Split(code, "\n")
	for i := range lines {
		if lines[i] != "" {
			lines[i] = indent + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}

func dedentCode(code string) string {
	lines := strings.Split(code, "\n")
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

func wrapText(width int, text string) string {
	if width <= 0 {
		return text
	}
	
	var result []string
	words := strings.Fields(text)
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

func extractPackageName(key string) string {
	parts := strings.Split(key, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return key
}

func sortDocumentation(pkg *doc.Package) {
	// Sort various documentation elements
	sort.Slice(pkg.Types, func(i, j int) bool {
		return pkg.Types[i].Name < pkg.Types[j].Name
	})
	sort.Slice(pkg.Funcs, func(i, j int) bool {
		return pkg.Funcs[i].Name < pkg.Funcs[j].Name
	})
	sort.Slice(pkg.Consts, func(i, j int) bool {
		return pkg.Consts[i].Names[0] < pkg.Consts[j].Names[0]
	})
	sort.Slice(pkg.Vars, func(i, j int) bool {
		return pkg.Vars[i].Names[0] < pkg.Vars[j].Names[0]
	})
}

// Standard documentation templates
const packageTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>{{.Name}} - Go Documentation</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        h1 { color: #375EAB; }
        h2 { color: #375EAB; border-bottom: 1px solid #375EAB; }
        pre { background: #f4f4f4; padding: 10px; overflow-x: auto; }
        code { background: #f4f4f4; padding: 2px 4px; }
        .signature { display: block; margin: 10px 0; }
    </style>
</head>
<body>
    <h1>Package {{.Name}}</h1>
    <p>{{.Doc}}</p>
    
    {{if .Consts}}
    <h2>Constants</h2>
    {{range .Consts}}
        <h3>{{range .Names}}{{.}} {{end}}</h3>
        <pre>{{.Decl}}</pre>
        <p>{{.Doc}}</p>
    {{end}}
    {{end}}
    
    {{if .Vars}}
    <h2>Variables</h2>
    {{range .Vars}}
        <h3>{{range .Names}}{{.}} {{end}}</h3>
        <pre>{{.Decl}}</pre>
        <p>{{.Doc}}</p>
    {{end}}
    {{end}}
    
    {{if .Funcs}}
    <h2>Functions</h2>
    {{range .Funcs}}
        <h3 id="{{.Name}}">func {{.Name}}</h3>
        <pre>{{.Decl}}</pre>
        <p>{{.Doc}}</p>
    {{end}}
    {{end}}
    
    {{if .Types}}
    <h2>Types</h2>
    {{range .Types}}
        <h3 id="{{.Name}}">type {{.Name}}</h3>
        <pre>{{.Decl}}</pre>
        <p>{{.Doc}}</p>
        
        {{if .Methods}}
        <h4>Methods</h4>
        {{range .Methods}}
            <h5 id="{{.Recv}}.{{.Name}}">func ({{.Recv}}) {{.Name}}</h5>
            <pre>{{.Decl}}</pre>
            <p>{{.Doc}}</p>
        {{end}}
        {{end}}
    {{end}}
    {{end}}
</body>
</html>`

const typeTemplate = `<div class="type-doc">
    <h3>type {{.Type.Name}}</h3>
    <pre>{{.Type.Decl}}</pre>
    <p>{{.Type.Doc}}</p>
</div>`

const functionTemplate = `<div class="func-doc">
    <h3>func {{.Function.Name}}</h3>
    <pre>{{.Function.Decl}}</pre>
    <p>{{.Function.Doc}}</p>
</div>`

const methodTemplate = `<div class="method-doc">
    <h3>func ({{.Method.Recv}}) {{.Method.Name}}</h3>
    <pre>{{.Method.Decl}}</pre>
    <p>{{.Method.Doc}}</p>
</div>`

const constantTemplate = `<div class="const-doc">
    <h3>{{range .Const.Names}}{{.}} {{end}}</h3>
    <pre>{{.Const.Decl}}</pre>
    <p>{{.Const.Doc}}</p>
</div>`

const variableTemplate = `<div class="var-doc">
    <h3>{{range .Var.Names}}{{.}} {{end}}</h3>
    <pre>{{.Var.Decl}}</pre>
    <p>{{.Var.Doc}}</p>
</div>`

const indexTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>Documentation Index</title>
</head>
<body>
    <h1>Documentation Index</h1>
    <h2>Packages</h2>
    <ul>
    {{range .Packages}}
        <li><a href="/pkg/{{.}}">{{.}}</a></li>
    {{end}}
    </ul>
    <p>Generated: {{.Generated}}</p>
</body>
</html>`

const apiTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
</head>
<body>
    <h1>{{.Title}}</h1>
    {{range .Packages}}
        <section>
            <h2>{{.Name}}</h2>
            <p>{{.Doc}}</p>
        </section>
    {{end}}
</body>
</html>`

// ExportHTML exports documentation as HTML
func (dc *DocCache) ExportHTML(w io.Writer, key string) error {
	dc.mu.RLock()
	doc, exists := dc.docCache[key]
	dc.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("documentation not found: %s", key)
	}
	
	_, err := w.Write([]byte(doc.HTML))
	return err
}

// ExportMarkdown exports documentation as Markdown
func (dc *DocCache) ExportMarkdown(w io.Writer, key string) error {
	dc.mu.RLock()
	doc, exists := dc.docCache[key]
	dc.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("documentation not found: %s", key)
	}
	
	if doc.Markdown == "" {
		return fmt.Errorf("markdown not available for: %s", key)
	}
	
	_, err := w.Write([]byte(doc.Markdown))
	return err
}

// ExportJSON exports documentation as JSON
func (dc *DocCache) ExportJSON(w io.Writer, key string) error {
	dc.mu.RLock()
	doc, exists := dc.docCache[key]
	dc.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("documentation not found: %s", key)
	}
	
	if doc.JSON == "" {
		return fmt.Errorf("JSON not available for: %s", key)
	}
	
	_, err := w.Write([]byte(doc.JSON))
	return err
}