package transpiler

import (
	"strings"
	"testing"
)

func TestCatchDecoratorParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string // Expected error types in catch decorators
	}{
		{
			name: "Single error type",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Catch(BadRequestError)
@Get("/users")
func (c *UserController) GetUsers() {}
`,
			expected: []string{"BadRequestError"},
		},
		{
			name: "Multiple error types",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Catch(BadRequestError, ValidationError)
@Get("/users")
func (c *UserController) GetUsers() {}
`,
			expected: []string{"BadRequestError", "ValidationError"},
		},
		{
			name: "Global error handler",
			input: `
package main

@Controller("/api")
@Catch()
type UserController struct {}

@Get("/users")
func (c *UserController) GetUsers() {}
`,
			expected: []string{}, // Empty means global handler
		},
		{
			name: "Controller and method level handlers",
			input: `
package main

@Controller("/api")
@Catch(NotFoundError)
type UserController struct {}

@Catch(ValidationError)
@Post("/users")
func (c *UserController) CreateUser() {}
`,
			expected: []string{"NotFoundError", "ValidationError"},
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

			// Collect all catch decorators
			var foundErrorTypes []string

			// Check controller-level decorators
			for _, decorator := range controller.Decorators {
				if decorator.Name == "Catch" {
					for _, arg := range decorator.Args {
						if errorType, ok := arg.Value.(string); ok {
							foundErrorTypes = append(foundErrorTypes, errorType)
						}
					}
				}
			}

			// Check method-level decorators
			for _, method := range controller.Methods {
				for _, decorator := range method.Decorators {
					if decorator.Name == "Catch" {
						for _, arg := range decorator.Args {
							if errorType, ok := arg.Value.(string); ok {
								foundErrorTypes = append(foundErrorTypes, errorType)
							}
						}
					}
				}
			}

			// Verify expected error types
			if len(foundErrorTypes) != len(tt.expected) {
				t.Errorf("Expected %d error types, got %d", len(tt.expected), len(foundErrorTypes))
			}

			for i, expectedType := range tt.expected {
				if i >= len(foundErrorTypes) || foundErrorTypes[i] != expectedType {
					t.Errorf("Expected error type %s at index %d, got %s", expectedType, i, foundErrorTypes[i])
				}
			}
		})
	}
}

func TestCatchDecoratorCodeGeneration(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedSnippets []string // Code snippets we expect to find
	}{
		{
			name: "Single error type handler",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Catch(BadRequestError)
@Get("/users")
func (c *UserController) GetUsers() {}
`,
			expectedSnippets: []string{
				"func (c *UserController) handleGetUsersError(err error, ctx *httpPackage.RequestContext)",
				"case *BadRequestError:",
				"ctx.JSON(400",
				"server.RegisterMethodErrorFilter(\"GetUsers\", \"BadRequestError\"",
			},
		},
		{
			name: "Multiple error types handler",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Catch(BadRequestError, ValidationError)
@Post("/users")
func (c *UserController) CreateUser() {}
`,
			expectedSnippets: []string{
				"func (c *UserController) handleCreateUserError(err error, ctx *httpPackage.RequestContext)",
				"case *BadRequestError:",
				"case *ValidationError:",
				"ctx.JSON(422",
				"server.RegisterMethodErrorFilter(\"CreateUser\", \"BadRequestError\"",
				"server.RegisterMethodErrorFilter(\"CreateUser\", \"ValidationError\"",
			},
		},
		{
			name: "Global controller error handler",
			input: `
package main

@Controller("/api")
@Catch()
type UserController struct {}

@Get("/users")
func (c *UserController) GetUsers() {}
`,
			expectedSnippets: []string{
				"func (c *UserController) handleError(err error, ctx *httpPackage.RequestContext)",
				"// Global error handler - catches all errors",
				"server.RegisterGlobalErrorFilter(c.handleError)",
			},
		},
		{
			name: "Controller and method level handlers",
			input: `
package main

@Controller("/api")
@Catch(NotFoundError)
type UserController struct {}

@Catch(ValidationError)
@Post("/users")
func (c *UserController) CreateUser() {}
`,
			expectedSnippets: []string{
				"func (c *UserController) handleNotFoundErrorError(err error, ctx *httpPackage.RequestContext)",
				"func (c *UserController) handleCreateUserError(err error, ctx *httpPackage.RequestContext)",
				"case *NotFoundError:",
				"case *ValidationError:",
				"ctx.JSON(404",
				"ctx.JSON(422",
				"server.RegisterErrorFilter(\"NotFoundError\"",
				"server.RegisterMethodErrorFilter(\"CreateUser\", \"ValidationError\"",
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

func TestCatchDecoratorErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
		errorMsg    string
	}{
		{
			name: "Valid single error type",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Catch(BadRequestError)
@Get("/users")
func (c *UserController) GetUsers() {}
`,
			shouldError: false,
		},
		{
			name: "Valid multiple error types",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Catch(BadRequestError, ValidationError, NotFoundError)
@Get("/users")
func (c *UserController) GetUsers() {}
`,
			shouldError: false,
		},
		{
			name: "Valid global error handler",
			input: `
package main

@Controller("/api")
@Catch()
type UserController struct {}
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

func TestCatchDecoratorTypeMapping(t *testing.T) {
	// Test that @Catch decorator is properly mapped
	decoratorType := GetDecoratorType("Catch")
	if decoratorType != CatchDecorator {
		t.Errorf("Expected CatchDecorator, got %v", decoratorType)
	}

	// Test that it's recognized as an error handling decorator
	if !IsErrorHandlingDecorator(decoratorType) {
		t.Error("CatchDecorator should be recognized as an error handling decorator")
	}
}

func TestCatchDecoratorComplexScenarios(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedSnippets []string
	}{
		{
			name: "Multiple methods with different error handlers",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Catch(NotFoundError)
@Get("/users/:id")
func (c *UserController) GetUser() {}

@Catch(ValidationError)
@Post("/users")
func (c *UserController) CreateUser() {}

@Catch(UnauthorizedError)
@Delete("/users/:id")
func (c *UserController) DeleteUser() {}
`,
			expectedSnippets: []string{
				"handleGetUserError",
				"handleCreateUserError", 
				"handleDeleteUserError",
				"case *NotFoundError:",
				"case *ValidationError:",
				"case *UnauthorizedError:",
				"ctx.JSON(404",
				"ctx.JSON(422",
				"ctx.JSON(401",
			},
		},
		{
			name: "Controller with multiple error handlers",
			input: `
package main

@Controller("/api")
@Catch(InternalServerError)
type UserController struct {}

@Catch(ValidationError)
@Post("/users")
func (c *UserController) CreateUser() {}

@Catch(NotFoundError)
@Get("/users/:id")
func (c *UserController) GetUser() {}
`,
			expectedSnippets: []string{
				"handleInternalServerErrorError",
				"handleCreateUserError",
				"handleGetUserError",
				"case *InternalServerError:",
				"case *ValidationError:",
				"case *NotFoundError:",
				"ctx.JSON(500",
				"ctx.JSON(422",
				"ctx.JSON(404",
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