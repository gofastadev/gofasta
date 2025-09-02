// Package core provides tests for text/template compiled cache.
package core

import (
	"strings"
	"sync"
	"testing"
	"text/template"
)

func TestNewTemplateCache(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		tc := NewTemplateCache(nil)
		if tc == nil {
			t.Fatal("Expected non-nil template cache")
		}
		if tc.config == nil {
			t.Error("Expected non-nil config")
		}
		if !tc.config.EnableMetrics {
			t.Error("Expected metrics to be enabled by default")
		}
		if !tc.config.PrecompileOnAdd {
			t.Error("Expected precompile on add to be enabled by default")
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &TemplateCacheConfig{
			MaxTemplates:      100,
			MaxCacheSizeMB:    50,
			EnableDedup:       false,
			EnableMetrics:     false,
			Delims:            [2]string{"[[", "]]"},
			PrecompileOnAdd:   false,
			LazyCompilation:   true,
			ConcurrentCompile: false,
			CompileWorkers:    2,
		}
		tc := NewTemplateCache(config)
		if tc == nil {
			t.Fatal("Expected non-nil template cache")
		}
		if tc.config.MaxTemplates != 100 {
			t.Errorf("Expected max templates 100, got %d", tc.config.MaxTemplates)
		}
		if tc.config.Delims[0] != "[[" || tc.config.Delims[1] != "]]" {
			t.Errorf("Expected custom delimiters [[]], got %v", tc.config.Delims)
		}
	})
}

func TestAddTemplate(t *testing.T) {
	tc := NewTemplateCache(nil)

	t.Run("add simple template", func(t *testing.T) {
		err := tc.AddTemplate("simple", "Hello {{.Name}}!")
		if err != nil {
			t.Fatalf("Failed to add template: %v", err)
		}

		// Check template was stored
		tc.mu.RLock()
		tmpl, exists := tc.templates["simple"]
		tc.mu.RUnlock()

		if !exists {
			t.Error("Template was not stored")
		}
		if tmpl.Name != "simple" {
			t.Errorf("Expected template name 'simple', got %s", tmpl.Name)
		}
		if tmpl.Source != "Hello {{.Name}}!" {
			t.Errorf("Expected source 'Hello {{.Name}}!', got %s", tmpl.Source)
		}
	})

	t.Run("add template with metadata", func(t *testing.T) {
		metadata := map[string]interface{}{
			"version": "1.0",
			"author":  "test",
		}
		err := tc.AddTemplateWithMetadata("meta", "{{.Value}}", metadata)
		if err != nil {
			t.Fatalf("Failed to add template with metadata: %v", err)
		}

		tc.mu.RLock()
		tmpl, exists := tc.templates["meta"]
		tc.mu.RUnlock()

		if !exists {
			t.Error("Template was not stored")
		}
		if tmpl.Metadata == nil {
			t.Error("Expected metadata to be stored")
		}
		if tmpl.Metadata["version"] != "1.0" {
			t.Errorf("Expected version 1.0, got %v", tmpl.Metadata["version"])
		}
	})

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