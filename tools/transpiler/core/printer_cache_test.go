// Package core provides tests for printer cache with template pre-compilation.
package core

import (
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
	"sync"
	"testing"
	"text/template"
)

func TestNewPrinterCache(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		pc := NewPrinterCache(nil)
		if pc == nil {
			t.Fatal("Expected non-nil printer cache")
		}
		if pc.config == nil {
			t.Error("Expected non-nil config")
		}
		if !pc.config.EnableMetrics {
			t.Error("Expected metrics to be enabled by default")
		}
		if !pc.config.PrecompileTemplates {
			t.Error("Expected template precompilation to be enabled by default")
		}
	})
	
	t.Run("with custom config", func(t *testing.T) {
		config := &PrinterCacheConfig{
			Mode:                printer.RawFormat,
			Tabwidth:            4,
			PrecompileTemplates: false,
			MaxTemplates:        50,
			BufferPoolSize:      25,
			EnableMetrics:       false,
		}
		pc := NewPrinterCache(config)
		if pc == nil {
			t.Fatal("Expected non-nil printer cache")
		}
		if pc.config.Tabwidth != 4 {
			t.Errorf("Expected tabwidth 4, got %d", pc.config.Tabwidth)
		}
		if pc.config.MaxTemplates != 50 {
			t.Errorf("Expected max templates 50, got %d", pc.config.MaxTemplates)
		}
	})
}

func TestCompileTemplate(t *testing.T) {
	pc := NewPrinterCache(nil)
	
	t.Run("compile simple template", func(t *testing.T) {
		err := pc.CompileTemplate("simple", "Hello {{.Name}}!")
		if err != nil {
			t.Fatalf("Failed to compile template: %v", err)
		}
		
		// Check template was stored
		pc.mu.RLock()
		tmpl, exists := pc.templates["simple"]
		pc.mu.RUnlock()
		
		if !exists {
			t.Error("Template was not stored")
		}
		if tmpl.Name != "simple" {
			t.Errorf("Expected template name 'simple', got %s", tmpl.Name)
		}
		if tmpl.Uses != 0 {
			t.Errorf("Expected 0 uses, got %d", tmpl.Uses)
		}
	})
	
	t.Run("compile template with functions", func(t *testing.T) {
		err := pc.CompileTemplate("func", "{{title .name}}")
		if err != nil {
			t.Fatalf("Failed to compile template with function: %v", err)
		}
		
		// Test execution
		result, err := pc.ExecuteTemplate("func", map[string]string{"name": "test"})
		if err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}
		if result != "Test" {
			t.Errorf("Expected 'Test', got %s", result)
		}
	})
	
	t.Run("compile invalid template", func(t *testing.T) {
		err := pc.CompileTemplate("invalid", "{{.Name")
		if err == nil {
			t.Error("Expected error for invalid template")
		}
		if !strings.Contains(err.Error(), "failed to compile template") {
			t.Errorf("Expected compile error message, got: %v", err)
		}
	})
	
	t.Run("template limit enforcement", func(t *testing.T) {
		pc := NewPrinterCache(&PrinterCacheConfig{
			MaxTemplates: 3,
		})
		
		// Add templates up to limit
		for i := 0; i < 5; i++ {
			name := strings.Repeat("t", i+1)
			err := pc.CompileTemplate(name, "Template {{.}}")
			if err != nil {
				t.Fatalf("Failed to compile template %s: %v", name, err)
			}
			
			// Use the template to set its usage count
			if i < 3 {
				for j := 0; j < i; j++ {
					_, _ = pc.GetTemplate(name)
				}
			}
		}
		
		// Should only have 3 templates
		pc.mu.RLock()
		count := len(pc.templates)
		pc.mu.RUnlock()
		
		if count != 3 {
			t.Errorf("Expected 3 templates (limit), got %d", count)
		}
	})
}

