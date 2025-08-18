package transpiler

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestLexerEdgeCasesComprehensive tests lexer edge cases
func TestLexerEdgeCasesComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected func(*testing.T, []Token)
	}{
		{
			name:  "Unicode characters",
			input: "ηλλο κόσμε", // Greek text
			expected: func(t *testing.T, tokens []Token) {
				// Should tokenize as identifiers
				if len(tokens) < 3 { // At least 2 idents + EOF
					t.Errorf("Expected at least 3 tokens, got %d", len(tokens))
				}
			},
		},
		{
			name:  "Very long identifier",
			input: strings.Repeat("a", 1000),
			expected: func(t *testing.T, tokens []Token) {
				if len(tokens) != 2 { // IDENT + EOF
					t.Errorf("Expected 2 tokens, got %d", len(tokens))
				}
				if tokens[0].Type != IDENT {
					t.Errorf("Expected IDENT, got %s", tokenTypeNames[tokens[0].Type])
				}
				if len(tokens[0].Literal) != 1000 {
					t.Errorf("Expected literal length 1000, got %d", len(tokens[0].Literal))
				}
			},
		},
		{
			name:  "Multiple dots",
			input: "...",
			expected: func(t *testing.T, tokens []Token) {
				if tokens[0].Type != SPREAD {
					t.Errorf("Expected SPREAD, got %s", tokenTypeNames[tokens[0].Type])
				}
			},
		},
		{
			name:  "Partial dots",
			input: "..",
			expected: func(t *testing.T, tokens []Token) {
				if tokens[0].Type != PERIOD || tokens[1].Type != PERIOD {
					t.Errorf("Expected two PERIOD tokens")
				}
			},
		},
		{
			name:  "String with newlines",
			input: `"line1\nline2\nline3"`,
			expected: func(t *testing.T, tokens []Token) {
				if tokens[0].Type != STRING {
					t.Errorf("Expected STRING, got %s", tokenTypeNames[tokens[0].Type])
				}
			},
		},
		{
			name:  "Empty string",
			input: `""`,
			expected: func(t *testing.T, tokens []Token) {
				if tokens[0].Type != STRING {
					t.Errorf("Expected STRING, got %s", tokenTypeNames[tokens[0].Type])
				}
				if tokens[0].Literal != "" {
					t.Errorf("Expected empty literal, got %q", tokens[0].Literal)
				}
			},
		},
		{
			name:  "Numbers with underscores",
			input: "1_000_000",
			expected: func(t *testing.T, tokens []Token) {
				// Go-style number separators
				if tokens[0].Type != INT {
					t.Errorf("Expected INT, got %s", tokenTypeNames[tokens[0].Type])
				}
			},
		},
		{
			name:  "Decimal numbers",
			input: "123.456",
			expected: func(t *testing.T, tokens []Token) {
				if tokens[0].Type != FLOAT {
					t.Errorf("Expected FLOAT, got %s", tokenTypeNames[tokens[0].Type])
				}
			},
		},
		{
			name:  "Scientific notation",
			input: "1.23e10",
			expected: func(t *testing.T, tokens []Token) {
				// Should be parsed as multiple tokens unless we add support
				if len(tokens) < 3 {
					t.Errorf("Expected at least 3 tokens for scientific notation")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			var tokens []Token

			for {
				token := lexer.NextToken()
				tokens = append(tokens, token)
				if token.Type == EOF {
					break
				}
			}

			tt.expected(t, tokens)
		})
	}
}

