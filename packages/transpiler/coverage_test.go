package transpiler

import (
	"context"
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestFullCoverage aims for 100% code coverage
func TestFullCoverage(t *testing.T) {
	// Test all lexer token types
	t.Run("All token types", func(t *testing.T) {
		inputs := []string{
			"@Controller",                   // DECORATOR, IDENT
			`"string"`,                      // STRING
			`'char'`,                        // CHAR
			"123",                           // INT
			"123.45",                        // FLOAT
			"true false",                    // BOOLEAN
			"= + - * / %",                   // Operators
			"== != < > <= >=",               // Comparisons
			"&& || !",                       // Logical
			"; , . : ?",                     // Punctuation
			"... =>",                        // Special operators
			"( ) { } [ ]",                   // Delimiters
			"package import type struct",    // Keywords
			"interface func var const",      // More keywords
			"if else for return",            // Control keywords
			"int string bool float64 error", // Go types
			"map chan",                      // Channel/map types
			"// comment",                    // Single line comment
			"/* comment */",                 // Multi line comment
		}

		for _, input := range inputs {
			lexer := NewLexer(input)
			for {
				token := lexer.NextToken()
				if token.Type == EOF {
					break
				}
			}
		}
	})

	// Test parser with comprehensive input
	t.Run("Parser comprehensive", func(t *testing.T) {
		input := `package main

import "fmt"
import "net/http"

@Controller("/api/v1/users")
@UseGuards("auth", "ratelimit")
@UseMiddleware("logging")
type UserController struct {
	UserService *UserService ` + "`inject:\"\"`" + `
	Logger      *Logger      ` + "`inject:\"logger\"`" + `
}

@Get("/:id")
@UseGuards("resource-owner")
@UsePipes("validation")
@HttpCode(200)
func GetUser(@Param("id") id string, @Query("include") include string) {
}

@Post("")
@HttpCode(201)
@UseFilters("validation-exception")
func CreateUser(@Body() user CreateUserDTO) {
}

@Put("/:id")
func UpdateUser(@Param("id") id string, @Body() user UpdateUserDTO) {
}

@Delete("/:id")
@HttpCode(204)
func DeleteUser(@Param("id") id string) {
}

@Injectable()
type UserService struct {
	Repository *UserRepository ` + "`inject:\"\"`" + `
	Cache      *CacheService   ` + "`inject:\"cache\"`" + `
}

func FindById(id string) (*User, error) {
	return nil, nil
}

func Create(user CreateUserDTO) (*User, error) {
	return nil, nil
}

@Module({
	controllers: ["UserController", "AuthController"],
	providers: ["UserService", "AuthService", "CacheService"],
	imports: ["DatabaseModule", "AuthModule"],
	exports: ["UserService"]
})
type UserModule struct {
}`

		file, err := ParseGofaFile(input)
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		// Generate code to test code generation
		generator := NewCodeGenerator("main")
		code, err := generator.GenerateGoCode(file)
		if err != nil {
			t.Fatalf("Code generation failed: %v", err)
		}

		if !strings.Contains(code, "package main") {
			t.Error("Generated code should contain package declaration")
		}
	})

	// Test edge cases and error paths
	t.Run("Error paths", func(t *testing.T) {
		// Test lexer errors
		lexer := NewLexer("@")
		token := lexer.NextToken()
		if token.Type != DECORATOR {
			t.Error("Expected DECORATOR token")
		}

		// Test invalid characters
		lexer = NewLexer("#")
		token = lexer.NextToken()
		if token.Type != ILLEGAL {
			t.Error("Expected ILLEGAL token for invalid character")
		}

		// Test parser errors
		_, err := ParseGofaFile("invalid syntax @#@")
		if err == nil {
			t.Error("Expected parse error for invalid syntax")
		}

		// Test incomplete strings
		lexer = NewLexer(`"unclosed string`)
		for {
			token := lexer.NextToken()
			if token.Type == EOF {
				break
			}
		}

		// Test incomplete comments
		lexer = NewLexer("/* unclosed comment")
		for {
			token := lexer.NextToken()
			if token.Type == EOF {
				break
			}
		}
	})

	// Test AST walking
	t.Run("AST walking", func(t *testing.T) {
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
							Params: []*ParameterNode{
								{Name: "id", Type: "string"},
							},
						},
					},
				},
				&ServiceDeclaration{
					Name: "TestService",
					Decorators: []*DecoratorNode{
						{Name: "Injectable"},
					},
				},
				&ModuleDeclaration{
					Name: "TestModule",
					Decorators: []*DecoratorNode{
						{Name: "Module"},
					},
				},
			},
		}

		visitor := &countingVisitor{count: new(int)}
		Walk(visitor, file)

		if *visitor.count == 0 {
			t.Error("Visitor should have counted nodes")
		}
	})

	// Test parallel transpilation
	t.Run("Parallel transpilation", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create test files
		testFiles := map[string]string{
			"controller.gofa": `package main

@Controller("/test")
type TestController struct {}

@Get("/")
func Test() {}`,

			"service.gofa": `package main

@Injectable()
type TestService struct {}

func DoSomething() {}`,

			"invalid.gofa": "invalid content @#@",
		}

		for filename, content := range testFiles {
			filePath := filepath.Join(tempDir, filename)
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
		}

		opts := TranspileOptions{
			MaxWorkers:     2,
			OutputDir:      filepath.Join(tempDir, "output"),
			PreserveStruct: true,
			Verbose:        true,
		}

		transpiler := NewParallelTranspiler(opts)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		results, err := transpiler.TranspileDirectory(ctx, tempDir)
		if err != nil {
			t.Fatalf("Parallel transpilation failed: %v", err)
		}

		// Should have results for all files
		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}

		// Test statistics
		stats := GetStats(results)
		PrintStats(stats)

		// Test with files list
		filePaths := []string{
			filepath.Join(tempDir, "controller.gofa"),
			filepath.Join(tempDir, "service.gofa"),
		}

		results2, err := transpiler.TranspileFiles(ctx, filePaths)
		if err != nil {
			t.Fatalf("TranspileFiles failed: %v", err)
		}

		if len(results2) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results2))
		}
	})

	// Test code generation helpers
	t.Run("Code generation helpers", func(t *testing.T) {
		generator := NewCodeGenerator("test")

		// Test decorator utilities
		decorators := []*DecoratorNode{
			{Name: "Controller", Args: []DecoratorArg{{Value: "/api"}}},
			{Name: "UseGuards", Args: []DecoratorArg{{Value: "auth"}}},
		}

		if !generator.hasDecorator(decorators, "Controller") {
			t.Error("Should find Controller decorator")
		}

		decorator := generator.getDecorator(decorators, "Controller")
		if decorator == nil {
			t.Error("Should return Controller decorator")
		}

		value := generator.getDecoratorArgValue(decorator, 0)
		if value != "/api" {
			t.Errorf("Expected '/api', got '%s'", value)
		}

		// Test path combination
		path := generator.combineRoutePaths("/api", "/users")
		if path != "/api/users" {
			t.Errorf("Expected '/api/users', got '%s'", path)
		}

		// Test route info
		method := &MethodNode{
			Name: "GetUsers",
			Decorators: []*DecoratorNode{
				{Name: "Get", Args: []DecoratorArg{{Value: ""}}},
			},
		}

		routeInfo := generator.getRouteInfo(method)
		if routeInfo.Method != "GET" {
			t.Errorf("Expected GET method, got %s", routeInfo.Method)
		}

		// Test parameter extraction
		method.Params = []*ParameterNode{
			{
				Name: "user",
				Type: "User",
				Decorators: []*DecoratorNode{
					{Name: "Body"},
				},
			},
			{
				Name: "id",
				Type: "string",
				Decorators: []*DecoratorNode{
					{Name: "Param", Args: []DecoratorArg{{Value: "id"}}},
				},
			},
		}

		// This exercises the parameter extraction code paths
		generator.generateParameterExtraction(method)
	})

	// Test file operations
	t.Run("File operations", func(t *testing.T) {
		tempDir := t.TempDir()

		// Test file finding
		if err := os.WriteFile(filepath.Join(tempDir, "test.gofa"), []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		transpiler := NewParallelTranspiler(TranspileOptions{})
		files, err := transpiler.findGofaFiles(tempDir)
		if err != nil {
			t.Fatalf("findGofaFiles failed: %v", err)
		}

		if len(files) != 1 {
			t.Errorf("Expected 1 file, got %d", len(files))
		}

		// Test output path generation
		outputPath := transpiler.getOutputPath(tempDir, files[0])
		if !strings.HasSuffix(outputPath, ".go") {
			t.Error("Output path should have .go extension")
		}

		// Test single file output path
		singlePath := transpiler.getOutputPathForFile(files[0])
		if !strings.HasSuffix(singlePath, ".go") {
			t.Error("Single file output path should have .go extension")
		}
	})

	// Test CLI components (without the problematic parts)
	t.Run("CLI components", func(t *testing.T) {
		cli := NewCLI("test-version")

		// Test version command
		err := cli.versionCommand([]string{})
		if err != nil {
			t.Errorf("Version command failed: %v", err)
		}

		// Test usage printing
		cli.printUsage()

		// Test batch transpiler
		opts := TranspileOptions{MaxWorkers: 1}
		batchTranspiler := NewBatchTranspiler(opts)
		batchTranspiler.Stop()

		// Test watch mode
		watchMode := NewWatchMode(opts, ".", 100*time.Millisecond)
		watchMode.Stop()
	})

	// Test decorator type helpers
	t.Run("Decorator types", func(t *testing.T) {
		// Test all decorator type classifications
		decoratorTests := []struct {
			name     string
			expected DecoratorType
		}{
			{"Controller", ControllerDecorator},
			{"Injectable", InjectableDecorator},
			{"Get", GetDecorator},
			{"Post", PostDecorator},
			{"Body", BodyDecorator},
			{"Param", ParamDecorator},
			{"UseGuards", UseGuardsDecorator},
			{"UnknownDecorator", CustomDecorator},
		}

		for _, tt := range decoratorTests {
			result := GetDecoratorType(tt.name)
			if result != tt.expected {
				t.Errorf("GetDecoratorType(%s) = %v, expected %v", tt.name, result, tt.expected)
			}
		}

		// Test classification functions
		if !IsRouteDecorator(GetDecorator) {
			t.Error("Get should be a route decorator")
		}
		if !IsParameterDecorator(BodyDecorator) {
			t.Error("Body should be a parameter decorator")
		}
		if !IsClassDecorator(ControllerDecorator) {
			t.Error("Controller should be a class decorator")
		}
		if !IsCrossCuttingDecorator(UseGuardsDecorator) {
			t.Error("UseGuards should be a cross-cutting decorator")
		}
	})

	// Test tokenization utilities
	t.Run("Tokenization utilities", func(t *testing.T) {
		input := `@Controller("/test")
type TestController struct {}`

		tokens, err := TokenizeFile(input)
		if err != nil {
			t.Fatalf("TokenizeFile failed: %v", err)
		}

		if !IsValidGoFastaFile(tokens) {
			t.Error("Should be valid Gofasta file")
		}

		decoratorTokens := FindDecoratorTokens(tokens)
		if len(decoratorTokens) == 0 {
			t.Error("Should find decorator tokens")
		}

		filteredTokens := FilterTokens(tokens, DECORATOR)
		if len(filteredTokens) == 0 {
			t.Error("Should find filtered tokens")
		}
	})
}