func TestPrinterCacheGetTemplate(t *testing.T) {
	pc := NewPrinterCache(nil)
	
	// Compile a template
	err := pc.CompileTemplate("test", "{{.}}")
	if err != nil {
		t.Fatalf("Failed to compile template: %v", err)
	}
	
	t.Run("get existing template", func(t *testing.T) {
		tmpl, err := pc.GetTemplate("test")
		if err != nil {
			t.Fatalf("Failed to get template: %v", err)
		}
		if tmpl == nil {
			t.Error("Expected non-nil template")
		}
		if tmpl.Name != "test" {
			t.Errorf("Expected template name 'test', got %s", tmpl.Name)
		}
		if tmpl.Uses != 1 {
			t.Errorf("Expected 1 use, got %d", tmpl.Uses)
		}
		
		// Check metrics
		if pc.hits != 1 {
			t.Errorf("Expected 1 hit, got %d", pc.hits)
		}
	})
	
	t.Run("get non-existent template", func(t *testing.T) {
		tmpl, err := pc.GetTemplate("nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent template")
		}
		if tmpl != nil {
			t.Error("Expected nil template")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
		
		// Check metrics
		if pc.misses != 1 {
			t.Errorf("Expected 1 miss, got %d", pc.misses)
		}
	})
}

func TestPrinterCacheExecuteTemplate(t *testing.T) {
	pc := NewPrinterCache(nil)
	
	// Compile various templates
	templates := map[string]string{
		"greeting":   "Hello, {{.Name}}!",
		"list":       "{{range .Items}}- {{.}}\n{{end}}",
		"condition":  "{{if .Show}}Visible{{else}}Hidden{{end}}",
		"functest":   "{{upper .Text}}",
	}
	
	for name, source := range templates {
		err := pc.CompileTemplate(name, source)
		if err != nil {
			t.Fatalf("Failed to compile template %s: %v", name, err)
		}
	}
	
	t.Run("execute greeting template", func(t *testing.T) {
		result, err := pc.ExecuteTemplate("greeting", map[string]string{
			"Name": "World",
		})
		if err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}
		if result != "Hello, World!" {
			t.Errorf("Expected 'Hello, World!', got %s", result)
		}
	})
	
	t.Run("execute list template", func(t *testing.T) {
		result, err := pc.ExecuteTemplate("list", map[string][]string{
			"Items": {"one", "two", "three"},
		})
		if err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}
		expected := "- one\n- two\n- three\n"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})
	
	t.Run("execute conditional template", func(t *testing.T) {
		result1, err := pc.ExecuteTemplate("condition", map[string]bool{
			"Show": true,
		})
		if err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}
		if result1 != "Visible" {
			t.Errorf("Expected 'Visible', got %s", result1)
		}
		
		result2, err := pc.ExecuteTemplate("condition", map[string]bool{
			"Show": false,
		})
		if err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}
		if result2 != "Hidden" {
			t.Errorf("Expected 'Hidden', got %s", result2)
		}
	})
	
	t.Run("execute template with function", func(t *testing.T) {
		result, err := pc.ExecuteTemplate("functest", map[string]string{
			"Text": "hello",
		})
		if err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}
		if result != "HELLO" {
			t.Errorf("Expected 'HELLO', got %s", result)
		}
	})
	
	t.Run("execute non-existent template", func(t *testing.T) {
		result, err := pc.ExecuteTemplate("nonexistent", nil)
		if err == nil {
			t.Error("Expected error for non-existent template")
		}
		if result != "" {
			t.Errorf("Expected empty result, got %s", result)
		}
	})
}

func TestPrintNode(t *testing.T) {
	pc := NewPrinterCache(nil)
	
	t.Run("print identifier", func(t *testing.T) {
		node := &ast.Ident{Name: "TestIdent"}
		result, err := pc.PrintNode(node)
		if err != nil {
			t.Fatalf("Failed to print node: %v", err)
		}
		if result != "TestIdent" {
			t.Errorf("Expected 'TestIdent', got %s", result)
		}
		
		// Check metrics
		if pc.printOps != 1 {
			t.Errorf("Expected 1 print operation, got %d", pc.printOps)
		}
	})
	
	t.Run("print basic literal", func(t *testing.T) {
		node := &ast.BasicLit{
			Kind:  token.STRING,
			Value: `"test string"`,
		}
		result, err := pc.PrintNode(node)
		if err != nil {
			t.Fatalf("Failed to print node: %v", err)
		}
		if result != `"test string"` {
			t.Errorf("Expected '\"test string\"', got %s", result)
		}
	})
}

func TestFormatNode(t *testing.T) {
	pc := NewPrinterCache(nil)
	
	t.Run("format expression", func(t *testing.T) {
		// Create a binary expression: a + b
		node := &ast.BinaryExpr{
			X:  &ast.Ident{Name: "a"},
			Op: token.ADD,
			Y:  &ast.Ident{Name: "b"},
		}
		
		result, err := pc.FormatNode(node)
		if err != nil {
			t.Fatalf("Failed to format node: %v", err)
		}
		if result != "a + b" {
			t.Errorf("Expected 'a + b', got %s", result)
		}
	})
}

