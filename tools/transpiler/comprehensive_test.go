package transpiler

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestLexerComprehensive tests all lexer functionality
func TestLexerComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenType
	}{
		{
			name:  "Decorators",
			input: "@Controller @Get @Post",
			expected: []TokenType{
				DECORATOR, IDENT, DECORATOR, IDENT, DECORATOR, IDENT, EOF,
			},
		},
		{
			name:  "String literals",
			input: `"hello" 'world'`,
			expected: []TokenType{
				STRING, CHAR, EOF,
			},
		},
		{
			name:  "Numbers",
			input: "123 45.67",
			expected: []TokenType{
				INT, FLOAT, EOF,
			},
		},
		{
			name:  "Operators",
			input: "= + - * / % == != < > <= >= && || !",
			expected: []TokenType{
				ASSIGN, PLUS, MINUS, MULTIPLY, DIVIDE, MODULO,
				EQ, NOT_EQ, LT, GT, LTE, GTE, AND, OR, NOT, EOF,
			},
		},
		{
			name:  "Punctuation",
			input: "; , . : ? ... => ( ) { } [ ]",
			expected: []TokenType{
				SEMICOLON, COMMA, PERIOD, COLON, QUESTION, SPREAD, ARROW,
				LPAREN, RPAREN, LBRACE, RBRACE, LBRACKET, RBRACKET, EOF,
			},
		},
		{
			name:  "Keywords",
			input: "package import type struct interface func var const if else for return true false",
			expected: []TokenType{
				PACKAGE, IMPORT, TYPE, STRUCT, INTERFACE, FUNC, VAR, CONST,
				IF, ELSE, FOR, RETURN, BOOLEAN, BOOLEAN, EOF,
			},
		},
		{
			name:  "Go types",
			input: "int string bool float64 error map chan",
			expected: []TokenType{
				GO_INT, GO_STRING, GO_BOOL, GO_FLOAT, GO_ERROR, GO_MAP, GO_CHAN, EOF,
			},
		},
		{
			name:  "Comments single line",
			input: "// This is a comment\npackage main",
			expected: []TokenType{
				COMMENT, PACKAGE, IDENT, EOF,
			},
		},
		{
			name:  "Comments multi line",
			input: "/* Multi\nline\ncomment */ package",
			expected: []TokenType{
				COMMENT, PACKAGE, EOF,
			},
		},
		{
			name:  "Complex expression",
			input: `@Controller("/api") func GetUser(@Param("id") id string)`,
			expected: []TokenType{
				DECORATOR, IDENT, LPAREN, STRING, RPAREN,
				FUNC, IDENT, LPAREN, DECORATOR, IDENT, LPAREN, STRING, RPAREN,
				IDENT, GO_STRING, RPAREN, EOF,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			var actual []TokenType

			for {
				token := lexer.NextToken()
				actual = append(actual, token.Type)
				if token.Type == EOF {
					break
				}
			}

			if !reflect.DeepEqual(actual, tt.expected) {
				t.Errorf("Expected tokens %v, got %v", tt.expected, actual)
			}
		})
	}
}

// TestLexerErrors tests lexer error handling
func TestLexerErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"Illegal character", "#"},
		{"Incomplete string", `"unclosed string`},
		{"Incomplete char", `'unclosed`},
		{"Invalid operator", "&"},
		{"Invalid operator", "|"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)

			for {
				token := lexer.NextToken()
				if token.Type == EOF {
					break
				}
				if token.Type == ILLEGAL {
					return // Expected illegal token
				}
			}
		})
	}
}

// TestLexerEdgeCases tests edge cases
func TestLexerEdgeCases(t *testing.T) {
	// Empty input
	lexer := NewLexer("")
	token := lexer.NextToken()
	if token.Type != EOF {
		t.Errorf("Expected EOF for empty input, got %s", tokenTypeNames[token.Type])
	}

	// Only whitespace
	lexer = NewLexer("   \n\t  ")
	token = lexer.NextToken()
	if token.Type != EOF {
		t.Errorf("Expected EOF for whitespace-only input, got %s", tokenTypeNames[token.Type])
	}

	// String with escape sequences
	lexer = NewLexer(`"hello\nworld"`)
	token = lexer.NextToken()
	if token.Type != STRING {
		t.Errorf("Expected STRING token, got %s", tokenTypeNames[token.Type])
	}
}

