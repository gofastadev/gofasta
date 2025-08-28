package parsing

import (
	"strings"
	"testing"
	
	"github.com/healtronlabs/gofasta/tools/transpiler/core"
)

// TestParserFixForInfiniteLoop tests that parser no longer hangs on complex input
func TestParserFixForInfiniteLoop(t *testing.T) {
	input := `package main

import "fmt"

@Controller("/test")
type TestController struct {
	Service *TestService ` + "`inject:\"\"`" + `
}

@Get("/")
func GetTest() {
}`

	_, err := ParseGofaFile(input)
	if err != nil {
		t.Logf("Parse error (expected): %v", err)
		// This is okay - we just want to make sure it doesn't hang
	}
}

// TestParserWithSimpleInput tests parser with minimal valid input
func TestParserWithSimpleInput(t *testing.T) {
	input := `package main

type TestController struct {
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Simple parse failed: %v", err)
	}
	
	if file == nil || file.Package.Name != "main" {
		t.Error("Failed to parse simple input correctly")
	}
	
	if len(file.Declarations) != 1 {
		t.Errorf("Expected 1 declaration, got %d", len(file.Declarations))
	}
}

// TestParserWithDecorator tests parsing with a simple decorator
func TestParserWithDecorator(t *testing.T) {
	input := `package main

@Controller("/test")
type TestController struct {
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Decorator parse failed: %v", err)
	}
	
	if file == nil || len(file.Declarations) != 1 {
		t.Error("Failed to parse decorated struct")
	}
	
	controller, ok := file.Declarations[0].(*core.ControllerDeclaration)
	if !ok {
		t.Error("Expected ControllerDeclaration")
	} else if len(controller.Decorators) != 1 {
		t.Errorf("Expected 1 decorator, got %d", len(controller.Decorators))
	}
}

// TestParserErrorRecoveryFixed tests that parser recovers from errors without hanging
func TestParserErrorRecoveryFixed(t *testing.T) {
	tests := []string{
		"@Controller",          // incomplete decorator
		"type struct {}",       // missing type name
		"@Get(/invalid)",       // invalid decorator args
		"func InvalidFunction(", // incomplete function
	}
	
	for _, input := range tests {
		t.Run("input: "+input, func(t *testing.T) {
			_, err := ParseGofaFile(input)
			// We expect errors, but the parser shouldn't hang
			if err == nil {
				t.Log("Unexpectedly succeeded, but that's okay")
			} else {
				t.Logf("Got expected error: %v", err)
			}
		})
	}
}

