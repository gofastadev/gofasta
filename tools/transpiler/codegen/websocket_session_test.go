package codegen

import (
	"strings"
	"testing"
)

func TestWebSocketSessionBasic(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Session("userId") userId string,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var userId string",
		"if sessionValue := client.Session().Get(\"userId\"); sessionValue != nil {",
		"if strValue, ok := sessionValue.(string); ok {",
		"userId = strValue",
	}

	testWebSocketSessionGeneration(t, input, expected, "basic WebSocket session parameter")
}

func TestWebSocketSessionIntegerType(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Session("userLevel") userLevel int,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var userLevel int",
		"if sessionValue := client.Session().Get(\"userLevel\"); sessionValue != nil {",
		"if intValue, ok := sessionValue.(int); ok {",
		"userLevel = intValue",
		"} else if strValue, ok := sessionValue.(string); ok {",
		"if parsedInt, err := strconv.Atoi(strValue); err == nil {",
		"userLevel = parsedInt",
	}

	testWebSocketSessionGeneration(t, input, expected, "WebSocket integer session parameter")
}

func TestWebSocketSessionBooleanType(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Session("isAdmin") isAdmin bool,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var isAdmin bool",
		"if sessionValue := client.Session().Get(\"isAdmin\"); sessionValue != nil {",
		"if boolValue, ok := sessionValue.(bool); ok {",
		"isAdmin = boolValue",
		"} else if strValue, ok := sessionValue.(string); ok {",
		"if parsedBool, err := strconv.ParseBool(strValue); err == nil {",
		"isAdmin = parsedBool",
	}

	testWebSocketSessionGeneration(t, input, expected, "WebSocket boolean session parameter")
}

func TestWebSocketSessionWithCustomClientName(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Session("userId") userId string,
	@ConnectedSocket() wsClient *WebSocketClient
) {
}
`

	expected := []string{
		"var userId string",
		"if sessionValue := wsClient.Session().Get(\"userId\"); sessionValue != nil {",
		"if strValue, ok := sessionValue.(string); ok {",
		"userId = strValue",
	}

	testWebSocketSessionGeneration(t, input, expected, "WebSocket session with custom client parameter name")
}

func TestWebSocketSessionEntireSession(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Session() sessionData map[string]interface{},
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"sessionData := client.Session().GetAll()",
	}

	testWebSocketSessionGeneration(t, input, expected, "WebSocket entire session extraction")
}

func TestWebSocketSessionStore(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Session() session *Session,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"session := client.Session()",
	}

	testWebSocketSessionGeneration(t, input, expected, "WebSocket session store extraction")
}

func TestWebSocketSessionCustomType(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Session("user") user *User,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var user *User",
		"if sessionValue := client.Session().Get(\"user\"); sessionValue != nil {",
		"if typedValue, ok := sessionValue.(*User); ok {",
		"user = typedValue",
	}

	testWebSocketSessionGeneration(t, input, expected, "WebSocket custom type session parameter")
}

func TestWebSocketSessionComplex(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@MessageBody() data *ChatMessage,
	@Session("userId") userId string,
	@Session("userLevel") userLevel int,
	@Session("isAdmin") isAdmin bool,
	@Session("preferences") preferences *UserPreferences,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		// MessageBody parameter (should be handled by WebSocket-specific logic)
		"// Extract message body",
		"var data *ChatMessage",
		
		// userId parameter (string)
		"var userId string",
		"if sessionValue := client.Session().Get(\"userId\"); sessionValue != nil {",
		"if strValue, ok := sessionValue.(string); ok {",
		"userId = strValue",
		
		// userLevel parameter (int)
		"var userLevel int",
		"if sessionValue := client.Session().Get(\"userLevel\"); sessionValue != nil {",
		"if intValue, ok := sessionValue.(int); ok {",
		"userLevel = intValue",
		"} else if strValue, ok := sessionValue.(string); ok {",
		"if parsedInt, err := strconv.Atoi(strValue); err == nil {",
		"userLevel = parsedInt",
		
		// isAdmin parameter (bool)
		"var isAdmin bool",
		"if sessionValue := client.Session().Get(\"isAdmin\"); sessionValue != nil {",
		"if boolValue, ok := sessionValue.(bool); ok {",
		"isAdmin = boolValue",
		"} else if strValue, ok := sessionValue.(string); ok {",
		"if parsedBool, err := strconv.ParseBool(strValue); err == nil {",
		"isAdmin = parsedBool",
		
		// preferences parameter (custom type)
		"var preferences *UserPreferences",
		"if sessionValue := client.Session().Get(\"preferences\"); sessionValue != nil {",
		"if typedValue, ok := sessionValue.(*UserPreferences); ok {",
		"preferences = typedValue",
	}

	testWebSocketSessionGeneration(t, input, expected, "complex WebSocket session parameters")
}

func TestWebSocketSessionLifecycleHandler(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@OnGatewayConnection()
func HandleConnection(
	@Session("userId") userId string,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var userId string",
		"if sessionValue := client.Session().Get(\"userId\"); sessionValue != nil {",
		"if strValue, ok := sessionValue.(string); ok {",
		"userId = strValue",
	}

	testWebSocketSessionGeneration(t, input, expected, "WebSocket session in lifecycle handler")
}

func TestWebSocketSessionStandaloneFunction(t *testing.T) {
	input := `
