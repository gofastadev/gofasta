package core

import (
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
	"time"
)

func TestNewConstantCache(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		cache := NewConstantCache(nil)
		if cache == nil {
			t.Fatal("Expected non-nil cache")
		}
		if cache.config == nil {
			t.Fatal("Expected non-nil config")
		}
	})
	
	t.Run("custom config", func(t *testing.T) {
		config := &ConstantCacheConfig{
			MaxCacheEntries: 5000,
			TTL:            20 * time.Minute,
			MaxPrecision:   256,
		}
		cache := NewConstantCache(config)
		if cache.config.MaxCacheEntries != 5000 {
			t.Errorf("Expected MaxCacheEntries=5000, got %d", cache.config.MaxCacheEntries)
		}
		if cache.config.MaxPrecision != 256 {
			t.Errorf("Expected MaxPrecision=256, got %d", cache.config.MaxPrecision)
		}
	})
}

func TestEvaluateBasicLiterals(t *testing.T) {
	cache := NewConstantCache(nil)
	
	tests := []struct {
		expr     string
		expected interface{}
	}{
		{"42", int64(42)},
		{"-42", int64(-42)},
		{"3.14", 3.14},
		{`"hello"`, "hello"},
		{`'a'`, int64('a')},
		{"0x10", int64(16)},
		{"0755", int64(493)},
	}
	
	for _, test := range tests {
		expr, err := parser.ParseExpr(test.expr)
		if err != nil {
			t.Fatal(err)
		}
		
		val, ok := cache.EvaluateExpr(expr, nil)
		if !ok {
			t.Errorf("Failed to evaluate %s", test.expr)
			continue
		}
		
		switch expected := test.expected.(type) {
		case int64:
			if i, ok := constant.Int64Val(val); !ok || i != expected {
				t.Errorf("Expected %d, got %v", expected, val)
			}
		case float64:
			if f, ok := constant.Float64Val(val); !ok || f != expected {
				t.Errorf("Expected %f, got %v", expected, val)
			}
		case string:
			if s := constant.StringVal(val); s != expected {
				t.Errorf("Expected %q, got %q", expected, s)
			}
		}
	}
}

func TestEvaluateBinaryExpressions(t *testing.T) {
	cache := NewConstantCache(nil)
	
	tests := []struct {
		expr     string
		expected int64
	}{
		{"1 + 2", 3},
		{"5 - 3", 2},
		{"4 * 3", 12},
		{"10 / 2", 5},
		{"10 % 3", 1},
		{"1 << 3", 8},
		{"8 >> 2", 2},
		{"5 & 3", 1},
		{"5 | 3", 7},
		{"5 ^ 3", 6},
	}
	
	for _, test := range tests {
		expr, err := parser.ParseExpr(test.expr)
		if err != nil {
			t.Fatal(err)
		}
		
		val, ok := cache.EvaluateExpr(expr, nil)
		if !ok {
			t.Errorf("Failed to evaluate %s", test.expr)
			continue
		}
		
		if i, ok := constant.Int64Val(val); !ok || i != test.expected {
			t.Errorf("Expected %d for %s, got %v", test.expected, test.expr, val)
		}
	}
}

