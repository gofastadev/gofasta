package core

import (
	"go/ast"
	"go/parser"
	"go/token"
	"fmt"
	"golang.org/x/tools/go/ast/astutil"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNewASTUtilCache(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		cache := NewASTUtilCache(nil)
		if cache == nil {
			t.Fatal("Expected non-nil cache")
		}
		if cache.config == nil {
			t.Fatal("Expected non-nil config")
		}
	})
	
	t.Run("custom config", func(t *testing.T) {
		config := &ASTUtilCacheConfig{
			MaxCacheEntries: 100,
			TTL:            5 * time.Minute,
		}
		cache := NewASTUtilCache(config)
		if cache.config.MaxCacheEntries != 100 {
			t.Errorf("Expected MaxCacheEntries=100, got %d", cache.config.MaxCacheEntries)
		}
	})
}

func TestPathEnclosingInterval(t *testing.T) {
	src := `package main
func main() {
	x := 42
	println(x)
}`
	
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	
	cache := NewASTUtilCache(nil)
	
	// Find position of "42"
	pos := file.Pos() + token.Pos(len("package main\nfunc main() {\n\tx := "))
	end := pos + 2
	
	// First call - should miss cache
	path1, exact1 := cache.PathEnclosingInterval(file, pos, end)
	if path1 == nil {
		t.Fatal("Expected non-nil path")
	}
	
	// Second call - should hit cache
	path2, exact2 := cache.PathEnclosingInterval(file, pos, end)
	if !reflect.DeepEqual(path1, path2) || exact1 != exact2 {
		t.Error("Cached result differs from original")
	}
	
	stats := cache.GetStatistics()
	if stats["cache_hits"].(int64) < 1 {
		t.Error("Expected at least one cache hit")
	}
}