// TestParserWithInvalidTokens tests parser with various invalid tokens
func TestParserWithInvalidTokens(t *testing.T) {
	// Test with tokens that might cause infinite loops
	inputs := []string{
		"###invalid###",
		"@@@@@",
		"(((((",
		"}}}}",
		strings.Repeat("@Controller", 100), // Repetitive tokens
	}
	
	for i, input := range inputs {
		t.Run("invalid_input_"+string(rune('A'+i)), func(t *testing.T) {
			_, err := ParseGofaFile(input)
			// Should get an error without hanging
			if err == nil {
				t.Error("Expected error for invalid input")
			} else {
				t.Logf("Got expected error: %v", err)
			}
		})
	}
}
// TestWebSocketGatewayParsingSimple tests basic WebSocket gateway parsing with port
func TestWebSocketGatewayParsingSimple(t *testing.T) {
	input := `package main

@WebSocketGateway(8080)
type ChatGateway struct {
	@Inject("chatService")
	chatService *ChatService
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("WebSocket gateway parse failed: %v", err)
	}
	
	if file == nil || len(file.Declarations) != 1 {
		t.Fatalf("Expected 1 declaration, got %d", len(file.Declarations))
	}
	
	gateway, ok := file.Declarations[0].(*core.WebSocketGatewayDeclaration)
	if !ok {
		t.Fatalf("Expected WebSocketGatewayDeclaration, got %T", file.Declarations[0])
	}
	
	// Check basic properties
	if gateway.Name != "ChatGateway" {
		t.Errorf("Expected name \"ChatGateway\", got \"%s\"", gateway.Name)
	}
	
	// Check that we have the WebSocketGateway decorator at the struct level
	found := false
	for _, decorator := range gateway.Decorators {
		if decorator.Name == "WebSocketGateway" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find WebSocketGateway decorator")
	}
	
	// Check port extraction
	if gateway.Port == nil {
		t.Error("Expected port to be set")
	} else if *gateway.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", *gateway.Port)
	}
	
	// Check fields
	if len(gateway.Fields) != 1 {
		t.Errorf("Expected 1 field, got %d", len(gateway.Fields))
	}
	
	if len(gateway.Fields) > 0 && gateway.Fields[0].Name != "chatService" {
		t.Errorf("Expected field name \"chatService\", got \"%s\"", gateway.Fields[0].Name)
	}
}

// TestWebSocketGatewayParsingAdvanced tests WebSocket gateway with complex configuration
func TestWebSocketGatewayParsingAdvanced(t *testing.T) {
	input := `package main

@WebSocketGateway({
	port: 8080,
	namespace: "/chat",
	cors: true
})
type AdvancedGateway struct {
	@Inject("service")
	service *Service
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Advanced WebSocket gateway parse failed: %v", err)
	}
	
	if file == nil || len(file.Declarations) != 1 {
		t.Fatalf("Expected 1 declaration, got %d", len(file.Declarations))
	}
	
	gateway, ok := file.Declarations[0].(*core.WebSocketGatewayDeclaration)
	if !ok {
		t.Fatalf("Expected WebSocketGatewayDeclaration, got %T", file.Declarations[0])
	}
	
	// Check basic properties
	if gateway.Name != "AdvancedGateway" {
		t.Errorf("Expected name \"AdvancedGateway\", got \"%s\"", gateway.Name)
	}
	
	// Check port extraction
	if gateway.Port == nil {
		t.Error("Expected port to be set")
	} else if *gateway.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", *gateway.Port)
	}
	
	// Check namespace extraction
	if gateway.Namespace == nil {
		t.Error("Expected namespace to be set")
	} else if *gateway.Namespace != "/chat" {
		t.Errorf("Expected namespace \"/chat\", got \"%s\"", *gateway.Namespace)
	}
	
	// Check additional configuration
	if gateway.Config == nil {
		t.Error("Expected config to be initialized")
	}
	
	if corsValue, exists := gateway.Config["cors"]; !exists {
		t.Error("Expected \"cors\" in config")
	} else if cors, ok := corsValue.(bool); !ok || !cors {
		t.Errorf("Expected \"cors\" to be true, got %v", corsValue)
	}
}

