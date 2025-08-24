package codegen

import (
	"strings"
	"testing"
)

func TestHttpCodeDecoratorParsing(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedStatus int
	}{
		{
			name: "Basic @HttpCode(201) for POST",
			input: `
package main

@Controller("/api")
type UserController struct {}

@HttpCode(201)
@Post("/users")
func CreateUser() {}
`,
			expectedStatus: 201,
		},
		{
			name: "HttpCode(204) for DELETE",
			input: `
package main

@Controller("/api")
type UserController struct {}

@HttpCode(204)
@Delete("/users/:id")
func DeleteUser() {}
`,
			expectedStatus: 204,
		},
		{
			name: "HttpCode(202) for async operations",
			input: `
package main

@Controller("/api")
type TaskController struct {}

@HttpCode(202)
@Post("/tasks")
func CreateAsyncTask() {}
`,
			expectedStatus: 202,
		},
		{
			name: "HttpCode(304) for not modified",
			input: `
package main

@Controller("/api")
type CacheController struct {}

@HttpCode(304)
@Get("/data")
func GetCachedData() {}
`,
			expectedStatus: 304,
		},
		{
			name: "Method without HttpCode decorator",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Get("/users")
func GetUsers() {}
`,
			expectedStatus: 0, // No HttpCode decorator
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			parser := NewParser(lexer)
			file, err := parser.ParseFile()

			if err != nil {
				t.Fatalf("Parsing failed: %v", err)
			}

			if len(file.Declarations) == 0 {
				t.Fatal("No declarations found")
			}

			controller := file.Declarations[0].(*ControllerDeclaration)
			if len(controller.Methods) == 0 {
				t.Fatal("No methods found")
			}

			method := controller.Methods[0]
			
			// Check for HttpCode decorator
			var foundStatus int
			for _, decorator := range method.Decorators {
				if decorator.Name == "HttpCode" && len(decorator.Args) > 0 {
					if statusCode, ok := decorator.Args[0].Value.(int); ok {
						foundStatus = statusCode
						break
					}
				}
			}

			if foundStatus != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, foundStatus)
			}
		})
	}
}

func TestHttpCodeDecoratorCodeGeneration(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedSnippets []string
	}{
		{
			name: "HttpCode(201) generates ctx.Status(201)",
			input: `
package main

@Controller("/api")
type UserController struct {}

@HttpCode(201)
@Post("/users")
func CreateUser() {}
`,
			expectedSnippets: []string{
				"ctx.Status(201)",
				"ctx.JSON(201,",
			},
		},
		{
			name: "HttpCode(204) generates ctx.Status(204)",
			input: `
package main

@Controller("/api")
type UserController struct {}

@HttpCode(204)
@Delete("/users/:id")
func DeleteUser() {}
`,
			expectedSnippets: []string{
				"ctx.Status(204)",
				"ctx.JSON(204,",
			},
		},
		{
			name: "HttpCode(202) for async processing",
			input: `
package main

@Controller("/api")
type ProcessController struct {}

@HttpCode(202)
@Post("/process")
func StartAsyncProcess() {}
`,
			expectedSnippets: []string{
				"ctx.Status(202)",
				"ctx.JSON(202,",
			},
		},
		{
			name: "Method without HttpCode uses default 200",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Get("/users")
func GetUsers() {}
`,
			expectedSnippets: []string{
				"ctx.JSON(200,", // Default status code
			},
		},
		{
			name: "Multiple methods with different HttpCode values",
			input: `
package main

@Controller("/api")
type UserController struct {}

@HttpCode(201)
@Post("/users")
func CreateUser() {}

@HttpCode(204)
@Delete("/users/:id")
func DeleteUser() {}

@Get("/users")
func GetUsers() {}
`,
			expectedSnippets: []string{
				"ctx.Status(201)",
				"ctx.JSON(201,",
				"ctx.Status(204)",
				"ctx.JSON(204,",
				"ctx.JSON(200,", // Default for GetUsers
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			file, err := ParseGofaFile(tt.input)
			if err != nil {
				t.Fatalf("Parsing failed: %v", err)
			}

			// Generate code
			generator := NewCodeGenerator("main")
			generatedCode, err := generator.GenerateGoCode(file)
			if err != nil {
				t.Fatalf("Code generation failed: %v", err)
			}

			// Check for expected code snippets
			for _, snippet := range tt.expectedSnippets {
				if !strings.Contains(generatedCode, snippet) {
					t.Errorf("Expected code snippet not found: %s\nGenerated code:\n%s", snippet, generatedCode)
				}
			}
		})
	}
}

func TestHttpCodeDecoratorValidation(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		shouldError     bool
		errorMsg        string
	}{
		{
			name: "Valid HTTP status codes",
			input: `
package main

@Controller("/api")
type UserController struct {}

@HttpCode(200)
@Get("/users")
func GetUsers() {}

@HttpCode(201)
@Post("/users")
func CreateUser() {}

@HttpCode(204)
@Delete("/users/:id")
func DeleteUser() {}

@HttpCode(304)
@Get("/cached")
func GetCached() {}

@HttpCode(400)
@Get("/error")
func TriggerError() {}

@HttpCode(404)
@Get("/notfound")
func NotFound() {}

@HttpCode(500)
@Get("/servererror")
func ServerError() {}
`,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := ParseGofaFile(tt.input)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but parsing succeeded")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected parsing error: %v", err)
			}

			// Generate code to ensure no runtime errors
			generator := NewCodeGenerator("main")
			_, err = generator.GenerateGoCode(file)
			if err != nil {
				t.Fatalf("Code generation failed: %v", err)
			}
		})
	}
}

