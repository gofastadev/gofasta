package codegen

import (
	"fmt"
	"strings"
	"testing"
)

func TestHeaderDecoratorBasic(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Headers("Authorization") auth string) {
}
`

	expected := []string{
		"var auth string",
		"headerValue := ctx.GetHeader(\"Authorization\")",
		"auth = headerValue",
	}

	testHeaderGeneration(t, input, expected, "basic header parameter")
}

func TestHeaderDecoratorWithDefaultValue(t *testing.T) {
	input := `
package main

@Controller("/api/users") 
type UserController struct {
}

@Get("")
func GetUsers(@Headers("Content-Type", { defaultValue: "application/json" }) contentType string) {
}
`

	expected := []string{
		"var contentType string",
		"headerValue := ctx.GetHeader(\"Content-Type\")",
		"if headerValue == \"\" {",
		"headerValue = \"application/json\"",
		"contentType = headerValue",
	}

	testHeaderGeneration(t, input, expected, "header parameter with default value")
}

func TestHeaderDecoratorRequired(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Headers("X-API-Key", { required: true }) apiKey string) {
}
`

	expected := []string{
		"var apiKey string",
		"headerValue := ctx.GetHeader(\"X-API-Key\")",
		"if headerValue == \"\" {",
		"ctx.JSON(400, map[string]string{\"error\": \"Header 'X-API-Key' is required\"})",
		"return",
		"apiKey = headerValue",
	}

	testHeaderGeneration(t, input, expected, "required header parameter")
}

func TestHeaderDecoratorIntegerType(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Headers("X-Request-ID") requestId int) {
}
`

	expected := []string{
		"var requestId int",
		"headerValue := ctx.GetHeader(\"X-Request-ID\")",
		"if headerValue != \"\" {",
		"if parsedInt, err := strconv.Atoi(headerValue); err == nil {",
		"requestId = parsedInt",
		"} else {",
		"ctx.JSON(400, map[string]string{\"error\": \"Invalid integer value for header 'requestId'\"})",
		"return",
	}

	testHeaderGeneration(t, input, expected, "integer header parameter")
}

func TestHeaderDecoratorBooleanType(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Headers("X-Debug") debug bool) {
}
`

	expected := []string{
		"var debug bool",
		"headerValue := ctx.GetHeader(\"X-Debug\")",
		"if headerValue != \"\" {",
		"if parsedBool, err := strconv.ParseBool(headerValue); err == nil {",
		"debug = parsedBool",
		"} else {",
		"ctx.JSON(400, map[string]string{\"error\": \"Invalid boolean value for header 'debug' (use true/false)\"})",
		"return",
	}

	testHeaderGeneration(t, input, expected, "boolean header parameter")
}

func TestHeaderDecoratorFloatType(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Headers("X-Rate-Limit") rateLimit float64) {
}
`

	expected := []string{
		"var rateLimit float64",
		"headerValue := ctx.GetHeader(\"X-Rate-Limit\")",
		"if headerValue != \"\" {",
		"if parsedFloat, err := strconv.ParseFloat(headerValue, 64); err == nil {",
		"rateLimit = parsedFloat",
		"} else {",
		"ctx.JSON(400, map[string]string{\"error\": \"Invalid float value for header 'rateLimit'\"})",
		"return",
	}

	testHeaderGeneration(t, input, expected, "float header parameter")
}

func TestHeaderDecoratorArrayType(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Headers("Accept") acceptTypes []string) {
}
`

	expected := []string{
		"var acceptTypes []string",
		"headerValue := ctx.GetHeader(\"Accept\")",
		"if headerValue != \"\" {",
		"acceptTypes = strings.Split(headerValue, \",\")",
		"for i, v := range acceptTypes {",
		"acceptTypes[i] = strings.TrimSpace(v)",
	}

	testHeaderGeneration(t, input, expected, "array header parameter")
}

func TestHeaderDecoratorArrayWithCustomSeparator(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Headers("X-Tags", { type: "array", separator: "|" }) tags []string) {
}
`

	expected := []string{
		"var tags []string",
		"headerValue := ctx.GetHeader(\"X-Tags\")",
		"if headerValue != \"\" {",
		"tags = strings.Split(headerValue, \"|\")",
		"for i, v := range tags {",
		"tags[i] = strings.TrimSpace(v)",
	}

	testHeaderGeneration(t, input, expected, "array header parameter with custom separator")
}

func TestHeaderDecoratorWithTransform(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Headers("User-Agent", { transform: "lowercase" }) userAgent string) {
}
`

	expected := []string{
		"var userAgent string",
		"headerValue := ctx.GetHeader(\"User-Agent\")",
		"headerValue = strings.ToLower(headerValue)",
		"userAgent = headerValue",
	}

	testHeaderGeneration(t, input, expected, "header parameter with lowercase transform")
}

func TestHeaderDecoratorWithUppercaseTransform(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Headers("X-Custom", { transform: "uppercase" }) custom string) {
}
`

	expected := []string{
		"var custom string",
		"headerValue := ctx.GetHeader(\"X-Custom\")",
		"headerValue = strings.ToUpper(headerValue)",
		"custom = headerValue",
	}

	testHeaderGeneration(t, input, expected, "header parameter with uppercase transform")
}

func TestHeaderDecoratorWithTrimTransform(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Headers("X-Token", { transform: "trim" }) token string) {
}
`

	expected := []string{
		"var token string",
		"headerValue := ctx.GetHeader(\"X-Token\")",
		"headerValue = strings.TrimSpace(headerValue)",
		"token = headerValue",
	}

	testHeaderGeneration(t, input, expected, "header parameter with trim transform")
}

