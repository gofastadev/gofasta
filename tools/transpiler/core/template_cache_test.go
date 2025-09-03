// Package core provides text/template with compiled template cache - test file.
// This implements tests for Phase 1.2b: text/template with compiled template cache.
package core

import (
	"strings"
	"sync"
	"testing"
	"text/template"
	"time"
)

// TestDefaultTemplateCacheConfig tests the default configuration
func TestDefaultTemplateCacheConfig(t *testing.T) {
	config := DefaultTemplateCacheConfig()
	
	if config == nil {
		t.Fatal("expected non-nil config")
	}
	if config.MaxTemplates != 500 {
		t.Errorf("expected MaxTemplates to be 500, got %d", config.MaxTemplates)
	}
	if config.MaxCacheSizeMB != 100 {
		t.Errorf("expected MaxCacheSizeMB to be 100, got %d", config.MaxCacheSizeMB)
	}
	if !config.EnableDedup {
		t.Error("expected EnableDedup to be true")
	}
	if !config.EnableMetrics {
		t.Error("expected EnableMetrics to be true")
	}
	if config.Delims[0] != "{{" || config.Delims[1] != "}}" {
		t.Errorf("expected default delimiters {{/}}, got %v", config.Delims)
	}
	if config.FuncMap == nil {
		t.Error("expected FuncMap to be non-nil")
	}
	if config.StrictMode {
		t.Error("expected StrictMode to be false")
	}
	if !config.PrecompileOnAdd {
		t.Error("expected PrecompileOnAdd to be true")
	}
	if config.LazyCompilation {
		t.Error("expected LazyCompilation to be false")
	}
	if !config.ConcurrentCompile {
		t.Error("expected ConcurrentCompile to be true")
	}
	if config.CompileWorkers != 4 {
		t.Errorf("expected CompileWorkers to be 4, got %d", config.CompileWorkers)
	}
}

// TestDefaultTemplateFuncMap tests the default template function map
func TestDefaultTemplateFuncMap(t *testing.T) {
	funcs := DefaultTemplateFuncMap()
	
	expectedFuncs := []string{
		"upper", "lower", "title", "trim", "replace", "contains",
		"hasPrefix", "hasSuffix", "split", "join", "sprintf",
		"quote", "unquote", "indent", "dedent", "wrap",
		"basename", "dirname", "ext", "clean", "abs",
		"default", "empty", "coalesce", "ternary",
		"first", "last", "reverse", "uniq", "dict", "list",
		"now", "date", "timestamp",
	}
	
	for _, name := range expectedFuncs {
		if _, exists := funcs[name]; !exists {
			t.Errorf("expected function %s to exist", name)
		}
	}
}

func TestNewTemplateCache(t *testing.T) {
	tests := []struct {
		name   string
		config *TemplateCacheConfig
	}{
		{
			name:   "with nil config",
			config: nil,
		},
		{
			name:   "with default config",
			config: DefaultTemplateCacheConfig(),
		},
		{
			name: "with custom config",
			config: &TemplateCacheConfig{
				MaxTemplates:      100,
				MaxCacheSizeMB:    50,
				EnableDedup:       false,
				EnableMetrics:     false,
				Delims:            [2]string{"[[", "]]"},
				FuncMap:           make(template.FuncMap),
				StrictMode:        true,
				EnableAutoReload:  false,
				PrecompileOnAdd:   false,
				LazyCompilation:   true,
				ConcurrentCompile: false,
				CompileWorkers:    2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := NewTemplateCache(tt.config)
			if tc == nil {
				t.Fatal("expected non-nil template cache")
			}
			if tc.config == nil {
				t.Fatal("expected config to be initialized")
			}
			if tc.templates == nil {
				t.Fatal("expected templates to be initialized")
			}
			if tc.hashIndex == nil {
				t.Fatal("expected hashIndex to be initialized")
			}
			if tc.sets == nil {
				t.Fatal("expected sets to be initialized")
			}
		})
	}
}

