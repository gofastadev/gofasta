package codegen

import (
	"strings"
	"testing"
)

func TestReqDecoratorBasic(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Req() req *RequestContext) {
}
`

	expected := []string{
		"req := ctx",
	}

	testReqGeneration(t, input, expected, "basic request context parameter")
}

func TestReqDecoratorWithHTTPRequest(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Req() req *Request) {
}
`

	expected := []string{
		"req := ctx.GetRequest()",
	}

	testReqGeneration(t, input, expected, "HTTP request parameter")
}

func TestReqDecoratorWithRequestContextType(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Req() ctx *RequestContext) {
}
`

	expected := []string{
		"ctx := ctx",
	}

	testReqGeneration(t, input, expected, "RequestContext type parameter")
}

func TestReqDecoratorMultipleParameters(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Req() req *Request, @Query("id") id string) {
}
`

	expected := []string{
		"req := ctx.GetRequest()",
		"var id string",
		"queryValue := ctx.GetQuery(\"id\")",
		"id = queryValue",
	}

	testReqGeneration(t, input, expected, "request parameter with other parameters")
}

func TestReqDecoratorWithBody(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Post("")
func CreateUser(@Req() req *Request, @Body() user User) {
}
`

	expected := []string{
		"req := ctx.GetRequest()",
		"var user User",
		"if err := ctx.ParseJSON(&user); err != nil {",
		"ctx.JSON(400, map[string]string{\"error\": \"Invalid request body\"})",
		"return",
		"}",
	}

	testReqGeneration(t, input, expected, "request parameter with body parameter")
}

func TestReqDecoratorWithHeaders(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Req() req *Request, @Headers("Authorization") auth string) {
}
`

	expected := []string{
		"req := ctx.GetRequest()",
		"var auth string",
		"headerValue := ctx.GetHeader(\"Authorization\")",
		"auth = headerValue",
	}

	testReqGeneration(t, input, expected, "request parameter with headers parameter")
}

func TestReqDecoratorWithParam(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("/:id")
func GetUser(@Req() req *Request, @Param("id") id string) {
}
`

	expected := []string{
		"req := ctx.GetRequest()",
		"idValue := ctx.GetParam(\"id\")",
		"id := idValue",
	}

	testReqGeneration(t, input, expected, "request parameter with path parameter")
}

func TestReqDecoratorComplexExample(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Put("/:id")
func UpdateUser(
	@Req() req *Request,
	@Param("id") id string,
	@Body() updateData UpdateUserDto,
	@Query("validate", { defaultValue: "true" }) validate bool,
	@Headers("Authorization", { required: true }) authToken string
) {
}
`

	expected := []string{
		"req := ctx.GetRequest()",
		"idValue := ctx.GetParam(\"id\")",
		"id := idValue",
		"var updateData UpdateUserDto",
		"if err := ctx.ParseJSON(&updateData); err != nil {",
		"ctx.JSON(400, map[string]string{\"error\": \"Invalid request body\"})",
		"return",
		"}",
		"var validate bool",
		"queryValue := ctx.GetQuery(\"validate\")",
		"if queryValue == \"\" {",
		"queryValue = \"true\"",
		"}",
		"if queryValue != \"\" {",
		"if parsedBool, err := strconv.ParseBool(queryValue); err == nil {",
		"validate = parsedBool",
		"} else {",
		"ctx.JSON(400, map[string]string{\"error\": \"Invalid boolean value for parameter 'validate' (use true/false)\"})",
		"return",
		"}",
		"}",
		"var authToken string",
		"headerValue := ctx.GetHeader(\"Authorization\")",
		"if headerValue == \"\" {",
		"ctx.JSON(400, map[string]string{\"error\": \"Header 'Authorization' is required\"})",
		"return",
		"}",
		"authToken = headerValue",
	}

	testReqGeneration(t, input, expected, "complex request parameter with multiple decorators")
}

func TestReqDecoratorDefaultTypeHandling(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Req() req interface{}) {
}
`

	expected := []string{
		"req := ctx",
	}

	testReqGeneration(t, input, expected, "request parameter with interface{} type (defaults to context)")
}

// Helper function for testing @Req() decorator generation
func testReqGeneration(t *testing.T, input string, expected []string, description string) {
	t.Helper()
	
	// Parse the input
	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse input for %s: %v", description, err)
	}

	// Generate code
	generator := NewCodeGenerator("main")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code for %s: %v", description, err)
	}

	// Check that all expected strings are present
	for _, expectedStr := range expected {
		if !strings.Contains(output, expectedStr) {
			t.Errorf("For %s, expected to find '%s' in generated code", description, expectedStr)
			t.Errorf("Generated code:\n%s", output)
		}
	}
}

func TestReqDecoratorImports(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Req() req *Request) {
}
`

	// Parse and generate code
	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}

	generator := NewCodeGenerator("main")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Check that necessary imports are included
	expectedImports := []string{
		`"net/http"`,
		`"github.com/healtronlabs/gofasta/packages/http"`,
		`"github.com/healtronlabs/gofasta/packages/core"`,
		`"strconv"`,
		`"strings"`,
	}

	for _, expectedImport := range expectedImports {
		if !strings.Contains(output, expectedImport) {
			t.Errorf("Expected import '%s' not found in generated code", expectedImport)
			t.Errorf("Generated code:\n%s", output)
		}
	}
}