func TestHeaderDecoratorComplex(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(
	@Headers("Authorization", { required: true, transform: "trim" }) auth string,
	@Headers("X-Request-ID", { defaultValue: "0" }) requestId int,
	@Headers("Content-Type", { defaultValue: "application/json", transform: "lowercase" }) contentType string,
	@Headers("Accept", { type: "array" }) acceptTypes []string,
	@Headers("X-Debug") debug bool
) {
}
`

	expected := []string{
		// auth parameter
		"var auth string",
		"headerValue := ctx.GetHeader(\"Authorization\")",
		"if headerValue == \"\" {",
		"ctx.JSON(400, map[string]string{\"error\": \"Header 'Authorization' is required\"})",
		"return",
		"headerValue = strings.TrimSpace(headerValue)",
		"auth = headerValue",
		
		// requestId parameter  
		"var requestId int",
		"headerValue := ctx.GetHeader(\"X-Request-ID\")",
		"if headerValue == \"\" {",
		"headerValue = \"0\"",
		"if parsedInt, err := strconv.Atoi(headerValue); err == nil {",
		"requestId = parsedInt",
		
		// contentType parameter
		"var contentType string",
		"headerValue := ctx.GetHeader(\"Content-Type\")",
		"if headerValue == \"\" {",
		"headerValue = \"application/json\"",
		"headerValue = strings.ToLower(headerValue)",
		"contentType = headerValue",
		
		// acceptTypes parameter
		"var acceptTypes []string",
		"headerValue := ctx.GetHeader(\"Accept\")",
		"if headerValue != \"\" {",
		"acceptTypes = strings.Split(headerValue, \",\")",
		"for i, v := range acceptTypes {",
		"acceptTypes[i] = strings.TrimSpace(v)",
		
		// debug parameter
		"var debug bool",
		"headerValue := ctx.GetHeader(\"X-Debug\")",
		"if headerValue != \"\" {",
		"if parsedBool, err := strconv.ParseBool(headerValue); err == nil {",
		"debug = parsedBool",
	}

	testHeaderGeneration(t, input, expected, "complex multiple header parameters")
}

func TestHeaderDecoratorWithoutName(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Headers() userAgent string) {
}
`

	expected := []string{
		"var userAgent string",
		"headerValue := ctx.GetHeader(\"userAgent\")",
		"userAgent = headerValue",
	}

	testHeaderGeneration(t, input, expected, "header parameter without explicit name (uses parameter name)")
}

func TestHeaderDecoratorCaseInsensitiveSupport(t *testing.T) {
	input := `
package main

@Controller("/api/users")
type UserController struct {
}

@Get("")
func GetUsers(@Headers("content-type", { caseInsensitive: false }) contentType string) {
}
`

	expected := []string{
		"var contentType string",
		"headerValue := ctx.GetHeader(\"content-type\")",
		"contentType = headerValue",
	}

	testHeaderGeneration(t, input, expected, "header parameter with case sensitivity option")
}

func testHeaderGeneration(t *testing.T, input string, expectedStrings []string, testName string) {
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

func TestHeaderDecoratorErrorCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		shouldParse bool
	}{
		{
			name: "Header parameter without name",
			input: `
package main
@Controller("/api")
type TestController struct {}
@Get("")
func Test(@Headers() param string) {}
`,
			shouldParse: true, // Should parse but use parameter name as header name
		},
		{
			name: "Header parameter with empty options",
			input: `
package main
@Controller("/api")
type TestController struct {}
@Get("")
func Test(@Headers("X-Custom", {}) param string) {}
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

func TestHeaderDecoratorInference(t *testing.T) {
	input := `
package main

@Controller("/api/auth")
type AuthController struct {
}

@Post("")
func Login(@Headers("Authorization") auth string, @Headers("X-Timestamp") timestamp int, @Headers("X-Secure") secure bool, @Headers("Accept") accept []string) {
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
			paramName: "auth",
			paramType: "string",
			shouldContain: []string{
				"var auth string",
				"auth = headerValue",
			},
		},
		{
			paramName: "timestamp", 
			paramType: "int",
			shouldContain: []string{
				"var timestamp int",
				"strconv.Atoi(headerValue)",
				"timestamp = parsedInt",
			},
		},
		{
			paramName: "secure",
			paramType: "bool", 
			shouldContain: []string{
				"var secure bool",
				"strconv.ParseBool(headerValue)",
				"secure = parsedBool",
			},
		},
		{
			paramName: "accept",
			paramType: "[]string",
			shouldContain: []string{
				"var accept []string",
				"accept = strings.Split(headerValue, \",\")",
				"accept[i] = strings.TrimSpace(v)",
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

func TestHeaderDecoratorCommonHeaders(t *testing.T) {
	input := `
package main

@Controller("/api/files")
type FileController struct {
}

@Post("")
func Upload(
	@Headers("Content-Type") contentType string,
	@Headers("Content-Length") contentLength int,
	@Headers("Authorization", { required: true }) auth string,
	@Headers("User-Agent") userAgent string,
	@Headers("X-Forwarded-For") forwardedFor string
) {
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

	// Verify common headers are handled correctly
	commonHeaders := []string{
		"Content-Type",
		"Content-Length", 
		"Authorization",
		"User-Agent",
		"X-Forwarded-For",
	}

	for _, header := range commonHeaders {
		if !strings.Contains(goCode, fmt.Sprintf("ctx.GetHeader(\"%s\")", header)) {
			t.Errorf("Generated code missing header extraction for: %s", header)
		}
	}

	// Verify required validation for Authorization
	if !strings.Contains(goCode, "Header 'Authorization' is required") {
		t.Error("Generated code missing required validation for Authorization header")
	}

	// Verify type conversion for Content-Length
	if !strings.Contains(goCode, "strconv.Atoi(headerValue)") {
		t.Error("Generated code missing integer conversion for Content-Length header")
	}
}