// TestAddTemplate tests adding templates to the cache
func TestAddTemplate(t *testing.T) {
	tc := NewTemplateCache(nil)
	
	tests := []struct {
		name       string
		tmplName   string
		tmplSource string
		wantErr    bool
	}{
		{
			name:       "simple template",
			tmplName:   "simple",
			tmplSource: "Hello {{.Name}}",
			wantErr:    false,
		},
		{
			name:       "template with functions",
			tmplName:   "with_funcs",
			tmplSource: "{{.Name | upper}}",
			wantErr:    false,
		},
		{
			name:       "complex template",
			tmplSource: "{{range .Items}}{{.Name | title}}: {{.Value}}\n{{end}}",
			tmplName:   "complex",
			wantErr:    false,
		},
		{
			name:       "invalid template",
			tmplName:   "invalid",
			tmplSource: "{{.Name | invalidFunc}}",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tc.AddTemplate(tt.tmplName, tt.tmplSource)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				// Verify template was stored
				tmpl, getErr := tc.GetTemplate(tt.tmplName)
				if getErr != nil {
					t.Errorf("GetTemplate() error = %v", getErr)
				}
				if tmpl.Name != tt.tmplName {
					t.Errorf("Template name = %q, want %q", tmpl.Name, tt.tmplName)
				}
				if tmpl.Source != tt.tmplSource {
					t.Errorf("Template source = %q, want %q", tmpl.Source, tt.tmplSource)
				}
			}
		})
	}
}

// TestAddTemplateWithMetadata tests adding templates with metadata
func TestAddTemplateWithMetadata(t *testing.T) {
	tc := NewTemplateCache(nil)
	
	metadata := map[string]interface{}{
		"author":  "test",
		"version": "1.0",
		"tags":    []string{"test", "example"},
	}
	
	err := tc.AddTemplateWithMetadata("meta_test", "Hello {{.Name}}", metadata)
	if err != nil {
		t.Fatalf("AddTemplateWithMetadata() error = %v", err)
	}
	
	tmpl, err := tc.GetTemplate("meta_test")
	if err != nil {
		t.Fatalf("GetTemplate() error = %v", err)
	}
	
	if tmpl.Metadata["author"] != "test" {
		t.Errorf("Metadata author = %v, want 'test'", tmpl.Metadata["author"])
	}
	if tmpl.Metadata["version"] != "1.0" {
		t.Errorf("Metadata version = %v, want '1.0'", tmpl.Metadata["version"])
	}
}

// TestTemplateEviction tests template eviction when limits are reached
func TestTemplateEviction(t *testing.T) {
	config := DefaultTemplateCacheConfig()
	config.MaxTemplates = 3
	tc := NewTemplateCache(config)
	
	// Add templates up to limit
	tc.AddTemplate("tmpl1", "Template 1: {{.Value}}")
	tc.AddTemplate("tmpl2", "Template 2: {{.Value}}")
	tc.AddTemplate("tmpl3", "Template 3: {{.Value}}")
	
	// Use tmpl1 to make it more recent
	tc.Execute("tmpl1", map[string]string{"Value": "test"})
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	
	// Use tmpl3 to make it most recent
	tc.Execute("tmpl3", map[string]string{"Value": "test"})
	
	// Add fourth template - should evict tmpl2 (least recently used)
	err := tc.AddTemplate("tmpl4", "Template 4: {{.Value}}")
	if err != nil {
		t.Fatalf("AddTemplate() error = %v", err)
	}
	
	// tmpl1 should still exist
	_, err = tc.GetTemplate("tmpl1")
	if err != nil {
		t.Error("tmpl1 should still exist after eviction")
	}
	
	// tmpl2 should be evicted
	_, err = tc.GetTemplate("tmpl2")
	if err == nil {
		t.Error("tmpl2 should have been evicted")
	}
	
	// tmpl3 should still exist
	_, err = tc.GetTemplate("tmpl3")
	if err != nil {
		t.Error("tmpl3 should still exist after eviction")
	}
	
	// tmpl4 should exist
	_, err = tc.GetTemplate("tmpl4")
	if err != nil {
		t.Error("tmpl4 should exist after addition")
	}
}