// TestSubscribeMessageParsingSimple tests basic @SubscribeMessage decorator parsing
func TestSubscribeMessageParsingSimple(t *testing.T) {
	input := `package main

@WebSocketGateway(8080)
type ChatGateway struct {
	service *ChatService

	@SubscribeMessage("message")
	func handleMessage(client *WebSocketClient, data *ChatMessage) {
		// Handle message
	}
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("SubscribeMessage parse failed: %v", err)
	}
	
	if file == nil || len(file.Declarations) != 1 {
		t.Fatalf("Expected 1 declaration, got %d", len(file.Declarations))
	}
	
	gateway, ok := file.Declarations[0].(*core.WebSocketGatewayDeclaration)
	if !ok {
		t.Fatalf("Expected WebSocketGatewayDeclaration, got %T", file.Declarations[0])
	}
	
	// Check that we have one method
	if len(gateway.Methods) != 1 {
		t.Fatalf("Expected 1 method, got %d", len(gateway.Methods))
	}
	
	method := gateway.Methods[0]
	if method.Name != "handleMessage" {
		t.Errorf("Expected method name \"handleMessage\", got \"%s\"", method.Name)
	}
	
	// Check that the method has the SubscribeMessage decorator
	if len(method.Decorators) != 1 {
		t.Fatalf("Expected 1 decorator on method, got %d", len(method.Decorators))
	}
	
	decorator := method.Decorators[0]
	if decorator.Name != "SubscribeMessage" {
		t.Errorf("Expected decorator name \"SubscribeMessage\", got \"%s\"", decorator.Name)
	}
	
	// Check decorator argument
	if len(decorator.Args) != 1 {
		t.Fatalf("Expected 1 decorator argument, got %d", len(decorator.Args))
	}
	
	arg := decorator.Args[0]
	if arg.Key != "" {
		t.Errorf("Expected no key for decorator argument, got \"%s\"", arg.Key)
	}
	
	if argValue, ok := arg.Value.(string); !ok || argValue != "message" {
		t.Errorf("Expected argument value \"message\", got %v", arg.Value)
	}
}

// TestSubscribeMessageParsingMultiple tests multiple @SubscribeMessage decorators
func TestSubscribeMessageParsingMultiple(t *testing.T) {
	input := `package main

@WebSocketGateway(3000)
type GameGateway struct {
	@SubscribeMessage("join_game")
	func handleJoinGame(client *WebSocketClient, data *JoinGameData) {
		// Handle join game
	}

	@SubscribeMessage("leave_game") 
	func handleLeaveGame(client *WebSocketClient, data *LeaveGameData) {
		// Handle leave game
	}

	@SubscribeMessage("game_action")
	func handleGameAction(client *WebSocketClient, data *GameActionData) {
		// Handle game action
	}
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Multiple SubscribeMessage parse failed: %v", err)
	}
	
	gateway, ok := file.Declarations[0].(*core.WebSocketGatewayDeclaration)
	if !ok {
		t.Fatalf("Expected WebSocketGatewayDeclaration, got %T", file.Declarations[0])
	}
	
	// Check that we have three methods
	if len(gateway.Methods) != 3 {
		t.Fatalf("Expected 3 methods, got %d", len(gateway.Methods))
	}
	
	expectedMethods := []struct {
		name    string
		message string
	}{
		{"handleJoinGame", "join_game"},
		{"handleLeaveGame", "leave_game"},
		{"handleGameAction", "game_action"},
	}
	
	for i, expected := range expectedMethods {
		method := gateway.Methods[i]
		if method.Name != expected.name {
			t.Errorf("Expected method name \"%s\", got \"%s\"", expected.name, method.Name)
		}
		
		if len(method.Decorators) != 1 {
			t.Errorf("Expected 1 decorator on method %s, got %d", method.Name, len(method.Decorators))
			continue
		}
		
		decorator := method.Decorators[0]
		if decorator.Name != "SubscribeMessage" {
			t.Errorf("Expected decorator \"SubscribeMessage\" on method %s, got \"%s\"", method.Name, decorator.Name)
		}
		
		if len(decorator.Args) != 1 {
			t.Errorf("Expected 1 argument on decorator for method %s, got %d", method.Name, len(decorator.Args))
			continue
		}
		
		arg := decorator.Args[0]
		if argValue, ok := arg.Value.(string); !ok || argValue != expected.message {
			t.Errorf("Expected argument \"%s\" on method %s, got %v", expected.message, method.Name, arg.Value)
		}
	}
}

