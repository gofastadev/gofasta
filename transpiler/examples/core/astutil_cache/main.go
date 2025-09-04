// Example demonstrating golang.org/x/tools/go/ast/astutil with AST operation cache (Phase 1.2e)
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"

	"reflect"

	"github.com/healtronlabs/gofasta/transpiler/core"
	"golang.org/x/tools/go/ast/astutil"
)

func main() {
	// Create AST util cache with custom configuration
	config := &core.ASTUtilCacheConfig{
		MaxCacheEntries:  1000,
		CachePathOps:     true,
		CacheApplyOps:    true,
		CacheImportOps:   true,
		ConcurrentOps:    true,
		OperationWorkers: 4,
	}
	cache := core.NewASTUtilCache(config)

	// Sample Go source code
	src := `package main

import (
	"fmt"
	"strings"
)

type Person struct {
	Name string
	Age  int
}

func main() {
	p := Person{Name: "Alice", Age: 30}
	fmt.Printf("Person: %+v\n", p)
	greeting := strings.Join([]string{"Hello", p.Name}, " ")
	fmt.Println(greeting)
}

func helper(x int) int {
	return x * 2
}`

	// Parse the source code
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "example.go", src, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== AST Util Cache Example ===\n")

	// 1. Path Enclosing Interval
	fmt.Println("1. Finding path enclosing interval:")
	// Find the position of "Person" struct
	pos := file.Pos() + token.Pos(len(`package main

import (
	"fmt"
	"strings"
)

type `))
	end := pos + token.Pos(len("Person"))

	path, exact := cache.PathEnclosingInterval(file, pos, end)
	fmt.Printf("   Found %d nodes in path, exact=%v\n", len(path), exact)
	if len(path) > 0 {
		fmt.Printf("   First node type: %T\n", path[0])
	}

	// 2. Apply transformation
	fmt.Println("\n2. Applying AST transformation:")
	countIdents := 0
	pre := func(c *astutil.Cursor) bool {
		if _, ok := c.Node().(*ast.Ident); ok {
			countIdents++
		}
		return true
	}

	result := cache.Apply(file, pre, nil)
	fmt.Printf("   Found %d identifiers\n", countIdents)
	fmt.Printf("   Result type: %T\n", result)

	// 3. Import management
	fmt.Println("\n3. Import management:")

	// Add an import
	added := cache.AddImport(fset, file, "time")
	fmt.Printf("   Added 'time' import: %v\n", added)

	// Add a named import
	added = cache.AddNamedImport(fset, file, "io", "io/ioutil")
	fmt.Printf("   Added named import 'io': %v\n", added)

	// Check if import is used
	uses := cache.UsesImport(file, "fmt")
	fmt.Printf("   File uses 'fmt': %v\n", uses)

	uses = cache.UsesImport(file, "time")
	fmt.Printf("   File uses 'time': %v (should be false)\n", uses)

	// 4. Find functions
	fmt.Println("\n4. Finding declarations:")
	funcs := cache.FindFuncDecls(file)
	fmt.Printf("   Found %d functions:\n", len(funcs))
	for _, fn := range funcs {
		fmt.Printf("     - %s\n", fn.Name.Name)
	}

	// 5. Find type specifications
	types := cache.FindTypeSpecs(file)
	fmt.Printf("   Found %d types:\n", len(types))
	for _, t := range types {
		fmt.Printf("     - %s\n", t.Name.Name)
	}

	// 6. Rename identifier
	fmt.Println("\n5. Renaming identifiers:")
	count := cache.RenameIdent(file, "Person", "Employee")
	fmt.Printf("   Renamed %d occurrences of 'Person' to 'Employee'\n", count)

	// 7. Remove unused imports
	fmt.Println("\n6. Removing unused imports:")
	removed := cache.RemoveUnusedImports(fset, file)
	fmt.Printf("   Removed %d unused imports\n", removed)

	// 8. Extract comments
	fmt.Println("\n7. Extracting comments:")
	comments := cache.ExtractComments(file)
	fmt.Printf("   Found %d comment groups\n", len(comments))

	// 9. Check for side effects
	fmt.Println("\n8. Checking for side effects:")
	exprs := []string{
		"42",
		"x + y",
		"fmt.Println()",
	}
	for _, exprStr := range exprs {
		expr, err := parser.ParseExpr(exprStr)
		if err == nil {
			hasSideEffects := cache.HasSideEffects(expr)
			fmt.Printf("   '%s' has side effects: %v\n", exprStr, hasSideEffects)
		}
	}

	// 10. Get enclosing function
	fmt.Println("\n9. Finding enclosing function:")
	// Find a node inside main function
	var targetNode ast.Node
	ast.Inspect(file, func(n ast.Node) bool {
		if lit, ok := n.(*ast.BasicLit); ok && lit.Value == `"Alice"` {
			targetNode = lit
			return false
		}
		return true
	})

	if targetNode != nil {
		enclosing := cache.GetEnclosingFunction(file, targetNode)
		if enclosing != nil {
			fmt.Printf("   Node is inside function: %s\n", enclosing.Name.Name)
		}
	}

	// 11. Batch operations on multiple files
	fmt.Println("\n10. Batch operations:")
	srcs := []string{
		`package main; func f1() { x := 1 }`,
		`package main; func f2() { y := 2 }`,
		`package main; func f3() { z := 3 }`,
	}

	var files []*ast.File
	for i, s := range srcs {
		f, err := parser.ParseFile(fset, fmt.Sprintf("file%d.go", i), s, 0)
		if err == nil {
			files = append(files, f)
		}
	}

	// Apply transformation to all files concurrently
	transformCount := 0
	transform := func(c *astutil.Cursor) bool {
		if _, ok := c.Node().(*ast.FuncDecl); ok {
			transformCount++
		}
		return true
	}

	results := cache.BatchApply(files, transform, nil)
	fmt.Printf("   Processed %d files, found %d functions\n", len(results), transformCount)

	// 12. AST Walker
	fmt.Println("\n11. Using AST Walker:")
	walker := cache.NewASTWalker()

	// Find all nodes of a specific type
	nodes := walker.FindNodes(file, reflect.TypeOf(&ast.StructType{}))
	fmt.Printf("   Found %d struct types\n", len(nodes))

	// Find nodes by name
	nodes = walker.FindNodesByName(file, "main")
	fmt.Printf("   Found %d nodes named 'main'\n", len(nodes))

	// Show cache statistics
	fmt.Println("\n=== Cache Statistics ===")
	stats := cache.GetStatistics()
	fmt.Printf("Path cache size: %d\n", stats["path_cache_size"])
	fmt.Printf("Apply cache size: %d\n", stats["apply_cache_size"])
	fmt.Printf("Import cache size: %d\n", stats["import_cache_size"])
	fmt.Printf("Cache hits: %d\n", stats["cache_hits"])
	fmt.Printf("Cache misses: %d\n", stats["cache_misses"])
	if stats["cache_hits"].(int64)+stats["cache_misses"].(int64) > 0 {
		fmt.Printf("Hit rate: %.2f%%\n", stats["hit_rate"])
	}
	fmt.Printf("Total operations: %d\n", stats["total_operations"])
}
