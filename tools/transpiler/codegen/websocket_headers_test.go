package codegen

import (
	"strings"
	"testing"
)

func TestWebSocketHeadersBasic(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Headers("Authorization") auth string,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var auth string",
		"headerValue := client.Handshake().Header.Get(\"Authorization\")",
		"auth = headerValue",
	}

	testWebSocketHeaderGeneration(t, input, expected, "basic WebSocket header parameter")
}

func TestWebSocketHeadersWithDefaultValue(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Headers("Content-Type", { defaultValue: "application/json" }) contentType string,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var contentType string",
		"headerValue := client.Handshake().Header.Get(\"Content-Type\")",
		"if headerValue == \"\" {",
		"headerValue = \"application/json\"",
		"contentType = headerValue",
	}

	testWebSocketHeaderGeneration(t, input, expected, "WebSocket header parameter with default value")
}

func TestWebSocketHeadersRequired(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Headers("X-API-Key", { required: true }) apiKey string,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var apiKey string",
		"headerValue := client.Handshake().Header.Get(\"X-API-Key\")",
		"if headerValue == \"\" {",
		"ctx.JSON(400, map[string]string{\"error\": \"Header 'X-API-Key' is required\"})",
		"return",
		"apiKey = headerValue",
	}

	testWebSocketHeaderGeneration(t, input, expected, "required WebSocket header parameter")
}

func TestWebSocketHeadersWithCustomClientName(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Headers("Authorization") auth string,
	@ConnectedSocket() wsClient *WebSocketClient
) {
}
`

	expected := []string{
		"var auth string",
		"headerValue := wsClient.Handshake().Header.Get(\"Authorization\")",
		"auth = headerValue",
	}

	testWebSocketHeaderGeneration(t, input, expected, "WebSocket header with custom client parameter name")
}

func TestWebSocketHeadersIntegerType(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Headers("X-Request-ID") requestId int,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var requestId int",
		"headerValue := client.Handshake().Header.Get(\"X-Request-ID\")",
		"if headerValue != \"\" {",
		"if parsedInt, err := strconv.Atoi(headerValue); err == nil {",
		"requestId = parsedInt",
		"} else {",
		"ctx.JSON(400, map[string]string{\"error\": \"Invalid integer value for header 'requestId'\"})",
		"return",
	}

	testWebSocketHeaderGeneration(t, input, expected, "WebSocket integer header parameter")
}

func TestWebSocketHeadersBooleanType(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Headers("X-Debug") debug bool,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var debug bool",
		"headerValue := client.Handshake().Header.Get(\"X-Debug\")",
		"if headerValue != \"\" {",
		"if parsedBool, err := strconv.ParseBool(headerValue); err == nil {",
		"debug = parsedBool",
		"} else {",
		"ctx.JSON(400, map[string]string{\"error\": \"Invalid boolean value for header 'debug' (use true/false)\"})",
		"return",
	}

	testWebSocketHeaderGeneration(t, input, expected, "WebSocket boolean header parameter")
}

func TestWebSocketHeadersArrayType(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Headers("Accept") acceptTypes []string,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var acceptTypes []string",
		"headerValue := client.Handshake().Header.Get(\"Accept\")",
		"if headerValue != \"\" {",
		"acceptTypes = strings.Split(headerValue, \",\")",
		"for i, v := range acceptTypes {",
		"acceptTypes[i] = strings.TrimSpace(v)",
	}

	testWebSocketHeaderGeneration(t, input, expected, "WebSocket array header parameter")
}

func TestWebSocketHeadersWithTransform(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Headers("User-Agent", { transform: "lowercase" }) userAgent string,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var userAgent string",
		"headerValue := client.Handshake().Header.Get(\"User-Agent\")",
		"headerValue = strings.ToLower(headerValue)",
		"userAgent = headerValue",
	}

	testWebSocketHeaderGeneration(t, input, expected, "WebSocket header parameter with transform")
}

func TestWebSocketHeadersComplex(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@MessageBody() data *ChatMessage,
	@Headers("Authorization", { required: true, transform: "trim" }) auth string,
	@Headers("X-Request-ID", { defaultValue: "0" }) requestId int,
	@Headers("Accept", { type: "array" }) acceptTypes []string,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		// MessageBody parameter (should be handled by WebSocket-specific logic)
		"// Extract message body",
		"var data *ChatMessage",
		
		// auth parameter (required, trimmed)
		"var auth string",
		"headerValue := client.Handshake().Header.Get(\"Authorization\")",
		"if headerValue == \"\" {",
		"ctx.JSON(400, map[string]string{\"error\": \"Header 'Authorization' is required\"})",
		"return",
		"headerValue = strings.TrimSpace(headerValue)",
		"auth = headerValue",
		
		// requestId parameter (with default)
		"var requestId int", 
		"headerValue := client.Handshake().Header.Get(\"X-Request-ID\")",
		"if headerValue == \"\" {",
		"headerValue = \"0\"",
		"if parsedInt, err := strconv.Atoi(headerValue); err == nil {",
		"requestId = parsedInt",
		
		// acceptTypes parameter (array)
		"var acceptTypes []string",
		"headerValue := client.Handshake().Header.Get(\"Accept\")",
		"if headerValue != \"\" {",
		"acceptTypes = strings.Split(headerValue, \",\")",
		"for i, v := range acceptTypes {",
		"acceptTypes[i] = strings.TrimSpace(v)",
	}

	testWebSocketHeaderGeneration(t, input, expected, "complex WebSocket header parameters")
}

func TestWebSocketHeadersLifecycleHandler(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@OnGatewayConnection()
func HandleConnection(
	@Headers("Origin") origin string,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var origin string",
		"headerValue := client.Handshake().Header.Get(\"Origin\")",
		"origin = headerValue",
	}

	testWebSocketHeaderGeneration(t, input, expected, "WebSocket header in lifecycle handler")
}

func TestWebSocketHeadersStandaloneFunction(t *testing.T) {
	input := `
package main

@SubscribeMessage("test")
func HandleTestMessage(
	@Headers("Authorization") auth string,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var auth string",
		"headerValue := client.Handshake().Header.Get(\"Authorization\")",
		"auth = headerValue",
	}

	testWebSocketHeaderGeneration(t, input, expected, "WebSocket header in standalone function")
}

// Helper function for testing WebSocket header generation
func testWebSocketHeaderGeneration(t *testing.T, input string, expectedStrings []string, testName string) {
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
		"\"github.com/healtronlabs/gofasta/packages/websocket\"",
	}

	for _, requiredImport := range requiredImports {
		if !strings.Contains(goCode, requiredImport) {
			t.Errorf("Generated code for %s missing required import: %s", testName, requiredImport)
		}
	}
}

func TestWebSocketVsHTTPHeaderContext(t *testing.T) {
	// Test both HTTP and WebSocket contexts in same file to ensure proper context switching
	input := `
package main

@Controller("/api/users")
type UserController struct {}

@Get("")
func GetUsers(@Headers("Authorization") auth string) {
}

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Headers("Authorization") auth string,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse mixed context file: %v", err)
	}

	generator := NewCodeGenerator("main")
	goCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate Go code for mixed context: %v", err)
	}

	// Should contain HTTP context header access
	if !strings.Contains(goCode, "headerValue := ctx.GetHeader(\"Authorization\")") {
		t.Error("Generated code missing HTTP context header access")
	}

	// Should contain WebSocket context header access
	if !strings.Contains(goCode, "headerValue := client.Handshake().Header.Get(\"Authorization\")") {
		t.Error("Generated code missing WebSocket context header access")
	}

	t.Logf("Generated mixed context code:\n%s", goCode)
}