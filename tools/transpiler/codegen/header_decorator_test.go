package codegen

import (
	"strings"
	"testing"
)

func TestHeaderDecoratorParsing(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedName   string
		expectedValue  string
	}{
		{
			name: "Basic @Header with name and value",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Header("Cache-Control", "no-cache")
@Get("/users")
func GetUsers() {}
`,
			expectedName:  "Cache-Control",
			expectedValue: "no-cache",
		},
		{
			name: "Content-Type header",
			input: `
package main

@Controller("/api")
type FileController struct {}

@Header("Content-Type", "application/pdf")
@Get("/download")
func DownloadFile() {}
`,
			expectedName:  "Content-Type",
			expectedValue: "application/pdf",
		},
		{
			name: "Custom API header",
			input: `
package main

@Controller("/api")
type APIController struct {}

@Header("X-API-Version", "v2.1")
@Get("/info")
func GetAPIInfo() {}
`,
			expectedName:  "X-API-Version",
			expectedValue: "v2.1",
		},
		{
			name: "CORS header",
			input: `
package main

@Controller("/api")
type CORSController struct {}

@Header("Access-Control-Allow-Origin", "*")
@Get("/public")
func GetPublicData() {}
`,
			expectedName:  "Access-Control-Allow-Origin",
			expectedValue: "*",
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
			
			// Check for Header decorator
			var foundName, foundValue string
			for _, decorator := range method.Decorators {
				if decorator.Name == "Header" {
					if len(decorator.Args) > 0 {
						if name, ok := decorator.Args[0].Value.(string); ok {
							foundName = name
						}
					}
					if len(decorator.Args) > 1 {
						if value, ok := decorator.Args[1].Value.(string); ok {
							foundValue = value
						}
					}
					break
				}
			}

			if foundName != tt.expectedName {
				t.Errorf("Expected header name %s, got %s", tt.expectedName, foundName)
			}
			if foundValue != tt.expectedValue {
				t.Errorf("Expected header value %s, got %s", tt.expectedValue, foundValue)
			}
		})
	}
}

func TestHeaderDecoratorCodeGeneration(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedSnippets []string
	}{
		{
			name: "Single header generates ctx.Header call",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Header("Cache-Control", "no-cache")
@Get("/users")
func GetUsers() {}
`,
			expectedSnippets: []string{
				"ctx.Header(\"Cache-Control\", \"no-cache\")",
			},
		},
		{
			name: "Multiple headers generate multiple ctx.Header calls",
			input: `
package main

@Controller("/api")
type FileController struct {}

@Header("Content-Type", "application/pdf")
@Header("Content-Disposition", "attachment; filename=report.pdf")
@Header("Cache-Control", "private")
@Get("/download")
func DownloadFile() {}
`,
			expectedSnippets: []string{
				"ctx.Header(\"Content-Type\", \"application/pdf\")",
				"ctx.Header(\"Content-Disposition\", \"attachment; filename=report.pdf\")",
				"ctx.Header(\"Cache-Control\", \"private\")",
			},
		},
		{
			name: "CORS headers",
			input: `
package main

@Controller("/api")
type CORSController struct {}

@Header("Access-Control-Allow-Origin", "*")
@Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
@Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
@Options("/cors")
func HandleCORS() {}
`,
			expectedSnippets: []string{
				"ctx.Header(\"Access-Control-Allow-Origin\", \"*\")",
				"ctx.Header(\"Access-Control-Allow-Methods\", \"GET, POST, PUT, DELETE\")",
				"ctx.Header(\"Access-Control-Allow-Headers\", \"Content-Type, Authorization\")",
			},
		},
		{
			name: "API versioning headers",
			input: `
package main

@Controller("/api")
type APIController struct {}

@Header("X-API-Version", "v2.1")
@Header("X-Rate-Limit", "1000")
@Header("X-Rate-Limit-Remaining", "995")
@Get("/data")
func GetData() {}
`,
			expectedSnippets: []string{
				"ctx.Header(\"X-API-Version\", \"v2.1\")",
				"ctx.Header(\"X-Rate-Limit\", \"1000\")",
				"ctx.Header(\"X-Rate-Limit-Remaining\", \"995\")",
			},
		},
		{
			name: "Security headers",
			input: `
package main

@Controller("/api")
type SecurityController struct {}

@Header("X-Content-Type-Options", "nosniff")
@Header("X-Frame-Options", "DENY")
@Header("X-XSS-Protection", "1; mode=block")
@Header("Strict-Transport-Security", "max-age=31536000")
@Get("/secure")
func GetSecureData() {}
`,
			expectedSnippets: []string{
				"ctx.Header(\"X-Content-Type-Options\", \"nosniff\")",
				"ctx.Header(\"X-Frame-Options\", \"DENY\")",
				"ctx.Header(\"X-XSS-Protection\", \"1; mode=block\")",
				"ctx.Header(\"Strict-Transport-Security\", \"max-age=31536000\")",
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

func TestHeaderDecoratorValidation(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		shouldError     bool
		errorMsg        string
	}{
		{
			name: "Valid single header",
			input: `
package main

@Controller("/api")
type UserController struct {}

@Header("Cache-Control", "no-cache")
@Get("/users")
func GetUsers() {}
`,
			shouldError: false,
		},
		{
			name: "Valid multiple headers",
			input: `
package main

@Controller("/api")
type FileController struct {}

@Header("Content-Type", "application/json")
@Header("X-API-Version", "v1.0")
@Header("Cache-Control", "max-age=3600")
@Get("/data")
func GetData() {}
`,
			shouldError: false,
		},
		{
			name: "Valid security headers",
			input: `
package main

@Controller("/api")
type SecurityController struct {}

@Header("X-Content-Type-Options", "nosniff")
@Header("X-Frame-Options", "SAMEORIGIN")
@Header("X-XSS-Protection", "1; mode=block")
@Get("/secure")
func GetSecure() {}
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

func TestHeaderDecoratorTypeMapping(t *testing.T) {
	// Test that @Header decorator is properly mapped
	decoratorType := GetDecoratorType("Header")
	if decoratorType != HeaderDecoratorType {
		t.Errorf("Expected HeaderDecoratorType, got %v", decoratorType)
	}
}

func TestHeaderDecoratorWithOtherDecorators(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedSnippets []string
	}{
		{
			name: "Header with HttpCode decorator",
			input: `
package main

@Controller("/api")
type UserController struct {}

@HttpCode(201)
@Header("Location", "/api/users/123")
@Header("X-Resource-Created", "true")
@Post("/users")
func CreateUser() {}
`,
			expectedSnippets: []string{
				"ctx.Status(201)",
				"ctx.Header(\"Location\", \"/api/users/123\")",
				"ctx.Header(\"X-Resource-Created\", \"true\")",
			},
		},
		{
			name: "Header with parameters",
			input: `
package main

@Controller("/api")
type DownloadController struct {}

@Header("Content-Type", "application/octet-stream")
@Header("Content-Disposition", "attachment")
@Get("/download/:filename")
func DownloadFile(@Param("filename") filename string) {}
`,
			expectedSnippets: []string{
				"filenameValue := ctx.GetParam(\"filename\")",
				"filename := filenameValue",
				"ctx.Header(\"Content-Type\", \"application/octet-stream\")",
				"ctx.Header(\"Content-Disposition\", \"attachment\")",
			},
		},
		{
			name: "Header with Catch decorator",
			input: `
package main

@Controller("/api")
type APIController struct {}

@Header("X-API-Version", "v1.0")
@Catch(NotFoundError)
@Get("/data/:id")
func GetData(@Param("id") id string) {}
`,
			expectedSnippets: []string{
				"ctx.Header(\"X-API-Version\", \"v1.0\")",
				"idValue := ctx.GetParam(\"id\")",
				"id := idValue",
				"handleGetDataError",
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
		})
	}
}

func TestHeaderDecoratorComplexScenarios(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedSnippets []string
	}{
		{
			name: "File download with comprehensive headers",
			input: `
package main

@Controller("/api/files")
type FileDownloadController struct {}

@Header("Content-Type", "application/pdf")
@Header("Content-Disposition", "attachment; filename=document.pdf")
@Header("Content-Length", "1048576")
@Header("Cache-Control", "private, no-cache")
@Header("X-Download-Options", "noopen")
@Get("/download/:id")
func DownloadPDF(@Param("id") id string) {}
`,
			expectedSnippets: []string{
				"ctx.Header(\"Content-Type\", \"application/pdf\")",
				"ctx.Header(\"Content-Disposition\", \"attachment; filename=document.pdf\")",
				"ctx.Header(\"Content-Length\", \"1048576\")",
				"ctx.Header(\"Cache-Control\", \"private, no-cache\")",
				"ctx.Header(\"X-Download-Options\", \"noopen\")",
			},
		},
		{
			name: "REST API with versioning and rate limiting",
			input: `
package main

@Controller("/api/v2")
type APIv2Controller struct {}

@Header("X-API-Version", "2.0")
@Header("X-Rate-Limit", "1000")
@Header("X-Rate-Limit-Window", "3600")
@Header("X-Rate-Limit-Remaining", "999")
@Get("/users")
func GetUsers() {}

@Header("X-API-Version", "2.0")
@Header("X-Rate-Limit", "100")
@Header("X-Rate-Limit-Window", "3600")
@Post("/users")
func CreateUser() {}
`,
			expectedSnippets: []string{
				"ctx.Header(\"X-API-Version\", \"2.0\")",
				"ctx.Header(\"X-Rate-Limit\", \"1000\")",
				"ctx.Header(\"X-Rate-Limit\", \"100\")",
				"ctx.Header(\"X-Rate-Limit-Window\", \"3600\")",
				"ctx.Header(\"X-Rate-Limit-Remaining\", \"999\")",
			},
		},
		{
			name: "Security-focused API with all security headers",
			input: `
package main

@Controller("/api/secure")
type SecureController struct {}

@Header("X-Content-Type-Options", "nosniff")
@Header("X-Frame-Options", "DENY")
@Header("X-XSS-Protection", "1; mode=block")
@Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
@Header("Content-Security-Policy", "default-src 'self'")
@Header("Referrer-Policy", "strict-origin-when-cross-origin")
@Get("/data")
func GetSecureData() {}
`,
			expectedSnippets: []string{
				"ctx.Header(\"X-Content-Type-Options\", \"nosniff\")",
				"ctx.Header(\"X-Frame-Options\", \"DENY\")",
				"ctx.Header(\"X-XSS-Protection\", \"1; mode=block\")",
				"ctx.Header(\"Strict-Transport-Security\", \"max-age=31536000; includeSubDomains\")",
				"ctx.Header(\"Content-Security-Policy\", \"default-src 'self'\")",
				"ctx.Header(\"Referrer-Policy\", \"strict-origin-when-cross-origin\")",
			},
		},
		{
			name: "CORS preflight response",
			input: `
package main

@Controller("/api")
type CORSController struct {}

@Header("Access-Control-Allow-Origin", "https://example.com")
@Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
@Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
@Header("Access-Control-Allow-Credentials", "true")
@Header("Access-Control-Max-Age", "86400")
@Options("/users")
func HandleCORSPreflight() {}
`,
			expectedSnippets: []string{
				"ctx.Header(\"Access-Control-Allow-Origin\", \"https://example.com\")",
				"ctx.Header(\"Access-Control-Allow-Methods\", \"GET, POST, PUT, DELETE, OPTIONS\")",
				"ctx.Header(\"Access-Control-Allow-Headers\", \"Content-Type, Authorization, X-Requested-With\")",
				"ctx.Header(\"Access-Control-Allow-Credentials\", \"true\")",
				"ctx.Header(\"Access-Control-Max-Age\", \"86400\")",
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

func TestHeaderDecoratorOrder(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedOrder   []string
	}{
		{
			name: "Headers should appear after status code but before parameter extraction",
			input: `
package main

@Controller("/api")
type OrderController struct {}

@HttpCode(201)
@Header("X-Custom", "value")
@Header("Location", "/api/users/123")
@Post("/users")
func CreateUser(@Body() userData CreateUserDto) {}
`,
			expectedOrder: []string{
				"ctx.Status(201)",
				"ctx.Header(\"X-Custom\", \"value\")",
				"ctx.Header(\"Location\", \"/api/users/123\")",
				"var userData CreateUserDto",
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

			// Check that the order is correct by finding indices
			prevIndex := -1
			for i, expectedSnippet := range tt.expectedOrder {
				index := strings.Index(generatedCode, expectedSnippet)
				if index == -1 {
					t.Errorf("Expected snippet not found: %s", expectedSnippet)
					continue
				}
				if index <= prevIndex {
					t.Errorf("Expected snippet %d (%s) to appear after snippet %d, but order is wrong", 
						i, expectedSnippet, i-1)
				}
				prevIndex = index
			}
		})
	}
}