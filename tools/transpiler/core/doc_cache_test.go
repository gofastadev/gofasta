// Package core provides tests for documentation cache.
package core

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"strings"
	"testing"
	"time"
)

func TestNewDocCache(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		dc := NewDocCache(nil)
		if dc == nil {
			t.Fatal("Expected non-nil doc cache")
		}
		if dc.config == nil {
			t.Error("Expected non-nil config")
		}
		if !dc.config.EnableMetrics {
			t.Error("Expected metrics to be enabled by default")
		}
	})
	
	t.Run("with custom config", func(t *testing.T) {
		config := &DocCacheConfig{
			MaxTemplates:         50,
			MaxDocs:              250,
			MaxCacheSizeMB:       25,
			EnableMetrics:        false,
			EnableMarkdown:       false,
			EnableJSON:           false,
			IncludePrivate:       true,
			ConcurrentGeneration: false,
		}
		dc := NewDocCache(config)
		if dc == nil {
			t.Fatal("Expected non-nil doc cache")
		}
		if dc.config.MaxTemplates != 50 {
			t.Errorf("Expected max templates 50, got %d", dc.config.MaxTemplates)
		}
		if dc.config.IncludePrivate != true {
			t.Error("Expected include private to be true")
		}
	})
}

func TestLoadTemplates(t *testing.T) {
	dc := NewDocCache(nil)
	
	t.Run("load standard templates", func(t *testing.T) {
		err := dc.LoadTemplates("")
		if err != nil {
			t.Fatalf("Failed to load templates: %v", err)
		}
		
		// Check that standard templates were loaded
		expectedTemplates := []string{
			"package", "type", "func", "method",
			"const", "var", "index", "api",
		}
		
		dc.mu.RLock()
		defer dc.mu.RUnlock()
		
		for _, name := range expectedTemplates {
			if _, exists := dc.templates[name]; !exists {
				t.Errorf("Expected template %s to be loaded", name)
			}
		}
	})
}

func TestExtractPackageDoc(t *testing.T) {
	dc := NewDocCache(nil)
	
	t.Run("extract package documentation", func(t *testing.T) {
		// Create a simple package
		src := `
package testpkg

// TestType is a test type.
type TestType struct {
	Field string
}

// TestFunc is a test function.
func TestFunc() {}
`
		
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse file: %v", err)
		}
		
		pkg := &ast.Package{
			Name: "testpkg",
			Files: map[string]*ast.File{
				"test.go": file,
			},
		}
		
		docPkg, err := dc.ExtractPackageDoc(pkg, "example.com/testpkg")
		if err != nil {
			t.Fatalf("Failed to extract package doc: %v", err)
		}
		
		if docPkg == nil {
			t.Fatal("Expected non-nil package doc")
		}
		if docPkg.Name != "testpkg" {
			t.Errorf("Expected package name 'testpkg', got %s", docPkg.Name)
		}
		if docPkg.ImportPath != "example.com/testpkg" {
			t.Errorf("Expected import path 'example.com/testpkg', got %s", docPkg.ImportPath)
		}
		
		// Check caching
		cached, err := dc.ExtractPackageDoc(pkg, "example.com/testpkg")
		if err != nil {
			t.Fatalf("Failed to get cached doc: %v", err)
		}
		if cached != docPkg {
			t.Error("Expected to get cached package doc")
		}
	})
}

