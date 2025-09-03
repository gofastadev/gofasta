// Package core provides template-based code generation with pre-compilation.
// This implements Phase 1.3c: Implement template-based code generation with pre-compilation.
package core

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"
	"time"
)

// CodeGenerator provides template-based code generation with pre-compilation
type CodeGenerator struct {
	config    *GeneratorConfig
	templates map[string]*GeneratorTemplate
	funcs     template.FuncMap
	mu        sync.RWMutex

	// Metrics
	generations    int64
	compilations   int64
	cacheHits      int64
	cacheMisses    int64
	totalDuration  int64
}

// GeneratorConfig contains configuration for code generator
type GeneratorConfig struct {
	// Template settings
	TemplateDir        string
	PrecompileAll      bool
	EnableCache        bool
	CacheTTL           time.Duration
	StrictMode         bool
	
	// Output settings
	FormatOutput       bool
	AddHeaders         bool
	HeaderTemplate     string
	IndentStyle        string // "tabs" or "spaces"
	IndentSize         int
	
	// Performance settings
	ConcurrentGenerate bool
	WorkerCount        int
	BufferSize         int
	EnableMetrics      bool
}

// GeneratorTemplate represents a pre-compiled generator template
type GeneratorTemplate struct {
	Name         string
	Source       string
	Template     *template.Template
	CompiledAt   time.Time
	LastUsed     time.Time
	UseCount     int64
	Dependencies []string
	Metadata     map[string]interface{}
}

// GenerationContext contains context for code generation
type GenerationContext struct {
	PackageName  string
	Imports      []string
	Decorators   []Decorator
	Types        []TypeDefinition
	Functions    []FunctionDefinition
	Variables    []VariableDefinition
	Constants    []ConstantDefinition
	Metadata     map[string]interface{}
}

// TypeDefinition represents a type in generated code
type TypeDefinition struct {
	Name       string
	Kind       string // "struct", "interface", "alias", "enum"
	Fields     []FieldDefinition
	Methods    []MethodDefinition
	Embedded   []string
	Tags       map[string]string
	Doc        string
	Decorators []Decorator
}

// FieldDefinition represents a field in a struct
type FieldDefinition struct {
	Name       string
	Type       string
	Tag        string
	Doc        string
	Decorators []Decorator
	Optional   bool
	Default    interface{}
}

// MethodDefinition represents a method
type MethodDefinition struct {
	Name       string
	Receiver   string
	Parameters []ParameterDefinition
	Returns    []string
	Body       string
	Doc        string
	Decorators []Decorator
}

// FunctionDefinition represents a function
type FunctionDefinition struct {
	Name       string
	Parameters []ParameterDefinition
	Returns    []string
	Body       string
	Doc        string
	Decorators []Decorator
	Async      bool
}

// ParameterDefinition represents a function parameter
type ParameterDefinition struct {
	Name     string
	Type     string
	Variadic bool
	Optional bool
	Default  interface{}
}

// VariableDefinition represents a variable
type VariableDefinition struct {
	Name  string
	Type  string
	Value interface{}
	Doc   string
}

// ConstantDefinition represents a constant
type ConstantDefinition struct {
	Name  string
	Type  string
	Value interface{}
	Doc   string
}

// GenerationResult contains the result of code generation
type GenerationResult struct {
	Code       string
	Formatted  bool
	Duration   time.Duration
	Template   string
	Warnings   []string
}

