package codegen

import (
	"strings"
	"testing"
)

func TestHostParamDecoratorBasic(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@HostParam() host string) {
}
`

	expected := []string{
		"host := ctx.GetHost()",
	}

	testHostParamGeneration(t, input, expected, "basic host parameter")
}

func TestHostParamDecoratorWithOtherParameters(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@HostParam() host string, @Query("page") page string) {
}
`

	expected := []string{
		"host := ctx.GetHost()",
		"var page string",
		"queryValue := ctx.GetQuery(\"page\")",
		"page = queryValue",
	}

	testHostParamGeneration(t, input, expected, "host parameter with query parameter")
}

func TestHostParamDecoratorWithHeaders(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@HostParam() host string, @Headers("User-Agent") userAgent string) {
}
`

	expected := []string{
		"host := ctx.GetHost()",
		"var userAgent string",
		"headerValue := ctx.GetHeader(\"User-Agent\")",
		"userAgent = headerValue",
	}

	testHostParamGeneration(t, input, expected, "host parameter with headers parameter")
}

func TestHostParamDecoratorWithBody(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Post("")
func CreateUser(@HostParam() host string, @Body() user User) {
}
`

	expected := []string{
		"host := ctx.GetHost()",
		"var user User",
		"if err := ctx.ParseJSON(&user); err != nil {",
		"ctx.JSON(400, map[string]string{\"error\": \"Invalid request body\"})",
		"return",
		"}",
	}

	testHostParamGeneration(t, input, expected, "host parameter with body parameter")
}

func TestHostParamDecoratorWithSession(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@HostParam() host string, @Session("userId") userId string) {
}
`

	expected := []string{
		"host := ctx.GetHost()",
		"var userId string",
		"if sessionValue := ctx.GetSession(\"userId\"); sessionValue != nil {",
		"if strValue, ok := sessionValue.(string); ok {",
		"userId = strValue",
		"}",
		"}",
	}

	testHostParamGeneration(t, input, expected, "host parameter with session parameter")
}

func TestHostParamDecoratorWithReqAndRes(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@HostParam() host string, @Req() req *Request, @Res() res *ResponseWriter) {
}
`

	expected := []string{
		"host := ctx.GetHost()",
		"req := ctx.GetRequest()",
		"res := ctx.GetResponseWriter()",
	}

	testHostParamGeneration(t, input, expected, "host parameter with request and response parameters")
}

func TestHostParamDecoratorWithIp(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@HostParam() host string, @Ip() clientIP string) {
}
`

	expected := []string{
		"host := ctx.GetHost()",
		"clientIP := ctx.GetClientIP()",
	}

	testHostParamGeneration(t, input, expected, "host parameter with IP parameter")
}

func TestHostParamDecoratorMultiTenantExample(t *testing.T) {
	input := `
package main

@Controller("/api/tenant") 
type TenantController struct {
}

@Get("")
func GetTenantData(
	@HostParam() host string,
	@Session("tenantId") tenantId string,
	@Headers("Authorization", { required: true }) authToken string
) {
}
`

	expected := []string{
		"host := ctx.GetHost()",
		"var tenantId string",
		"if sessionValue := ctx.GetSession(\"tenantId\"); sessionValue != nil {",
		"if strValue, ok := sessionValue.(string); ok {",
		"tenantId = strValue",
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

	testHostParamGeneration(t, input, expected, "host parameter for multi-tenant application")
}

func TestHostParamDecoratorSubdomainRoutingExample(t *testing.T) {
	input := `
package main

@Controller("/api/routing") 
type RoutingController struct {
}

@Get("/subdomain-info")
func GetSubdomainInfo(
	@HostParam() host string,
	@Query("includePort", { defaultValue: "false" }) includePort bool,
	@Headers("X-Forwarded-Host") forwardedHost string
) {
}
`

	expected := []string{
		"host := ctx.GetHost()",
		"var includePort bool",
		"queryValue := ctx.GetQuery(\"includePort\")",
		"if queryValue == \"\" {",
		"queryValue = \"false\"",
		"}",
		"if queryValue != \"\" {",
		"if parsedBool, err := strconv.ParseBool(queryValue); err == nil {",
		"includePort = parsedBool",
		"} else {",
		"ctx.JSON(400, map[string]string{\"error\": \"Invalid boolean value for parameter 'includePort' (use true/false)\"})",
		"return",
		"}",
		"}",
		"var forwardedHost string",
		"headerValue := ctx.GetHeader(\"X-Forwarded-Host\")",
		"forwardedHost = headerValue",
	}

	testHostParamGeneration(t, input, expected, "host parameter for subdomain routing")
}

func TestHostParamDecoratorCorsExample(t *testing.T) {
	input := `
package main

@Controller("/api/cors") 
type CorsController struct {
}