func TestGeneratePackageDoc(t *testing.T) {
	dc := NewDocCache(nil)
	dc.LoadTemplates("")
	
	t.Run("generate HTML documentation", func(t *testing.T) {
		// Create a doc.Package
		docPkg := &doc.Package{
			Name:       "testpkg",
			ImportPath: "example.com/testpkg",
			Doc:        "Package testpkg provides testing functionality.",
			Funcs: []*doc.Func{
				{
					Name: "TestFunc",
					Doc:  "TestFunc does something.",
				},
			},
			Types: []*doc.Type{
				{
					Name: "TestType",
					Doc:  "TestType represents something.",
				},
			},
		}
		
		html, err := dc.GeneratePackageDoc(docPkg)
		if err != nil {
			t.Fatalf("Failed to generate package doc: %v", err)
		}
		
		if html == "" {
			t.Error("Expected non-empty HTML")
		}
		
		// Check that HTML contains expected elements
		if !strings.Contains(html, "Package testpkg") {
			t.Error("Expected HTML to contain package name")
		}
		if !strings.Contains(html, "TestFunc") {
			t.Error("Expected HTML to contain function name")
		}
		if !strings.Contains(html, "TestType") {
			t.Error("Expected HTML to contain type name")
		}
		
		// Check caching
		dc.mu.RLock()
		cached, exists := dc.docCache["example.com/testpkg"]
		dc.mu.RUnlock()
		
		if !exists {
			t.Error("Expected documentation to be cached")
		}
		if cached.HTML != html {
			t.Error("Expected cached HTML to match generated HTML")
		}
	})
}

func TestGenerateTypeDoc(t *testing.T) {
	dc := NewDocCache(nil)
	dc.LoadTemplates("")
	
	t.Run("generate type documentation", func(t *testing.T) {
		docType := &doc.Type{
			Name: "MyType",
			Doc:  "MyType is a custom type.",
			Methods: []*doc.Func{
				{
					Name: "Method1",
					Doc:  "Method1 does something.",
					Recv: "MyType",
				},
			},
		}
		
		html, err := dc.GenerateTypeDoc(docType, "example.com/pkg")
		if err != nil {
			t.Fatalf("Failed to generate type doc: %v", err)
		}
		
		if html == "" {
			t.Error("Expected non-empty HTML")
		}
		
		// Check content
		if !strings.Contains(html, "MyType") {
			t.Error("Expected HTML to contain type name")
		}
	})
}

func TestGenerateFunctionDoc(t *testing.T) {
	dc := NewDocCache(nil)
	dc.LoadTemplates("")
	
	t.Run("generate function documentation", func(t *testing.T) {
		docFunc := &doc.Func{
			Name: "MyFunction",
			Doc:  "MyFunction performs an operation.",
		}
		
		html, err := dc.GenerateFunctionDoc(docFunc, "example.com/pkg")
		if err != nil {
			t.Fatalf("Failed to generate function doc: %v", err)
		}
		
		if html == "" {
			t.Error("Expected non-empty HTML")
		}
		
		// Check content
		if !strings.Contains(html, "MyFunction") {
			t.Error("Expected HTML to contain function name")
		}
	})
}

func TestBatchGenerate(t *testing.T) {
	dc := NewDocCache(&DocCacheConfig{
		ConcurrentGeneration: true,
		GenerationWorkers:    2,
	})
	dc.LoadTemplates("")
	
	t.Run("batch generate documentation", func(t *testing.T) {
		items := []DocItem{
			{
				Key:  "func1",
				Type: "func",
				Data: map[string]interface{}{
					"Function": &doc.Func{Name: "Func1"},
					"Package":  "pkg1",
				},
			},
			{
				Key:  "func2",
				Type: "func",
				Data: map[string]interface{}{
					"Function": &doc.Func{Name: "Func2"},
					"Package":  "pkg2",
				},
			},
			{
				Key:  "type1",
				Type: "type",
				Data: map[string]interface{}{
					"Type":    &doc.Type{Name: "Type1"},
					"Package": "pkg1",
				},
			},
		}
		
		docs, err := dc.BatchGenerate(items)
		if err != nil {
			t.Fatalf("Failed to batch generate: %v", err)
		}
		
		if len(docs) != 3 {
			t.Errorf("Expected 3 documents, got %d", len(docs))
		}
		
		for _, item := range items {
			if _, exists := docs[item.Key]; !exists {
				t.Errorf("Expected document for key %s", item.Key)
			}
		}
	})
}

