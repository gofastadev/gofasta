// Example demonstrating go/constant with constant evaluation cache (Phase 1.2g)
package main

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"log"

	"github.com/healtronlabs/gofasta/transpiler/core"
)

func main() {
	// Create constant cache with custom configuration
	config := &core.ConstantCacheConfig{
		MaxCacheEntries: 5000,
		EnableMetrics:   true,
		CacheBinaryOps:  true,
		CacheUnaryOps:   true,
		CacheCompareOps: true,
		CacheShiftOps:   true,
		MaxPrecision:    256,
		ConcurrentEval:  true,
		EvalWorkers:     4,
	}
	cache := core.NewConstantCache(config)

	fmt.Println("=== Constant Cache Example ===\n")

	// 1. Basic constant creation
	fmt.Println("1. Creating basic constants:")
	intVal := cache.MakeInt64(42)
	floatVal := cache.MakeFloat64(3.14159)
	strVal := cache.MakeString("Hello, World!")
	boolVal := cache.MakeBool(true)

	fmt.Printf("   Integer: %v\n", intVal)
	fmt.Printf("   Float: %v\n", floatVal)
	fmt.Printf("   String: %v\n", strVal)
	fmt.Printf("   Boolean: %v\n", boolVal)

	// 2. Binary operations
	fmt.Println("\n2. Binary operations:")
	x := cache.MakeInt64(10)
	y := cache.MakeInt64(3)

	operations := []struct {
		op   token.Token
		name string
	}{
		{token.ADD, "Addition"},
		{token.SUB, "Subtraction"},
		{token.MUL, "Multiplication"},
		{token.QUO, "Division"},
		{token.REM, "Remainder"},
		{token.AND, "Bitwise AND"},
		{token.OR, "Bitwise OR"},
		{token.XOR, "Bitwise XOR"},
	}

	for _, op := range operations {
		result, ok := cache.BinaryOp(x, op.op, y)
		if ok {
			if i, ok := constant.Int64Val(result); ok {
				fmt.Printf("   %s (10 %s 3) = %d\n", op.name, op.op, i)
			}
		}
	}

	// 3. Unary operations
	fmt.Println("\n3. Unary operations:")
	unaryOps := []struct {
		op   token.Token
		val  constant.Value
		name string
	}{
		{token.SUB, cache.MakeInt64(42), "Negation"},
		{token.ADD, cache.MakeInt64(42), "Plus"},
		{token.XOR, cache.MakeInt64(5), "Bitwise NOT"},
		{token.NOT, cache.MakeBool(true), "Logical NOT"},
	}

	for _, op := range unaryOps {
		result, ok := cache.UnaryOp(op.op, op.val)
		if ok {
			fmt.Printf("   %s: %v -> %v\n", op.name, op.val, result)
		}
	}

	// 4. Comparisons
	fmt.Println("\n4. Comparison operations:")
	a := cache.MakeInt64(10)
	b := cache.MakeInt64(20)

	compareOps := []struct {
		op   token.Token
		name string
	}{
		{token.EQL, "Equal"},
		{token.NEQ, "Not Equal"},
		{token.LSS, "Less Than"},
		{token.LEQ, "Less or Equal"},
		{token.GTR, "Greater Than"},
		{token.GEQ, "Greater or Equal"},
	}

	for _, op := range compareOps {
		result, ok := cache.Compare(a, op.op, b)
		if ok {
			fmt.Printf("   10 %s 20: %v\n", op.name, result)
		}
	}

	// 5. Shift operations
	fmt.Println("\n5. Shift operations:")
	shiftVal := cache.MakeInt64(8)

	leftShift, _ := cache.Shift(shiftVal, token.SHL, 2)
	rightShift, _ := cache.Shift(shiftVal, token.SHR, 2)

	if l, ok := constant.Int64Val(leftShift); ok {
		fmt.Printf("   8 << 2 = %d\n", l)
	}
	if r, ok := constant.Int64Val(rightShift); ok {
		fmt.Printf("   8 >> 2 = %d\n", r)
	}

	// 6. Evaluate expressions
	fmt.Println("\n6. Evaluating expressions:")
	expressions := []string{
		"42",
		"3.14",
		`"hello"`,
		"1 + 2 * 3",
		"(10 - 5) * 2",
		"100 / 10 / 2",
		"1 << 3 | 1",
		"15 & 7 ^ 3",
		"-42",
		"!false", // This won't parse directly
	}

	for _, expr := range expressions {
		parsed, err := parser.ParseExpr(expr)
		if err != nil {
			continue
		}

		val, ok := cache.EvaluateExpr(parsed, nil)
		if ok {
			fmt.Printf("   %s = %v\n", expr, val)
		}
	}

	// 7. Type conversions
	fmt.Println("\n7. Type conversions:")

	// Convert int to float
	intConst := cache.MakeInt64(42)
	f, ok := cache.ToFloat64(intConst)
	if ok {
		fmt.Printf("   Int to Float: 42 -> %.2f\n", f)
	}

	// Convert float to int (if whole number)
	floatConst := cache.MakeFloat64(100.0)
	i, ok := cache.ToInt64(floatConst)
	if ok {
		fmt.Printf("   Float to Int: 100.0 -> %d\n", i)
	}

	// Convert to string
	s := cache.ToString(cache.MakeInt64(123))
	fmt.Printf("   Int to String: 123 -> %q\n", s)

	s = cache.ToString(cache.MakeBool(true))
	fmt.Printf("   Bool to String: true -> %q\n", s)

	// 8. Complex expressions from source code
	fmt.Println("\n8. Evaluating complex expressions from code:")
	src := `package main

const (
	MaxSize = 100
	MinSize = 10
	DefaultSize = (MaxSize + MinSize) / 2
	
	Pi = 3.14159
	Tau = Pi * 2
	
	Greeting = "Hello, " + "World!"
	
	BitMask = 0xFF << 8
	Flags = 0x01 | 0x02 | 0x04
)

func main() {
	const localConst = 42 * 2
	x := MaxSize - MinSize
	y := DefaultSize * 2
}`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "constants.go", src, 0)
	if err != nil {
		log.Fatal(err)
	}

	// Type check to get constant values
	conf := types.Config{}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	pkg, _ := conf.Check("main", fset, []*ast.File{file}, info)
	_ = pkg

	// Find and evaluate constant expressions
	fmt.Println("   Constants found in code:")
	ast.Inspect(file, func(n ast.Node) bool {
		if spec, ok := n.(*ast.ValueSpec); ok {
			for i, name := range spec.Names {
				if i < len(spec.Values) {
					val, ok := cache.EvaluateExpr(spec.Values[i], info)
					if ok {
						fmt.Printf("     %s = %v\n", name.Name, val)
					}
				}
			}
		}
		return true
	})

	// 9. Batch evaluation
	fmt.Println("\n9. Batch evaluation of expressions:")
	batchExprs := []string{
		"1 + 1",
		"2 * 3",
		"10 / 2",
		"7 % 3",
		"1 << 4",
	}

	var parsedExprs []ast.Expr
	for _, expr := range batchExprs {
		parsed, err := parser.ParseExpr(expr)
		if err == nil {
			parsedExprs = append(parsedExprs, parsed)
		}
	}

	results := cache.BatchEvaluate(parsedExprs, nil)
	fmt.Println("   Batch results:")
	for i, result := range results {
		if result != nil && i < len(batchExprs) {
			fmt.Printf("     %s = %v\n", batchExprs[i], result)
		}
	}

	// 10. Check if expressions are constant
	fmt.Println("\n10. Checking constant expressions:")
	checkExprs := []string{
		"42",
		"1 + 2",
		"x", // Variable, not constant
	}

	for _, expr := range checkExprs {
		parsed, err := parser.ParseExpr(expr)
		if err == nil {
			isConst := cache.IsConstantExpr(parsed, nil)
			fmt.Printf("   '%s' is constant: %v\n", expr, isConst)
		}
	}

	// 11. Optimize constants in code
	fmt.Println("\n11. Constant optimization:")
	optimizeSrc := `package main
func main() {
	a := 1 + 2 + 3
	b := 10 * 2 / 5
	c := (4 + 5) * 2
}`

	optFile, _ := parser.ParseFile(fset, "optimize.go", optimizeSrc, 0)
	optimized := cache.OptimizeConstants(optFile, nil)
	fmt.Printf("   Optimized %d constant expressions\n", optimized)

	// 12. Warmup cache with common values
	fmt.Println("\n12. Cache warmup:")
	cache.WarmupCache()
	fmt.Println("   Cache warmed up with common constants")

	// Show cache statistics
	fmt.Println("\n=== Cache Statistics ===")
	stats := cache.GetStatistics()
	fmt.Printf("Eval cache size: %d\n", stats["eval_cache_size"])
	fmt.Printf("Binary cache size: %d\n", stats["binary_cache_size"])
	fmt.Printf("Unary cache size: %d\n", stats["unary_cache_size"])
	fmt.Printf("Compare cache size: %d\n", stats["compare_cache_size"])
	fmt.Printf("Shift cache size: %d\n", stats["shift_cache_size"])
	fmt.Printf("Cache hits: %d\n", stats["cache_hits"])
	fmt.Printf("Cache misses: %d\n", stats["cache_misses"])
	if stats["cache_hits"].(int64)+stats["cache_misses"].(int64) > 0 {
		fmt.Printf("Hit rate: %.2f%%\n", stats["hit_rate"])
	}
	fmt.Printf("Total evaluations: %d\n", stats["total_evaluations"])
	fmt.Printf("Binary operations: %d\n", stats["binary_operations"])
	fmt.Printf("Unary operations: %d\n", stats["unary_operations"])
	fmt.Printf("Compare operations: %d\n", stats["compare_operations"])
	fmt.Printf("Shift operations: %d\n", stats["shift_operations"])

	// Clear cache
	fmt.Println("\n13. Clearing cache:")
	cache.Clear()
	stats = cache.GetStatistics()
	fmt.Printf("   Cache size after clear: %d\n", stats["eval_cache_size"])
}