// TestTemplateDeduplication tests template deduplication
func TestTemplateDeduplication(t *testing.T) {
	t.Run("deduplication", func(t *testing.T) {
		tc := NewTemplateCache(&TemplateCacheConfig{
			EnableDedup: true,
		})

		// Add identical templates with different names
		err1 := tc.AddTemplate("tmpl1", "Same content {{.}}")
		err2 := tc.AddTemplate("tmpl2", "Same content {{.}}")

		if err1 != nil || err2 != nil {
			t.Fatalf("Failed to add templates: %v, %v", err1, err2)
		}

		tc.mu.RLock()
		tmpl1 := tc.templates["tmpl1"]
		tmpl2 := tc.templates["tmpl2"]
		tc.mu.RUnlock()

		// Should point to same template due to deduplication
		if tmpl1 != tmpl2 {
			t.Error("Expected templates to be deduplicated")
		}
	})

	t.Run("template limit enforcement", func(t *testing.T) {
		tc := NewTemplateCache(&TemplateCacheConfig{
			MaxTemplates: 3,
		})

		// Add templates up to and beyond limit
		for i := 0; i < 5; i++ {
			name := strings.Repeat("t", i+1)
			err := tc.AddTemplate(name, "Template {{.}}")
			if err != nil {
				t.Fatalf("Failed to add template %s: %v", name, err)
			}
		}

		tc.mu.RLock()
		count := len(tc.templates)
		tc.mu.RUnlock()

		if count > 3 {
			t.Errorf("Expected max 3 templates, got %d", count)
		}
	})
}

func TestTemplateCacheGetTemplate(t *testing.T) {
	tc := NewTemplateCache(nil)

	// Add a template
	err := tc.AddTemplate("test", "{{.Value}}")
	if err != nil {
		t.Fatalf("Failed to add template: %v", err)
	}

	t.Run("get existing template", func(t *testing.T) {
		tmpl, err := tc.GetTemplate("test")
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
		if tc.hits != 1 {
			t.Errorf("Expected 1 hit, got %d", tc.hits)
		}
	})

	t.Run("get non-existent template", func(t *testing.T) {
		tmpl, err := tc.GetTemplate("nonexistent")
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
		if tc.misses != 1 {
			t.Errorf("Expected 1 miss, got %d", tc.misses)
		}
	})

	t.Run("lazy compilation", func(t *testing.T) {
		tc := NewTemplateCache(&TemplateCacheConfig{
			PrecompileOnAdd: false,
			LazyCompilation: true,
		})

		err := tc.AddTemplate("lazy", "{{.Value}}")
		if err != nil {
			t.Fatalf("Failed to add template: %v", err)
		}

		// Template should not be compiled yet
		tc.mu.RLock()
		tmpl := tc.templates["lazy"]
		tc.mu.RUnlock()

		if tmpl.Template != nil {
			t.Error("Expected template to not be compiled yet")
		}

		// Getting template should trigger compilation
		compiled, err := tc.GetTemplate("lazy")
		if err != nil {
			t.Fatalf("Failed to get template: %v", err)
		}
		if compiled.Template == nil {
			t.Error("Expected template to be compiled after get")
		}
	})
}