func TestHttpCodeDecoratorTypeMapping(t *testing.T) {
	// Test that @HttpCode decorator is properly mapped
	decoratorType := GetDecoratorType("HttpCode")
	if decoratorType != HttpCodeDecorator {
		t.Errorf("Expected HttpCodeDecorator, got %v", decoratorType)
	}
}

func TestHttpCodeDecoratorWithOtherDecorators(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedSnippets []string
	}{
		{
			name: "HttpCode with Catch decorator",
			input: `
package main

@Controller("/api")
type UserController struct {}

@HttpCode(201)
@Catch(ValidationError)
@Post("/users")
func CreateUser() {}
`,
			expectedSnippets: []string{
				"ctx.Status(201)",
				"ctx.JSON(201,",
				"handleCreateUserError",
			},
		},
		{
			name: "HttpCode with route parameters",
			input: `
package main

@Controller("/api")
type UserController struct {}

@HttpCode(204)
@Delete("/users/:id")
func DeleteUser(@Param("id") id string) {}
`,
			expectedSnippets: []string{
				"ctx.Status(204)",
				"idValue := ctx.GetParam(\"id\")",
				"id := idValue",
				"ctx.JSON(204,",
			},
		},
		{
			name: "HttpCode with body parameter",
			input: `
package main

@Controller("/api")
type UserController struct {}

@HttpCode(201)
@Post("/users")
func CreateUser(@Body() userData CreateUserDto) {}

type CreateUserDto struct {
	Name string
}
`,
			expectedSnippets: []string{
				"ctx.Status(201)",
				"var userData CreateUserDto",
				"ctx.ParseJSON(&userData)",
				"ctx.JSON(201,",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := ParseGofaFile(tt.input)
			if err != nil {
				t.Fatalf("Parsing failed: %v", err)
			}

			generator := NewCodeGenerator("main")
			generatedCode, err := generator.GenerateGoCode(file)
			if err != nil {
				t.Fatalf("Code generation failed: %v", err)
			}

			for _, snippet := range tt.expectedSnippets {
				if !strings.Contains(generatedCode, snippet) {
					t.Errorf("Expected code snippet not found: %s", snippet)
				}
			}
		})
	}
}

func TestHttpCodeDecoratorComplexScenarios(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedSnippets []string
	}{
		{
			name: "REST API with appropriate status codes",
			input: `
package main

@Controller("/api/v1/users")
type UserController struct {}

@Get("/")
func GetUsers() {}

@HttpCode(201)
@Post("/")
func CreateUser() {}

@Get("/:id")
func GetUser() {}

@Put("/:id")
func UpdateUser() {}

@HttpCode(204)
@Delete("/:id")
func DeleteUser() {}
`,
			expectedSnippets: []string{
				// GET endpoints use default 200
				"ctx.JSON(200,", // GetUsers and GetUser
				// POST creates resource with 201
				"ctx.Status(201)",
				"ctx.JSON(201,",
				// PUT updates with default 200
				// DELETE with 204 No Content
				"ctx.Status(204)",
				"ctx.JSON(204,",
			},
		},
		{
			name: "Async operations with 202 Accepted",
			input: `
package main

@Controller("/api/jobs")
type JobController struct {}

@HttpCode(202)
@Post("/")
func StartJob() {}

@HttpCode(202)
@Put("/:id/restart")
func RestartJob() {}

@Get("/:id/status")
func GetJobStatus() {}
`,
			expectedSnippets: []string{
				"ctx.Status(202)",
				"ctx.JSON(202,",
				"ctx.JSON(200,", // Status check uses default 200
			},
		},
		{
			name: "Caching endpoints with 304 Not Modified",
			input: `
package main

@Controller("/api/cache")
type CacheController struct {}

@HttpCode(304)
@Get("/data")
func GetCachedData() {}

@HttpCode(204)
@Delete("/cache")
func ClearCache() {}
`,
			expectedSnippets: []string{
				"ctx.Status(304)",
				"ctx.JSON(304,",
				"ctx.Status(204)",
				"ctx.JSON(204,",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := ParseGofaFile(tt.input)
			if err != nil {
				t.Fatalf("Parsing failed: %v", err)
			}

			generator := NewCodeGenerator("main")
			generatedCode, err := generator.GenerateGoCode(file)
			if err != nil {
				t.Fatalf("Code generation failed: %v", err)
			}

			for _, snippet := range tt.expectedSnippets {
				if !strings.Contains(generatedCode, snippet) {
					t.Errorf("Expected code snippet not found: %s", snippet)
				}
			}
		})
	}
}

func TestHttpCodeDecoratorStringParsing(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedStatus int
	}{
		{
			name: "String status code parsing",
			input: `
package main

@Controller("/api")
type UserController struct {}

@HttpCode("201")
@Post("/users")
func CreateUser() {}
`,
			expectedStatus: 201,
		},
		{
			name: "Invalid string status code",
			input: `
package main

@Controller("/api")
type UserController struct {}

@HttpCode("invalid")
@Post("/users")
func CreateUser() {}
`,
			expectedStatus: 0, // Should fail to parse and return 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := ParseGofaFile(tt.input)
			if err != nil {
				t.Fatalf("Parsing failed: %v", err)
			}

			generator := NewCodeGenerator("main")
			controller := file.Declarations[0].(*ControllerDeclaration)
			method := controller.Methods[0]
			
			statusCode := generator.getHttpStatusCode(method)
			
			if statusCode != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, statusCode)
			}
		})
	}
}