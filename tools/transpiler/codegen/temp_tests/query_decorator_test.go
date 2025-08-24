package codegen

import (
	"strings"
	"testing"
)

func TestQueryDecoratorBasic(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Query("page") page string) {
}
`

	expected := []string{
		"var page string",
		"queryValue := ctx.GetQuery(\"page\")",
		"page = queryValue",
	}

	testQueryGeneration(t, input, expected, "basic query parameter")
}

func TestQueryDecoratorWithDefaultValue(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Query("sort", { defaultValue: "asc" }) sort string) {
}
`

	expected := []string{
		"var sort string",
		"queryValue := ctx.GetQuery(\"sort\")",
		"if queryValue == \"\" {",
		"queryValue = \"asc\"",
		"sort = queryValue",
	}

	testQueryGeneration(t, input, expected, "query parameter with default value")
}

func TestQueryDecoratorRequired(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Query("limit", { required: true }) limit string) {
}
`

	expected := []string{
		"var limit string",
		"queryValue := ctx.GetQuery(\"limit\")",
		"if queryValue == \"\" {",
		"ctx.JSON(400, map[string]string{\"error\": \"Query parameter 'limit' is required\"})",
		"return",
		"limit = queryValue",
	}

	testQueryGeneration(t, input, expected, "required query parameter")
}

func TestQueryDecoratorIntegerType(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Query("page") page int) {
}
`

	expected := []string{
		"var page int",
		"queryValue := ctx.GetQuery(\"page\")",
		"if queryValue != \"\" {",
		"if parsedInt, err := strconv.Atoi(queryValue); err == nil {",
		"page = parsedInt",
		"} else {",
		"ctx.JSON(400, map[string]string{\"error\": \"Invalid integer value for parameter 'page'\"})",
		"return",
	}

	testQueryGeneration(t, input, expected, "integer query parameter")
}

func TestQueryDecoratorBooleanType(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Query("active") active bool) {
}
`

	expected := []string{
		"var active bool",
		"queryValue := ctx.GetQuery(\"active\")",
		"if queryValue != \"\" {",
		"if parsedBool, err := strconv.ParseBool(queryValue); err == nil {",
		"active = parsedBool",
		"} else {",
		"ctx.JSON(400, map[string]string{\"error\": \"Invalid boolean value for parameter 'active' (use true/false)\"})",
		"return",
	}

	testQueryGeneration(t, input, expected, "boolean query parameter")
}

func TestQueryDecoratorFloatType(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Query("rating") rating float64) {
}
`

	expected := []string{
		"var rating float64",
		"queryValue := ctx.GetQuery(\"rating\")",
		"if queryValue != \"\" {",
		"if parsedFloat, err := strconv.ParseFloat(queryValue, 64); err == nil {",
		"rating = parsedFloat",
		"} else {",
		"ctx.JSON(400, map[string]string{\"error\": \"Invalid float value for parameter 'rating'\"})",
		"return",
	}

	testQueryGeneration(t, input, expected, "float query parameter")
}

func TestQueryDecoratorArrayType(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Query("tags") tags []string) {
}
`

	expected := []string{
		"var tags []string",
		"queryValue := ctx.GetQuery(\"tags\")",
		"if queryValue != \"\" {",
		"tags = strings.Split(queryValue, \",\")",
		"for i, v := range tags {",
		"tags[i] = strings.TrimSpace(v)",
	}

	testQueryGeneration(t, input, expected, "array query parameter")
}

func TestQueryDecoratorArrayWithCustomSeparator(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Query("ids", { type: "array", separator: "|" }) ids []string) {
}
`

	expected := []string{
		"var ids []string",
		"queryValue := ctx.GetQuery(\"ids\")",
		"if queryValue != \"\" {",
		"ids = strings.Split(queryValue, \"|\")",
		"for i, v := range ids {",
		"ids[i] = strings.TrimSpace(v)",
	}

	testQueryGeneration(t, input, expected, "array query parameter with custom separator")
}