func TestExecute(t *testing.T) {
	tc := NewTemplateCache(nil)

	// Add various templates
	templates := map[string]string{
		"greeting":  "Hello, {{.Name}}!",
		"list":      "{{range .Items}}- {{.}}\n{{end}}",
		"condition": "{{if .Show}}Visible{{else}}Hidden{{end}}",
		"functest":  "{{upper .Text}}",
	}

	for name, source := range templates {
		err := tc.AddTemplate(name, source)
		if err != nil {
			t.Fatalf("Failed to add template %s: %v", name, err)
		}
	}

	t.Run("execute greeting template", func(t *testing.T) {
		result, err := tc.Execute("greeting", map[string]string{
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
		result, err := tc.Execute("list", map[string][]string{
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
		result1, err := tc.Execute("condition", map[string]bool{
			"Show": true,
		})
		if err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}
		if result1 != "Visible" {
			t.Errorf("Expected 'Visible', got %s", result1)
		}

		result2, err := tc.Execute("condition", map[string]bool{
			"Show": false,
		})
		if err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}
		if result2 != "Hidden" {
			t.Errorf("Expected 'Hidden', got %s", result2)
		}
	})

	t.Run("execute with function", func(t *testing.T) {
		result, err := tc.Execute("functest", map[string]string{
			"Text": "hello",
		})
		if err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}
		if result != "HELLO" {
			t.Errorf("Expected 'HELLO', got %s", result)
		}
	})
}

func TestCreateSet(t *testing.T) {
	tc := NewTemplateCache(nil)

	t.Run("create template set", func(t *testing.T) {
		templates := map[string]string{
			"header": "<h1>{{.Title}}</h1>",
			"body":   "<p>{{.Content}}</p>",
			"footer": "<footer>{{.Copyright}}</footer>",
		}

		err := tc.CreateSet("page", templates)
		if err != nil {
			t.Fatalf("Failed to create template set: %v", err)
		}

		// Check set was created
		tc.mu.RLock()
		_, exists := tc.sets["page"]
		tc.mu.RUnlock()

		if !exists {
			t.Error("Template set was not created")
		}

		// Check individual templates were added
		for name := range templates {
			fullName := "page." + name
			tc.mu.RLock()
			_, exists := tc.templates[fullName]
			tc.mu.RUnlock()

			if !exists {
				t.Errorf("Expected template %s to exist", fullName)
			}
		}
	})

	t.Run("execute template from set", func(t *testing.T) {
		templates := map[string]string{
			"main": "Main: {{.Value}}",
			"sub":  "Sub: {{.Value}}",
		}

		err := tc.CreateSet("test", templates)
		if err != nil {
			t.Fatalf("Failed to create template set: %v", err)
		}

		result, err := tc.ExecuteSet("test", "main", map[string]string{
			"Value": "test",
		})
		if err != nil {
			t.Fatalf("Failed to execute template from set: %v", err)
		}
		if result != "Main: test" {
			t.Errorf("Expected 'Main: test', got %s", result)
		}
	})
}

func TestBatchCompile(t *testing.T) {
	t.Run("sequential compilation", func(t *testing.T) {
		tc := NewTemplateCache(&TemplateCacheConfig{
			ConcurrentCompile: false,
		})

		templates := map[string]string{
			"tmpl1": "Template 1: {{.}}",
			"tmpl2": "Template 2: {{.}}",
			"tmpl3": "Template 3: {{.}}",
		}

		err := tc.BatchCompile(templates)
		if err != nil {
			t.Fatalf("Failed to batch compile: %v", err)
		}

		// Check all templates were added
		for name := range templates {
			tc.mu.RLock()
			_, exists := tc.templates[name]
			tc.mu.RUnlock()

			if !exists {
				t.Errorf("Expected template %s to exist", name)
			}
		}
	})

	t.Run("concurrent compilation", func(t *testing.T) {
		tc := NewTemplateCache(&TemplateCacheConfig{
			ConcurrentCompile: true,
			CompileWorkers:    2,
		})

		templates := make(map[string]string)
		for i := 0; i < 10; i++ {
			name := strings.Repeat("t", i+1)
			templates[name] = "Template {{.}}"
		}

		err := tc.BatchCompile(templates)
		if err != nil {
			t.Fatalf("Failed to batch compile: %v", err)
		}

		// Check all templates were added
		tc.mu.RLock()
		count := len(tc.templates)
		tc.mu.RUnlock()

		if count != 10 {
			t.Errorf("Expected 10 templates, got %d", count)
		}
	})

	t.Run("batch compile with error", func(t *testing.T) {
		tc := NewTemplateCache(nil)

		templates := map[string]string{
			"valid":   "{{.Value}}",
			"invalid": "{{.Value",
		}

		err := tc.BatchCompile(templates)
		if err == nil {
			t.Error("Expected error for invalid template")
		}
		if !strings.Contains(err.Error(), "batch compile errors") {
			t.Errorf("Expected batch compile error message, got: %v", err)
		}
	})
}

func TestTemplateCacheGetStatistics(t *testing.T) {
	tc := NewTemplateCache(nil)

	// Perform various operations
	_ = tc.AddTemplate("test1", "{{.}}")
	_ = tc.AddTemplate("test2", "{{.}}")
	_, _ = tc.GetTemplate("test1")
	_, _ = tc.GetTemplate("nonexistent")
	_, _ = tc.Execute("test1", "data")

	stats := tc.GetStatistics()

	// Check statistics
	if stats["template_count"].(int) != 2 {
		t.Errorf("Expected 2 templates, got %v", stats["template_count"])
	}
	if stats["cache_hits"].(int64) < 1 {
		t.Errorf("Expected at least 1 hit, got %v", stats["cache_hits"])
	}
	if stats["cache_misses"].(int64) != 1 {
		t.Errorf("Expected 1 miss, got %v", stats["cache_misses"])
	}
	if stats["executions"].(int64) != 1 {
		t.Errorf("Expected 1 execution, got %v", stats["executions"])
	}

	hitRate := stats["hit_rate"].(float64)
	if hitRate < 50.0 {
		t.Errorf("Expected hit rate >= 50%%, got %.2f%%", hitRate)
	}
}

func TestTemplateCacheClear(t *testing.T) {
	tc := NewTemplateCache(nil)

	// Add templates and perform operations
	_ = tc.AddTemplate("test1", "{{.}}")
	_ = tc.AddTemplate("test2", "{{.}}")
	_ = tc.CreateSet("set1", map[string]string{"a": "{{.}}"})
	_, _ = tc.GetTemplate("test1")

	// Clear cache
	tc.Clear()

	// Check everything was cleared
	tc.mu.RLock()
	templateCount := len(tc.templates)
	setCount := len(tc.sets)
	tc.mu.RUnlock()

	if templateCount != 0 {
		t.Errorf("Expected 0 templates after clear, got %d", templateCount)
	}
	if setCount != 0 {
		t.Errorf("Expected 0 sets after clear, got %d", setCount)
	}
	if tc.hits != 0 {
		t.Errorf("Expected 0 hits after clear, got %d", tc.hits)
	}
	if tc.executions != 0 {
		t.Errorf("Expected 0 executions after clear, got %d", tc.executions)
	}
}

func TestTemplateCacheHelperFunctions(t *testing.T) {
	t.Run("string functions", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
			fn       func(string) string
		}{
			{"quote", "test", `"test"`, quote},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.fn(tt.input)
				if result != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result)
				}
			})
		}
	})

	t.Run("indent", func(t *testing.T) {
		input := "line1\nline2\nline3"
		result := indent(4, input)
		expected := "    line1\n    line2\n    line3"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("dedent", func(t *testing.T) {
		input := "    line1\n    line2\n        line3"
		result := dedent(input)
		expected := "line1\nline2\n    line3"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("wrap", func(t *testing.T) {
		input := "This is a long line that needs to be wrapped"
		result := wrap(20, input)
		lines := strings.Split(result, "\n")
		for _, line := range lines {
			if len(line) > 20 {
				t.Errorf("Line exceeds width 20: %q", line)
			}
		}
	})

	t.Run("logic functions", func(t *testing.T) {
		// Test defaultValue
		if defaultValue("default", "") != "default" {
			t.Error("Expected default value for empty string")
		}
		if defaultValue("default", "value") != "value" {
			t.Error("Expected actual value for non-empty string")
		}

		// Test isEmpty
		if !isEmpty(nil) {
			t.Error("Expected nil to be empty")
		}
		if !isEmpty("") {
			t.Error("Expected empty string to be empty")
		}
		if isEmpty("value") {
			t.Error("Expected non-empty string to not be empty")
		}

		// Test ternary
		if ternary(true, "yes", "no") != "yes" {
			t.Error("Expected 'yes' for true condition")
		}
		if ternary(false, "yes", "no") != "no" {
			t.Error("Expected 'no' for false condition")
		}
	})

	t.Run("collection functions", func(t *testing.T) {
		list := []interface{}{"a", "b", "c"}

		// Test first
		if first(list) != "a" {
			t.Error("Expected first element to be 'a'")
		}

		// Test last
		if last(list) != "c" {
			t.Error("Expected last element to be 'c'")
		}

		// Test reverse
		reversed := reverse(list)
		if reversed[0] != "c" || reversed[2] != "a" {
			t.Error("Expected reversed list")
		}

		// Test unique
		duplicates := []interface{}{"a", "b", "a", "c", "b"}
		uniq := unique(duplicates)
		if len(uniq) != 3 {
			t.Errorf("Expected 3 unique elements, got %d", len(uniq))
		}

		// Test dict
		d := dict("key1", "value1", "key2", "value2")
		if d["key1"] != "value1" {
			t.Error("Expected dict to contain key1=value1")
		}
	})
}

func TestTemplateCacheConcurrentAccess(t *testing.T) {
	tc := NewTemplateCache(nil)

	// Pre-add some templates
	for i := 0; i < 10; i++ {
		name := strings.Repeat("t", i+1)
		_ = tc.AddTemplate(name, "Template {{.}}")
	}

	var wg sync.WaitGroup

	// Concurrent template addition
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			name := strings.Repeat("a", i+1)
			_ = tc.AddTemplate(name, "{{.}}")
		}
	}()

	// Concurrent template execution
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_, _ = tc.Execute("t", i)
		}
	}()

	// Concurrent template retrieval
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_, _ = tc.GetTemplate("t")
		}
	}()

	// Concurrent statistics reading
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = tc.GetStatistics()
		}
	}()

	wg.Wait()

	// Verify operations completed
	stats := tc.GetStatistics()
	if stats["executions"].(int64) < 50 {
		t.Error("Expected at least 50 executions")
	}
}