// TestParserEdgeCases tests parser edge cases
func TestParserEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
		validate    func(*testing.T, *GofaFile, error)
	}{
		{
			name:        "Empty package name",
			input:       "package",
			shouldError: true,
			validate: func(t *testing.T, file *GofaFile, err error) {
				if err == nil {
					t.Error("Expected error for empty package name")
				}
			},
		},
		{
			name: "Multiple packages",
			input: `package main
package other`,
			shouldError: true, // Multiple package declarations are invalid
			validate: func(t *testing.T, file *GofaFile, err error) {
				if err != nil && file != nil && file.Package != nil && file.Package.Name == "main" {
					// This is acceptable - parser takes first package
				}
				// If file is nil due to error, that's also acceptable for this edge case
			},
		},
		{
			name: "Decorator without target",
			input: `package main
@Controller("/test")`,
			shouldError: false,
			validate: func(t *testing.T, file *GofaFile, err error) {
				if len(file.Decorators) != 1 {
					t.Errorf("Expected 1 file-level decorator, got %d", len(file.Decorators))
				}
			},
		},
		{
			name: "Nested decorators",
			input: `package main

@Controller("/outer")
@UseGuards("auth")
type OuterController struct {
	@Injectable()
	service *Service
}`,
			shouldError: false,
			validate: func(t *testing.T, file *GofaFile, err error) {
				if len(file.Declarations) != 1 {
					t.Errorf("Expected 1 declaration, got %d", len(file.Declarations))
				}
			},
		},
		{
			name: "Complex types",
			input: `package main

type ComplexController struct {
	MapField    map[string]interface{}
	SliceField  [][]string
	ChanField   chan<- int
	FuncField   func(string) error
	PointerPtr  **User
}`,
			shouldError: false,
			validate: func(t *testing.T, file *GofaFile, err error) {
				controller := file.Declarations[0].(*ControllerDeclaration)
				if len(controller.Fields) != 5 {
					t.Errorf("Expected 5 fields, got %d", len(controller.Fields))
				}
			},
		},
		{
			name: "Method with complex parameters",
			input: `package main

type TestController struct {}

func ComplexMethod(
	a string,
	b []interface{},
	c map[string]int,
	d func(string) error,
	e chan bool,
) (result []string, err error) {}`,
			shouldError: false,
			validate: func(t *testing.T, file *GofaFile, err error) {
				controller := file.Declarations[0].(*ControllerDeclaration)
				if len(controller.Methods) != 1 {
					t.Errorf("Expected 1 method, got %d", len(controller.Methods))
				}
				method := controller.Methods[0]
				if len(method.Params) != 5 {
					t.Errorf("Expected 5 parameters, got %d", len(method.Params))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := ParseGofaFile(tt.input)
			
			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
			}
			
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, file, err)
			}
		})
	}
}

// TestCodeGenerationEdgeCases tests code generation edge cases
func TestCodeGenerationEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		file     *GofaFile
		contains []string
		notContains []string
	}{
		{
			name: "Empty controller",
			file: &GofaFile{
				Package: &ast.Ident{Name: "main"},
				Declarations: []GofaDeclaration{
					&ControllerDeclaration{
						Name:       "EmptyController",
						Decorators: []*DecoratorNode{},
						Fields:     []*FieldNode{},
						Methods:    []*MethodNode{},
					},
				},
			},
			contains: []string{
				"package main",
				"type EmptyController struct",
				"func (c *EmptyController) RegisterRoutes",
			},
		},
		{
			name: "Service without Injectable",
			file: &GofaFile{
				Package: &ast.Ident{Name: "main"},
				Declarations: []GofaDeclaration{
					&ServiceDeclaration{
						Name:       "PlainService",
						Decorators: []*DecoratorNode{},
						Fields:     []*FieldNode{},
						Methods:    []*MethodNode{},
					},
				},
			},
			contains: []string{
				"package main",
				"type PlainService struct",
			},
			notContains: []string{
				"func (s *PlainService) Initialize",
			},
		},
		{
			name: "Module with empty configuration",
			file: &GofaFile{
				Package: &ast.Ident{Name: "main"},
				Declarations: []GofaDeclaration{
					&ModuleDeclaration{
						Name: "EmptyModule",
						Decorators: []*DecoratorNode{
							{
								Name: "Module",
								Args: []DecoratorArg{},
							},
						},
					},
				},
			},
			contains: []string{
				"package main",
				"type EmptyModule struct",
				"core.BaseModule",
				"func (m *EmptyModule) Configure",
			},
		},
		{
			name: "Controller with method naming edge cases",
			file: &GofaFile{
				Package: &ast.Ident{Name: "main"},
				Declarations: []GofaDeclaration{
					&ControllerDeclaration{
						Name:       "TestController",
						Decorators: []*DecoratorNode{},
						Fields:     []*FieldNode{},
						Methods: []*MethodNode{
							{
								Name: "getUserByID", // lowercase start
								Decorators: []*DecoratorNode{
									{Name: "Get", Args: []DecoratorArg{{Value: "/:id"}}},
								},
							},
							{
								Name: "GETALLCAPS", // all caps
								Decorators: []*DecoratorNode{
									{Name: "Get", Args: []DecoratorArg{{Value: "/all"}}},
								},
							},
						},
					},
				},
			},
			contains: []string{
				"func (c *TestController) getUserByID",
				"func (c *TestController) GETALLCAPS",
				"server.Get(\"/all\", c.GETALLCAPS)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewCodeGenerator("main")
			code, err := generator.GenerateGoCode(tt.file)
			if err != nil {
				t.Fatalf("Code generation failed: %v", err)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(code, expected) {
					t.Errorf("Generated code missing: %s\nGenerated:\n%s", expected, code)
				}
			}

			for _, notExpected := range tt.notContains {
				if strings.Contains(code, notExpected) {
					t.Errorf("Generated code should not contain: %s", notExpected)
				}
			}
		})
	}
}

