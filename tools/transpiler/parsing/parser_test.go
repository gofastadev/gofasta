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