// Built-in templates
var builtinTemplates = map[string]string{
	"struct": `{{if .Doc}}// {{.Doc}}
{{end}}{{range .Decorators}}// {{.Raw}}
{{end}}type {{.Name}} struct {
{{range .Fields}}{{if .Doc}}	// {{.Doc}}
{{end}}{{range .Decorators}}	// {{.Raw}}
{{end}}	{{.Name}} {{.Type}}{{if .Tag}} ` + "`{{.Tag}}`" + `{{end}}
{{end}}}
`,

	"interface": `{{if .Doc}}// {{.Doc}}
{{end}}type {{.Name}} interface {
{{range .Methods}}	{{.Name}}({{range $i, $p := .Parameters}}{{if $i}}, {{end}}{{$p.Name}} {{$p.Type}}{{end}}){{if .Returns}} ({{range $i, $r := .Returns}}{{if $i}}, {{end}}{{$r}}{{end}}){{end}}
{{end}}}
`,

	"function": `{{if .Doc}}// {{.Doc}}
{{end}}{{range .Decorators}}// {{.Raw}}
{{end}}func {{.Name}}({{range $i, $p := .Parameters}}{{if $i}}, {{end}}{{$p.Name}} {{$p.Type}}{{end}}){{if .Returns}} ({{range $i, $r := .Returns}}{{if $i}}, {{end}}{{$r}}{{end}}){{end}} {
{{.Body}}
}
`,

	"method": `{{if .Doc}}// {{.Doc}}
{{end}}{{range .Decorators}}// {{.Raw}}
{{end}}func ({{.Receiver}}) {{.Name}}({{range $i, $p := .Parameters}}{{if $i}}, {{end}}{{$p.Name}} {{$p.Type}}{{end}}){{if .Returns}} ({{range $i, $r := .Returns}}{{if $i}}, {{end}}{{$r}}{{end}}){{end}} {
{{.Body}}
}
`,

	"package": `{{if .HeaderTemplate}}{{.HeaderTemplate}}
{{end}}package {{.PackageName}}

{{if .Imports}}import (
{{range .Imports}}	"{{.}}"
{{end}})
{{end}}

{{range .Constants}}const {{.Name}}{{if .Type}} {{.Type}}{{end}} = {{.Value}}
{{end}}
{{range .Variables}}var {{.Name}}{{if .Type}} {{.Type}}{{end}}{{if .Value}} = {{.Value}}{{end}}
{{end}}
{{range .Types}}{{template "type" .}}
{{end}}
{{range .Functions}}{{template "function" .}}
{{end}}`,

	"rest_controller": `package {{.PackageName}}

import (
	"encoding/json"
	"net/http"
)

{{range .Types}}
// {{.Name}}Controller handles {{.Name}} REST endpoints
type {{.Name}}Controller struct {
	service *{{.Name}}Service
}

// New{{.Name}}Controller creates a new controller
func New{{.Name}}Controller(service *{{.Name}}Service) *{{.Name}}Controller {
	return &{{.Name}}Controller{service: service}
}

{{range .Methods}}
// {{.Name}} {{.Doc}}
{{range .Decorators}}// {{.Raw}}
{{end}}func (c *{{$.Name}}Controller) {{.Name}}(w http.ResponseWriter, r *http.Request) {
	// Implementation
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
{{end}}
{{end}}`,
}

// DefaultGeneratorConfig returns the default configuration
func DefaultGeneratorConfig() *GeneratorConfig {
	return &GeneratorConfig{
		TemplateDir:        "./templates",
		PrecompileAll:      true,
		EnableCache:        true,
		CacheTTL:           5 * time.Minute,
		StrictMode:         false,
		FormatOutput:       true,
		AddHeaders:         true,
		HeaderTemplate:     "// Code generated by GoFasta. DO NOT EDIT.",
		IndentStyle:        "tabs",
		IndentSize:         4,
		ConcurrentGenerate: true,
		WorkerCount:        4,
		BufferSize:         4096,
		EnableMetrics:      true,
	}
}

// NewCodeGenerator creates a new code generator
func NewCodeGenerator(config *GeneratorConfig) *CodeGenerator {
	if config == nil {
		config = DefaultGeneratorConfig()
	}

	cg := &CodeGenerator{
		config:    config,
		templates: make(map[string]*GeneratorTemplate),
		funcs:     createTemplateFuncs(),
	}

	// Register built-in templates
	cg.registerBuiltinTemplates()

	// Precompile if configured
	if config.PrecompileAll {
		cg.precompileAll()
	}

	return cg
}

// createTemplateFuncs creates template helper functions
func createTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		// String functions
		"lower":      strings.ToLower,
		"upper":      strings.ToUpper,
		"title":      strings.Title,
		"camelCase":  toCamelCase,
		"snakeCase":  toSnakeCase,
		"kebabCase":  toKebabCase,
		"plural":     toPlural,
		"singular":   toSingular,
		
		// Type functions
		"isPointer":  isPointer,
		"isSlice":    isSlice,
		"isMap":      isMap,
		"baseType":   getBaseType,
		
		// Code generation helpers
		"indent":     indentCode,
		"comment":    comment,
		"imports":    generateImports,
		"tags":       generateTags,
		
		// Control flow
		"first":      firstItem,
		"last":       lastItem,
		"contains":   contains,
		"join":       strings.Join,
	}
}