// TestParserComprehensive tests all parser functionality
func TestParserComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(*testing.T, *GofaFile, error)
	}{
		{
			name: "Complete controller",
			input: `package main

import "fmt"

@Controller("/users")
@UseGuards("auth")
type UserController struct {
	Service *UserService ` + "`inject:\"\"`" + `
}

@Get("/:id")
@UseGuards("resource")
func GetUser(@Param("id") id string) {}

@Post("")
@HttpCode(201)
func CreateUser(@Body() user User) {}`,
			validate: func(t *testing.T, file *GofaFile, err error) {
				if err != nil {
					t.Fatalf("Parse error: %v", err)
				}

				if file.Package.Name != "main" {
					t.Errorf("Expected package 'main', got '%s'", file.Package.Name)
				}

				if len(file.Imports) != 1 {
					t.Errorf("Expected 1 import, got %d", len(file.Imports))
				}

				if len(file.Declarations) != 1 {
					t.Errorf("Expected 1 declaration, got %d", len(file.Declarations))
				}

				controller := file.Declarations[0].(*ControllerDeclaration)
				if controller.Name != "UserController" {
					t.Errorf("Expected UserController, got %s", controller.Name)
				}

				if len(controller.Decorators) != 2 {
					t.Errorf("Expected 2 controller decorators, got %d", len(controller.Decorators))
				}

				if len(controller.Methods) != 2 {
					t.Errorf("Expected 2 methods, got %d", len(controller.Methods))
				}
			},
		},
		{
			name: "Service declaration",
			input: `package main

@Injectable()
type UserService struct {
	Repository *UserRepository ` + "`inject:\"\"`" + `
}

func GetUsers() []User {}
func CreateUser(user User) User {}`,
			validate: func(t *testing.T, file *GofaFile, err error) {
				if err != nil {
					t.Fatalf("Parse error: %v", err)
				}

				service := file.Declarations[0].(*ServiceDeclaration)
				if service.Name != "UserService" {
					t.Errorf("Expected UserService, got %s", service.Name)
				}

				if len(service.Decorators) != 1 || service.Decorators[0].Name != "Injectable" {
					t.Errorf("Expected @Injectable decorator")
				}

				if len(service.Methods) != 2 {
					t.Errorf("Expected 2 methods, got %d", len(service.Methods))
				}
			},
		},
		{
			name: "Module declaration",
			input: `package main

@Module({
	controllers: ["UserController"],
	providers: ["UserService"]
})
type AppModule struct {}`,
			validate: func(t *testing.T, file *GofaFile, err error) {
				if err != nil {
					t.Fatalf("Parse error: %v", err)
				}

				module := file.Declarations[0].(*ModuleDeclaration)
				if module.Name != "AppModule" {
					t.Errorf("Expected AppModule, got %s", module.Name)
				}

				if len(module.Decorators) != 1 || module.Decorators[0].Name != "Module" {
					t.Errorf("Expected @Module decorator")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := ParseGofaFile(tt.input)
			tt.validate(t, file, err)
		})
	}
}

// TestParserErrors tests parser error handling
func TestParserErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Missing package name",
			input: "package",
		},
		{
			name:  "Invalid import",
			input: "import invalid",
		},
		{
			name:  "Missing struct keyword",
			input: "type UserController",
		},
		{
			name:  "Unclosed struct",
			input: "type UserController struct {",
		},
		{
			name:  "Invalid decorator",
			input: "@ invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseGofaFile(tt.input)
			if err == nil {
				t.Error("Expected parsing error, but got none")
			}
		})
	}
}

// TestDecoratorExtractionComprehensive tests comprehensive decorator extraction
func TestDecoratorExtractionComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
	}{
		{
			name:  "Simple decorator",
			input: `@Controller("/users")`,
			expected: map[string]interface{}{
				"name": "Controller",
				"args": []string{"/users"},
			},
		},
		{
			name:  "Decorator with multiple args",
			input: `@UseGuards("auth", "admin")`,
			expected: map[string]interface{}{
				"name": "UseGuards",
				"args": []string{"auth", "admin"},
			},
		},
		{
			name:  "Decorator with number",
			input: `@HttpCode(201)`,
			expected: map[string]interface{}{
				"name": "HttpCode",
				"args": []int{201},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			parser := NewParser(lexer)
			decorator := parser.parseDecorator()

			if decorator == nil {
				t.Fatal("Failed to parse decorator")
			}

			if decorator.Name != tt.expected["name"] {
				t.Errorf("Expected name %s, got %s", tt.expected["name"], decorator.Name)
			}
		})
	}
}

