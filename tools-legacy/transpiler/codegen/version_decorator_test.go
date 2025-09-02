package codegen

import (
	"strings"
	"testing"
)

func TestVersionDecoratorTranspilation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string // What we expect to see in the generated code
	}{
		{
			name: "Simple numeric version",
			input: `package main

@Version("1")
@Controller("/users")
type UsersV1Controller struct {}

@Get("/")
func (c *UsersV1Controller) getUsers() []User {
	return []User{}
}`,
			expected: []string{
				"UsersV1Controller",
				"RegisterRoutes",
				`server.Get("/v1/users", c.getUsers)`,
			},
		},
		{
			name: "Version with 'v' prefix",
			input: `package main

@Version("v2")
@Controller("/api/users")
type UsersV2Controller struct {}

@Get("/")
func (c *UsersV2Controller) getUsers() []User {
	return []User{}
}`,
			expected: []string{
				"UsersV2Controller",
				"RegisterRoutes",
				`server.Get("/v2/api/users", c.getUsers)`,
			},
		},
		{
			name: "Version with '/v' prefix",
			input: `package main

@Version("/v3")
@Controller("/api/products")
type ProductV3Controller struct {}

@Get("/:id")
func (c *ProductV3Controller) getProduct() Product {
	return Product{}
}`,
			expected: []string{
				"ProductV3Controller",
				"RegisterRoutes",
				`server.Get("/v3/api/products/:id", c.getProduct)`,
			},
		},
		{
			name: "Multiple HTTP methods with versioning",
			input: `package main

@Version("2")
@Controller("/items")
type ItemController struct {}

@Get("/")
func (c *ItemController) getItems() []Item {
	return []Item{}
}

@Post("/")
func (c *ItemController) createItem() Item {
	return Item{}
}

@Put("/:id")
func (c *ItemController) updateItem() Item {
	return Item{}
}

@Delete("/:id")
func (c *ItemController) deleteItem() {
}`,
			expected: []string{
				"ItemController",
				"RegisterRoutes",
				`server.Get("/v2/items", c.getItems)`,
				`server.Post("/v2/items", c.createItem)`,
				`server.Put("/v2/items/:id", c.updateItem)`,
				`server.Delete("/v2/items/:id", c.deleteItem)`,
			},
		},
		{
			name: "Controller without version - should not have version prefix",
			input: `package main

@Controller("/health")
type HealthController struct {}

@Get("/")
func (c *HealthController) checkHealth() HealthStatus {
	return HealthStatus{}
}`,
			expected: []string{
				"HealthController",
				"RegisterRoutes",
				`server.Get("/health", c.checkHealth)`,
			},
		},
		{
			name: "Mixed controllers with and without versioning",
			input: `package main

@Version("1")
@Controller("/v1-only")
type V1Controller struct {}

@Get("/")
func (c *V1Controller) getV1Data() string {
	return "v1"
}

@Controller("/no-version")  
type NoVersionController struct {}

@Get("/")
func (c *NoVersionController) getData() string {
	return "no version"
}`,
			expected: []string{
				"V1Controller",
				"NoVersionController",
				`server.Get("/v1/v1-only", c.getV1Data)`,
				`server.Get("/no-version", c.getData)`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			lexer := NewLexer(tt.input)
			parser := NewParser(lexer)
			file, err := parser.ParseFile()
			if err != nil {
				t.Fatalf("Failed to parse input: %v", err)
			}

			// Generate Go code
			generator := NewCodeGenerator("main")
			goCode, err := generator.GenerateGoCode(file)
			if err != nil {
				t.Fatalf("Failed to generate Go code: %v", err)
			}

			// Check if expected strings are present in generated code
			for _, expected := range tt.expected {
				if !strings.Contains(goCode, expected) {
					t.Errorf("Expected to find '%s' in generated code but didn't.\nGenerated code:\n%s", expected, goCode)
				}
			}

			// Print generated code for debugging (optional)
			// t.Logf("Generated code:\n%s", goCode)
		})
	}
}

func TestVersionDecoratorEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "Empty version string should be ignored",
			input: `package main

@Version("")
@Controller("/test")
type TestController struct {}

@Get("/")
func (c *TestController) getTest() string {
	return "test"
}`,
			expected: []string{
				`server.Get("/test", c.getTest)`, // No version prefix
			},
		},
		{
			name: "Version decorator without argument should be ignored",
			input: `package main

@Version()
@Controller("/test")
type TestController struct {}

@Get("/")
func (c *TestController) getTest() string {
	return "test"
}`,
			expected: []string{
				`server.Get("/test", c.getTest)`, // No version prefix
			},
		},
		{
			name: "Complex version strings",
			input: `package main

@Version("v1.2.3")
@Controller("/api")
type ApiController struct {}

@Get("/data")
func (c *ApiController) getData() string {
	return "data"
}`,
			expected: []string{
				`server.Get("/v1.2.3/api/data", c.getData)`,
			},
		},
		{
			name: "Case normalization - follows industry best practice",
			input: `package main

@Version("V2")
@Controller("/uppercase")
type UppercaseController struct {}

@Get("/")
func (c *UppercaseController) get() string {
	return "uppercase"
}

@Version("/V3")
@Controller("/slash-uppercase") 
type SlashUppercaseController struct {}

@Get("/")
func (c *SlashUppercaseController) get() string {
	return "slash-uppercase"
}`,
			expected: []string{
				`server.Get("/v2/uppercase", c.get)`,      // V2 → v2 (normalized)
				`server.Get("/v3/slash-uppercase", c.get)`, // V3 → v3 (normalized)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			lexer := NewLexer(tt.input)
			parser := NewParser(lexer)
			file, err := parser.ParseFile()
			if err != nil {
				t.Fatalf("Failed to parse input: %v", err)
			}

			// Generate Go code
			generator := NewCodeGenerator("main")
			goCode, err := generator.GenerateGoCode(file)
			if err != nil {
				t.Fatalf("Failed to generate Go code: %v", err)
			}

			// Check if expected strings are present in generated code
			for _, expected := range tt.expected {
				if !strings.Contains(goCode, expected) {
					t.Errorf("Expected to find '%s' in generated code but didn't.\nGenerated code:\n%s", expected, goCode)
				}
			}
		})
	}
}