// registerBuiltinTemplates registers built-in templates
func (cg *CodeGenerator) registerBuiltinTemplates() {
	for name, source := range builtinTemplates {
		cg.AddTemplate(name, source, nil)
	}
}

// precompileAll precompiles all templates
func (cg *CodeGenerator) precompileAll() {
	cg.mu.Lock()
	defer cg.mu.Unlock()

	for _, tmpl := range cg.templates {
		if tmpl.Template == nil {
			cg.compileTemplate(tmpl)
		}
	}
}

// AddTemplate adds a new template
func (cg *CodeGenerator) AddTemplate(name, source string, metadata map[string]interface{}) error {
	cg.mu.Lock()
	defer cg.mu.Unlock()

	tmpl := &GeneratorTemplate{
		Name:       name,
		Source:     source,
		Metadata:   metadata,
		CompiledAt: time.Now(),
	}

	// Compile if precompile is enabled
	if cg.config.PrecompileAll {
		if err := cg.compileTemplate(tmpl); err != nil {
			return err
		}
	}

	cg.templates[name] = tmpl
	return nil
}

// compileTemplate compiles a template
func (cg *CodeGenerator) compileTemplate(gt *GeneratorTemplate) error {
	tmpl := template.New(gt.Name).Funcs(cg.funcs)

	if cg.config.StrictMode {
		tmpl = tmpl.Option("missingkey=error")
	} else {
		tmpl = tmpl.Option("missingkey=default")
	}

	// Parse the template
	parsed, err := tmpl.Parse(gt.Source)
	if err != nil {
		return fmt.Errorf("failed to compile template %s: %w", gt.Name, err)
	}

	// Add sub-templates
	for name, source := range builtinTemplates {
		if name != gt.Name {
			_, err = parsed.New(name).Parse(source)
			if err != nil {
				return fmt.Errorf("failed to add sub-template %s: %w", name, err)
			}
		}
	}

	gt.Template = parsed
	gt.CompiledAt = time.Now()
	atomic.AddInt64(&cg.compilations, 1)

	return nil
}

// Generate generates code using a template
func (cg *CodeGenerator) Generate(templateName string, context interface{}) (*GenerationResult, error) {
	start := time.Now()

	// Get template
	tmpl, err := cg.getTemplate(templateName)
	if err != nil {
		return nil, err
	}

	// Generate code
	var buf bytes.Buffer
	if err := tmpl.Template.Execute(&buf, context); err != nil {
		return nil, fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	code := buf.String()

	// Format if configured
	if cg.config.FormatOutput {
		if formatted, err := format.Source([]byte(code)); err == nil {
			code = string(formatted)
		}
	}

	// Add header if configured
	if cg.config.AddHeaders && cg.config.HeaderTemplate != "" {
		code = cg.config.HeaderTemplate + "\n\n" + code
	}

	// Update metrics
	atomic.AddInt64(&cg.generations, 1)
	duration := time.Since(start)
	atomic.AddInt64(&cg.totalDuration, int64(duration))
	atomic.AddInt64(&tmpl.UseCount, 1)
	tmpl.LastUsed = time.Now()

	return &GenerationResult{
		Code:      code,
		Formatted: cg.config.FormatOutput,
		Duration:  duration,
		Template:  templateName,
	}, nil
}

// GenerateStruct generates a struct
func (cg *CodeGenerator) GenerateStruct(def TypeDefinition) (string, error) {
	result, err := cg.Generate("struct", def)
	if err != nil {
		return "", err
	}
	return result.Code, nil
}

// GenerateInterface generates an interface
func (cg *CodeGenerator) GenerateInterface(def TypeDefinition) (string, error) {
	result, err := cg.Generate("interface", def)
	if err != nil {
		return "", err
	}
	return result.Code, nil
}

// GenerateFunction generates a function
func (cg *CodeGenerator) GenerateFunction(def FunctionDefinition) (string, error) {
	result, err := cg.Generate("function", def)
	if err != nil {
		return "", err
	}
	return result.Code, nil
}

// GeneratePackage generates a complete package
func (cg *CodeGenerator) GeneratePackage(ctx GenerationContext) (string, error) {
	// Add header template to context
	if cg.config.HeaderTemplate != "" {
		if ctx.Metadata == nil {
			ctx.Metadata = make(map[string]interface{})
		}
		ctx.Metadata["HeaderTemplate"] = cg.config.HeaderTemplate
	}

	result, err := cg.Generate("package", ctx)
	if err != nil {
		return "", err
	}
	return result.Code, nil
}

// GenerateRESTController generates a REST controller
func (cg *CodeGenerator) GenerateRESTController(ctx GenerationContext) (string, error) {
	result, err := cg.Generate("rest_controller", ctx)
	if err != nil {
		return "", err
	}
	return result.Code, nil
}

// GenerateToWriter generates code to a writer
func (cg *CodeGenerator) GenerateToWriter(templateName string, context interface{}, w io.Writer) error {
	result, err := cg.Generate(templateName, context)
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(result.Code))
	return err
}