// TestASTWalkEdgeCases tests AST walking edge cases
func TestASTWalkEdgeCases(t *testing.T) {
	// Create a complex AST
	file := &GofaFile{
		Package: &ast.Ident{Name: "test"},
		Declarations: []GofaDeclaration{
			&ControllerDeclaration{
				Name: "TestController",
				Decorators: []*DecoratorNode{
					{Name: "Controller", Args: []DecoratorArg{{Value: "/test"}}},
				},
				Fields: []*FieldNode{
					{Name: "Service", Type: "*TestService"},
				},
				Methods: []*MethodNode{
					{
						Name: "TestMethod",
						Decorators: []*DecoratorNode{
							{Name: "Get", Args: []DecoratorArg{{Value: "/"}}},
						},
						Params: []*ParameterNode{
							{
								Name: "id",
								Type: "string",
								Decorators: []*DecoratorNode{
									{Name: "Param", Args: []DecoratorArg{{Value: "id"}}},
								},
							},
						},
					},
				},
			},
		},
	}

	// Test visitor that collects all node types
	nodeTypes := make(map[string]int)
	visitor := &typeCountingVisitor{counts: nodeTypes}

	Walk(visitor, file)

	expectedTypes := []string{
		"*transpiler.GofaFile",
		"*transpiler.ControllerDeclaration", 
		"*transpiler.DecoratorNode",
		"*transpiler.FieldNode",
		"*transpiler.MethodNode",
		"*transpiler.ParameterNode",
	}

	for _, expectedType := range expectedTypes {
		if nodeTypes[expectedType] == 0 {
			t.Errorf("Expected to visit %s but didn't", expectedType)
		}
	}

	// Test visitor that returns nil (stops walking)
	stopVisitor := &stopWalkingVisitor{}
	Walk(stopVisitor, file)
	// Should not panic
}

type typeCountingVisitor struct {
	counts map[string]int
}

func (v *typeCountingVisitor) Visit(node GofaASTNode) Visitor {
	nodeType := reflect.TypeOf(node).String()
	v.counts[nodeType]++
	return v
}

type stopWalkingVisitor struct{}

func (v *stopWalkingVisitor) Visit(node GofaASTNode) Visitor {
	return nil // Stop walking
}

// TestParserErrorRecovery tests parser error recovery
func TestParserErrorRecovery(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Missing closing brace",
			input: "type TestController struct { field string",
		},
		{
			name:  "Invalid decorator syntax",
			input: "@Controller(/invalid/path)",
		},
		{
			name:  "Malformed function",
			input: "func InvalidFunction(",
		},
		{
			name:  "Incomplete import",
			input: "import",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			parser := NewParser(lexer)
			
			_, err := parser.ParseFile()
			if err == nil {
				t.Error("Expected parsing error but got none")
			}

			// Verify parser collected errors
			if len(parser.Errors()) == 0 {
				t.Error("Expected parser errors to be collected")
			}
		})
	}
}