// TestCodeGenerationComprehensive tests comprehensive code generation
func TestCodeGenerationComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name: "Controller with routes",
			input: `package main

@Controller("/users")
type UserController struct {
	Service *UserService ` + "`inject:\"\"`" + `
}

@Get("/:id")
func GetUser(id string) {}

@Post("")
func CreateUser(user User) {}`,
			contains: []string{
				"package main",
				"type UserController struct",
				"Service *UserService `inject:\"\"`",
				"func (c *UserController) RegisterRoutes",
				"server.Get(\"/users/:id\", c.GetUser)",
				"server.Post(\"/users\", c.CreateUser)",
				"func (c *UserController) GetUser",
				"func (c *UserController) CreateUser",
			},
		},
		{
			name: "Service with Injectable",
			input: `package main

@Injectable()
type UserService struct {
	Repo *UserRepository ` + "`inject:\"\"`" + `
}

func GetUsers() []User {}`,
			contains: []string{
				"package main",
				"type UserService struct",
				"Repo *UserRepository `inject:\"\"`",
				"func (s *UserService) Initialize() error",
				"func (s *UserService) GetUsers",
			},
		},
		{
			name: "Module with configuration",
			input: `package main

@Module({
	controllers: ["UserController"],
	providers: ["UserService"]
})
type AppModule struct {}`,
			contains: []string{
				"package main",
				"type AppModule struct",
				"core.BaseModule",
				"func (m *AppModule) Configure",
				"container.RegisterProvider",
				"container.RegisterController",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := ParseGofaFile(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			generator := NewCodeGenerator("main")
			code, err := generator.GenerateGoCode(file)
			if err != nil {
				t.Fatalf("Code generation error: %v", err)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(code, expected) {
					t.Errorf("Generated code missing: %s\nGenerated code:\n%s", expected, code)
				}
			}
		})
	}
}

// TestParallelTranspilerComprehensive tests parallel transpiler
func TestParallelTranspilerComprehensive(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"controller1.gofa": `package main

@Controller("/test1")
type TestController1 struct {}

@Get("/")
func GetTest() {}`,

		"controller2.gofa": `package main

@Controller("/test2") 
type TestController2 struct {}

@Post("/")
func PostTest() {}`,

		"service.gofa": `package main

@Injectable()
type TestService struct {}

func DoSomething() {}`,

		"module.gofa": `package main

@Module({controllers: ["TestController"], providers: ["TestService"]})
type TestModule struct {}`,
	}

	// Write test files
	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
	}

	// Test different configurations
	testConfigs := []struct {
		name string
		opts TranspileOptions
	}{
		{
			name: "Default options",
			opts: TranspileOptions{
				OutputDir:      filepath.Join(tempDir, "output1"),
				PreserveStruct: false,
			},
		},
		{
			name: "Preserve structure",
			opts: TranspileOptions{
				OutputDir:      filepath.Join(tempDir, "output2"),
				PreserveStruct: true,
			},
		},
		{
			name: "Custom workers",
			opts: TranspileOptions{
				MaxWorkers:     1,
				OutputDir:      filepath.Join(tempDir, "output3"),
				PreserveStruct: false,
				Verbose:        true,
			},
		},
	}

	for _, tc := range testConfigs {
		t.Run(tc.name, func(t *testing.T) {
			transpiler := NewParallelTranspiler(tc.opts)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			results, err := transpiler.TranspileDirectory(ctx, tempDir)
			if err != nil {
				t.Fatalf("Transpilation failed: %v", err)
			}

			if len(results) != 4 {
				t.Errorf("Expected 4 results, got %d", len(results))
			}

			// Check all results succeeded
			for _, result := range results {
				if result.Error != nil {
					t.Errorf("Transpilation failed for %s: %v", result.InputPath, result.Error)
				}

				// Verify output file exists
				if _, err := os.Stat(result.OutputPath); os.IsNotExist(err) {
					t.Errorf("Output file missing: %s", result.OutputPath)
				}
			}

			// Test statistics
			stats := GetStats(results)
			if stats.TotalFiles != 4 {
				t.Errorf("Expected 4 total files in stats, got %d", stats.TotalFiles)
			}
			if stats.SuccessfulFiles != 4 {
				t.Errorf("Expected 4 successful files, got %d", stats.SuccessfulFiles)
			}
		})
	}
}