func TestCustomDelimiters(t *testing.T) {
	tc := NewTemplateCache(&TemplateCacheConfig{
		Delims: [2]string{"[[", "]]"},
	})

	err := tc.AddTemplate("custom", "Hello [[.Name]]!")
	if err != nil {
		t.Fatalf("Failed to add template with custom delimiters: %v", err)
	}

	result, err := tc.Execute("custom", map[string]string{
		"Name": "World",
	})
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	if result != "Hello World!" {
		t.Errorf("Expected 'Hello World!', got %s", result)
	}
}

func TestCustomFunctions(t *testing.T) {
	customFuncs := template.FuncMap{
		"double": func(n int) int { return n * 2 },
		"concat": func(a, b string) string { return a + b },
	}

	tc := NewTemplateCache(&TemplateCacheConfig{
		FuncMap: customFuncs,
	})

	err := tc.AddTemplate("custom", "{{double .Value}}")
	if err != nil {
		t.Fatalf("Failed to add template with custom function: %v", err)
	}

	result, err := tc.Execute("custom", map[string]int{
		"Value": 5,
	})
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	if result != "10" {
		t.Errorf("Expected '10', got %s", result)
	}
}

func BenchmarkAddTemplate(b *testing.B) {
	tc := NewTemplateCache(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := strings.Repeat("t", i%100+1)
		_ = tc.AddTemplate(name, "Template {{.Value}}")
	}
}