func TestEvaluateUnaryExpressions(t *testing.T) {
	cache := NewConstantCache(nil)
	
	tests := []struct {
		expr     string
		expected interface{}
	}{
		{"-42", int64(-42)},
		{"+42", int64(42)},
		{"^5", int64(-6)},
		{"!true", false},
		{"!false", true},
	}
	
	for _, test := range tests {
		// Handle boolean literals specially
		var expr ast.Expr
		var err error
		
		if test.expr == "!true" || test.expr == "!false" {
			// Create UnaryExpr manually for boolean
			ident := &ast.Ident{Name: test.expr[1:]}
			expr = &ast.UnaryExpr{
				Op: token.NOT,
				X:  ident,
			}
			
			// For this test, treat true/false as constants
			info := &types.Info{
				Uses: map[*ast.Ident]types.Object{},
			}
			if test.expr == "!true" {
				info.Uses[ident] = types.NewConst(
					token.NoPos, nil, "true", types.Typ[types.Bool], constant.MakeBool(true))
			} else {
				info.Uses[ident] = types.NewConst(
					token.NoPos, nil, "false", types.Typ[types.Bool], constant.MakeBool(false))
			}
			
			val, ok := cache.EvaluateExpr(expr, info)
			if !ok {
				t.Errorf("Failed to evaluate %s", test.expr)
				continue
			}
			
			if b := constant.BoolVal(val); b != test.expected.(bool) {
				t.Errorf("Expected %v for %s, got %v", test.expected, test.expr, b)
			}
		} else {
			expr, err = parser.ParseExpr(test.expr)
			if err != nil {
				t.Fatal(err)
			}
			
			val, ok := cache.EvaluateExpr(expr, nil)
			if !ok {
				t.Errorf("Failed to evaluate %s", test.expr)
				continue
			}
			
			if i, ok := constant.Int64Val(val); !ok || i != test.expected.(int64) {
				t.Errorf("Expected %d for %s, got %v", test.expected, test.expr, val)
			}
		}
	}
}

func TestBinaryOp(t *testing.T) {
	cache := NewConstantCache(nil)
	
	x := constant.MakeInt64(10)
	y := constant.MakeInt64(3)
	
	tests := []struct {
		op       token.Token
		expected int64
	}{
		{token.ADD, 13},
		{token.SUB, 7},
		{token.MUL, 30},
		{token.QUO, 3},
		{token.REM, 1},
	}
	
	for _, test := range tests {
		result, ok := cache.BinaryOp(x, test.op, y)
		if !ok {
			t.Errorf("BinaryOp failed for %v", test.op)
			continue
		}
		
		if i, ok := constant.Int64Val(result); !ok || i != test.expected {
			t.Errorf("Expected %d for op %v, got %v", test.expected, test.op, result)
		}
	}
	
	// Test caching
	stats := cache.GetStatistics()
	initialOps := stats["binary_operations"].(int64)
	
	// Repeat an operation - should hit cache
	cache.BinaryOp(x, token.ADD, y)
	
	stats = cache.GetStatistics()
	if stats["cache_hits"].(int64) == 0 {
		t.Error("Expected cache hit for repeated operation")
	}
	if stats["binary_operations"].(int64) != initialOps {
		t.Error("Binary operations count should not increase for cached operation")
	}
}

func TestUnaryOp(t *testing.T) {
	cache := NewConstantCache(nil)
	
	tests := []struct {
		op       token.Token
		operand  constant.Value
		expected interface{}
	}{
		{token.SUB, constant.MakeInt64(42), int64(-42)},
		{token.ADD, constant.MakeInt64(42), int64(42)},
		{token.XOR, constant.MakeInt64(5), int64(-6)},
		{token.NOT, constant.MakeBool(true), false},
	}
	
	for _, test := range tests {
		result, ok := cache.UnaryOp(test.op, test.operand)
		if !ok {
			t.Errorf("UnaryOp failed for %v", test.op)
			continue
		}
		
		switch expected := test.expected.(type) {
		case int64:
			if i, ok := constant.Int64Val(result); !ok || i != expected {
				t.Errorf("Expected %d, got %v", expected, result)
			}
		case bool:
			if b := constant.BoolVal(result); b != expected {
				t.Errorf("Expected %v, got %v", expected, b)
			}
		}
	}
}

func TestCompare(t *testing.T) {
	cache := NewConstantCache(nil)
	
	x := constant.MakeInt64(10)
	y := constant.MakeInt64(20)
	
	tests := []struct {
		op       token.Token
		expected bool
	}{
		{token.EQL, false},
		{token.NEQ, true},
		{token.LSS, true},
		{token.LEQ, true},
		{token.GTR, false},
		{token.GEQ, false},
	}
	
	for _, test := range tests {
		result, ok := cache.Compare(x, test.op, y)
		if !ok {
			t.Errorf("Compare failed for %v", test.op)
			continue
		}
		
		if result != test.expected {
			t.Errorf("Expected %v for %v, got %v", test.expected, test.op, result)
		}
	}
}