func TestQueryDecoratorWithTransform(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Query("name", { transform: "lowercase" }) name string) {
}
`

	expected := []string{
		"var name string",
		"queryValue := ctx.GetQuery(\"name\")",
		"queryValue = strings.ToLower(queryValue)",
		"name = queryValue",
	}

	testQueryGeneration(t, input, expected, "query parameter with lowercase transform")
}

func TestQueryDecoratorComplex(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(
	@Query("page", { defaultValue: "1" }) page int,
	@Query("limit", { required: true }) limit int,
	@Query("sort", { defaultValue: "name", transform: "lowercase" }) sort string,
	@Query("tags", { type: "array" }) tags []string,
	@Query("active") active bool
) {
}
`

	expected := []string{
		// page parameter
		"var page int",
		"queryValue := ctx.GetQuery(\"page\")",
		"if queryValue == \"\" {",
		"queryValue = \"1\"",
		"if parsedInt, err := strconv.Atoi(queryValue); err == nil {",
		"page = parsedInt",
		
		// limit parameter  
		"var limit int",
		"queryValue := ctx.GetQuery(\"limit\")",
		"if queryValue == \"\" {",
		"ctx.JSON(400, map[string]string{\"error\": \"Query parameter 'limit' is required\"})",
		"return",
		"if parsedInt, err := strconv.Atoi(queryValue); err == nil {",
		"limit = parsedInt",
		
		// sort parameter
		"var sort string",
		"queryValue := ctx.GetQuery(\"sort\")",
		"if queryValue == \"\" {",
		"queryValue = \"name\"",
		"queryValue = strings.ToLower(queryValue)",
		"sort = queryValue",
		
		// tags parameter
		"var tags []string",
		"queryValue := ctx.GetQuery(\"tags\")",
		"if queryValue != \"\" {",
		"tags = strings.Split(queryValue, \",\")",
		"for i, v := range tags {",
		"tags[i] = strings.TrimSpace(v)",
		
		// active parameter
		"var active bool",
		"queryValue := ctx.GetQuery(\"active\")",
		"if queryValue != \"\" {",
		"if parsedBool, err := strconv.ParseBool(queryValue); err == nil {",
		"active = parsedBool",
	}

	testQueryGeneration(t, input, expected, "complex multiple query parameters")
}

func testQueryGeneration(t *testing.T, input string, expectedStrings []string, testName string) {
	// Parse the input
	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse .gofa file for %s: %v", testName, err)
	}

	// Generate code
	generator := NewCodeGenerator("main")
	goCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate Go code for %s: %v", testName, err)
	}

	// Check that all expected strings are present
	for _, expectedStr := range expectedStrings {
		if !strings.Contains(goCode, expectedStr) {
			t.Errorf("Generated code for %s missing expected string: %q", testName, expectedStr)
			t.Logf("Generated code:\n%s", goCode)
		}
	}

	// Ensure necessary imports are included
	requiredImports := []string{
		"\"strconv\"",
		"\"strings\"",
		"\"github.com/healtronlabs/gofasta/packages/http\"",
	}

	for _, requiredImport := range requiredImports {
		if !strings.Contains(goCode, requiredImport) {
			t.Errorf("Generated code for %s missing required import: %s", testName, requiredImport)
		}
	}
}

func TestQueryDecoratorErrorCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		shouldParse bool
	}{
		{
			name: "Query parameter without name",
			input: `
package main
@Controller("/api")
type TestController struct {}
@Get("")
func Test(@Query() param string) {}
`,
			shouldParse: true, // Should parse but use parameter name as query name
		},
		{
			name: "Query parameter with empty options",
			input: `
package main
@Controller("/api")
type TestController struct {}
@Get("")
func Test(@Query("param", {}) param string) {}
`,
			shouldParse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseGofaFile(tt.input)
			
			if tt.shouldParse && err != nil {
				t.Errorf("Expected parsing to succeed for %s, but got error: %v", tt.name, err)
			} else if !tt.shouldParse && err == nil {
				t.Errorf("Expected parsing to fail for %s, but it succeeded", tt.name)
			}
		})
	}
}

func TestQueryDecoratorInference(t *testing.T) {
	input := `
package main

@Controller("/api/products")
type ProductController struct {
}

@Get("")
func GetProducts(@Query("category") category string, @Query("inStock") inStock bool, @Query("minPrice") minPrice float64, @Query("tags") tags []string) {
}
`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse .gofa file: %v", err)
	}

	generator := NewCodeGenerator("main")
	goCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate Go code: %v", err)
	}

	// Test type inference from parameter types
	testCases := []struct {
		paramName string
		paramType string
		shouldContain []string
	}{
		{
			paramName: "category",
			paramType: "string",
			shouldContain: []string{
				"var category string",
				"category = queryValue",
			},
		},
		{
			paramName: "inStock", 
			paramType: "bool",
			shouldContain: []string{
				"var inStock bool",
				"strconv.ParseBool(queryValue)",
				"inStock = parsedBool",
			},
		},
		{
			paramName: "minPrice",
			paramType: "float64", 
			shouldContain: []string{
				"var minPrice float64",
				"strconv.ParseFloat(queryValue, 64)",
				"minPrice = parsedFloat",
			},
		},
		{
			paramName: "tags",
			paramType: "[]string",
			shouldContain: []string{
				"var tags []string",
				"tags = strings.Split(queryValue, \",\")",
				"tags[i] = strings.TrimSpace(v)",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.paramName, func(t *testing.T) {
			for _, expected := range tc.shouldContain {
				if !strings.Contains(goCode, expected) {
					t.Errorf("Generated code missing expected string for %s (%s): %q", tc.paramName, tc.paramType, expected)
				}
			}
		})
	}
}