func BenchmarkTemplateCacheExecuteTemplate(b *testing.B) {
	tc := NewTemplateCache(nil)
	_ = tc.AddTemplate("bench", "Hello {{.Name}}!")

	data := map[string]string{"Name": "World"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tc.Execute("bench", data)
	}
}

func BenchmarkBatchCompile(b *testing.B) {
	tc := NewTemplateCache(&TemplateCacheConfig{
		ConcurrentCompile: true,
		CompileWorkers:    4,
	})

	templates := make(map[string]string)
	for i := 0; i < 20; i++ {
		name := strings.Repeat("t", i+1)
		templates[name] = "Template {{.}}"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tc.BatchCompile(templates)
		tc.Clear() // Clear for next iteration
	}
}

func BenchmarkConcurrentExecute(b *testing.B) {
	tc := NewTemplateCache(nil)
	_ = tc.AddTemplate("bench", "{{.Value}}")

	b.RunParallel(func(pb *testing.PB) {
		data := map[string]int{"Value": 42}
		for pb.Next() {
			_, _ = tc.Execute("bench", data)
		}
	})
}

// TestUnquote tests string unquoting with escape handling
func TestUnquote(t *testing.T) {
	t.Run("simple quoted string", func(t *testing.T) {
		result, err := unquote(`"hello world"`)
		if err != nil {
			t.Fatalf("unquote() error = %v", err)
		}
		if result != "hello world" {
			t.Errorf("unquote() = %q, want %q", result, "hello world")
		}
	})
	
	t.Run("string without quotes", func(t *testing.T) {
		result, err := unquote("no quotes")
		if err != nil {
			t.Fatalf("unquote() error = %v", err)
		}
		if result != "no quotes" {
			t.Errorf("unquote() = %q, want %q", result, "no quotes")
		}
	})
	
	t.Run("empty string", func(t *testing.T) {
		result, err := unquote(`""`)
		if err != nil {
			t.Fatalf("unquote() error = %v", err)
		}
		if result != "" {
			t.Errorf("unquote() = %q, want %q", result, "")
		}
	})
	
	t.Run("string with single quote", func(t *testing.T) {
		result, err := unquote(`"single quote"`)
		if err != nil {
			t.Fatalf("unquote() error = %v", err)
		}
		if result != "single quote" {
			t.Errorf("unquote() = %q, want %q", result, "single quote")
		}
	})
	
	t.Run("string with escaped content", func(t *testing.T) {
		result, err := unquote(`"with \"escaped\" quotes"`)
		if err != nil {
			t.Fatalf("unquote() error = %v", err)
		}
		expected := `with \"escaped\" quotes`
		if result != expected {
			t.Errorf("unquote() = %q, want %q", result, expected)
		}
	})
}