// TestLexerPositionTracking tests position tracking in lexer
func TestLexerPositionTracking(t *testing.T) {
	input := `line1
line2
line3`

	lexer := NewLexer(input)
	
	var tokens []Token
	for {
		token := lexer.NextToken()
		tokens = append(tokens, token)
		if token.Type == EOF {
			break
		}
	}

	// Verify line numbers are tracked correctly
	expectedLines := []int{1, 2, 3, 3}
	for i, token := range tokens {
		if i < len(expectedLines) && token.Line != expectedLines[i] {
			t.Errorf("Token %d: expected line %d, got %d", 
				i, expectedLines[i], token.Line)
		}
	}
}

// TestCodeGeneratorHelperEdgeCases tests code generator helper edge cases
func TestCodeGeneratorHelperEdgeCases(t *testing.T) {
	generator := NewCodeGenerator("test")

	// Test empty decorator lists
	if generator.hasDecorator([]*DecoratorNode{}, "NonExistent") {
		t.Error("Should not find decorator in empty list")
	}

	decorator := generator.getDecorator([]*DecoratorNode{}, "NonExistent")
	if decorator != nil {
		t.Error("Should return nil for non-existent decorator")
	}

	// Test decorator with no arguments
	emptyDecorator := &DecoratorNode{Name: "Empty", Args: []DecoratorArg{}}
	argValue := generator.getDecoratorArgValue(emptyDecorator, 0)
	if argValue != "" {
		t.Errorf("Expected empty string for missing arg, got %q", argValue)
	}

	// Test path combination edge cases
	tests := []struct {
		controller string
		method     string
		expected   string
	}{
		{"", "", "/"},
		{"/", "/", "/"},
		{"/api/", "/users/", "/api/users"},
		{"/api", "", "/api"},
		{"", "/users", "/users"},
		{"/api//", "//users//", "/api/users"},
	}

	for _, tt := range tests {
		result := generator.combineRoutePaths(tt.controller, tt.method)
		if result != tt.expected {
			t.Errorf("combineRoutePaths(%q, %q) = %q, expected %q",
				tt.controller, tt.method, result, tt.expected)
		}
	}

	// Test route info with no decorators
	method := &MethodNode{
		Name:       "UnknownMethod",
		Decorators: []*DecoratorNode{},
	}

	routeInfo := generator.getRouteInfo(method)
	if routeInfo.Method != "" {
		t.Errorf("Expected empty method for undecorated method, got %q", routeInfo.Method)
	}

	// Test route info with method naming convention
	conventionMethod := &MethodNode{
		Name:       "GetUserProfile",
		Decorators: []*DecoratorNode{},
	}

	routeInfo = generator.getRouteInfo(conventionMethod)
	if routeInfo.Method != "GET" {
		t.Errorf("Expected GET for GetUserProfile, got %q", routeInfo.Method)
	}
	if routeInfo.Path != "/userprofile" {
		t.Errorf("Expected /userprofile path, got %q", routeInfo.Path)
	}
}

