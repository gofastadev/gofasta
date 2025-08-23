package transpiler

import (
	"fmt"
	"strings"
	"testing"
)

func TestRedirectDecoratorParsing(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedURL    string
		expectedStatus int
	}{
		{
			name: "Basic @Redirect with URL and status code",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Redirect("https://example.com", 302)
@Get("/old-route")
func RedirectOldRoute() {}
`,
			expectedURL:    "https://example.com",
			expectedStatus: 302,
		},
		{
			name: "Redirect with default status code",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Redirect("https://newsite.com")
@Get("/moved")
func RedirectMoved() {}
`,
			expectedURL:    "https://newsite.com",
			expectedStatus: 302, // Default
		},
		{
			name: "Redirect with 301 permanent status",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Redirect("https://permanent.com", 301)
@Get("/permanent-move")
func PermanentRedirect() {}
`,
			expectedURL:    "https://permanent.com",
			expectedStatus: 301,
		},
		{
			name: "Redirect with 307 temporary status",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Redirect("https://temporary.com", 307)
@Get("/temp-move")
func TemporaryRedirect() {}
`,
			expectedURL:    "https://temporary.com",
			expectedStatus: 307,
		},
		{
			name: "Redirect with 308 permanent status",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Redirect("https://permanent308.com", 308)
@Get("/perm308")
func Permanent308Redirect() {}
`,
			expectedURL:    "https://permanent308.com",
			expectedStatus: 308,
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

			// Check for Redirect decorator
			var foundURL string
			var foundStatus int = 302 // Default
			for _, decorator := range method.Decorators {
				if decorator.Name == "Redirect" {
					if len(decorator.Args) > 0 {
						if url, ok := decorator.Args[0].Value.(string); ok {
							foundURL = url
						}
					}
					if len(decorator.Args) > 1 {
						if status, ok := decorator.Args[1].Value.(int); ok {
							foundStatus = status
						}
					}
					break
				}
			}

			if foundURL != tt.expectedURL {
				t.Errorf("Expected URL %s, got %s", tt.expectedURL, foundURL)
			}
			if foundStatus != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, foundStatus)
			}
		})
	}
}

func TestRedirectDecoratorCodeGeneration(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedSnippets []string
	}{
		{
			name: "Basic redirect generates ctx.Redirect",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Redirect("https://example.com", 302)
@Get("/old-route")
func RedirectOldRoute() {}
`,
			expectedSnippets: []string{
				"ctx.Redirect(302, \"https://example.com\")",
			},
		},
		{
			name: "Redirect with default status code",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Redirect("https://newsite.com")
@Get("/moved")
func RedirectMoved() {}
`,
			expectedSnippets: []string{
				"ctx.Redirect(302, \"https://newsite.com\")",
			},
		},
		{
			name: "Permanent redirect with 301",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Redirect("https://permanent.com", 301)
@Get("/permanent")
func PermanentRedirect() {}
`,
			expectedSnippets: []string{
				"ctx.Redirect(301, \"https://permanent.com\")",
			},
		},
		{
			name: "Temporary redirect with 307",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Redirect("https://temp.com", 307)
@Get("/temp")
func TempRedirect() {}
`,
			expectedSnippets: []string{
				"ctx.Redirect(307, \"https://temp.com\")",
			},
		},
		{
			name: "Multiple redirect methods",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Redirect("https://site1.com", 301)
@Get("/old1")
func Redirect1() {}

@Redirect("https://site2.com", 302)
@Get("/old2")
func Redirect2() {}

@Redirect("https://site3.com")
@Get("/old3")
func Redirect3() {}
`,
			expectedSnippets: []string{
				"ctx.Redirect(301, \"https://site1.com\")",
				"ctx.Redirect(302, \"https://site2.com\")",
				"ctx.Redirect(302, \"https://site3.com\")", // Default status
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

func TestRedirectDecoratorValidation(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
		errorMsg    string
	}{
		{
			name: "Valid redirect with URL and status",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Redirect("https://example.com", 302)
@Get("/redirect")
func TestRedirect() {}
`,
			shouldError: false,
		},
		{
			name: "Valid redirect with URL only",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Redirect("https://example.com")
@Get("/redirect")
func TestRedirect() {}
`,
			shouldError: false,
		},
		{
			name: "Valid redirect with different status codes",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Redirect("https://example.com", 301)
@Get("/redirect1")
func TestRedirect1() {}

@Redirect("https://example.com", 302)
@Get("/redirect2")
func TestRedirect2() {}

@Redirect("https://example.com", 307)
@Get("/redirect3")
func TestRedirect3() {}

@Redirect("https://example.com", 308)
@Get("/redirect4")
func TestRedirect4() {}
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

func TestRedirectDecoratorTypeMapping(t *testing.T) {
	// Test that @Redirect decorator is properly mapped
	decoratorType := GetDecoratorType("Redirect")
	if decoratorType != RedirectDecorator {
		t.Errorf("Expected RedirectDecorator, got %v", decoratorType)
	}
}

func TestRedirectDecoratorWithOtherDecorators(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedSnippets []string
	}{
		{
			name: "Redirect takes precedence over HttpCode",
			input: `
package main

@Controller("/api")
type UserController struct {}

@HttpCode(201)
@Redirect("https://example.com", 302)
@Get("/redirect")
func TestRedirect() {}
`,
			expectedSnippets: []string{
				"ctx.Redirect(302, \"https://example.com\")",
			},
			// Should NOT contain HttpCode handling since redirect takes precedence
		},
		{
			name: "Redirect with parameters",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Redirect("https://example.com", 302)
@Get("/redirect/:id")
func TestRedirect(@Param("id") id string) {}
`,
			expectedSnippets: []string{
				"idValue := ctx.GetParam(\"id\")",
				"id := idValue",
				"ctx.Redirect(302, \"https://example.com\")",
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
					t.Errorf("Expected code snippet not found: %s\nGenerated code:\n%s", snippet, generatedCode)
				}
			}

			// Ensure HttpCode is not present when Redirect is used
			if strings.Contains(tt.input, "@HttpCode") && strings.Contains(tt.input, "@Redirect") {
				if strings.Contains(generatedCode, "ctx.Status(") {
					t.Errorf("HttpCode should not be generated when Redirect decorator is present")
				}
			}
		})
	}
}

func TestRedirectDecoratorComplexScenarios(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedSnippets []string
	}{
		{
			name: "API versioning redirects",
			input: `
package main

@Controller("/api/v1")
type V1Controller struct {}

@Redirect("/api/v2/users", 301)
@Get("/users")
func RedirectToV2Users() {}

@Redirect("/api/v2/posts", 301)
@Get("/posts")
func RedirectToV2Posts() {}
`,
			expectedSnippets: []string{
				"ctx.Redirect(301, \"/api/v2/users\")",
				"ctx.Redirect(301, \"/api/v2/posts\")",
			},
		},
		{
			name: "Domain migration redirects",
			input: `
package main

@Controller("/")
type MigrationController struct {}

@Redirect("https://newdomain.com/", 301)
@Get("/")
func RedirectRoot() {}

@Redirect("https://newdomain.com/about", 301)
@Get("/about")
func RedirectAbout() {}

@Redirect("https://newdomain.com/contact", 301)
@Get("/contact")
func RedirectContact() {}
`,
			expectedSnippets: []string{
				"ctx.Redirect(301, \"https://newdomain.com/\")",
				"ctx.Redirect(301, \"https://newdomain.com/about\")",
				"ctx.Redirect(301, \"https://newdomain.com/contact\")",
			},
		},
		{
			name: "Maintenance mode redirects",
			input: `
package main

@Controller("/app")
type AppController struct {}

@Redirect("/maintenance", 302)
@Get("/dashboard")
func RedirectToDashboard() {}

@Redirect("/maintenance", 302)
@Get("/settings")
func RedirectToSettings() {}
`,
			expectedSnippets: []string{
				"ctx.Redirect(302, \"/maintenance\")",
			},
		},
		{
			name: "SEO-friendly redirects",
			input: `
package main

@Controller("/")
type SEOController struct {}

@Redirect("/products", 301)
@Get("/items")
func RedirectItems() {}

@Redirect("/blog", 301)
@Get("/news")
func RedirectNews() {}

@Redirect("/support", 301)
@Get("/help")
func RedirectHelp() {}
`,
			expectedSnippets: []string{
				"ctx.Redirect(301, \"/products\")",
				"ctx.Redirect(301, \"/blog\")",
				"ctx.Redirect(301, \"/support\")",
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

func TestRedirectDecoratorStringParsing(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedURL    string
		expectedStatus int
	}{
		{
			name: "String status code parsing",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Redirect("https://example.com", "301")
@Get("/redirect")
func TestRedirect() {}
`,
			expectedURL:    "https://example.com",
			expectedStatus: 301,
		},
		{
			name: "Invalid string status code defaults to 302",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Redirect("https://example.com", "invalid")
@Get("/redirect")
func TestRedirect() {}
`,
			expectedURL:    "https://example.com",
			expectedStatus: 302, // Should default to 302 when parsing fails
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

			expectedRedirect := fmt.Sprintf("ctx.Redirect(%d, \"%s\")", tt.expectedStatus, tt.expectedURL)
			if !strings.Contains(generatedCode, expectedRedirect) {
				t.Errorf("Expected redirect not found: %s\nGenerated code:\n%s", expectedRedirect, generatedCode)
			}
		})
	}
}