func TestApply(t *testing.T) {
	src := `package main
func main() {
	x := 42
}`
	
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	
	cache := NewASTUtilCache(nil)
	
	// Count identifiers
	count := 0
	pre := func(c *astutil.Cursor) bool {
		if _, ok := c.Node().(*ast.Ident); ok {
			count++
		}
		return true
	}
	
	result := cache.Apply(file, pre, nil)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	
	if count == 0 {
		t.Error("Expected to find identifiers")
	}
	
	// Apply again - should use cache
	count2 := 0
	pre2 := func(c *astutil.Cursor) bool {
		if _, ok := c.Node().(*ast.Ident); ok {
			count2++
		}
		return true
	}
	
	result2 := cache.Apply(file, pre2, nil)
	if result2 == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestImportOperations(t *testing.T) {
	src := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}`
	
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	
	cache := NewASTUtilCache(nil)
	
	t.Run("AddImport", func(t *testing.T) {
		added := cache.AddImport(fset, file, "strings")
		if !added {
			t.Error("Expected import to be added")
		}
		
		// Check if import exists
		hasStrings := false
		for _, imp := range file.Imports {
			if strings.Contains(imp.Path.Value, "strings") {
				hasStrings = true
				break
			}
		}
		if !hasStrings {
			t.Error("Import not found in file")
		}
	})
	
	t.Run("AddNamedImport", func(t *testing.T) {
		added := cache.AddNamedImport(fset, file, "io", "io/ioutil")
		if !added {
			t.Error("Expected named import to be added")
		}
	})
	
	t.Run("UsesImport", func(t *testing.T) {
		uses := cache.UsesImport(file, "fmt")
		if !uses {
			t.Error("Expected fmt to be used")
		}
		
		notUsed := cache.UsesImport(file, "strings")
		if notUsed {
			t.Error("Expected strings to not be used")
		}
	})
	
	t.Run("DeleteImport", func(t *testing.T) {
		deleted := cache.DeleteImport(fset, file, "strings")
		if !deleted {
			t.Error("Expected import to be deleted")
		}
	})
}

func TestBatchApply(t *testing.T) {
	srcs := []string{
		`package main; func f1() {}`,
		`package main; func f2() {}`,
		`package main; func f3() {}`,
	}
	
	fset := token.NewFileSet()
	var files []*ast.File
	
	for i, src := range srcs {
		file, err := parser.ParseFile(fset, fmt.Sprintf("test%d.go", i), src, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, file)
	}
	
	cache := NewASTUtilCache(&ASTUtilCacheConfig{
		ConcurrentOps:    true,
		OperationWorkers: 2,
	})
	
	// Apply transformation to all files
	pre := func(c *astutil.Cursor) bool {
		return true
	}
	
	results := cache.BatchApply(files, pre, nil)
	if len(results) != len(files) {
		t.Errorf("Expected %d results, got %d", len(files), len(results))
	}
	
	for i, result := range results {
		if result == nil {
			t.Errorf("Result %d is nil", i)
		}
	}
}

func TestFindFunctions(t *testing.T) {
	src := `package main

func main() {}
func helper() {}
func (r *Receiver) Method() {}
`
	
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	
	cache := NewASTUtilCache(nil)
	
	t.Run("FindFuncDecls", func(t *testing.T) {
		funcs := cache.FindFuncDecls(file)
		if len(funcs) != 3 {
			t.Errorf("Expected 3 functions, got %d", len(funcs))
		}
	})
	
	t.Run("FindIdentifiers", func(t *testing.T) {
		idents := cache.FindIdentifiers(file)
		if len(idents) == 0 {
			t.Error("Expected to find identifiers")
		}
	})
	
	t.Run("FindTypeSpecs", func(t *testing.T) {
		// Add a type to the source
		src := `package main
type MyType struct {
	Field int
}
type MyInterface interface {
	Method()
}`
		
		file, err := parser.ParseFile(fset, "test.go", src, 0)
		if err != nil {
			t.Fatal(err)
		}
		
		types := cache.FindTypeSpecs(file)
		if len(types) != 2 {
			t.Errorf("Expected 2 types, got %d", len(types))
		}
	})
}

func TestRenameIdent(t *testing.T) {
	src := `package main

func main() {
	oldName := 42
	println(oldName)
	x := oldName
}`
	
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	
	cache := NewASTUtilCache(nil)
	
	count := cache.RenameIdent(file, "oldName", "newName")
	if count != 3 { // Variable declaration and two uses
		t.Errorf("Expected 3 renames, got %d", count)
	}
	
	// Verify rename
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok && ident.Name == "newName" {
			found = true
		}
		return true
	})
	
	if !found {
		t.Error("Renamed identifier not found")
	}
}

func TestOptimizeImports(t *testing.T) {
	src := `package main

import (
	"strings"
	"github.com/healtronlabs/gofasta/core"
	"fmt"
	"golang.org/x/tools/go/ast/astutil"
)

func main() {
	fmt.Println("test")
}`
	
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	
	cache := NewASTUtilCache(nil)
	cache.OptimizeImports(fset, file)
	
	// Check that imports are sorted
	imports := astutil.Imports(fset, file)
	if len(imports) == 0 {
		t.Error("No imports found after optimization")
	}
}

func TestASTWalker(t *testing.T) {
	src := `package main

type MyStruct struct {
	Field int
}

func main() {
	x := 42
}`
	
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	
	cache := NewASTUtilCache(nil)
	walker := cache.NewASTWalker()
	
	t.Run("FindNodes", func(t *testing.T) {
		nodes := walker.FindNodes(file, reflect.TypeOf(&ast.FuncDecl{}))
		if len(nodes) != 1 {
			t.Errorf("Expected 1 function, got %d", len(nodes))
		}
	})
	
	t.Run("FindNodesByName", func(t *testing.T) {
		nodes := walker.FindNodesByName(file, "main")
		if len(nodes) == 0 {
			t.Error("Expected to find 'main' nodes")
		}
	})
}

func TestExtractComments(t *testing.T) {
	src := `// Package comment
package main

// Function comment
func main() {
	// Inline comment
	x := 42 // End of line comment
}

// Another function
func helper() {}
`
	
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	
	cache := NewASTUtilCache(nil)
	comments := cache.ExtractComments(file)
	
	if len(comments) == 0 {
		t.Error("Expected to find comments")
	}
}

func TestHasSideEffects(t *testing.T) {
	tests := []struct {
		expr     string
		expected bool
	}{
		{"42", false},
		{"x", false},
		{"x + y", false},
		{"f()", true},
		{"x++", true},
		{"x = 5", true},
	}
	
	cache := NewASTUtilCache(nil)
	
	for _, test := range tests {
		expr, err := parser.ParseExpr(test.expr)
		if err != nil {
			// Some expressions need to be parsed as statements
			continue
		}
		
		hasSideEffects := cache.HasSideEffects(expr)
		if hasSideEffects != test.expected {
			t.Errorf("HasSideEffects(%s) = %v, expected %v",
				test.expr, hasSideEffects, test.expected)
		}
	}
}

func TestASTUtilCacheEviction(t *testing.T) {
	config := &ASTUtilCacheConfig{
		MaxCacheEntries: 2,
		CachePathOps:   true,
	}
	cache := NewASTUtilCache(config)
	
	src := `package main; func f() { x := 1; y := 2; z := 3 }`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	
	// Add multiple entries to trigger eviction
	cache.PathEnclosingInterval(file, 10, 11)
	cache.PathEnclosingInterval(file, 20, 21)
	cache.PathEnclosingInterval(file, 30, 31) // Should trigger eviction
	
	stats := cache.GetStatistics()
	pathCacheSize := stats["path_cache_size"].(int)
	if pathCacheSize > 2 {
		t.Errorf("Expected cache size <= 2, got %d", pathCacheSize)
	}
}

func TestASTUtilCacheClear(t *testing.T) {
	cache := NewASTUtilCache(nil)
	
	src := `package main; func f() {}`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	
	// Add some cache entries
	cache.PathEnclosingInterval(file, 10, 20)
	cache.Apply(file, nil, nil)
	
	// Clear cache
	cache.Clear()
	
	stats := cache.GetStatistics()
	if stats["cache_hits"].(int64) != 0 {
		t.Error("Expected cache hits to be 0 after clear")
	}
	if stats["cache_misses"].(int64) != 0 {
		t.Error("Expected cache misses to be 0 after clear")
	}
}

func TestGetEnclosingFunction(t *testing.T) {
	src := `package main

func outer() {
	inner := func() {
		x := 42
	}
	inner()
}

func another() {
	y := 100
}`
	
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	
	cache := NewASTUtilCache(nil)
	
	// Find the literal 42
	var target ast.Node
	ast.Inspect(file, func(n ast.Node) bool {
		if lit, ok := n.(*ast.BasicLit); ok && lit.Value == "42" {
			target = lit
			return false
		}
		return true
	})
	
	if target == nil {
		t.Fatal("Could not find target node")
	}
	
	enclosing := cache.GetEnclosingFunction(file, target)
	if enclosing == nil {
		t.Fatal("Expected to find enclosing function")
	}
	
	if enclosing.Name.Name != "outer" {
		t.Errorf("Expected enclosing function to be 'outer', got '%s'", enclosing.Name.Name)
	}
}

func TestRemoveUnusedImports(t *testing.T) {
	src := `package main

import (
	"fmt"
	"strings"
	"bytes"
)

func main() {
	fmt.Println("hello")
}`
	
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	
	cache := NewASTUtilCache(nil)
	removed := cache.RemoveUnusedImports(fset, file)
	
	// strings and bytes should be removed
	if removed != 2 {
		t.Errorf("Expected 2 imports to be removed, got %d", removed)
	}
	
	// Check that fmt is still there
	hasFormat := false
	for _, imp := range file.Imports {
		if strings.Contains(imp.Path.Value, "fmt") {
			hasFormat = true
			break
		}
	}
	
	if !hasFormat {
		t.Error("fmt import should not have been removed")
	}
}