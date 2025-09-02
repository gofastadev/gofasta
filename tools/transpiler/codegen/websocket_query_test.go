package codegen

import (
	"strings"
	"testing"
)

func TestWebSocketQueryBasic(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Query("room") roomId string,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var roomId string",
		"queryValue := client.Handshake().URL.Query().Get(\"room\")",
		"roomId = queryValue",
	}

	testWebSocketQueryGeneration(t, input, expected, "basic WebSocket query parameter")
}

func TestWebSocketQueryWithDefaultValue(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Query("page", { defaultValue: "1" }) page string,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var page string",
		"queryValue := client.Handshake().URL.Query().Get(\"page\")",
		"if queryValue == \"\" {",
		"queryValue = \"1\"",
		"page = queryValue",
	}

	testWebSocketQueryGeneration(t, input, expected, "WebSocket query parameter with default value")
}

func TestWebSocketQueryRequired(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Query("token", { required: true }) token string,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var token string",
		"queryValue := client.Handshake().URL.Query().Get(\"token\")",
		"if queryValue == \"\" {",
		"ctx.JSON(400, map[string]string{\"error\": \"Query parameter 'token' is required\"})",
		"return",
		"token = queryValue",
	}

	testWebSocketQueryGeneration(t, input, expected, "required WebSocket query parameter")
}

func TestWebSocketQueryWithCustomClientName(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Query("room") roomId string,
	@ConnectedSocket() wsClient *WebSocketClient
) {
}
`

	expected := []string{
		"var roomId string",
		"queryValue := wsClient.Handshake().URL.Query().Get(\"room\")",
		"roomId = queryValue",
	}

	testWebSocketQueryGeneration(t, input, expected, "WebSocket query with custom client parameter name")
}

func TestWebSocketQueryIntegerType(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Query("limit") limit int,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var limit int",
		"queryValue := client.Handshake().URL.Query().Get(\"limit\")",
		"if queryValue != \"\" {",
		"if parsedInt, err := strconv.Atoi(queryValue); err == nil {",
		"limit = parsedInt",
		"} else {",
		"ctx.JSON(400, map[string]string{\"error\": \"Invalid integer value for parameter 'limit'\"})",
		"return",
	}

	testWebSocketQueryGeneration(t, input, expected, "WebSocket integer query parameter")
}

func TestWebSocketQueryBooleanType(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Query("debug") debug bool,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var debug bool",
		"queryValue := client.Handshake().URL.Query().Get(\"debug\")",
		"if queryValue != \"\" {",
		"if parsedBool, err := strconv.ParseBool(queryValue); err == nil {",
		"debug = parsedBool",
		"} else {",
		"ctx.JSON(400, map[string]string{\"error\": \"Invalid boolean value for parameter 'debug' (use true/false)\"})",
		"return",
	}

	testWebSocketQueryGeneration(t, input, expected, "WebSocket boolean query parameter")
}

func TestWebSocketQueryArrayType(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Query("tags") tags []string,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var tags []string",
		"queryValue := client.Handshake().URL.Query().Get(\"tags\")",
		"if queryValue != \"\" {",
		"tags = strings.Split(queryValue, \",\")",
		"for i, v := range tags {",
		"tags[i] = strings.TrimSpace(v)",
	}

	testWebSocketQueryGeneration(t, input, expected, "WebSocket array query parameter")
}

func TestWebSocketQueryWithTransform(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Query("name", { transform: "lowercase" }) name string,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var name string",
		"queryValue := client.Handshake().URL.Query().Get(\"name\")",
		"queryValue = strings.ToLower(queryValue)",
		"name = queryValue",
	}

	testWebSocketQueryGeneration(t, input, expected, "WebSocket query parameter with transform")
}

func TestWebSocketQueryComplex(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@MessageBody() data *ChatMessage,
	@Query("room", { required: true, transform: "trim" }) room string,
	@Query("page", { defaultValue: "1" }) page int,
	@Query("tags", { type: "array" }) tags []string,
	@Query("debug") debug bool,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		// MessageBody parameter (should be handled by WebSocket-specific logic)
		"// Extract message body",
		"var data *ChatMessage",
		
		// room parameter (required, trimmed)
		"var room string",
		"queryValue := client.Handshake().URL.Query().Get(\"room\")",
		"if queryValue == \"\" {",
		"ctx.JSON(400, map[string]string{\"error\": \"Query parameter 'room' is required\"})",
		"return",
		"queryValue = strings.TrimSpace(queryValue)",
		"room = queryValue",
		
		// page parameter (with default)
		"var page int", 
		"queryValue := client.Handshake().URL.Query().Get(\"page\")",
		"if queryValue == \"\" {",
		"queryValue = \"1\"",
		"if parsedInt, err := strconv.Atoi(queryValue); err == nil {",
		"page = parsedInt",
		
		// tags parameter (array)
		"var tags []string",
		"queryValue := client.Handshake().URL.Query().Get(\"tags\")",
		"if queryValue != \"\" {",
		"tags = strings.Split(queryValue, \",\")",
		"for i, v := range tags {",
		"tags[i] = strings.TrimSpace(v)",
		
		// debug parameter (boolean)
		"var debug bool",
		"queryValue := client.Handshake().URL.Query().Get(\"debug\")",
		"if queryValue != \"\" {",
		"if parsedBool, err := strconv.ParseBool(queryValue); err == nil {",
		"debug = parsedBool",
	}

	testWebSocketQueryGeneration(t, input, expected, "complex WebSocket query parameters")
}

func TestWebSocketQueryLifecycleHandler(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@OnGatewayConnection()
func HandleConnection(
	@Query("room") room string,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var room string",
		"queryValue := client.Handshake().URL.Query().Get(\"room\")",
		"room = queryValue",
	}

	testWebSocketQueryGeneration(t, input, expected, "WebSocket query in lifecycle handler")
}

func TestWebSocketQueryStandaloneFunction(t *testing.T) {
	input := `