func TestShift(t *testing.T) {
	cache := NewConstantCache(nil)
	
	x := constant.MakeInt64(8)
	
	tests := []struct {
		op       token.Token
		shift    uint
		expected int64
	}{
		{token.SHL, 2, 32},
		{token.SHR, 2, 2},
	}
	
	for _, test := range tests {
		result, ok := cache.Shift(x, test.op, test.shift)
		if !ok {
			t.Errorf("Shift failed for %v", test.op)
			continue
		}
		
		if i, ok := constant.Int64Val(result); !ok || i != test.expected {
			t.Errorf("Expected %d for shift %v by %d, got %v", 
				test.expected, test.op, test.shift, result)
		}
	}
}

func TestMakeFunctions(t *testing.T) {
	cache := NewConstantCache(nil)
	
	t.Run("MakeInt64", func(t *testing.T) {
		val := cache.MakeInt64(42)
		if i, ok := constant.Int64Val(val); !ok || i != 42 {
			t.Errorf("Expected 42, got %v", val)
		}
	})
	
	t.Run("MakeUint64", func(t *testing.T) {
		val := cache.MakeUint64(42)
		if u, ok := constant.Uint64Val(val); !ok || u != 42 {
			t.Errorf("Expected 42, got %v", val)
		}
	})
	
	t.Run("MakeFloat64", func(t *testing.T) {
		val := cache.MakeFloat64(3.14)
		if f, ok := constant.Float64Val(val); !ok || f != 3.14 {
			t.Errorf("Expected 3.14, got %v", val)
		}
	})
	
	t.Run("MakeString", func(t *testing.T) {
		val := cache.MakeString("hello")
		if s := constant.StringVal(val); s != "hello" {
			t.Errorf("Expected 'hello', got %q", s)
		}
	})
	
	t.Run("MakeBool", func(t *testing.T) {
		val := cache.MakeBool(true)
		if b := constant.BoolVal(val); !b {
			t.Error("Expected true, got false")
		}
	})
}

func TestValueConversions(t *testing.T) {
	cache := NewConstantCache(nil)
	
	t.Run("ToFloat64", func(t *testing.T) {
		intVal := constant.MakeInt64(42)
		f, ok := cache.ToFloat64(intVal)
		if !ok || f != 42.0 {
			t.Errorf("Expected 42.0, got %f", f)
		}
		
		floatVal := constant.MakeFloat64(3.14)
		f, ok = cache.ToFloat64(floatVal)
		if !ok || f != 3.14 {
			t.Errorf("Expected 3.14, got %f", f)
		}
	})
	
	t.Run("ToInt64", func(t *testing.T) {
		intVal := constant.MakeInt64(42)
		i, ok := cache.ToInt64(intVal)
		if !ok || i != 42 {
			t.Errorf("Expected 42, got %d", i)
		}
		
		floatVal := constant.MakeFloat64(42.0)
		i, ok = cache.ToInt64(floatVal)
		if !ok || i != 42 {
			t.Errorf("Expected 42, got %d", i)
		}
		
		// Non-integer float should fail
		floatVal = constant.MakeFloat64(3.14)
		_, ok = cache.ToInt64(floatVal)
		if ok {
			t.Error("Expected conversion to fail for non-integer float")
		}
	})
	
	t.Run("ToString", func(t *testing.T) {
		strVal := constant.MakeString("hello")
		s := cache.ToString(strVal)
		if s != "hello" {
			t.Errorf("Expected 'hello', got %q", s)
		}
		
		boolVal := constant.MakeBool(true)
		s = cache.ToString(boolVal)
		if s != "true" {
			t.Errorf("Expected 'true', got %q", s)
		}
		
		intVal := constant.MakeInt64(42)
		s = cache.ToString(intVal)
		if s != "42" {
			t.Errorf("Expected '42', got %q", s)
		}
	})
}