// GenerateBatch generates multiple files concurrently
func (cg *CodeGenerator) GenerateBatch(requests map[string]GenerationRequest) (map[string]*GenerationResult, error) {
	if !cg.config.ConcurrentGenerate || len(requests) <= 1 {
		// Generate sequentially
		results := make(map[string]*GenerationResult)
		for name, req := range requests {
			result, err := cg.Generate(req.Template, req.Context)
			if err != nil {
				return results, err
			}
			results[name] = result
		}
		return results, nil
	}

	// Generate concurrently
	type workResult struct {
		name   string
		result *GenerationResult
		err    error
	}

	resultChan := make(chan workResult, len(requests))
	sem := make(chan struct{}, cg.config.WorkerCount)

	var wg sync.WaitGroup
	for name, req := range requests {
		wg.Add(1)
		go func(n string, r GenerationRequest) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := cg.Generate(r.Template, r.Context)
			resultChan <- workResult{name: n, result: result, err: err}
		}(name, req)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	results := make(map[string]*GenerationResult)
	for work := range resultChan {
		if work.err != nil {
			return results, work.err
		}
		results[work.name] = work.result
	}

	return results, nil
}

// GenerationRequest represents a generation request
type GenerationRequest struct {
	Template string
	Context  interface{}
}

// getTemplate retrieves a template
func (cg *CodeGenerator) getTemplate(name string) (*GeneratorTemplate, error) {
	cg.mu.RLock()
	tmpl, exists := cg.templates[name]
	cg.mu.RUnlock()

	if !exists {
		atomic.AddInt64(&cg.cacheMisses, 1)
		return nil, fmt.Errorf("template %s not found", name)
	}

	// Compile if not yet compiled
	if tmpl.Template == nil {
		cg.mu.Lock()
		if tmpl.Template == nil {
			if err := cg.compileTemplate(tmpl); err != nil {
				cg.mu.Unlock()
				return nil, err
			}
		}
		cg.mu.Unlock()
	}

	atomic.AddInt64(&cg.cacheHits, 1)
	return tmpl, nil
}

// LoadTemplateFromFile loads a template from a file
func (cg *CodeGenerator) LoadTemplateFromFile(name, path string) error {
	content, err := readFile(path)
	if err != nil {
		return fmt.Errorf("failed to read template file %s: %w", path, err)
	}

	metadata := map[string]interface{}{
		"source_file": path,
		"loaded_at":   time.Now(),
	}

	return cg.AddTemplate(name, string(content), metadata)
}

// LoadTemplatesFromDir loads all templates from a directory
func (cg *CodeGenerator) LoadTemplatesFromDir(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.tmpl"))
	if err != nil {
		return err
	}

	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".tmpl")
		if err := cg.LoadTemplateFromFile(name, file); err != nil {
			return err
		}
	}

	return nil
}