package main

@SubscribeMessage("test")
func HandleTestMessage(
	@Query("room") room string,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var room string",
		"queryValue := client.Handshake().URL.Query().Get(\"room\")",
		"room = queryValue",
	}

	testWebSocketQueryGeneration(t, input, expected, "WebSocket query in standalone function")
}

func TestWebSocketQueryArrayWithCustomSeparator(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Query("filters", { type: "array", separator: "|" }) filters []string,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var filters []string",
		"queryValue := client.Handshake().URL.Query().Get(\"filters\")",
		"if queryValue != \"\" {",
		"filters = strings.Split(queryValue, \"|\")",
		"for i, v := range filters {",
		"filters[i] = strings.TrimSpace(v)",
	}

	testWebSocketQueryGeneration(t, input, expected, "WebSocket array query parameter with custom separator")
}

// Helper function for testing WebSocket query generation
func testWebSocketQueryGeneration(t *testing.T, input string, expectedStrings []string, testName string) {
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

func TestWebSocketVsHTTPQueryContext(t *testing.T) {
	// Test both HTTP and WebSocket contexts in same file to ensure proper context switching
	input := `
package main

@Controller("/api/users")
type UserController struct {}

@Get("")
func GetUsers(@Query("page") page string) {
}

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Query("room") room string,
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

	// Should contain HTTP context query access
	if !strings.Contains(goCode, "queryValue := ctx.GetQuery(\"page\")") {
		t.Error("Generated code missing HTTP context query access")
	}

	// Should contain WebSocket context query access
	if !strings.Contains(goCode, "queryValue := client.Handshake().URL.Query().Get(\"room\")") {
		t.Error("Generated code missing WebSocket context query access")
	}

	t.Logf("Generated mixed context code:\n%s", goCode)
}