@Options("/preflight")
func HandlePreflight(
	@HostParam() host string,
	@Headers("Origin") origin string,
	@Headers("Access-Control-Request-Method") requestMethod string,
	@Res() res *ResponseWriter
) {
}
`

	expected := []string{
		"host := ctx.GetHost()",
		"var origin string",
		"headerValue := ctx.GetHeader(\"Origin\")",
		"origin = headerValue",
		"var requestMethod string",
		"headerValue := ctx.GetHeader(\"Access-Control-Request-Method\")",
		"requestMethod = headerValue",
		"res := ctx.GetResponseWriter()",
	}

	testHostParamGeneration(t, input, expected, "host parameter for CORS handling")
}

func TestHostParamDecoratorLoadBalancingExample(t *testing.T) {
	input := `
package main

@Controller("/api/lb") 
type LoadBalancerController struct {
}

@Get("/health")
func HealthCheck(
	@HostParam() host string,
	@Headers("X-Forwarded-For") forwardedFor string,
	@Ip() clientIP string,
	@Query("deep", { defaultValue: "false" }) deepCheck bool
) {
}
`

	expected := []string{
		"host := ctx.GetHost()",
		"var forwardedFor string",
		"headerValue := ctx.GetHeader(\"X-Forwarded-For\")",
		"forwardedFor = headerValue",
		"clientIP := ctx.GetClientIP()",
		"var deepCheck bool",
		"queryValue := ctx.GetQuery(\"deep\")",
		"if queryValue == \"\" {",
		"queryValue = \"false\"",
		"}",
		"if queryValue != \"\" {",
		"if parsedBool, err := strconv.ParseBool(queryValue); err == nil {",
		"deepCheck = parsedBool",
		"} else {",
		"ctx.JSON(400, map[string]string{\"error\": \"Invalid boolean value for parameter 'deepCheck' (use true/false)\"})",
		"return",
		"}",
		"}",
	}

	testHostParamGeneration(t, input, expected, "host parameter for load balancer health checks")
}

func TestHostParamDecoratorApiVersioningExample(t *testing.T) {
	input := `
package main

@Controller("/api/versioning") 
type VersioningController struct {
}

@Get("/version-info")
func GetVersionInfo(
	@HostParam() host string,
	@Headers("Accept", { defaultValue: "application/json" }) accept string,
	@Query("format") format string,
	@Session("clientVersion") clientVersion string
) {
}
`

	expected := []string{
		"host := ctx.GetHost()",
		"var accept string",
		"headerValue := ctx.GetHeader(\"Accept\")",
		"if headerValue == \"\" {",
		"headerValue = \"application/json\"",
		"}",
		"accept = headerValue",
		"var format string",
		"queryValue := ctx.GetQuery(\"format\")",
		"format = queryValue",
		"var clientVersion string",
		"if sessionValue := ctx.GetSession(\"clientVersion\"); sessionValue != nil {",
		"if strValue, ok := sessionValue.(string); ok {",
		"clientVersion = strValue",
		"}",
		"}",
	}

	testHostParamGeneration(t, input, expected, "host parameter for API versioning")
}

func TestHostParamDecoratorMultipleEndpoints(t *testing.T) {
	input := `
package main

@Controller("/api/multi") 
type MultiController struct {
}

@Get("/endpoint1")
func Endpoint1(@HostParam() host1 string) {
}

@Post("/endpoint2") 
func Endpoint2(@HostParam() host2 string, @Body() data RequestDto) {
}
`

	// Just test one method for simplicity
	expected := []string{
		"host1 := ctx.GetHost()",
	}

	testHostParamGeneration(t, input, expected, "multiple endpoints with host parameters")
}

func TestHostParamDecoratorComplexScenario(t *testing.T) {
	input := `
package main

@Controller("/api/complex") 
type ComplexController struct {
}

@Put("/update/:id")
func ComplexUpdate(
	@HostParam() host string,
	@Param("id") resourceId string,
	@Body() updateData UpdateDto,
	@Query("validate", { defaultValue: "true" }) validate bool,
	@Headers("Authorization", { required: true }) authToken string,
	@Session("userId") currentUser string,
	@Ip() clientIP string,
	@Req() req *Request,
	@Res() res *ResponseWriter
) {
}
`

	expected := []string{
		"host := ctx.GetHost()",
		"resourceIdValue := ctx.GetParam(\"id\")",
		"resourceId := resourceIdValue",
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
		"clientIP := ctx.GetClientIP()",
		"req := ctx.GetRequest()",
		"res := ctx.GetResponseWriter()",
	}

	testHostParamGeneration(t, input, expected, "complex scenario with all parameter decorators including host")
}

// Helper function for testing @HostParam() decorator generation
func testHostParamGeneration(t *testing.T, input string, expected []string, description string) {
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

func TestHostParamDecoratorImports(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@HostParam() host string) {
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