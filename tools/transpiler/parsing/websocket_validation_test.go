package parsing

import (
	"strings"
	"testing"
	"github.com/healtronlabs/gofasta/tools/transpiler/core"
)

func TestWebSocketFunctionValidation(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid OnGatewayConnection function",
			input: `@OnGatewayConnection()
func HandleConnection(client *WebSocketClient, headers map[string]string) {
}`,
			expectError: false,
		},
		{
			name: "OnGatewayConnection with invalid arguments",
			input: `@OnGatewayConnection("invalid")
func HandleConnection(client *WebSocketClient) {
}`,
			expectError: true,
			errorMsg:    "@OnGatewayConnection() decorator should not have arguments",
		},
		{
			name: "OnGatewayConnection with invalid parameter type",
			input: `@OnGatewayConnection()
func HandleConnection(client *WebSocketClient, invalidParam int) {
}`,
			expectError: true,
			errorMsg:    "invalid parameter type 'int' for @OnGatewayConnection function",
		},
		{
			name: "OnGatewayConnection with invalid return type",
			input: `@OnGatewayConnection()
func HandleConnection(client *WebSocketClient) string {
}`,
			expectError: true,
			errorMsg:    "should return void or error, got 'string'",
		},
		{
			name: "Valid OnGatewayDisconnect function",
			input: `@OnGatewayDisconnect()
func HandleDisconnect(client *WebSocketClient, reason string) {
}`,
			expectError: false,
		},
		{
			name: "OnGatewayDisconnect with invalid parameter type",
			input: `@OnGatewayDisconnect()
func HandleDisconnect(client *WebSocketClient, invalidParam float64) {
}`,
			expectError: true,
			errorMsg:    "invalid parameter type 'float64' for @OnGatewayDisconnect function",
		},
		{
			name: "Valid OnGatewayInit function",
			input: `@OnGatewayInit()
func HandleInit() {
}`,
			expectError: false,
		},
		{
			name: "OnGatewayInit with too many parameters",
			input: `@OnGatewayInit()
func HandleInit(param1 string, param2 string, param3 string) {
}`,
			expectError: true,
			errorMsg:    "should have minimal parameters, got 3 parameters",
		},
		{
			name: "Valid SubscribeMessage function with string event",
			input: `@SubscribeMessage("message")
func HandleMessage(data *ChatMessage, client *WebSocketClient) {
}`,
			expectError: false,
		},
		{
			name: "Valid SubscribeMessage function with event array",
			input: `@SubscribeMessage(["join_room", "leave_room"])
func HandleRoomActions(data *RoomAction, client *WebSocketClient) {
}`,
			expectError: false,
		},
		{
			name: "SubscribeMessage with no arguments",
			input: `@SubscribeMessage()
func HandleMessage(data *ChatMessage) {
}`,
			expectError: true,
			errorMsg:    "requires at least one event name argument",
		},
		{
			name: "SubscribeMessage with empty event name",
			input: `@SubscribeMessage("")
func HandleMessage(data *ChatMessage) {
}`,
			expectError: true,
			errorMsg:    "event name cannot be empty",
		},
		{
			name: "SubscribeMessage with empty event array",
			input: `@SubscribeMessage([])
func HandleMessage(data *ChatMessage) {
}`,
			expectError: true,
			errorMsg:    "event array cannot be empty",
		},
		{
			name: "SubscribeMessage with invalid event type",
			input: `@SubscribeMessage(123)
func HandleMessage(data *ChatMessage) {
}`,
			expectError: true,
			errorMsg:    "argument must be a string or array of strings",
		},
		{
			name: "SubscribeMessage with bool parameter type (now valid)",
			input: `@SubscribeMessage("message")
func HandleMessage(data *ChatMessage, flagParam bool) {
}`,
			expectError: false,
		},
		{
			name: "Multiple WebSocket decorators on same function",
			input: `@OnGatewayConnection()
@OnGatewayDisconnect()
func HandleConnection(client *WebSocketClient) {
}`,
			expectError: false, // Both decorators are WebSocket decorators, so validation will run for both
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := NewLexer(test.input)
			parser := NewParser(lexer)
			
			// Parse the file
			result, _ := parser.ParseFile()
			
			hasError := len(parser.errors) > 0
			if hasError != test.expectError {
				if test.expectError {
					t.Errorf("Expected parsing error but got none. Parsed successfully with %d declarations", len(result.Declarations))
				} else {
					t.Errorf("Expected no parsing error but got: %v", parser.errors)
				}
				return
			}
			
			if test.expectError && test.errorMsg != "" {
				found := false
				for _, err := range parser.errors {
					if strings.Contains(err, test.errorMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error message containing '%s', but got errors: %v", test.errorMsg, parser.errors)
				}
			}
		})
	}
}