// TestSubscribeMessageParsingWithEventArrays tests @SubscribeMessage with event arrays
func TestSubscribeMessageParsingWithEventArrays(t *testing.T) {
	input := `package main

@WebSocketGateway({port: 3000})
type MultiEventGateway struct {
	@SubscribeMessage(["join_room", "leave_room"])
	func handleRoomEvents(client *WebSocketClient, data *RoomEventData) {
		// Handle multiple room events
	}
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("SubscribeMessage with arrays parse failed: %v", err)
	}
	
	gateway, ok := file.Declarations[0].(*core.WebSocketGatewayDeclaration)
	if !ok {
		t.Fatalf("Expected WebSocketGatewayDeclaration, got %T", file.Declarations[0])
	}
	
	// Check that we have one method
	if len(gateway.Methods) != 1 {
		t.Fatalf("Expected 1 method, got %d", len(gateway.Methods))
	}
	
	method := gateway.Methods[0]
	if method.Name != "handleRoomEvents" {
		t.Errorf("Expected method name \"handleRoomEvents\", got \"%s\"", method.Name)
	}
	
	// Check decorator
	if len(method.Decorators) != 1 {
		t.Fatalf("Expected 1 decorator, got %d", len(method.Decorators))
	}
	
	decorator := method.Decorators[0]
	if decorator.Name != "SubscribeMessage" {
		t.Errorf("Expected decorator \"SubscribeMessage\", got \"%s\"", decorator.Name)
	}
	
	// Check decorator argument is array
	if len(decorator.Args) != 1 {
		t.Fatalf("Expected 1 decorator argument, got %d", len(decorator.Args))
	}
	
	arg := decorator.Args[0]
	if argArray, ok := arg.Value.([]interface{}); !ok {
		t.Errorf("Expected array argument, got %T", arg.Value)
	} else {
		expectedEvents := []string{"join_room", "leave_room"}
		if len(argArray) != len(expectedEvents) {
			t.Errorf("Expected %d events, got %d", len(expectedEvents), len(argArray))
		} else {
			for i, expected := range expectedEvents {
				if event, ok := argArray[i].(string); !ok || event != expected {
					t.Errorf("Expected event \"%s\" at index %d, got %v", expected, i, argArray[i])
				}
			}
		}
	}
}

// TestSubscribeMessageParsingMixed tests mixed methods with and without decorators
func TestSubscribeMessageParsingMixed(t *testing.T) {
	input := `package main

@WebSocketGateway(8080)
type MixedGateway struct {
	service *Service
	
	@SubscribeMessage("user_message")
	func handleUserMessage(client *WebSocketClient, message *UserMessage) {
		// Handle user message
	}
	
	func helperMethod() {
		// Regular method without decorator
	}
	
	@SubscribeMessage("admin_action")
	func handleAdminAction(client *WebSocketClient, action *AdminAction) {
		// Handle admin action
	}
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Mixed methods parse failed: %v", err)
	}
	
	gateway, ok := file.Declarations[0].(*core.WebSocketGatewayDeclaration)
	if !ok {
		t.Fatalf("Expected WebSocketGatewayDeclaration, got %T", file.Declarations[0])
	}
	
	// Check that we have three methods
	if len(gateway.Methods) != 3 {
		t.Fatalf("Expected 3 methods, got %d", len(gateway.Methods))
	}
	
	// First method should have SubscribeMessage decorator
	method1 := gateway.Methods[0]
	if method1.Name != "handleUserMessage" {
		t.Errorf("Expected first method name \"handleUserMessage\", got \"%s\"", method1.Name)
	}
	if len(method1.Decorators) != 1 {
		t.Errorf("Expected 1 decorator on first method, got %d", len(method1.Decorators))
	}
	
	// Second method should have no decorators
	method2 := gateway.Methods[1]
	if method2.Name != "helperMethod" {
		t.Errorf("Expected second method name \"helperMethod\", got \"%s\"", method2.Name)
	}
	if len(method2.Decorators) != 0 {
		t.Errorf("Expected 0 decorators on helper method, got %d", len(method2.Decorators))
	}
	
	// Third method should have SubscribeMessage decorator
	method3 := gateway.Methods[2]
	if method3.Name != "handleAdminAction" {
		t.Errorf("Expected third method name \"handleAdminAction\", got \"%s\"", method3.Name)
	}
	if len(method3.Decorators) != 1 {
		t.Errorf("Expected 1 decorator on third method, got %d", len(method3.Decorators))
	}
}