package main

@SubscribeMessage("test")
func HandleTestMessage(
	@Session("userId") userId string,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		"var userId string",
		"if sessionValue := client.Session().Get(\"userId\"); sessionValue != nil {",
		"if strValue, ok := sessionValue.(string); ok {",
		"userId = strValue",
	}

	testWebSocketSessionGeneration(t, input, expected, "WebSocket session in standalone function")
}

func TestWebSocketSessionMixed(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Headers("Authorization") authToken string,
	@Query("room") roomId string,
	@Session("userId") userId string,
	@Session("userLevel") userLevel int,
	@ConnectedSocket() client *WebSocketClient
) {
}
`

	expected := []string{
		// Headers parameter
		"var authToken string",
		"headerValue := client.Handshake().Header.Get(\"Authorization\")",
		"authToken = headerValue",
		
		// Query parameter
		"var roomId string",
		"queryValue := client.Handshake().URL.Query().Get(\"room\")",
		"roomId = queryValue",
		
		// Session parameters
		"var userId string",
		"if sessionValue := client.Session().Get(\"userId\"); sessionValue != nil {",
		"if strValue, ok := sessionValue.(string); ok {",
		"userId = strValue",
		
		"var userLevel int",
		"if sessionValue := client.Session().Get(\"userLevel\"); sessionValue != nil {",
		"if intValue, ok := sessionValue.(int); ok {",
		"userLevel = intValue",
	}

	testWebSocketSessionGeneration(t, input, expected, "mixed WebSocket Headers, Query, and Session parameters")
}

// Helper function for testing WebSocket session generation
func testWebSocketSessionGeneration(t *testing.T, input string, expectedStrings []string, testName string) {
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
		"\"github.com/healtronlabs/gofasta/packages/websocket\"",
	}

	for _, requiredImport := range requiredImports {
		if !strings.Contains(goCode, requiredImport) {
			t.Errorf("Generated code for %s missing required import: %s", testName, requiredImport)
		}
	}
}

func TestWebSocketVsHTTPSessionContext(t *testing.T) {
	// Test both HTTP and WebSocket contexts in same file to ensure proper context switching
	input := `
package main

@Controller("/api/users")
type UserController struct {}

@Get("")
func GetUsers(@Session("userId") userId string) {
}

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
	@Session("userId") userId string,
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

	// Should contain HTTP context session access
	if !strings.Contains(goCode, "if sessionValue := ctx.GetSession(\"userId\"); sessionValue != nil {") {
		t.Error("Generated code missing HTTP context session access")
	}

	// Should contain WebSocket context session access
	if !strings.Contains(goCode, "if sessionValue := client.Session().Get(\"userId\"); sessionValue != nil {") {
		t.Error("Generated code missing WebSocket context session access")
	}

	t.Logf("Generated mixed context code:\n%s", goCode)
}