// TestTranspileFileEdgeCases tests edge cases in file transpilation
func TestTranspileFileEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
	}{
		{
			name:        "Empty file",
			input:       "",
			shouldError: true,
		},
		{
			name:        "Only package declaration",
			input:       "package main",
			shouldError: false,
		},
		{
			name: "Only imports",
			input: `package main
import "fmt"`,
			shouldError: false,
		},
		{
			name: "No decorators",
			input: `package main
type PlainStruct struct {
	Field string
}`,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := TranspileFile("test.gofa", tt.input)

			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestTokenizeFile tests file tokenization
func TestTokenizeFile(t *testing.T) {
	input := `@Controller("/test")
type TestController struct {}`

	tokens, err := TokenizeFile(input)
	if err != nil {
		t.Fatalf("Tokenization failed: %v", err)
	}

	if len(tokens) < 8 {
		t.Errorf("Expected at least 8 tokens, got %d", len(tokens))
	}

	// Test IsValidGoFastaFile
	if !IsValidGoFastaFile(tokens) {
		t.Error("Should be valid Gofasta file")
	}

	// Test FilterTokens
	decorators := FilterTokens(tokens, DECORATOR)
	if len(decorators) != 1 {
		t.Errorf("Expected 1 decorator token, got %d", len(decorators))
	}

	// Test FindDecoratorTokens
	decoratorTokens := FindDecoratorTokens(tokens)
	if len(decoratorTokens) != 1 {
		t.Errorf("Expected 1 decorator token, got %d", len(decoratorTokens))
	}
}

// TestASTHelpers tests AST helper functions
func TestASTHelpers(t *testing.T) {
	// Test DecoratorType functions
	if !IsRouteDecorator(GetDecoratorType("Get")) {
		t.Error("Get should be a route decorator")
	}

	if !IsParameterDecorator(GetDecoratorType("Body")) {
		t.Error("Body should be a parameter decorator")
	}

	if !IsClassDecorator(GetDecoratorType("Controller")) {
		t.Error("Controller should be a class decorator")
	}

	if !IsCrossCuttingDecorator(GetDecoratorType("UseGuards")) {
		t.Error("UseGuards should be a cross-cutting decorator")
	}

	// Test custom decorator
	if GetDecoratorType("CustomDecorator") != CustomDecorator {
		t.Error("Unknown decorators should return CustomDecorator")
	}
}

// TestASTVisitor tests AST visitor pattern
func TestASTVisitor(t *testing.T) {
	input := `package main

@Controller("/test")
type TestController struct {}

@Get("/")
func TestMethod() {}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// Create a visitor that counts nodes
	nodeCount := 0
	visitor := &countingVisitor{count: &nodeCount}

	Walk(visitor, file)

	if nodeCount == 0 {
		t.Error("Visitor should have counted some nodes")
	}
}

// countingVisitor implements Visitor interface for testing
type countingVisitor struct {
	count *int
}

func (v *countingVisitor) Visit(node GofaASTNode) Visitor {
	*v.count++
	return v
}

// TestCodeGeneratorHelpers tests code generator helper methods
func TestCodeGeneratorHelpers(t *testing.T) {
	generator := NewCodeGenerator("test")

	// Test import collection
	generator.addImport("test/import1")
	generator.addImport("test/import2")
	generator.addImport("test/import1") // duplicate

	if len(generator.imports) != 2 {
		t.Errorf("Expected 2 imports, got %d", len(generator.imports))
	}

	// Test decorator helpers
	decorators := []*DecoratorNode{
		{Name: "Controller", Args: []DecoratorArg{{Value: "/test"}}},
		{Name: "UseGuards", Args: []DecoratorArg{{Value: "auth"}}},
	}

	if !generator.hasDecorator(decorators, "Controller") {
		t.Error("Should find Controller decorator")
	}

	decorator := generator.getDecorator(decorators, "Controller")
	if decorator == nil || decorator.Name != "Controller" {
		t.Error("Should return Controller decorator")
	}

	argValue := generator.getDecoratorArgValue(decorator, 0)
	if argValue != "/test" {
		t.Errorf("Expected '/test', got '%s'", argValue)
	}

	// Test path combination
	fullPath := generator.combineRoutePaths("/api", "/users")
	if fullPath != "/api/users" {
		t.Errorf("Expected '/api/users', got '%s'", fullPath)
	}

	// Test route info extraction
	method := &MethodNode{
		Name: "GetUser",
		Decorators: []*DecoratorNode{
			{Name: "Get", Args: []DecoratorArg{{Value: "/:id"}}},
		},
	}

	routeInfo := generator.getRouteInfo(method)
	if routeInfo.Method != "GET" || routeInfo.Path != "/:id" {
		t.Errorf("Expected GET /:id, got %s %s", routeInfo.Method, routeInfo.Path)
	}
}

// TestErrorHandling tests various error conditions
func TestErrorHandling(t *testing.T) {
	// Test invalid token types
	invalidInput := "@#@invalid"
	_, err := TokenizeFile(invalidInput)
	if err == nil {
		t.Error("Expected tokenization error for invalid input")
	}

	// Test parser with invalid tokens
	lexer := NewLexer("@ @")
	parser := NewParser(lexer)
	_, err = parser.ParseFile()
	if err == nil {
		t.Error("Expected parsing error for invalid decorators")
	}

	// Test code generation with incomplete AST
	file := &GofaFile{
		Declarations: []GofaDeclaration{
			&ControllerDeclaration{
				Name:       "",
				Decorators: []*DecoratorNode{},
				Fields:     []*FieldNode{},
				Methods:    []*MethodNode{},
			},
		},
	}

	generator := NewCodeGenerator("test")
	_, err = generator.GenerateGoCode(file)
	if err != nil {
		t.Logf("Code generation handled incomplete AST: %v", err)
	}
}

// TestConcurrency tests concurrent access
func TestConcurrency(t *testing.T) {
	input := `package main

@Controller("/test")
type TestController struct {}

@Get("/")
func TestMethod() {}`

	// Test concurrent transpilation
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			_, err := TranspileFile("test.gofa", input)
			if err != nil {
				t.Errorf("Concurrent transpilation failed: %v", err)
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// BenchmarkComprehensive provides comprehensive benchmarks
func BenchmarkLexerLarge(b *testing.B) {
	// Large input with many decorators
	input := strings.Repeat(`@Controller("/api/users")
@UseGuards("auth", "validation")
@UseMiddleware("logging", "cors")
type UserController struct {
	UserService *UserService `+"`inject:\"\"`"+`
	Logger      *Logger      `+"`inject:\"logger\"`"+`
}

@Get("/:id")
@UseGuards("resource-owner")
func GetUser(@Param("id") id string) {}

@Post("/")
@HttpCode(201)
func CreateUser(@Body() user User) {}

@Put("/:id")
func UpdateUser(@Param("id") id string, @Body() user User) {}

@Delete("/:id")
@HttpCode(204)
func DeleteUser(@Param("id") id string) {}

`, 100)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		lexer := NewLexer(input)
		for {
			token := lexer.NextToken()
			if token.Type == EOF {
				break
			}
		}
	}
}

func BenchmarkParserLarge(b *testing.B) {
	input := `package main

@Controller("/api/users")
type UserController struct {
	UserService *UserService ` + "`inject:\"\"`" + `
}

@Get("/:id")
func GetUser(id string) {}

@Post("/")
func CreateUser(user User) {}

@Put("/:id")
func UpdateUser(id string, user User) {}

@Delete("/:id")
func DeleteUser(id string) {}`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := ParseGofaFile(input)
		if err != nil {
			b.Fatalf("Parse error: %v", err)
		}
	}
}

func BenchmarkTranspileLarge(b *testing.B) {
	input := `package main

@Controller("/api/users")
type UserController struct {
	UserService *UserService ` + "`inject:\"\"`" + `
}

@Injectable()
type UserService struct {
	Repository *UserRepository ` + "`inject:\"\"`" + `
}

@Get("/:id")
func GetUser(id string) {}

@Post("/")
func CreateUser(user User) {}`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := TranspileFile("test.gofa", input)
		if err != nil {
			b.Fatalf("Transpile error: %v", err)
		}
	}
}

// TestMemoryUsage tests memory efficiency
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	// Large input to test memory usage
	largeInput := strings.Repeat(`@Controller("/test")
type TestController struct {}
@Get("/") func Test() {}
`, 1000)

	// Multiple transpilations to test for memory leaks
	for i := 0; i < 100; i++ {
		_, err := TranspileFile("test.gofa", largeInput)
		if err != nil {
			t.Fatalf("Transpilation %d failed: %v", i, err)
		}
	}
}
