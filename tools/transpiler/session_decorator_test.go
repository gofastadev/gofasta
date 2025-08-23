package transpiler

import (
	"strings"
	"testing"
)

func TestSessionDecoratorBasic(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Session() session *Session) {
}
`

	expected := []string{
		"session := ctx.GetSessionStore()",
	}

	testSessionGeneration(t, input, expected, "basic session parameter")
}

func TestSessionDecoratorWithKey(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Session("userId") userId string) {
}
`

	expected := []string{
		"var userId string",
		"if sessionValue := ctx.GetSession(\"userId\"); sessionValue != nil {",
		"if strValue, ok := sessionValue.(string); ok {",
		"userId = strValue",
		"}",
		"}",
	}

	testSessionGeneration(t, input, expected, "session parameter with string key")
}

func TestSessionDecoratorWithIntKey(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Session("count") count int) {
}
`

	expected := []string{
		"var count int",
		"if sessionValue := ctx.GetSession(\"count\"); sessionValue != nil {",
		"if intValue, ok := sessionValue.(int); ok {",
		"count = intValue",
		"} else if strValue, ok := sessionValue.(string); ok {",
		"if parsedInt, err := strconv.Atoi(strValue); err == nil {",
		"count = parsedInt",
		"}",
		"}",
		"}",
	}

	testSessionGeneration(t, input, expected, "session parameter with integer key")
}

func TestSessionDecoratorWithBoolKey(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Session("isLoggedIn") isLoggedIn bool) {
}
`

	expected := []string{
		"var isLoggedIn bool",
		"if sessionValue := ctx.GetSession(\"isLoggedIn\"); sessionValue != nil {",
		"if boolValue, ok := sessionValue.(bool); ok {",
		"isLoggedIn = boolValue",
		"} else if strValue, ok := sessionValue.(string); ok {",
		"if parsedBool, err := strconv.ParseBool(strValue); err == nil {",
		"isLoggedIn = parsedBool",
		"}",
		"}",
		"}",
	}

	testSessionGeneration(t, input, expected, "session parameter with boolean key")
}

func TestSessionDecoratorWithMapType(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Session() sessionData map[string]interface{}) {
}
`

	expected := []string{
		"sessionData := ctx.GetAllSessionData()",
	}

	testSessionGeneration(t, input, expected, "session parameter with map type")
}

func TestSessionDecoratorWithInterface(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Session() sessionData interface{}) {
}
`

	expected := []string{
		"sessionData := ctx.GetAllSessionData()",
	}

	testSessionGeneration(t, input, expected, "session parameter with interface{} type")
}

func TestSessionDecoratorMultipleParameters(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Session("userId") userId string, @Query("page") page string) {
}
`

	expected := []string{
		"var userId string",
		"if sessionValue := ctx.GetSession(\"userId\"); sessionValue != nil {",
		"if strValue, ok := sessionValue.(string); ok {",
		"userId = strValue",
		"}",
		"}",
		"var page string",
		"queryValue := ctx.GetQuery(\"page\")",
		"page = queryValue",
	}

	testSessionGeneration(t, input, expected, "session parameter with other parameters")
}

func TestSessionDecoratorWithBody(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Post("")
func CreateUser(@Session("userId") userId string, @Body() user User) {
}
`

	expected := []string{
		"var userId string",
		"if sessionValue := ctx.GetSession(\"userId\"); sessionValue != nil {",
		"if strValue, ok := sessionValue.(string); ok {",
		"userId = strValue",
		"}",
		"}",
		"var user User",
		"if err := ctx.ParseJSON(&user); err != nil {",
		"ctx.JSON(400, map[string]string{\"error\": \"Invalid request body\"})",
		"return",
		"}",
	}

	testSessionGeneration(t, input, expected, "session parameter with body parameter")
}

func TestSessionDecoratorWithHeaders(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Session("theme") theme string, @Headers("Authorization") auth string) {
}
`

	expected := []string{
		"var theme string",
		"if sessionValue := ctx.GetSession(\"theme\"); sessionValue != nil {",
		"if strValue, ok := sessionValue.(string); ok {",
		"theme = strValue",
		"}",
		"}",
		"var auth string",
		"headerValue := ctx.GetHeader(\"Authorization\")",
		"auth = headerValue",
	}

	testSessionGeneration(t, input, expected, "session parameter with headers parameter")
}

func TestSessionDecoratorWithReqAndRes(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Session("userId") userId string, @Req() req *Request, @Res() res *ResponseWriter) {
}
`

	expected := []string{
		"var userId string",
		"if sessionValue := ctx.GetSession(\"userId\"); sessionValue != nil {",
		"if strValue, ok := sessionValue.(string); ok {",
		"userId = strValue",
		"}",
		"}",
		"req := ctx.GetRequest()",
		"res := ctx.GetResponseWriter()",
	}

	testSessionGeneration(t, input, expected, "session parameter with request and response parameters")
}

func TestSessionDecoratorComplexExample(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Put("/:id")
func UpdateUser(
	@Session("userId") userId string,
	@Session("role") userRole string,
	@Param("id") id string,
	@Body() updateData UpdateUserDto,
	@Query("validate", { defaultValue: "true" }) validate bool,
	@Headers("Authorization", { required: true }) authToken string
) {
}
`

	expected := []string{
		"var userId string",
		"if sessionValue := ctx.GetSession(\"userId\"); sessionValue != nil {",
		"if strValue, ok := sessionValue.(string); ok {",
		"userId = strValue",
		"}",
		"}",
		"var userRole string",
		"if sessionValue := ctx.GetSession(\"role\"); sessionValue != nil {",
		"if strValue, ok := sessionValue.(string); ok {",
		"userRole = strValue",
		"}",
		"}",
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

	testSessionGeneration(t, input, expected, "complex session parameter with multiple decorators")
}

func TestSessionDecoratorCustomType(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Session("userProfile") profile UserProfile) {
}
`

	expected := []string{
		"var profile UserProfile",
		"if sessionValue := ctx.GetSession(\"userProfile\"); sessionValue != nil {",
		"if typedValue, ok := sessionValue.(UserProfile); ok {",
		"profile = typedValue",
		"}",
		"}",
	}

	testSessionGeneration(t, input, expected, "session parameter with custom type")
}

func TestSessionDecoratorWholeSessionWithCustomType(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Session() session *CustomSession) {
}
`

	expected := []string{
		"session := ctx.GetSessionStore()",
	}

	testSessionGeneration(t, input, expected, "whole session with custom session type")
}

// Helper function for testing @Session() decorator generation
func testSessionGeneration(t *testing.T, input string, expected []string, description string) {
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

func TestSessionDecoratorImports(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Session("userId") userId string) {
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