func TestGenerateIndex(t *testing.T) {
	dc := NewDocCache(nil)
	dc.LoadTemplates("")
	
	// Add some package docs
	dc.packageDocs["pkg1"] = &doc.Package{Name: "pkg1"}
	dc.packageDocs["pkg2"] = &doc.Package{Name: "pkg2"}
	dc.packageDocs["pkg3"] = &doc.Package{Name: "pkg3"}
	
	t.Run("generate documentation index", func(t *testing.T) {
		html, err := dc.GenerateIndex()
		if err != nil {
			t.Fatalf("Failed to generate index: %v", err)
		}
		
		if html == "" {
			t.Error("Expected non-empty HTML")
		}
		
		// Check that packages are listed
		if !strings.Contains(html, "pkg1") {
			t.Error("Expected index to contain pkg1")
		}
		if !strings.Contains(html, "pkg2") {
			t.Error("Expected index to contain pkg2")
		}
		if !strings.Contains(html, "pkg3") {
			t.Error("Expected index to contain pkg3")
		}
	})
}

func TestDocCacheEviction(t *testing.T) {
	dc := NewDocCache(&DocCacheConfig{
		MaxDocs: 3,
	})
	dc.LoadTemplates("")
	
	t.Run("evict oldest documentation", func(t *testing.T) {
		// Generate docs to fill cache
		for i := 0; i < 5; i++ {
			key := fmt.Sprintf("doc%d", i)
			data := map[string]interface{}{
				"Function": &doc.Func{Name: fmt.Sprintf("Func%d", i)},
				"Package":  "pkg",
			}
			_, err := dc.generateDoc("func", data, key)
			if err != nil {
				t.Fatalf("Failed to generate doc %d: %v", i, err)
			}
			time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		}
		
		// Check cache size
		dc.mu.RLock()
		cacheSize := len(dc.docCache)
		dc.mu.RUnlock()
		
		if cacheSize > 3 {
			t.Errorf("Expected cache size <= 3, got %d", cacheSize)
		}
		
		// Check that oldest docs were evicted
		if _, exists := dc.GetCachedDoc("doc0"); exists {
			t.Error("Expected doc0 to be evicted")
		}
		if _, exists := dc.GetCachedDoc("doc1"); exists {
			t.Error("Expected doc1 to be evicted")
		}
		
		// Check that newest docs are still cached
		if _, exists := dc.GetCachedDoc("doc4"); !exists {
			t.Error("Expected doc4 to be cached")
		}
	})
}

func TestDocCacheGetStatistics(t *testing.T) {
	dc := NewDocCache(nil)
	dc.LoadTemplates("")
	
	// Generate some activity
	docPkg := &doc.Package{
		Name:       "testpkg",
		ImportPath: "test",
	}
	_, _ = dc.GeneratePackageDoc(docPkg)
	_, _ = dc.GeneratePackageDoc(docPkg) // Hit cache
	
	stats := dc.GetStatistics()
	
	// Check statistics
	if stats["template_count"].(int) == 0 {
		t.Error("Expected templates to be loaded")
	}
	if stats["cached_docs"].(int) == 0 {
		t.Error("Expected cached docs")
	}
	if stats["cache_hits"].(int64) == 0 {
		t.Error("Expected cache hits")
	}
	if stats["generations"].(int64) == 0 {
		t.Error("Expected generations")
	}
}

func TestInvalidateDoc(t *testing.T) {
	dc := NewDocCache(nil)
	dc.LoadTemplates("")
	
	t.Run("invalidate cached documentation", func(t *testing.T) {
		// Generate a doc
		data := map[string]interface{}{
			"Function": &doc.Func{Name: "TestFunc"},
			"Package":  "pkg",
		}
		_, err := dc.generateDoc("func", data, "testdoc")
		if err != nil {
			t.Fatalf("Failed to generate doc: %v", err)
		}
		
		// Verify it's cached
		if _, exists := dc.GetCachedDoc("testdoc"); !exists {
			t.Error("Expected doc to be cached")
		}
		
		// Invalidate
		dc.InvalidateDoc("testdoc")
		
		// Verify it's removed
		if _, exists := dc.GetCachedDoc("testdoc"); exists {
			t.Error("Expected doc to be invalidated")
		}
	})
}