// TestCoalesce tests value coalescing with nil/empty checks
func TestCoalesce(t *testing.T) {
	t.Run("first non-empty value", func(t *testing.T) {
		result := coalesce("", "hello", "world")
		if result != "hello" {
			t.Errorf("coalesce() = %v, want %q", result, "hello")
		}
	})
	
	t.Run("nil then valid value", func(t *testing.T) {
		result := coalesce(nil, "valid")
		if result != "valid" {
			t.Errorf("coalesce() = %v, want %q", result, "valid")
		}
	})
	
	t.Run("all empty returns nil", func(t *testing.T) {
		result := coalesce("", nil)
		if result != nil {
			t.Errorf("coalesce() = %v, want nil", result)
		}
	})
	
	t.Run("integer values", func(t *testing.T) {
		// Note: 0 is not considered empty by isEmpty(), so it should return 0
		result := coalesce(0, 42, 100)
		if result != 0 {
			t.Errorf("coalesce() = %v, want %d", result, 0)
		}
	})
	
	t.Run("mixed types", func(t *testing.T) {
		result := coalesce(nil, "", "string", 123)
		if result != "string" {
			t.Errorf("coalesce() = %v, want %q", result, "string")
		}
	})
	
	t.Run("boolean values", func(t *testing.T) {
		// Note: false is not considered empty by isEmpty(), so it should return false
		result := coalesce(false, true)
		if result != false {
			t.Errorf("coalesce() = %v, want false", result)
		}
	})
	
	t.Run("empty slice", func(t *testing.T) {
		result := coalesce([]interface{}{}, "fallback")
		if result != "fallback" {
			t.Errorf("coalesce() = %v, want %q", result, "fallback")
		}
	})
}

// TestListFunction tests list creation and manipulation
func TestListFunction(t *testing.T) {
	t.Run("create empty list", func(t *testing.T) {
		result := list()
		if len(result) != 0 {
			t.Errorf("list() length = %d, want 0", len(result))
		}
		// Note: list() with no args returns nil, which has length 0
		if result != nil && len(result) != 0 {
			t.Errorf("list() = %v, want nil or empty slice", result)
		}
	})
	
	t.Run("create list with single item", func(t *testing.T) {
		result := list("single")
		if len(result) != 1 {
			t.Errorf("list() length = %d, want 1", len(result))
		}
		if result[0] != "single" {
			t.Errorf("list()[0] = %v, want %q", result[0], "single")
		}
	})
	
	t.Run("create list with multiple items", func(t *testing.T) {
		result := list("first", 42, true, nil)
		if len(result) != 4 {
			t.Errorf("list() length = %d, want 4", len(result))
		}
		
		expected := []interface{}{"first", 42, true, nil}
		for i, exp := range expected {
			if result[i] != exp {
				t.Errorf("list()[%d] = %v, want %v", i, result[i], exp)
			}
		}
	})
	
	t.Run("create list with mixed types", func(t *testing.T) {
		result := list("string", 123, 45.67, []int{1, 2, 3})
		if len(result) != 4 {
			t.Errorf("list() length = %d, want 4", len(result))
		}
		
		if result[0] != "string" {
			t.Errorf("list()[0] = %v, want %q", result[0], "string")
		}
		if result[1] != 123 {
			t.Errorf("list()[1] = %v, want %d", result[1], 123)
		}
		if result[2] != 45.67 {
			t.Errorf("list()[2] = %v, want %f", result[2], 45.67)
		}
	})
	
	t.Run("list preserves order", func(t *testing.T) {
		items := []interface{}{1, 2, 3, 4, 5}
		result := list(items...)
		
		for i, item := range items {
			if result[i] != item {
				t.Errorf("list()[%d] = %v, want %v (order not preserved)", i, result[i], item)
			}
		}
	})
}

