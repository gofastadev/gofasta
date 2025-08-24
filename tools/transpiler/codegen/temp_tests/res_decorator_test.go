package codegen

import (
	"strings"
	"testing"
)

func TestResDecoratorBasic(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Res() res *RequestContext) {
}
`

	expected := []string{
		"res := ctx",
	}

	testResGeneration(t, input, expected, "basic response context parameter")
}

func TestResDecoratorWithResponseWriter(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Res() res *ResponseWriter) {
}
`

	expected := []string{
		"res := ctx.GetResponseWriter()",
	}

	testResGeneration(t, input, expected, "response writer parameter")
}

func TestResDecoratorWithResponseContext(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Res() res *ResponseContext) {
}
`

	expected := []string{
		"res := ctx",
	}

	testResGeneration(t, input, expected, "ResponseContext type parameter")
}

func TestResDecoratorMultipleParameters(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Res() res *ResponseWriter, @Query("id") id string) {
}
`

	expected := []string{
		"res := ctx.GetResponseWriter()",
		"var id string",
		"queryValue := ctx.GetQuery(\"id\")",
		"id = queryValue",
	}

	testResGeneration(t, input, expected, "response parameter with other parameters")
}

func TestResDecoratorWithBody(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Post("")
func CreateUser(@Res() res *ResponseWriter, @Body() user User) {
}
`

	expected := []string{
		"res := ctx.GetResponseWriter()",
		"var user User",
		"if err := ctx.ParseJSON(&user); err != nil {",
		"ctx.JSON(400, map[string]string{\"error\": \"Invalid request body\"})",
		"return",
		"}",
	}

	testResGeneration(t, input, expected, "response parameter with body parameter")
}

func TestResDecoratorWithHeaders(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Res() res *ResponseWriter, @Headers("Authorization") auth string) {
}
`

	expected := []string{
		"res := ctx.GetResponseWriter()",
		"var auth string",
		"headerValue := ctx.GetHeader(\"Authorization\")",
		"auth = headerValue",
	}

	testResGeneration(t, input, expected, "response parameter with headers parameter")
}

func TestResDecoratorWithParam(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("/:id")
func GetUser(@Res() res *ResponseWriter, @Param("id") id string) {
}
`

	expected := []string{
		"res := ctx.GetResponseWriter()",
		"idValue := ctx.GetParam(\"id\")",
		"id := idValue",
	}

	testResGeneration(t, input, expected, "response parameter with path parameter")
}

func TestResDecoratorWithReqAndRes(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Req() req *Request, @Res() res *ResponseWriter) {
}
`

	expected := []string{
		"req := ctx.GetRequest()",
		"res := ctx.GetResponseWriter()",
	}

	testResGeneration(t, input, expected, "both request and response parameters")
}

func TestResDecoratorComplexExample(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Put("/:id")
func UpdateUser(
	@Res() res *RequestContext,
	@Param("id") id string,
	@Body() updateData UpdateUserDto,
	@Query("validate", { defaultValue: "true" }) validate bool,
	@Headers("Authorization", { required: true }) authToken string
) {
}
`

	expected := []string{
		"res := ctx",
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

	testResGeneration(t, input, expected, "complex response parameter with multiple decorators")
}

func TestResDecoratorDefaultTypeHandling(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Res() res interface{}) {
}
`

	expected := []string{
		"res := ctx",
	}

	testResGeneration(t, input, expected, "response parameter with interface{} type (defaults to context)")
}

func TestResDecoratorReqAndResWithOtherParams(t *testing.T) {
	input := `
package main

@Controller("/api/streaming") 
type StreamingController struct {
}

@Get("/data")
func StreamData(
	@Req() req *Request,
	@Res() res *ResponseWriter,
	@Query("format", { defaultValue: "json" }) format string,
	@Headers("Accept-Encoding") encoding string
) {
}
`

	expected := []string{
		"req := ctx.GetRequest()",
		"res := ctx.GetResponseWriter()",
		"var format string",
		"queryValue := ctx.GetQuery(\"format\")",
		"if queryValue == \"\" {",
		"queryValue = \"json\"",
		"}",
		"format = queryValue",
		"var encoding string",
		"headerValue := ctx.GetHeader(\"Accept-Encoding\")",
		"encoding = headerValue",
	}

	testResGeneration(t, input, expected, "request and response parameters with other decorators")
}

// Helper function for testing @Res() decorator generation
func testResGeneration(t *testing.T, input string, expected []string, description string) {
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

func TestResDecoratorImports(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Res() res *ResponseWriter) {
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