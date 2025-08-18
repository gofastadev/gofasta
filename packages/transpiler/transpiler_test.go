package transpiler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLexer tests the lexer functionality
func TestLexer(t *testing.T) {
	input := `@Controller("/api/users")
type UserController struct {
	UserService *UserService` + "`inject:\"\"`" + `
}`

	lexer := NewLexer(input)
	
	expectedTokens := []TokenType{
		DECORATOR, IDENT, LPAREN, STRING, RPAREN,
		TYPE, IDENT, STRUCT, LBRACE,
		IDENT, MULTIPLY, IDENT, STRING,
		RBRACE, EOF,
	}
	
	var actualTokens []TokenType
	for {
		token := lexer.NextToken()
		actualTokens = append(actualTokens, token.Type)
		if token.Type == EOF {
			break
		}
	}
	
	if len(actualTokens) != len(expectedTokens) {
		t.Fatalf("Expected %d tokens, got %d", len(expectedTokens), len(actualTokens))
	}
	
	for i, expected := range expectedTokens {
		if actualTokens[i] != expected {
			t.Errorf("Token %d: expected %s, got %s", i, tokenTypeNames[expected], tokenTypeNames[actualTokens[i]])
		}
	}
}

// TestParser tests the parser functionality
func TestParser(t *testing.T) {
	input := `package main

@Controller("/api/users")
type UserController struct {
	UserService *UserService` + "`inject:\"\"`" + `
}

@Get("/:id")
func GetUser(id string) {
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	
	if file.Package.Name != "main" {
		t.Errorf("Expected package 'main', got '%s'", file.Package.Name)
	}
	
	if len(file.Declarations) != 1 {
		t.Fatalf("Expected 1 declaration, got %d", len(file.Declarations))
	}
	
	controller, ok := file.Declarations[0].(*ControllerDeclaration)
	if !ok {
		t.Fatalf("Expected ControllerDeclaration, got %T", file.Declarations[0])
	}
	
	if controller.Name != "UserController" {
		t.Errorf("Expected controller name 'UserController', got '%s'", controller.Name)
	}
	
	if len(controller.Decorators) != 1 {
		t.Fatalf("Expected 1 decorator, got %d", len(controller.Decorators))
	}
	
	if controller.Decorators[0].Name != "Controller" {
		t.Errorf("Expected decorator 'Controller', got '%s'", controller.Decorators[0].Name)
	}
}

// TestCodeGeneration tests the code generation functionality
func TestCodeGeneration(t *testing.T) {
	input := `package main

@Controller("/users")
type UserController struct {
	UserService *UserService` + "`inject:\"\"`" + `
}

@Get("/:id")
func GetUser(id string) {
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	
	generator := NewCodeGenerator("main")
	goCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}
	
	// Check that generated code contains expected elements
	expectedElements := []string{
		"package main",
		"type UserController struct",
		"UserService *UserService",
		"func (c *UserController) RegisterRoutes",
		"func (c *UserController) GetUser",
	}
	
	for _, element := range expectedElements {
		if !strings.Contains(goCode, element) {
			t.Errorf("Generated code missing expected element: %s", element)
		}
	}
}

// TestTranspileFile tests the complete transpilation process
func TestTranspileFile(t *testing.T) {
	input := `package main

@Controller("/users")
type UserController struct {
	UserService *UserService` + "`inject:\"\"`" + `
}

@Get("/:id")  
func GetUser(id string) {
}`

	goCode, err := TranspileFile("test.gofa", input)
	if err != nil {
		t.Fatalf("Transpilation failed: %v", err)
	}
	
	if !strings.Contains(goCode, "package main") {
		t.Error("Generated code should contain package declaration")
	}
	
	if !strings.Contains(goCode, "type UserController struct") {
		t.Error("Generated code should contain UserController struct")
	}
}

// TestParallelTranspiler tests parallel transpilation
func TestParallelTranspiler(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()
	
	// Create test .gofa files
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

func DoSomething() string {
	return "test"
}`,
	}
	
	// Write test files
	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
	}
	
	// Create output directory
	outputDir := filepath.Join(tempDir, "output")
	
	// Setup transpiler
	opts := TranspileOptions{
		MaxWorkers:     2,
		OutputDir:      outputDir,
		FileExtension:  ".go",
		PreserveStruct: false,
		Verbose:        false,
	}
	
	transpiler := NewParallelTranspiler(opts)
	
	// Transpile directory
	ctx := context.Background()
	results, err := transpiler.TranspileDirectory(ctx, tempDir)
	if err != nil {
		t.Fatalf("Parallel transpilation failed: %v", err)
	}
	
	// Check results
	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}
	
	// Check that all files were successful
	for _, result := range results {
		if result.Error != nil {
			t.Errorf("Transpilation failed for %s: %v", result.InputPath, result.Error)
		}
		
		// Check that output file exists
		if _, err := os.Stat(result.OutputPath); os.IsNotExist(err) {
			t.Errorf("Output file does not exist: %s", result.OutputPath)
		}
	}
	
	// Check generated content
	for _, result := range results {
		content, err := os.ReadFile(result.OutputPath)
		if err != nil {
			t.Errorf("Failed to read generated file %s: %v", result.OutputPath, err)
			continue
		}
		
		contentStr := string(content)
		if !strings.Contains(contentStr, "package main") {
			t.Errorf("Generated file %s missing package declaration", result.OutputPath)
		}
	}
}

// TestTranspileStats tests statistics calculation
func TestTranspileStats(t *testing.T) {
	results := []TranspileResult{
		{
			InputPath:  "file1.gofa",
			OutputPath: "file1.go",
			Error:      nil,
			Duration:   100 * time.Millisecond,
		},
		{
			InputPath:  "file2.gofa", 
			OutputPath: "file2.go",
			Error:      nil,
			Duration:   200 * time.Millisecond,
		},
		{
			InputPath: "file3.gofa",
			Error:     fmt.Errorf("parsing error: invalid syntax"),
			Duration:  50 * time.Millisecond,
		},
	}
	
	stats := GetStats(results)
	
	if stats.TotalFiles != 3 {
		t.Errorf("Expected 3 total files, got %d", stats.TotalFiles)
	}
	
	if stats.SuccessfulFiles != 2 {
		t.Errorf("Expected 2 successful files, got %d", stats.SuccessfulFiles)
	}
	
	if stats.FailedFiles != 1 {
		t.Errorf("Expected 1 failed file, got %d", stats.FailedFiles)
	}
	
	expectedAvg := (100 + 200 + 50) * time.Millisecond / 3
	if stats.AverageDuration != expectedAvg {
		t.Errorf("Expected average duration %v, got %v", expectedAvg, stats.AverageDuration)
	}
	
	if stats.ErrorSummary["parsing_error"] != 1 {
		t.Errorf("Expected 1 parsing error, got %d", stats.ErrorSummary["parsing_error"])
	}
}

// TestDecoratorExtraction tests decorator extraction
func TestDecoratorExtraction(t *testing.T) {
	input := `@Controller("/users")
@UseGuards("auth")
@UseMiddleware("logging")
type UserController struct {}`
	
	lexer := NewLexer(input)
	parser := NewParser(lexer)
	file, err := parser.ParseFile()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	
	if len(file.Declarations) != 1 {
		t.Fatalf("Expected 1 declaration, got %d", len(file.Declarations))
	}
	
	controller := file.Declarations[0].(*ControllerDeclaration)
	
	expectedDecorators := []string{"Controller", "UseGuards", "UseMiddleware"}
	if len(controller.Decorators) != len(expectedDecorators) {
		t.Fatalf("Expected %d decorators, got %d", len(expectedDecorators), len(controller.Decorators))
	}
	
	for i, expected := range expectedDecorators {
		if controller.Decorators[i].Name != expected {
			t.Errorf("Expected decorator %s, got %s", expected, controller.Decorators[i].Name)
		}
	}
}

// TestParameterDecorators tests parameter decorator parsing
func TestParameterDecorators(t *testing.T) {
	input := `package main

type UserController struct {}

@Post("/")
func CreateUser(@Body() user User, @Param("id") id string) {}`
	
	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	
	controller := file.Declarations[0].(*ControllerDeclaration)
	method := controller.Methods[0]
	
	if len(method.Params) != 2 {
		t.Fatalf("Expected 2 parameters, got %d", len(method.Params))
	}
	
	// Check first parameter (@Body() user User)
	param1 := method.Params[0]
	if param1.Name != "user" {
		t.Errorf("Expected parameter name 'user', got '%s'", param1.Name)
	}
	
	if len(param1.Decorators) != 1 || param1.Decorators[0].Name != "Body" {
		t.Errorf("Expected @Body decorator on first parameter")
	}
	
	// Check second parameter (@Param("id") id string)
	param2 := method.Params[1]
	if param2.Name != "id" {
		t.Errorf("Expected parameter name 'id', got '%s'", param2.Name)
	}
	
	if len(param2.Decorators) != 1 || param2.Decorators[0].Name != "Param" {
		t.Errorf("Expected @Param decorator on second parameter")
	}
}

// BenchmarkLexer benchmarks the lexer performance
func BenchmarkLexer(b *testing.B) {
	input := `@Controller("/api/users")
@UseGuards("auth", "validation")
@UseMiddleware("logging", "cors")
type UserController struct {
	UserService *UserService` + "`inject:\"\"`" + `
	Logger      *Logger      ` + "`inject:\"logger\"`" + `
}

@Get("/:id")
@UseGuards("resource-owner")
func GetUser(@Param("id") id string) {}

@Post("/")
@HttpCode(201)
func CreateUser(@Body() user User) {}`
	
	b.ResetTimer()
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

// BenchmarkParser benchmarks the parser performance
func BenchmarkParser(b *testing.B) {
	input := `package main

@Controller("/api/users")
type UserController struct {
	UserService *UserService` + "`inject:\"\"`" + `
}

@Get("/:id")
func GetUser(id string) {}

@Post("/")
func CreateUser(user User) {}`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseGofaFile(input)
		if err != nil {
			b.Fatalf("Parse error: %v", err)
		}
	}
}

// BenchmarkTranspile benchmarks the complete transpilation process
func BenchmarkTranspile(b *testing.B) {
	input := `package main

@Controller("/api/users")
type UserController struct {
	UserService *UserService` + "`inject:\"\"`" + `
}

@Get("/:id")
func GetUser(id string) {}

@Post("/")
func CreateUser(user User) {}`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := TranspileFile("test.gofa", input)
		if err != nil {
			b.Fatalf("Transpile error: %v", err)
		}
	}
}