// TestFormatDate tests date formatting with various formats
func TestFormatDate(t *testing.T) {
	// Use a fixed time for consistent testing
	testTime := time.Date(2025, 9, 3, 23, 5, 30, 123456789, time.UTC)
	
	t.Run("RFC3339 format", func(t *testing.T) {
		result := formatDate(time.RFC3339, testTime)
		expected := "2025-09-03T23:05:30Z"
		if result != expected {
			t.Errorf("formatDate(RFC3339) = %q, want %q", result, expected)
		}
	})
	
	t.Run("kitchen format", func(t *testing.T) {
		result := formatDate(time.Kitchen, testTime)
		expected := "11:05PM"
		if result != expected {
			t.Errorf("formatDate(Kitchen) = %q, want %q", result, expected)
		}
	})
	
	t.Run("custom date format", func(t *testing.T) {
		result := formatDate("2006-01-02", testTime)
		expected := "2025-09-03"
		if result != expected {
			t.Errorf("formatDate(custom date) = %q, want %q", result, expected)
		}
	})
	
	t.Run("custom time format", func(t *testing.T) {
		result := formatDate("15:04:05", testTime)
		expected := "23:05:30"
		if result != expected {
			t.Errorf("formatDate(custom time) = %q, want %q", result, expected)
		}
	})
	
	t.Run("custom datetime format", func(t *testing.T) {
		result := formatDate("Jan 2, 2006 at 3:04pm", testTime)
		expected := "Sep 3, 2025 at 11:05pm"
		if result != expected {
			t.Errorf("formatDate(custom datetime) = %q, want %q", result, expected)
		}
	})
	
	t.Run("day of week format", func(t *testing.T) {
		result := formatDate("Monday", testTime)
		expected := "Wednesday" // September 3, 2025 is a Wednesday
		if result != expected {
			t.Errorf("formatDate(weekday) = %q, want %q", result, expected)
		}
	})
	
	t.Run("month format", func(t *testing.T) {
		result := formatDate("January", testTime)
		expected := "September"
		if result != expected {
			t.Errorf("formatDate(month) = %q, want %q", result, expected)
		}
	})
	
	t.Run("year format", func(t *testing.T) {
		result := formatDate("2006", testTime)
		expected := "2025"
		if result != expected {
			t.Errorf("formatDate(year) = %q, want %q", result, expected)
		}
	})
}

// TestTimestamp tests timestamp generation and formatting
func TestTimestamp(t *testing.T) {
	t.Run("returns current unix timestamp", func(t *testing.T) {
		before := time.Now().Unix()
		result := timestamp()
		after := time.Now().Unix()
		
		if result < before || result > after {
			t.Errorf("timestamp() = %d, expected between %d and %d", result, before, after)
		}
	})
	
	t.Run("returns valid unix timestamp format", func(t *testing.T) {
		result := timestamp()
		
		// Unix timestamps should be positive integers
		if result <= 0 {
			t.Errorf("timestamp() = %d, expected positive integer", result)
		}
		
		// Should be within reasonable range (after year 2020, before year 2030)
		// 1577836800 = Jan 1, 2020 00:00:00 UTC
		// 1893456000 = Jan 1, 2030 00:00:00 UTC
		if result < 1577836800 || result > 1893456000 {
			t.Errorf("timestamp() = %d, expected reasonable timestamp between 2020-2030", result)
		}
	})
	
	t.Run("subsequent calls are consistent", func(t *testing.T) {
		first := timestamp()
		second := timestamp()
		
		// Since Unix timestamps have second precision, they should be equal or second >= first
		if second < first {
			t.Errorf("timestamp() second call = %d, should be >= first call = %d", second, first)
		}
	})
	
	t.Run("can be converted to time", func(t *testing.T) {
		ts := timestamp()
		converted := time.Unix(ts, 0)
		
		// Should be close to now (within 1 second)
		now := time.Now()
		diff := now.Sub(converted)
		if diff < 0 {
			diff = -diff
		}
		
		if diff > time.Second {
			t.Errorf("timestamp() converted time difference = %v, expected < 1 second", diff)
		}
	})
	
	t.Run("multiple calls in sequence", func(t *testing.T) {
		timestamps := make([]int64, 5)
		for i := range timestamps {
			timestamps[i] = timestamp()
			if i > 0 {
				time.Sleep(1 * time.Millisecond)
			}
		}
		
		// All timestamps should be in ascending order (or equal due to precision)
		for i := 1; i < len(timestamps); i++ {
			if timestamps[i] < timestamps[i-1] {
				t.Errorf("timestamp()[%d] = %d < timestamp()[%d] = %d, expected non-decreasing", 
					i, timestamps[i], i-1, timestamps[i-1])
			}
		}
	})
}