func TestDocCacheHelperFunctions(t *testing.T) {
	t.Run("makeAnchor", func(t *testing.T) {
		result := makeAnchor("Test Section")
		expected := `<a id="test-section"></a>`
		if string(result) != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	})
	
	t.Run("makeLink", func(t *testing.T) {
		result := makeLink("Click here", "/path/to/page")
		if !strings.Contains(string(result), "href=\"/path/to/page\"") {
			t.Error("Expected link to contain correct href")
		}
		if !strings.Contains(string(result), "Click here") {
			t.Error("Expected link to contain text")
		}
	})
	
	t.Run("makeCodeBlock", func(t *testing.T) {
		result := makeCodeBlock("fmt.Println()", "go")
		if !strings.Contains(string(result), "language-go") {
			t.Error("Expected code block to have language class")
		}
		if !strings.Contains(string(result), "fmt.Println()") {
			t.Error("Expected code block to contain code")
		}
	})
	
	t.Run("indentCode", func(t *testing.T) {
		code := "line1\nline2\nline3"
		result := indentCode(4, code)
		expected := "    line1\n    line2\n    line3"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})
	
	t.Run("dedentCode", func(t *testing.T) {
		code := "    line1\n    line2\n        line3"
		result := dedentCode(code)
		expected := "line1\nline2\n    line3"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})
}

func TestDocCacheClear(t *testing.T) {
	dc := NewDocCache(nil)
	dc.LoadTemplates("")
	
	// Generate some data
	docPkg := &doc.Package{Name: "test"}
	_, _ = dc.GeneratePackageDoc(docPkg)
	
	// Clear cache
	dc.Clear()
	
	// Check everything was cleared
	dc.mu.RLock()
	templateCount := len(dc.templates)
	docCount := len(dc.docCache)
	packageCount := len(dc.packageDocs)
	dc.mu.RUnlock()
	
	if templateCount != 0 {
		t.Errorf("Expected 0 templates after clear, got %d", templateCount)
	}
	if docCount != 0 {
		t.Errorf("Expected 0 cached docs after clear, got %d", docCount)
	}
	if packageCount != 0 {
		t.Errorf("Expected 0 package docs after clear, got %d", packageCount)
	}
	if dc.hits != 0 {
		t.Error("Expected hits to be reset")
	}
	if dc.generations != 0 {
		t.Error("Expected generations to be reset")
	}
}

func BenchmarkGeneratePackageDoc(b *testing.B) {
	dc := NewDocCache(nil)
	dc.LoadTemplates("")
	
	docPkg := &doc.Package{
		Name:       "benchpkg",
		ImportPath: "bench",
		Doc:        "Benchmark package",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench_%d", i)
		_, _ = dc.generateDoc("package", docPkg, key)
	}
}

func BenchmarkBatchGenerate(b *testing.B) {
	dc := NewDocCache(&DocCacheConfig{
		ConcurrentGeneration: true,
		GenerationWorkers:    4,
	})
	dc.LoadTemplates("")
	
	items := make([]DocItem, 10)
	for i := 0; i < 10; i++ {
		items[i] = DocItem{
			Key:  fmt.Sprintf("item%d", i),
			Type: "func",
			Data: map[string]interface{}{
				"Function": &doc.Func{Name: fmt.Sprintf("Func%d", i)},
				"Package":  "pkg",
			},
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = dc.BatchGenerate(items)
		dc.Clear() // Clear for next iteration
	}
}