func TestFormatSource(t *testing.T) {
	pc := NewPrinterCache(nil)
	
	t.Run("format valid source", func(t *testing.T) {
		source := []byte("package main\nfunc main(){fmt.Println(\"hello\")}")
		formatted, err := pc.FormatSource(source)
		if err != nil {
			t.Fatalf("Failed to format source: %v", err)
		}
		
		// Check that it was formatted (should have proper spacing)
		if !strings.Contains(string(formatted), "func main() {") {
			t.Error("Expected formatted source to have proper spacing")
		}
	})
	
	t.Run("format invalid source", func(t *testing.T) {
		source := []byte("package main\nfunc main() {")
		_, err := pc.FormatSource(source)
		if err == nil {
			t.Error("Expected error for invalid source")
		}
	})
}

func TestBatchPrint(t *testing.T) {
	pc := NewPrinterCache(nil)
	
	t.Run("batch print small set", func(t *testing.T) {
		nodes := []ast.Node{
			&ast.Ident{Name: "first"},
			&ast.Ident{Name: "second"},
			&ast.Ident{Name: "third"},
		}
		
		results, err := pc.BatchPrint(nodes)
		if err != nil {
			t.Fatalf("Failed to batch print: %v", err)
		}
		
		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}
		if results[0] != "first" {
			t.Errorf("Expected 'first', got %s", results[0])
		}
		if results[1] != "second" {
			t.Errorf("Expected 'second', got %s", results[1])
		}
		if results[2] != "third" {
			t.Errorf("Expected 'third', got %s", results[2])
		}
	})
	
	t.Run("batch print large set", func(t *testing.T) {
		// Create 20 nodes to trigger parallel processing
		nodes := make([]ast.Node, 20)
		for i := 0; i < 20; i++ {
			nodes[i] = &ast.Ident{Name: strings.Repeat("n", i+1)}
		}
		
		results, err := pc.BatchPrint(nodes)
		if err != nil {
			t.Fatalf("Failed to batch print large set: %v", err)
		}
		
		if len(results) != 20 {
			t.Errorf("Expected 20 results, got %d", len(results))
		}
		
		// Verify each result
		for i, result := range results {
			expected := strings.Repeat("n", i+1)
			if result != expected {
				t.Errorf("Result %d: expected %s, got %s", i, expected, result)
			}
		}
	})
}

func TestPrinterCacheGetStatistics(t *testing.T) {
	pc := NewPrinterCache(nil)
	
	// Perform some operations
	_ = pc.CompileTemplate("test", "{{.}}")
	_, _ = pc.GetTemplate("test")
	_, _ = pc.GetTemplate("nonexistent")
	_, _ = pc.PrintNode(&ast.Ident{Name: "test"})
	
	stats := pc.GetStatistics()
	
	// Check statistics
	if stats["template_count"].(int) != 1 {
		t.Errorf("Expected 1 template, got %v", stats["template_count"])
	}
	if stats["template_hits"].(int64) != 1 {
		t.Errorf("Expected 1 hit, got %v", stats["template_hits"])
	}
	if stats["template_misses"].(int64) != 1 {
		t.Errorf("Expected 1 miss, got %v", stats["template_misses"])
	}
	if stats["print_operations"].(int64) != 1 {
		t.Errorf("Expected 1 print operation, got %v", stats["print_operations"])
	}
	
	hitRate := stats["hit_rate"].(float64)
	if hitRate != 50.0 {
		t.Errorf("Expected 50%% hit rate, got %.2f%%", hitRate)
	}
}

func TestPrinterCacheClear(t *testing.T) {
	pc := NewPrinterCache(nil)
	
	// Add templates and perform operations
	_ = pc.CompileTemplate("test1", "{{.}}")
	_ = pc.CompileTemplate("test2", "{{.}}")
	_, _ = pc.GetTemplate("test1")
	_, _ = pc.PrintNode(&ast.Ident{Name: "test"})
	
	// Clear cache
	pc.Clear()
	
	// Check everything was cleared
	pc.mu.RLock()
	templateCount := len(pc.templates)
	pc.mu.RUnlock()
	
	if templateCount != 0 {
		t.Errorf("Expected 0 templates after clear, got %d", templateCount)
	}
	if pc.hits != 0 {
		t.Errorf("Expected 0 hits after clear, got %d", pc.hits)
	}
	if pc.misses != 0 {
		t.Errorf("Expected 0 misses after clear, got %d", pc.misses)
	}
	if pc.printOps != 0 {
		t.Errorf("Expected 0 print ops after clear, got %d", pc.printOps)
	}
}