func TestWebSocketGatewayValidation(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid WebSocket gateway with port",
			input: `@WebSocketGateway(8080)
type ChatGateway struct {
    @Inject("chatService")
    chatService *ChatService
}`,
			expectError: false,
		},
		{
			name: "Valid WebSocket gateway with config",
			input: `@WebSocketGateway({
    port: 3000,
    namespace: "/chat",
    cors: true,
    transports: ["websocket", "polling"]
})
type ChatGateway struct {
    @Inject("chatService")
    chatService *ChatService
}`,
			expectError: false,
		},
		{
			name: "Valid WebSocket gateway with CORS object config",
			input: `@WebSocketGateway({
    port: 4000,
    namespace: "/notifications", 
    cors: {
        origin: "*",
        credentials: true
    }
})
type NotificationGateway struct {
    @Inject("service")
    service *Service
}`,
			expectError: false,
		},
		{
			name: "Valid WebSocket gateway with complex config including timeouts",
			input: `@WebSocketGateway({
    port: 3000,
    namespace: "/api/v1/websocket",
    cors: {
        origin: "*",
        credentials: true
    },
    transports: ["websocket"],
    pingTimeout: 5000,
    pingInterval: 2500
})
type ComplexTestGateway struct {
    @Inject("service")
    service *Service
}`,
			expectError: false,
		},
		{
			name: "WebSocket gateway with valid but minimal config",
			input: `@WebSocketGateway()
type ChatGateway struct {
    @Inject("chatService")
    chatService *ChatService
}`,
			expectError: false,
		},
		{
			name: "WebSocket gateway with invalid port",
			input: `@WebSocketGateway(99999)
type ChatGateway struct {
    chatService *ChatService
}`,
			expectError: true,
			errorMsg:    "port 99999 is invalid, must be between 1-65535",
		},
		{
			name: "WebSocket gateway with empty namespace",
			input: `@WebSocketGateway({namespace: ""})
type ChatGateway struct {
    chatService *ChatService
}`,
			expectError: true,
			errorMsg:    "config 'namespace' cannot be empty",
		},
		{
			name: "WebSocket gateway with invalid transport",
			input: `@WebSocketGateway({transports: ["websocket", "invalid"]})
type ChatGateway struct {
    chatService *ChatService
}`,
			expectError: true,
			errorMsg:    "invalid transport 'invalid', must be 'websocket' or 'polling'",
		},
		{
			name: "WebSocket gateway with invalid config key",
			input: `@WebSocketGateway({invalidKey: "value"})
type ChatGateway struct {
    chatService *ChatService
}`,
			expectError: true,
			errorMsg:    "config has invalid key 'invalidKey'",
		},
		{
			name: "Gateway method with lifecycle decorator",
			input: `@WebSocketGateway(8080)
type ChatGateway struct {
    @OnGatewayConnection()
    func HandleConnection(client *WebSocketClient) {
    }
}`,
			expectError: true,
			errorMsg:    "lifecycle decorator @OnGatewayConnection should not be used on gateway methods",
		},
		{
			name: "Gateway method with valid SubscribeMessage",
			input: `@WebSocketGateway(8080)
type ChatGateway struct {
    @SubscribeMessage("message")
    func HandleMessage(data *ChatMessage, client *WebSocketClient) {
    }
}`,
			expectError: false,
		},
		{
			name: "Gateway field without Inject decorator",
			input: `@WebSocketGateway(8080)
type ChatGateway struct {
    chatService *ChatService
}`,
			expectError: true,
			errorMsg:    "should use @Inject decorator for dependency injection",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := NewLexer(test.input)
			parser := NewParser(lexer)
			
			// Parse the file
			result, _ := parser.ParseFile()
			
			hasError := len(parser.errors) > 0
			if hasError != test.expectError {
				if test.expectError {
					t.Errorf("Expected parsing error but got none. Parsed successfully with %d declarations", len(result.Declarations))
				} else {
					t.Errorf("Expected no parsing error but got: %v", parser.errors)
				}
				return
			}
			
			if test.expectError && test.errorMsg != "" {
				found := false
				for _, err := range parser.errors {
					if strings.Contains(err, test.errorMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error message containing '%s', but got errors: %v", test.errorMsg, parser.errors)
				}
			}
		})
	}
}

func TestWebSocketValidationComplexCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsgs   []string
	}{
		{
			name: "Multiple validation errors in one function",
			input: `@SubscribeMessage()
func HandleMessage(invalidParam int) bool {
}`,
			expectError: true,
			errorMsgs: []string{
				"requires at least one event name argument",
			},
		},
		{
			name: "Multiple validation errors in gateway",
			input: `@WebSocketGateway(99999)
type ChatGateway struct {
    chatService *ChatService
    @OnGatewayConnection()
    func HandleConnection() {
    }
}`,
			expectError: true,
			errorMsgs: []string{
				"port 99999 is invalid",
				"should use @Inject decorator",
				"lifecycle decorator @OnGatewayConnection should not be used on gateway methods",
			},
		},
		{
			name: "Valid complex WebSocket gateway",
			input: `@WebSocketGateway({
    port: 3000,
    namespace: "/chat",
    cors: true
})
type ChatGateway struct {
    @Inject("chatService")
    chatService *ChatService
    
    @Inject("logger")
    logger *Logger
    
    @SubscribeMessage(["join_room", "leave_room"])
    func HandleRoomActions(data *RoomAction, client *WebSocketClient, ack *AckCallback) error {
    }
    
    @SubscribeMessage("message")
    func HandleMessage(data *ChatMessage, client *WebSocketClient) {
    }
}`,
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := NewLexer(test.input)
			parser := NewParser(lexer)
			
			// Parse the file
			result, _ := parser.ParseFile()
			
			hasError := len(parser.errors) > 0
			if hasError != test.expectError {
				if test.expectError {
					t.Errorf("Expected parsing error but got none. Parsed successfully with %d declarations", len(result.Declarations))
				} else {
					t.Errorf("Expected no parsing error but got: %v", parser.errors)
				}
				return
			}
			
			if test.expectError {
				for _, expectedMsg := range test.errorMsgs {
					found := false
					for _, err := range parser.errors {
						if strings.Contains(err, expectedMsg) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error message containing '%s', but got errors: %v", expectedMsg, parser.errors)
					}
				}
			}
		})
	}
}

func TestWebSocketValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
	}{
		{
			name: "SubscribeMessage with mixed valid/invalid events in array",
			input: `@SubscribeMessage(["valid_event", "", "another_valid"])
func HandleMessage(data *ChatMessage) {
}`,
			expectError: true,
			errorMsg:    "event name cannot be empty (array item 2)",
		},
		{
			name: "SubscribeMessage with non-string in event array",
			input: `@SubscribeMessage(["valid_event", 123, "another_valid"])
func HandleMessage(data *ChatMessage) {
}`,
			expectError: true,
			errorMsg:    "event array must contain only strings (invalid item 2)",
		},
		{
			name: "OnGatewayConnection with error return type (valid)",
			input: `@OnGatewayConnection()
func HandleConnection(client *WebSocketClient) error {
}`,
			expectError: false,
		},
		{
			name: "WebSocket gateway with zero port",
			input: `@WebSocketGateway(0)
type ChatGateway struct {
}`,
			expectError: true,
			errorMsg:    "port 0 is invalid, must be between 1-65535",
		},
		{
			name: "WebSocket gateway config with wrong type for cors",
			input: `@WebSocketGateway({cors: "true"})
type ChatGateway struct {
}`,
			expectError: true,
			errorMsg:    "config 'cors' must be a boolean or configuration object",
		},
		{
			name: "WebSocket gateway with negative pingTimeout",
			input: `@WebSocketGateway({pingTimeout: -1000})
type ChatGateway struct {
}`,
			expectError: true,
			errorMsg:    "config 'pingTimeout' must be",
		},
		{
			name: "WebSocket gateway with invalid pingInterval type",
			input: `@WebSocketGateway({pingInterval: "2500"})
type ChatGateway struct {
}`,
			expectError: true,
			errorMsg:    "config 'pingInterval' must be an integer",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := NewLexer(test.input)
			parser := NewParser(lexer)
			
			// Parse the file
			result, _ := parser.ParseFile()
			
			hasError := len(parser.errors) > 0
			if hasError != test.expectError {
				if test.expectError {
					t.Errorf("Expected parsing error but got none. Parsed successfully with %d declarations", len(result.Declarations))
				} else {
					t.Errorf("Expected no parsing error but got: %v", parser.errors)
				}
				return
			}
			
			if test.expectError && test.errorMsg != "" {
				found := false
				for _, err := range parser.errors {
					if strings.Contains(err, test.errorMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error message containing '%s', but got errors: %v", test.errorMsg, parser.errors)
				}
			}
		})
	}
}
func TestComplexWebSocketConfigurationParsing(t *testing.T) {
	// Test the exact configuration from websocket-gateway-complex-test.gofa
	input := `package main

@WebSocketGateway({
	port: 3000,
	namespace: "/api/v1/websocket", 
	cors: {
		origin: "*",
		credentials: true
	},
	transports: ["websocket"],
	pingTimeout: 5000,
	pingInterval: 2500
})
type ComplexTestGateway struct {
	@Inject("gameService")
	gameService *GameService
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	
	file, err := parser.ParseFile()
	if err != nil {
		t.Fatalf("Complex config parsing failed: %v", err)
	}
	
	if len(parser.errors) > 0 {
		t.Fatalf("Parser validation errors: %v", parser.errors)  
	}
	
	if len(file.Declarations) != 1 {
		t.Errorf("Expected 1 declaration, got %d", len(file.Declarations))
	}
	
	t.Logf("✅ Complex WebSocket configuration with pingTimeout and pingInterval parsed successfully!")
}

func TestWebSocketParameterTypeValidation(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid SubscribeMessage with slice parameter",
			input: `@SubscribeMessage("complex_action")
func ComplexHandler(
	@MessageBody() body string,
	@ConnectedSocket() client *WebSocketClient,
	@Rooms() rooms []string,
	@Headers() headers map[string]string
) {
}`,
			expectError: false,
		},
		{
			name: "Valid SubscribeMessage with various slice types",
			input: `@SubscribeMessage("data_event")
func HandleData(
	@MessageBody() data []interface{},
	@ConnectedSocket() client *WebSocketClient,
	@UserIds() userIds []string,
	@Scores() scores []int
) {
}`,
			expectError: false,
		},
		{
			name: "Valid SubscribeMessage with custom types and slices",
			input: `@SubscribeMessage("user_action")
func HandleUserAction(
	@MessageBody() action *UserAction,
	@ConnectedSocket() client *WebSocketClient,
	@Rooms() rooms []string,
	@Active() isActive bool,
	@Count() count int
) {
}`,
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := NewLexer(test.input)
			parser := NewParser(lexer)
			
			// Parse the file
			_, _ = parser.ParseFile()
			
			hasError := len(parser.errors) > 0
			if hasError != test.expectError {
				if test.expectError {
					t.Errorf("Expected parsing error but got none")
				} else {
					t.Errorf("Expected no parsing error but got: %v", parser.errors)
				}
				return
			}
			
			if test.expectError && test.errorMsg != "" {
				found := false
				for _, err := range parser.errors {
					if strings.Contains(err, test.errorMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error message containing '%s', but got errors: %v", test.errorMsg, parser.errors)
				}
			}
		})
	}
}

// TestWebSocketLifecycleParameterTypeValidation tests that lifecycle methods accept custom parameter types
func TestWebSocketLifecycleParameterTypeValidation(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid OnGatewayConnection with custom User parameter",
			input: `@OnGatewayConnection()
func HandlePlayerConnection(
	@ConnectedSocket() client *WebSocketClient,
	@CurrentUser() user *User,
	@Headers() headers map[string]string
) {
}`,
			expectError: false,
		},
		{
			name: "Valid OnGatewayDisconnect with custom User parameter",
			input: `@OnGatewayDisconnect()
func HandlePlayerDisconnection(
	@ConnectedSocket() client *WebSocketClient,
	@CurrentUser() user *User,
	@DisconnectReason() reason string
) {
}`,
			expectError: false,
		},
		{
			name: "Valid OnGatewayConnection with multiple custom types",
			input: `@OnGatewayConnection()
func HandleGameConnection(
	@ConnectedSocket() client *WebSocketClient,
	@CurrentUser() user *User,
	@GameState() state *GameState,
	@ClientIP() ip string
) {
}`,
			expectError: false,
		},
		{
			name: "Valid OnGatewayDisconnect with slice and custom types",
			input: `@OnGatewayDisconnect()
func HandleGameDisconnection(
	@ConnectedSocket() client *WebSocketClient,
	@CurrentUser() user *User,
	@Rooms() rooms []string,
	@GameSession() session *GameSession
) {
}`,
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := NewLexer(test.input)
			parser := NewParser(lexer)
			
			// Parse the file
			_, _ = parser.ParseFile()
			
			hasError := len(parser.errors) > 0
			if hasError != test.expectError {
				if test.expectError {
					t.Errorf("Expected parsing error but got none")
				} else {
					t.Errorf("Expected no parsing error but got: %v", parser.errors)
				}
				return
			}
			
			if test.expectError && test.errorMsg != "" {
				found := false
				for _, err := range parser.errors {
					if strings.Contains(err, test.errorMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error containing \"%s\" but got: %v", test.errorMsg, parser.errors)
				}
			}
		})
	}
}

// TestWebSocketFullIntegration tests complete WebSocket file parsing through main parser flow
func TestWebSocketFullIntegration(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		validate    func(*core.GofaFile) bool
	}{
		{
			name: "Complete WebSocket Gateway with lifecycle methods and message handlers",
			input: `package main

@WebSocketGateway(8080)
type GameGateway struct {
	@Inject("gameService")
	gameService *GameService
	
	@Inject("userService") 
	userService *UserService
}

@OnGatewayConnection()
func HandlePlayerConnection(
	@ConnectedSocket() client *WebSocketClient,
	@CurrentUser() user *User,
	@Headers() headers map[string]string
) {
	// Handle player connection
}

@OnGatewayDisconnect()
func HandlePlayerDisconnection(
	@ConnectedSocket() client *WebSocketClient,
	@CurrentUser() user *User,
	@DisconnectReason() reason string
) {
	// Handle player disconnection  
}

@SubscribeMessage("join_game")
func HandleJoinGame(
	@MessageBody() data *JoinGameRequest,
	@ConnectedSocket() client *WebSocketClient,
	@CurrentUser() user *User
) {
	// Handle join game
}

@SubscribeMessage(["move", "action"]) 
func HandleGameActions(
	@MessageBody() action *GameAction,
	@ConnectedSocket() client *WebSocketClient,
	@Rooms() rooms []string
) {
	// Handle game actions
}

@Injectable()
type GameService struct {}

@Injectable() 
type UserService struct {}

type JoinGameRequest struct {
	GameID string
	PlayerName string
}

type GameAction struct {
	Type string
	Data interface{}
}

type User struct {
	ID string
	Name string
}`,
			expectError: false,
			validate: func(file *core.GofaFile) bool {
				// Should have declarations for Gateway, functions, and services
				if len(file.Declarations) < 6 {
					return false
				}
				
				// First should be WebSocket Gateway
				if gateway, ok := file.Declarations[0].(*core.WebSocketGatewayDeclaration); ok {
					if gateway.Name != "GameGateway" || len(gateway.Fields) != 2 {
						return false
					}
				} else {
					return false
				}
				
				// Should have WebSocket function declarations
				wsFunc1, ok1 := file.Declarations[1].(*core.WebSocketFunctionDeclaration)
				wsFunc2, ok2 := file.Declarations[2].(*core.WebSocketFunctionDeclaration)
				wsFunc3, ok3 := file.Declarations[3].(*core.WebSocketFunctionDeclaration)
				wsFunc4, ok4 := file.Declarations[4].(*core.WebSocketFunctionDeclaration)
				
				if !ok1 || !ok2 || !ok3 || !ok4 {
					return false
				}
				
				// Validate function names
				expectedFuncs := []string{"HandlePlayerConnection", "HandlePlayerDisconnection", "HandleJoinGame", "HandleGameActions"}
				actualFuncs := []string{wsFunc1.Name, wsFunc2.Name, wsFunc3.Name, wsFunc4.Name}
				
				for i, expected := range expectedFuncs {
					if actualFuncs[i] != expected {
						return false
					}
				}
				
				return true
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := NewLexer(test.input)
			parser := NewParser(lexer)
			
			// Parse complete file using main parser flow
			file, err := parser.ParseFile()
			
			hasError := err != nil
			if hasError != test.expectError {
				if test.expectError {
					t.Errorf("Expected parsing error but got none")
				} else {
					t.Errorf("Expected no parsing error but got: %v", err)
				}
				return
			}
			
			// If parsing succeeded and we have validation, run it
			if !test.expectError && test.validate != nil {
				if !test.validate(file) {
					t.Errorf("File validation failed")
				}
			}
			
			// Verify no parser errors accumulated
			if len(parser.Errors()) > 0 {
				t.Errorf("Parser accumulated errors: %v", parser.Errors())
			}
		})
	}
}