func TestBatchEvaluate(t *testing.T) {
	cache := NewConstantCache(&ConstantCacheConfig{
		ConcurrentEval: true,
		EvalWorkers:   2,
	})
	
	exprs := []string{"1 + 2", "3 * 4", "10 / 2", "7 - 3"}
	expected := []int64{3, 12, 5, 4}
	
	var parsedExprs []ast.Expr
	for _, expr := range exprs {
		parsed, err := parser.ParseExpr(expr)
		if err != nil {
			t.Fatal(err)
		}
		parsedExprs = append(parsedExprs, parsed)
	}
	
	results := cache.BatchEvaluate(parsedExprs, nil)
	
	if len(results) != len(expected) {
		t.Fatalf("Expected %d results, got %d", len(expected), len(results))
	}
	
	for i, result := range results {
		if result == nil {
			t.Errorf("Result %d is nil", i)
			continue
		}
		
		if val, ok := constant.Int64Val(result); !ok || val != expected[i] {
			t.Errorf("Expected %d at index %d, got %v", expected[i], i, result)
		}
	}
}

func TestFoldConstants(t *testing.T) {
	src := `package main

func main() {
	x := 1 + 2 + 3
	y := 10 * 2 / 5
	z := (4 + 5) * 2
}`
	
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	
	cache := NewConstantCache(nil)
	
	// Count constant expressions
	count := 0
	ast.Inspect(file, func(n ast.Node) bool {
		if expr, ok := n.(ast.Expr); ok {
			if _, isConst := cache.EvaluateExpr(expr, nil); isConst {
				count++
			}
		}
		return true
	})
	
	if count == 0 {
		t.Error("Expected to find constant expressions")
	}
}

func TestIsConstantExpr(t *testing.T) {
	cache := NewConstantCache(nil)
	
	tests := []struct {
		expr     string
		expected bool
	}{
		{"42", true},
		{"1 + 2", true},
		{`"hello"`, true},
		{"x", false},       // Variable, not constant
		{"f()", false},     // Function call, not constant
	}
	
	for _, test := range tests {
		expr, err := parser.ParseExpr(test.expr)
		if err != nil {
			// Some expressions might not parse (like standalone identifiers)
			continue
		}
		
		isConst := cache.IsConstantExpr(expr, nil)
		if isConst != test.expected {
			t.Errorf("IsConstantExpr(%s) = %v, expected %v", 
				test.expr, isConst, test.expected)
		}
	}
}

func TestConstantCacheEviction(t *testing.T) {
	config := &ConstantCacheConfig{
		MaxCacheEntries: 2,
		CacheBinaryOps: true,
	}
	cache := NewConstantCache(config)
	
	// Perform multiple operations to trigger eviction
	cache.BinaryOp(constant.MakeInt64(1), token.ADD, constant.MakeInt64(2))
	cache.BinaryOp(constant.MakeInt64(3), token.ADD, constant.MakeInt64(4))
	cache.BinaryOp(constant.MakeInt64(5), token.ADD, constant.MakeInt64(6))
	
	// Parse and evaluate expressions to fill eval cache
	expr1, _ := parser.ParseExpr("1 + 2")
	expr2, _ := parser.ParseExpr("3 + 4")
	expr3, _ := parser.ParseExpr("5 + 6")
	
	cache.EvaluateExpr(expr1, nil)
	cache.EvaluateExpr(expr2, nil)
	cache.EvaluateExpr(expr3, nil) // Should trigger eviction
	
	stats := cache.GetStatistics()
	evalCacheSize := stats["eval_cache_size"].(int)
	if evalCacheSize > 2 {
		t.Errorf("Expected eval cache size <= 2, got %d", evalCacheSize)
	}
}