// TestTranspileFileCornerCases tests corner cases in file transpilation
func TestTranspileFileCornerCases(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		filename    string
		content     string
		shouldError bool
	}{
		{
			name:        "Very large file",
			filename:    "large.gofa",
			content:     strings.Repeat("@Controller(\"/test\")\ntype TestController struct {}\n", 1000),
			shouldError: false,
		},
		{
			name:        "File with only comments",
			filename:    "comments.gofa",
			content:     "// Just comments\n/* More comments */",
			shouldError: true, // No actual code
		},
		{
			name:        "Binary data",
			filename:    "binary.gofa",
			content:     string([]byte{0x00, 0x01, 0x02, 0xFF}),
			shouldError: true,
		},
		{
			name:     "Deeply nested structure",
			filename: "nested.gofa",
			content: `package main

@Controller("/level1")
type Level1Controller struct {
	Level2 struct {
		Level3 struct {
			Level4 struct {
				Level5 string
			}
		}
	}
}`,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.filename)
			if err := os.WriteFile(filePath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			_, err := TranspileFile(filePath, tt.content)
			
			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
			}
			
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestMemoryAndResourceManagement tests resource management
func TestMemoryAndResourceManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource management test in short mode")
	}

	// Test with many small files
	tempDir := t.TempDir()
	
	fileCount := 100
	for i := 0; i < fileCount; i++ {
		content := fmt.Sprintf(`package main

@Controller("/test%d")
type TestController%d struct {}

@Get("/")
func Test() {}`, i, i)

		filename := filepath.Join(tempDir, fmt.Sprintf("test%d.gofa", i))
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Transpile all files
	opts := TranspileOptions{
		MaxWorkers: 4,
		OutputDir:  filepath.Join(tempDir, "output"),
		Verbose:    false,
	}

	batchTranspiler := NewBatchTranspiler(opts)
	err := batchTranspiler.TranspileProject(tempDir)
	if err != nil {
		t.Fatalf("Batch transpilation failed: %v", err)
	}

	// Verify all output files exist
	for i := 0; i < fileCount; i++ {
		outputFile := filepath.Join(tempDir, "output", fmt.Sprintf("test%d.go", i))
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Errorf("Output file missing: %s", outputFile)
		}
	}
}

// TestThreadSafety tests thread safety
func TestThreadSafety(t *testing.T) {
	input := `package main

@Controller("/test")
type TestController struct {}

@Get("/")
func Test() {}`

	// Run concurrent transpilations
	done := make(chan error, 20)
	
	for i := 0; i < 20; i++ {
		go func(id int) {
			_, err := TranspileFile(fmt.Sprintf("test%d.gofa", id), input)
			done <- err
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < 20; i++ {
		if err := <-done; err != nil {
			t.Errorf("Concurrent transpilation %d failed: %v", i, err)
		}
	}
}

// TestTokenTypeConstants tests token type constants
func TestTokenTypeConstants(t *testing.T) {
	// Verify all token types have names
	for tokenType := ILLEGAL; tokenType <= GO_ERROR; tokenType++ {
		if _, exists := tokenTypeNames[tokenType]; !exists {
			t.Errorf("Token type %d missing from tokenTypeNames", int(tokenType))
		}
	}

	// Verify keywords mapping
	for keyword, tokenType := range keywords {
		if tokenType < PACKAGE {
			t.Errorf("Keyword %s has invalid token type %d", keyword, int(tokenType))
		}
	}

	// Test lookupIdent function
	if lookupIdent("package") != PACKAGE {
		t.Error("lookupIdent failed for 'package'")
	}
	
	if lookupIdent("randomIdentifier") != IDENT {
		t.Error("lookupIdent should return IDENT for unknown identifiers")
	}
}

// TestDecoratorTypeHelpers tests decorator type helper functions
func TestDecoratorTypeHelpers(t *testing.T) {
	tests := []struct {
		decorator string
		isRoute   bool
		isParam   bool
		isClass   bool
		isCross   bool
	}{
		{"Get", true, false, false, false},
		{"Post", true, false, false, false},
		{"Body", false, true, false, false},
		{"Param", false, true, false, false},
		{"Controller", false, false, true, false},
		{"Injectable", false, false, true, false},
		{"UseGuards", false, false, false, true},
		{"UseMiddleware", false, false, false, true},
		{"CustomDecorator", false, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.decorator, func(t *testing.T) {
			decoratorType := GetDecoratorType(tt.decorator)
			
			if IsRouteDecorator(decoratorType) != tt.isRoute {
				t.Errorf("IsRouteDecorator mismatch for %s", tt.decorator)
			}
			
			if IsParameterDecorator(decoratorType) != tt.isParam {
				t.Errorf("IsParameterDecorator mismatch for %s", tt.decorator)
			}
			
			if IsClassDecorator(decoratorType) != tt.isClass {
				t.Errorf("IsClassDecorator mismatch for %s", tt.decorator)
			}
			
			if IsCrossCuttingDecorator(decoratorType) != tt.isCross {
				t.Errorf("IsCrossCuttingDecorator mismatch for %s", tt.decorator)
			}
		})
	}
}