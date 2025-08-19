package transpiler

import (
	"strings"
	"testing"
)

func TestIpDecoratorBasic(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Ip() clientIP string) {
}
`

	expected := []string{
		"clientIP := ctx.GetClientIP()",
	}

	testIpGeneration(t, input, expected, "basic IP parameter")
}

func TestIpDecoratorWithOtherParameters(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Ip() clientIP string, @Query("page") page string) {
}
`

	expected := []string{
		"clientIP := ctx.GetClientIP()",
		"var page string",
		"queryValue := ctx.GetQuery(\"page\")",
		"page = queryValue",
	}

	testIpGeneration(t, input, expected, "IP parameter with query parameter")
}

func TestIpDecoratorWithHeaders(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Ip() clientIP string, @Headers("User-Agent") userAgent string) {
}
`

	expected := []string{
		"clientIP := ctx.GetClientIP()",
		"var userAgent string",
		"headerValue := ctx.GetHeader(\"User-Agent\")",
		"userAgent = headerValue",
	}

	testIpGeneration(t, input, expected, "IP parameter with headers parameter")
}

func TestIpDecoratorWithBody(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Post("")
func CreateUser(@Ip() clientIP string, @Body() user User) {
}
`

	expected := []string{
		"clientIP := ctx.GetClientIP()",
		"var user User",
		"if err := ctx.ParseJSON(&user); err != nil {",
		"ctx.JSON(400, map[string]string{\"error\": \"Invalid request body\"})",
		"return",
		"}",
	}

	testIpGeneration(t, input, expected, "IP parameter with body parameter")
}

func TestIpDecoratorWithSession(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Ip() clientIP string, @Session("userId") userId string) {
}
`

	expected := []string{
		"clientIP := ctx.GetClientIP()",
		"var userId string",
		"if sessionValue := ctx.GetSession(\"userId\"); sessionValue != nil {",
		"if strValue, ok := sessionValue.(string); ok {",
		"userId = strValue",
		"}",
		"}",
	}

	testIpGeneration(t, input, expected, "IP parameter with session parameter")
}

func TestIpDecoratorWithReqAndRes(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Ip() clientIP string, @Req() req *Request, @Res() res *ResponseWriter) {
}
`

	expected := []string{
		"clientIP := ctx.GetClientIP()",
		"req := ctx.GetRequest()",
		"res := ctx.GetResponseWriter()",
	}

	testIpGeneration(t, input, expected, "IP parameter with request and response parameters")
}

func TestIpDecoratorSecurityExample(t *testing.T) {
	input := `
package main

@Controller("/api/security") 
type SecurityController struct {
}

@Post("/login")
func LoginWithIpTracking(
	@Ip() clientIP string,
	@Body() credentials LoginDto,
	@Headers("User-Agent") userAgent string,
	@Session() session *Session
) {
}
`

	expected := []string{
		"clientIP := ctx.GetClientIP()",
		"var credentials LoginDto",
		"if err := ctx.ParseJSON(&credentials); err != nil {",
		"ctx.JSON(400, map[string]string{\"error\": \"Invalid request body\"})",
		"return",
		"}",
		"var userAgent string",
		"headerValue := ctx.GetHeader(\"User-Agent\")",
		"userAgent = headerValue",
		"session := ctx.GetSessionStore()",
	}

	testIpGeneration(t, input, expected, "IP parameter in security context with multiple decorators")
}

func TestIpDecoratorRateLimitingExample(t *testing.T) {
	input := `
package main

@Controller("/api/rate-limit") 
type RateLimitController struct {
}

@Get("/protected")
func ProtectedEndpoint(
	@Ip() clientIP string,
	@Query("action") action string,
	@Headers("Authorization", { required: true }) authToken string
) {
}
`

	expected := []string{
		"clientIP := ctx.GetClientIP()",
		"var action string",
		"queryValue := ctx.GetQuery(\"action\")",
		"action = queryValue",
		"var authToken string",
		"headerValue := ctx.GetHeader(\"Authorization\")",
		"if headerValue == \"\" {",
		"ctx.JSON(400, map[string]string{\"error\": \"Header 'Authorization' is required\"})",
		"return",
		"}",
		"authToken = headerValue",
	}

	testIpGeneration(t, input, expected, "IP parameter for rate limiting with validation")
}

