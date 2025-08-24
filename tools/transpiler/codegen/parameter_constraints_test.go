package codegen

import (
	"strings"
	"testing"
)

func TestParameterConstraintsTranspilation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string // What we expect to see in the generated code
	}{
		{
			name: "Basic integer constraint",
			input: `package main

@Controller("/api/users")
type UserController struct {}

@Get("/:id")
func GetUser(@Param("id", { constraints: ["int"] }) id string) {
	return User{}
}`,
			expected: []string{
				"UserController",
				`idValue := ctx.GetParam("id")`,
				`if _, err := strconv.Atoi(idValue); err != nil {`,
				`ctx.JSON(400, map[string]string{"error": "Parameter 'id' must be an integer"})`,
				`id := idValue`,
			},
		},
		{
			name: "Multiple constraints - integer with minimum",
			input: `package main

@Controller("/api/users")
type UserController struct {}

@Get("/:id")
func GetUser(@Param("id", { constraints: ["int", "min(1)"] }) id string) {
	return User{}
}`,
			expected: []string{
				"UserController",
				`idValue := ctx.GetParam("id")`,
				`if _, err := strconv.Atoi(idValue); err != nil {`,
				`ctx.JSON(400, map[string]string{"error": "Parameter 'id' must be an integer"})`,
				`if intVal, err := strconv.Atoi(idValue); err == nil {`,
				`if intVal < 1 {`,
				`ctx.JSON(400, map[string]string{"error": "Parameter 'id' must be at least 1"})`,
			},
		},
		{
			name: "Range constraint",
			input: `package main

@Controller("/api")
type ApiController struct {}

@Get("/:value")
func GetValue(@Param("value", { constraints: ["range(1,100)"] }) value string) {
	return string{}
}`,
			expected: []string{
				"ApiController",
				`valueValue := ctx.GetParam("value")`,
				`if intVal < 1 || intVal > 100 {`,
				`ctx.JSON(400, map[string]string{"error": "Parameter 'value' must be between 1 and 100"})`,
			},
		},
		{
			name: "String length constraints",
			input: `package main

@Controller("/api")
type ApiController struct {}

@Get("/:name")
func GetName(@Param("name", { constraints: ["minlength(3)", "maxlength(20)"] }) name string) {
	return string{}
}`,
			expected: []string{
				"ApiController",
				`nameValue := ctx.GetParam("name")`,
				`if len(nameValue) < 3 {`,
				`ctx.JSON(400, map[string]string{"error": "Parameter 'name' must be at least 3 characters long"})`,
				`if len(nameValue) > 20 {`,
				`ctx.JSON(400, map[string]string{"error": "Parameter 'name' must be at most 20 characters long"})`,
			},
		},
		{
			name: "Alpha constraint",
			input: `package main

@Controller("/api")
type ApiController struct {}

@Get("/:code")
func GetCode(@Param("code", { constraints: ["alpha"] }) code string) {
	return string{}
}`,
			expected: []string{
				"ApiController",
				`codeValue := ctx.GetParam("code")`,
				`if matched, _ := regexp.MatchString("^[a-zA-Z]+$", codeValue); !matched {`,
				`ctx.JSON(400, map[string]string{"error": "Parameter 'code' must contain only alphabetic characters"})`,
			},
		},
		{
			name: "GUID constraint",
			input: `package main

@Controller("/api")
type ApiController struct {}

@Get("/:resourceId")
func GetResource(@Param("resourceId", { constraints: ["guid"] }) resourceId string) {
	return Resource{}
}`,
			expected: []string{
				"ApiController",
				`resourceIdValue := ctx.GetParam("resourceId")`,
				`if _, err := uuid.Parse(resourceIdValue); err != nil {`,
				`ctx.JSON(400, map[string]string{"error": "Parameter 'resourceId' must be a valid GUID"})`,
			},
		},
		{
			name: "Regex constraint",
			input: `package main

@Controller("/api")
type ApiController struct {}

@Get("/:sku")
func GetProduct(@Param("sku", { constraints: ["regex(^[A-Z]{2}-\\\\d{4}$)"] }) sku string) {
	return Product{}
}`,
			expected: []string{
				"ApiController",
				`skuValue := ctx.GetParam("sku")`,
				`if matched, _ := regexp.MatchString("^[A-Z]{2}-\\\\d{4}$", skuValue); !matched {`,
				`ctx.JSON(400, map[string]string{"error": "Parameter 'sku' does not match required pattern"})`,
			},
		},
		{
			name: "Boolean constraint",
			input: `package main

@Controller("/api")
type ApiController struct {}

@Get("/:enabled")
func GetSetting(@Param("enabled", { constraints: ["bool"] }) enabled string) {
	return bool{}
}`,
			expected: []string{
				"ApiController",
				`enabledValue := ctx.GetParam("enabled")`,
				`if _, err := strconv.ParseBool(enabledValue); err != nil {`,
				`ctx.JSON(400, map[string]string{"error": "Parameter 'enabled' must be a boolean (true/false)"})`,
			},
		},
		{
			name: "Required parameter",
			input: `package main

@Controller("/api")
type ApiController struct {}

@Get("/:id")
func GetItem(@Param("id", { required: true, constraints: ["int"] }) id string) {
	return Item{}
}`,
			expected: []string{
				"ApiController",
				`idValue := ctx.GetParam("id")`,
				`if idValue == "" {`,
				`ctx.JSON(400, map[string]string{"error": "Parameter 'id' is required"})`,
				`if _, err := strconv.Atoi(idValue); err != nil {`,
			},
		},
		{
			name: "Parameter with transformation",
			input: `package main

@Controller("/api")
type ApiController struct {}

@Get("/:code")
func GetCode(@Param("code", { transform: "uppercase", constraints: ["alpha"] }) code string) {
	return string{}
}`,
			expected: []string{
				"ApiController",
				`codeValue := ctx.GetParam("code")`,
				`codeValue = strings.ToUpper(codeValue)`,
				`if matched, _ := regexp.MatchString("^[a-zA-Z]+$", codeValue); !matched {`,
			},
		},
		{
			name: "Exact length constraint",
			input: `package main

@Controller("/api")
type ApiController struct {}

@Get("/:pin")
func ValidatePin(@Param("pin", { constraints: ["length(4)", "int"] }) pin string) {
	return bool{}
}`,
			expected: []string{
				"ApiController",
				`pinValue := ctx.GetParam("pin")`,
				`if _, err := strconv.Atoi(pinValue); err != nil {`,
				`if len(pinValue) != 4 {`,
				`ctx.JSON(400, map[string]string{"error": "Parameter 'pin' must be exactly 4 characters long"})`,
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
					t.Errorf("Expected to find '%s' in generated code but didn't.\\nGenerated code:\\n%s", expected, goCode)
				}
			}

			// Print generated code for debugging (optional)
			// t.Logf("Generated code:\\n%s", goCode)
		})
	}
}