func TestConstantWarmupCache(t *testing.T) {
	cache := NewConstantCache(nil)
	
	// Warmup cache
	cache.WarmupCache()
	
	// Check that common values are pre-cached
	// These operations should hit the cache
	zero := cache.MakeInt64(0)
	one := cache.MakeInt64(1)
	
	result, _ := cache.BinaryOp(one, token.ADD, one)
	if i, ok := constant.Int64Val(result); !ok || i != 2 {
		t.Error("Expected 1 + 1 = 2")
	}
	
	result, _ = cache.BinaryOp(one, token.MUL, zero)
	if i, ok := constant.Int64Val(result); !ok || i != 0 {
		t.Error("Expected 1 * 0 = 0")
	}
}

func TestConstantCacheClear(t *testing.T) {
	cache := NewConstantCache(nil)
	
	// Add some cache entries
	cache.BinaryOp(constant.MakeInt64(1), token.ADD, constant.MakeInt64(2))
	cache.UnaryOp(token.SUB, constant.MakeInt64(42))
	cache.Compare(constant.MakeInt64(1), token.EQL, constant.MakeInt64(1))
	
	expr, _ := parser.ParseExpr("1 + 2")
	cache.EvaluateExpr(expr, nil)
	
	// Clear cache
	cache.Clear()
	
	// Verify everything is cleared
	stats := cache.GetStatistics()
	if stats["eval_cache_size"].(int) != 0 {
		t.Error("Expected eval cache to be empty")
	}
	if stats["binary_cache_size"].(int) != 0 {
		t.Error("Expected binary cache to be empty")
	}
	if stats["cache_hits"].(int64) != 0 {
		t.Error("Expected cache hits to be 0")
	}
	if stats["cache_misses"].(int64) != 0 {
		t.Error("Expected cache misses to be 0")
	}
}

func TestComplexExpressions(t *testing.T) {
	cache := NewConstantCache(nil)
	
	tests := []struct {
		expr     string
		expected int64
	}{
		{"(1 + 2) * 3", 9},
		{"10 - 2 * 3", 4},
		{"(10 - 2) * 3", 24},
		{"1 + 2 + 3 + 4", 10},
		{"100 / 10 / 2", 5},
		{"1 << 2 | 1", 5},
		{"15 & 7 ^ 3", 4},
	}
	
	for _, test := range tests {
		expr, err := parser.ParseExpr(test.expr)
		if err != nil {
			t.Fatal(err)
		}
		
		val, ok := cache.EvaluateExpr(expr, nil)
		if !ok {
			t.Errorf("Failed to evaluate %s", test.expr)
			continue
		}
		
		if i, ok := constant.Int64Val(val); !ok || i != test.expected {
			t.Errorf("Expected %d for %s, got %v", test.expected, test.expr, val)
		}
	}
}

func TestFloatOperations(t *testing.T) {
	cache := NewConstantCache(nil)
	
	tests := []struct {
		expr     string
		expected float64
	}{
		{"3.14 + 2.86", 6.0},
		{"10.5 - 0.5", 10.0},
		{"2.5 * 4.0", 10.0},
		{"10.0 / 4.0", 2.5},
	}
	
	for _, test := range tests {
		expr, err := parser.ParseExpr(test.expr)
		if err != nil {
			t.Fatal(err)
		}
		
		val, ok := cache.EvaluateExpr(expr, nil)
		if !ok {
			t.Errorf("Failed to evaluate %s", test.expr)
			continue
		}
		
		if f, ok := constant.Float64Val(val); !ok || f != test.expected {
			t.Errorf("Expected %f for %s, got %v", test.expected, test.expr, val)
		}
	}
}

func TestStringOperations(t *testing.T) {
	cache := NewConstantCache(nil)
	
	tests := []struct {
		expr     string
		expected string
	}{
		{`"hello" + " " + "world"`, "hello world"},
		{`"test"`, "test"},
	}
	
	for _, test := range tests {
		expr, err := parser.ParseExpr(test.expr)
		if err != nil {
			t.Fatal(err)
		}
		
		val, ok := cache.EvaluateExpr(expr, nil)
		if !ok {
			t.Errorf("Failed to evaluate %s", test.expr)
			continue
		}
		
		if s := constant.StringVal(val); s != test.expected {
			t.Errorf("Expected %q for %s, got %q", test.expected, test.expr, s)
		}
	}
}