func TestIpDecoratorAnalyticsExample(t *testing.T) {
	input := `
package main

@Controller("/api/analytics") 
type AnalyticsController struct {
}

@Get("/track")
func TrackEvent(
	@Ip() clientIP string,
	@Query("event") eventName string,
	@Session("userId") userId string,
	@Headers("Referer") referer string
) {
}
`

	expected := []string{
		"clientIP := ctx.GetClientIP()",
		"var eventName string",
		"queryValue := ctx.GetQuery(\"event\")",
		"eventName = queryValue",
		"var userId string",
		"if sessionValue := ctx.GetSession(\"userId\"); sessionValue != nil {",
		"if strValue, ok := sessionValue.(string); ok {",
		"userId = strValue",
		"}",
		"}",
		"var referer string",
		"headerValue := ctx.GetHeader(\"Referer\")",
		"referer = headerValue",
	}

	testIpGeneration(t, input, expected, "IP parameter for analytics tracking")
}

func TestIpDecoratorAuditLogExample(t *testing.T) {
	input := `
package main

@Controller("/api/audit") 
type AuditController struct {
}

@Delete("/:resourceId")
func DeleteResource(
	@Ip() clientIP string,
	@Param("resourceId") resourceId string,
	@Session("userId") userId string,
	@Headers("X-Audit-Reason", { required: true }) auditReason string
) {
}
`

	expected := []string{
		"clientIP := ctx.GetClientIP()",
		"resourceId := ctx.GetParam(\"resourceId\")",
		"var userId string",
		"if sessionValue := ctx.GetSession(\"userId\"); sessionValue != nil {",
		"if strValue, ok := sessionValue.(string); ok {",
		"userId = strValue",
		"}",
		"}",
		"var auditReason string",
		"headerValue := ctx.GetHeader(\"X-Audit-Reason\")",
		"if headerValue == \"\" {",
		"ctx.JSON(400, map[string]string{\"error\": \"Header 'X-Audit-Reason' is required\"})",
		"return",
		"}",
		"auditReason = headerValue",
	}

	testIpGeneration(t, input, expected, "IP parameter for audit logging")
}

func TestIpDecoratorMultipleEndpoints(t *testing.T) {
	input := `
package main

@Controller("/api/multi") 
type MultiController struct {
}

@Get("/endpoint1")
func Endpoint1(@Ip() ip1 string) {
}

@Post("/endpoint2") 
func Endpoint2(@Ip() ip2 string, @Body() data RequestDto) {
}
`

	// Just test one method for simplicity
	expected := []string{
		"ip1 := ctx.GetClientIP()",
	}

	testIpGeneration(t, input, expected, "multiple endpoints with IP parameters")
}

func TestIpDecoratorComplexScenario(t *testing.T) {
	input := `
package main

@Controller("/api/complex") 
type ComplexController struct {
}

@Put("/update/:id")
func ComplexUpdate(
	@Ip() clientIP string,
	@Param("id") resourceId string,
	@Body() updateData UpdateDto,
	@Query("validate", { defaultValue: "true" }) validate bool,
	@Headers("Authorization", { required: true }) authToken string,
	@Session("userId") currentUser string,
	@Req() req *Request,
	@Res() res *ResponseWriter
) {
}
`

	expected := []string{
		"clientIP := ctx.GetClientIP()",
		"resourceId := ctx.GetParam(\"id\")",
		"var updateData UpdateDto",
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
		"var currentUser string",
		"if sessionValue := ctx.GetSession(\"userId\"); sessionValue != nil {",
		"if strValue, ok := sessionValue.(string); ok {",
		"currentUser = strValue",
		"}",
		"}",
		"req := ctx.GetRequest()",
		"res := ctx.GetResponseWriter()",
	}

	testIpGeneration(t, input, expected, "complex scenario with all parameter decorators")
}

// Helper function for testing @Ip() decorator generation
func testIpGeneration(t *testing.T, input string, expected []string, description string) {
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

func TestIpDecoratorImports(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Ip() clientIP string) {
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