func TestParameterConstraintsComplexScenarios(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "Multiple parameters with different constraints",
			input: `package main

@Controller("/api/store")
type StoreController struct {}

@Get("/:storeId/product/:productId")
func GetProduct(
	@Param("storeId", { constraints: ["int", "range(1,999)"] }) storeId string,
	@Param("productId", { required: true, constraints: ["guid"] }) productId string
) {
	return Product{}
}`,
			expected: []string{
				"StoreController",
				`storeIdValue := ctx.GetParam("storeId")`,
				`productIdValue := ctx.GetParam("productId")`,
				`if productIdValue == "" {`,
				`ctx.JSON(400, map[string]string{"error": "Parameter 'productId' is required"})`,
				`if intVal < 1 || intVal > 999 {`,
				`if _, err := uuid.Parse(productIdValue); err != nil {`,
			},
		},
		{
			name: "Combined transformations and validations",
			input: `package main

@Controller("/api")
type ApiController struct {}

@Get("/:category/:subcategory")
func GetItems(
	@Param("category", { transform: "lowercase", constraints: ["alpha", "minlength(3)"] }) category string,
	@Param("subcategory", { transform: "trim", constraints: ["regex(^[a-zA-Z0-9_-]+$)"] }) subcategory string
) {
	return []Item{}
}`,
			expected: []string{
				"ApiController",
				`categoryValue = strings.ToLower(categoryValue)`,
				`subcategoryValue = strings.TrimSpace(subcategoryValue)`,
				`if matched, _ := regexp.MatchString("^[a-zA-Z]+$", categoryValue); !matched {`,
				`if len(categoryValue) < 3 {`,
				`if matched, _ := regexp.MatchString("^[a-zA-Z0-9_-]+$", subcategoryValue); !matched {`,
			},
		},
		{
			name: "Edge case - parameter without constraints should still work",
			input: `package main

@Controller("/api")
type ApiController struct {}

@Get("/:id")
func GetUser(@Param("id") id string) {
	return User{}
}`,
			expected: []string{
				"ApiController",
				`idValue := ctx.GetParam("id")`,
				`id := idValue`,
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
					t.Errorf("Expected to find '%s' in generated code but didn't.\\nGenerated code:\\n%s", expected, goCode)
				}
			}
		})
	}
}

func TestConstraintParsingEdgeCases(t *testing.T) {
	generator := NewCodeGenerator("test")
	
	// Test constraint parsing
	tests := []struct {
		constraintStr string
		expectedType  string
		expectedValue string
		expectedValue2 string
	}{
		{"int", "int", "", ""},
		{"min(5)", "min", "5", ""},
		{"max(100)", "max", "100", ""},
		{"range(1,50)", "range", "1", "50"},
		{"regex(^\\\\d+$)", "regex", "^\\\\d+$", ""},
		{"length(8)", "length", "8", ""},
		{"minlength(3)", "minlength", "3", ""},
		{"maxlength(20)", "maxlength", "20", ""},
		{"alpha", "alpha", "", ""},
		{"bool", "bool", "", ""},
		{"guid", "guid", "", ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.constraintStr, func(t *testing.T) {
			constraint := generator.parseConstraint(tt.constraintStr)
			
			if constraint.Type != tt.expectedType {
				t.Errorf("Expected constraint type '%s', got '%s'", tt.expectedType, constraint.Type)
			}
			
			if constraint.Value != tt.expectedValue {
				t.Errorf("Expected constraint value '%s', got '%s'", tt.expectedValue, constraint.Value)
			}
			
			if constraint.Value2 != tt.expectedValue2 {
				t.Errorf("Expected constraint value2 '%s', got '%s'", tt.expectedValue2, constraint.Value2)
			}
		})
	}
}