func TestTemplateFunctions(t *testing.T) {
	tests := []struct {
		name     string
		function func(string) string
		input    string
		expected string
	}{
		{"title", stringTitle, "hello", "Hello"},
		{"title empty", stringTitle, "", ""},
		{"lower", stringLower, "HELLO", "hello"},
		{"upper", stringUpper, "hello", "HELLO"},
		{"capitalize", stringCapitalize, "hELLO", "Hello"},
		{"capitalize empty", stringCapitalize, "", ""},
		{"comment", makeComment, "test comment", "// test comment"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
	
	t.Run("indent", func(t *testing.T) {
		input := "line1\nline2\nline3"
		result := stringIndent(2, input)
		expected := "\t\tline1\n\t\tline2\n\t\tline3"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})
	
	t.Run("dedent", func(t *testing.T) {
		input := "\t\tline1\n\t\tline2\n\t\t\tline3"
		result := stringDedent(input)
		expected := "line1\nline2\n\tline3"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})
	
	t.Run("multiline", func(t *testing.T) {
		input := "line1\nline2\nline3"
		result := makeMultiline(input)
		expected := "/*\n * line1\n * line2\n * line3\n */"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})
}

func TestPrinterCacheConcurrentAccess(t *testing.T) {
	pc := NewPrinterCache(nil)
	
	// Pre-compile some templates
	for i := 0; i < 10; i++ {
		name := strings.Repeat("t", i+1)
		_ = pc.CompileTemplate(name, "Template {{.}}")
	}
	
	var wg sync.WaitGroup
	
	// Concurrent template compilation
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			name := strings.Repeat("c", i+1)
			_ = pc.CompileTemplate(name, "{{.}}")
		}
	}()
	
	// Concurrent template execution
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_, _ = pc.ExecuteTemplate("t", i)
		}
	}()
	
	// Concurrent printing
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_, _ = pc.PrintNode(&ast.Ident{Name: "test"})
		}
	}()
	
	// Concurrent statistics reading
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = pc.GetStatistics()
		}
	}()
	
	wg.Wait()
	
	// Verify operations completed
	stats := pc.GetStatistics()
	if stats["print_operations"].(int64) < 100 {
		t.Error("Expected at least 100 print operations")
	}
}

func TestCustomTemplateFuncs(t *testing.T) {
	customFuncs := template.FuncMap{
		"double": func(n int) int { return n * 2 },
		"concat": func(a, b string) string { return a + b },
	}
	
	config := &PrinterCacheConfig{
		TemplateFuncs: customFuncs,
	}
	pc := NewPrinterCache(config)
	
	t.Run("use custom function", func(t *testing.T) {
		err := pc.CompileTemplate("custom", "{{double .Value}}")
		if err != nil {
			t.Fatalf("Failed to compile template with custom function: %v", err)
		}
		
		result, err := pc.ExecuteTemplate("custom", map[string]int{
			"Value": 5,
		})
		if err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}
		if result != "10" {
			t.Errorf("Expected '10', got %s", result)
		}
	})
}

func BenchmarkCompileTemplate(b *testing.B) {
	pc := NewPrinterCache(nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := strings.Repeat("t", i%100+1)
		_ = pc.CompileTemplate(name, "Template {{.Value}}")
	}
}

func BenchmarkPrinterCacheExecuteTemplate(b *testing.B) {
	pc := NewPrinterCache(nil)
	_ = pc.CompileTemplate("bench", "Hello {{.Name}}!")
	
	data := map[string]string{"Name": "World"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pc.ExecuteTemplate("bench", data)
	}
}

func BenchmarkPrintNode(b *testing.B) {
	pc := NewPrinterCache(nil)
	node := &ast.Ident{Name: "BenchmarkIdent"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pc.PrintNode(node)
	}
}

func BenchmarkBatchPrint(b *testing.B) {
	pc := NewPrinterCache(nil)
	
	// Create nodes for batch printing
	nodes := make([]ast.Node, 20)
	for i := 0; i < 20; i++ {
		nodes[i] = &ast.Ident{Name: strings.Repeat("n", i+1)}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pc.BatchPrint(nodes)
	}
}