// GetStatistics returns generator statistics
func (cg *CodeGenerator) GetStatistics() map[string]interface{} {
	cg.mu.RLock()
	templateCount := len(cg.templates)
	compiledCount := 0
	for _, tmpl := range cg.templates {
		if tmpl.Template != nil {
			compiledCount++
		}
	}
	cg.mu.RUnlock()

	generations := atomic.LoadInt64(&cg.generations)
	avgDuration := time.Duration(0)
	if generations > 0 {
		avgDuration = time.Duration(atomic.LoadInt64(&cg.totalDuration) / generations)
	}

	hits := atomic.LoadInt64(&cg.cacheHits)
	misses := atomic.LoadInt64(&cg.cacheMisses)
	hitRate := float64(0)
	if total := hits + misses; total > 0 {
		hitRate = float64(hits) * 100.0 / float64(total)
	}

	return map[string]interface{}{
		"template_count":    templateCount,
		"compiled_count":    compiledCount,
		"generations":       generations,
		"compilations":      atomic.LoadInt64(&cg.compilations),
		"cache_hits":        hits,
		"cache_misses":      misses,
		"cache_hit_rate":    hitRate,
		"avg_duration":      avgDuration.String(),
		"avg_duration_ns":   avgDuration.Nanoseconds(),
	}
}

// Clear clears all templates and resets statistics
func (cg *CodeGenerator) Clear() {
	cg.mu.Lock()
	defer cg.mu.Unlock()

	cg.templates = make(map[string]*GeneratorTemplate)
	atomic.StoreInt64(&cg.generations, 0)
	atomic.StoreInt64(&cg.compilations, 0)
	atomic.StoreInt64(&cg.cacheHits, 0)
	atomic.StoreInt64(&cg.cacheMisses, 0)
	atomic.StoreInt64(&cg.totalDuration, 0)

	// Re-register built-in templates
	cg.registerBuiltinTemplates()
}

// Helper functions for templates

func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		if i == 0 {
			parts[i] = strings.ToLower(parts[i])
		} else {
			parts[i] = strings.Title(parts[i])
		}
	}
	return strings.Join(parts, "")
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func toKebabCase(s string) string {
	return strings.ReplaceAll(toSnakeCase(s), "_", "-")
}

func toPlural(s string) string {
	if strings.HasSuffix(s, "y") {
		return s[:len(s)-1] + "ies"
	}
	if strings.HasSuffix(s, "s") {
		return s + "es"
	}
	return s + "s"
}

func toSingular(s string) string {
	if strings.HasSuffix(s, "ies") {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "es") {
		return s[:len(s)-2]
	}
	if strings.HasSuffix(s, "s") {
		return s[:len(s)-1]
	}
	return s
}

func isPointer(t string) bool {
	return strings.HasPrefix(t, "*")
}

func isSlice(t string) bool {
	return strings.HasPrefix(t, "[]")
}

func isMap(t string) bool {
	return strings.HasPrefix(t, "map[")
}

func getBaseType(t string) string {
	t = strings.TrimPrefix(t, "*")
	t = strings.TrimPrefix(t, "[]")
	if idx := strings.Index(t, "map["); idx == 0 {
		if end := strings.Index(t, "]"); end > 0 {
			t = t[end+1:]
		}
	}
	return t
}

func indentCode(n int, s string) string {
	indentStr := strings.Repeat("\t", n)
	lines := strings.Split(s, "\n")
	for i := range lines {
		if lines[i] != "" {
			lines[i] = indentStr + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}

func comment(s string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = "// " + lines[i]
	}
	return strings.Join(lines, "\n")
}

func generateImports(imports []string) string {
	if len(imports) == 0 {
		return ""
	}
	var buf strings.Builder
	buf.WriteString("import (\n")
	for _, imp := range imports {
		buf.WriteString(fmt.Sprintf("\t\"%s\"\n", imp))
	}
	buf.WriteString(")\n")
	return buf.String()
}

func generateTags(tags map[string]string) string {
	if len(tags) == 0 {
		return ""
	}
	var parts []string
	for key, value := range tags {
		parts = append(parts, fmt.Sprintf(`%s:"%s"`, key, value))
	}
	return "`" + strings.Join(parts, " ") + "`"
}

func firstItem(slice []interface{}) interface{} {
	if len(slice) > 0 {
		return slice[0]
	}
	return nil
}

func lastItem(slice []interface{}) interface{} {
	if len(slice) > 0 {
		return slice[len(slice)-1]
	}
	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func readFile(path string) ([]byte, error) {
	// This is a placeholder - in real implementation would use os.ReadFile
	return []byte(""), fmt.Errorf("file reading not implemented in this example")
}
