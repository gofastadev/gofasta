package codegen

import (
	"fmt"
	"strings"
	"testing"
)

func TestWebSocketGatewayDeclarationBasic(t *testing.T) {
	input := `package test

@WebSocketGateway(8080)
type ChatGateway struct {
    @Inject("chatService")
    chatService *ChatService
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Test WebSocket imports
	if !strings.Contains(output, `"github.com/healtronlabs/gofasta/packages/websocket"`) {
		t.Errorf("Expected WebSocket import not found in output:\n%s", output)
	}

	// Test gateway struct generation
	if !strings.Contains(output, "type ChatGateway struct {") {
		t.Errorf("Expected ChatGateway struct not found in output:\n%s", output)
	}

	// Test dependency injection tags
	if !strings.Contains(output, `chatService *ChatService `+"`inject:\"chatService\"`") {
		t.Errorf("Expected dependency injection tag not found in output:\n%s", output)
	}

	// Test WebSocket server setup function
	if !strings.Contains(output, "func (ws *ChatGateway) SetupChatGateway() *websocket.WebSocketServer {") {
		t.Errorf("Expected WebSocket server setup function not found in output:\n%s", output)
	}

	// Test WebSocket configuration
	if !strings.Contains(output, "websocket.NewWebSocketServer(&websocket.Config{") {
		t.Errorf("Expected WebSocket config not found in output:\n%s", output)
	}

	if !strings.Contains(output, "Port: 8080,") {
		t.Errorf("Expected port configuration not found in output:\n%s", output)
	}

	// Test handler registration function
	if !strings.Contains(output, "func (ws *ChatGateway) RegisterChatGatewayHandlers(server *websocket.WebSocketServer) {") {
		t.Errorf("Expected handler registration function not found in output:\n%s", output)
	}
}

func TestWebSocketGatewayAdvancedConfiguration(t *testing.T) {
	input := `package test

@WebSocketGateway({
    port: 3000,
    namespace: "/chat",
    cors: true
})
type AdvancedGateway struct {}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Test advanced configuration parsing
	if !strings.Contains(output, "Port:      3000,") {
		t.Errorf("Expected port 3000 not found in output:\n%s", output)
	}

	if !strings.Contains(output, `Namespace: "/chat",`) {
		t.Errorf("Expected namespace configuration not found in output:\n%s", output)
	}

	if !strings.Contains(output, "CORS:      true,") {
		t.Errorf("Expected CORS configuration not found in output:\n%s", output)
	}
}

// TestWebSocketComplexConfiguration tests all advanced configuration options
func TestWebSocketComplexConfiguration(t *testing.T) {
	input := `package test

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
type ComplexGateway struct {}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Test port configuration
	if !strings.Contains(output, "Port:      3000,") {
		t.Errorf("Expected port 3000 not found in output:\n%s", output)
	}

	// Test namespace configuration
	if !strings.Contains(output, `Namespace: "/api/v1/websocket",`) {
		t.Errorf("Expected namespace configuration not found in output:\n%s", output)
	}

	// Test complex CORS configuration
	if !strings.Contains(output, "CORS: websocket.CORSConfig{") {
		t.Errorf("Expected CORS config object not found in output:\n%s", output)
	}
	if !strings.Contains(output, `Origin:      "*",`) {
		t.Errorf("Expected CORS origin not found in output:\n%s", output)
	}
	if !strings.Contains(output, "Credentials: true,") {
		t.Errorf("Expected CORS credentials not found in output:\n%s", output)
	}

	// Test transports configuration
	if !strings.Contains(output, "Transports: []string{") {
		t.Errorf("Expected transports array not found in output:\n%s", output)
	}
	if !strings.Contains(output, `"websocket",`) {
		t.Errorf("Expected websocket transport not found in output:\n%s", output)
	}

	// Test ping configuration
	if !strings.Contains(output, "PingTimeout:  5000,") {
		t.Errorf("Expected ping timeout not found in output:\n%s", output)
	}
	if !strings.Contains(output, "PingInterval: 2500,") {
		t.Errorf("Expected ping interval not found in output:\n%s", output)
	}
}

// TestWebSocketDefaultConfiguration tests that defaults are not unnecessarily generated
func TestWebSocketDefaultConfiguration(t *testing.T) {
	input := `package test

@WebSocketGateway({
	port: 8080,
	transports: ["websocket", "polling"],
	pingTimeout: 20000,
	pingInterval: 25000
})
type DefaultGateway struct {}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Test that default ping values are not unnecessarily generated
	if strings.Contains(output, "PingTimeout: 20000,") {
		t.Errorf("Default ping timeout should not be generated in output:\n%s", output)
	}
	if strings.Contains(output, "PingInterval: 25000,") {
		t.Errorf("Default ping interval should not be generated in output:\n%s", output)
	}

	// Test that transports are properly generated
	if !strings.Contains(output, "Transports: []string{") {
		t.Errorf("Expected transports array not found in output:\n%s", output)
	}
	if !strings.Contains(output, `"websocket",`) {
		t.Errorf("Expected websocket transport not found in output:\n%s", output)
	}
	if !strings.Contains(output, `"polling",`) {
		t.Errorf("Expected polling transport not found in output:\n%s", output)
	}
}

// TestWebSocketTransportsConfiguration tests multiple transport configurations
func TestWebSocketTransportsConfiguration(t *testing.T) {
	input := `package test

@WebSocketGateway({
	port: 8080,
	transports: ["websocket", "polling"]
})
type TransportsGateway struct {}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Test supported transport types
	transports := []string{"websocket", "polling"}
	for _, transport := range transports {
		if !strings.Contains(output, fmt.Sprintf(`"%s",`, transport)) {
			t.Errorf("Expected transport %s not found in output:\n%s", transport, output)
		}
	}
}

func TestWebSocketMessageHandlerGeneration(t *testing.T) {
	input := `package test

@WebSocketGateway(8080)
type ChatGateway struct {}

@SubscribeMessage("message")
func HandleMessage(
    @MessageBody() data *ChatMessage,
    @ConnectedSocket() client *WebSocketClient,
    @MessageAck() ack *AckCallback
) error {
    return nil
}

@SubscribeMessage("join_room")
func HandleJoinRoom(
    @MessageBody() data *RoomData,
    @ConnectedSocket() client *WebSocketClient
) {
    // Handle room join
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Test message handler function generation
	if !strings.Contains(output, "func HandleMessage(data *ChatMessage, client *WebSocketClient, ack *AckCallback) error {") {
		t.Errorf("Expected HandleMessage function not found in output:\n%s", output)
	}

	if !strings.Contains(output, "func HandleJoinRoom(data *RoomData, client *WebSocketClient) {") {
		t.Errorf("Expected HandleJoinRoom function not found in output:\n%s", output)
	}

	// Test message handler registration
	if !strings.Contains(output, `server.OnMessage("message", HandleMessage)`) {
		t.Errorf("Expected message handler registration not found in output:\n%s", output)
	}

	if !strings.Contains(output, `server.OnMessage("join_room", HandleJoinRoom)`) {
		t.Errorf("Expected join_room handler registration not found in output:\n%s", output)
	}

	// Test handler comments
	if !strings.Contains(output, "// Handle WebSocket message: message") {
		t.Errorf("Expected message handler comment not found in output:\n%s", output)
	}

	if !strings.Contains(output, "// Handle WebSocket message: join_room") {
		t.Errorf("Expected join_room handler comment not found in output:\n%s", output)
	}
}

func TestWebSocketLifecycleHandlerGeneration(t *testing.T) {
	input := `package test

@WebSocketGateway(8080)
type Gateway struct {}

@OnGatewayConnection()
func HandleConnection(
    @ConnectedSocket() client *WebSocketClient,
    @Headers() headers map[string]string
) {
    // Handle connection
}

@OnGatewayDisconnect()
func HandleDisconnect(
    @ConnectedSocket() client *WebSocketClient,
    @DisconnectReason() reason string
) {
    // Handle disconnect
}

@OnGatewayInit()
func AfterInit() {
    // Initialize gateway
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Test lifecycle handler function generation
	if !strings.Contains(output, "func HandleConnection(client *WebSocketClient, headers map[string]string) {") {
		t.Errorf("Expected HandleConnection function not found in output:\n%s", output)
	}

	if !strings.Contains(output, "func HandleDisconnect(client *WebSocketClient, reason string) {") {
		t.Errorf("Expected HandleDisconnect function not found in output:\n%s", output)
	}

	if !strings.Contains(output, "func AfterInit() {") {
		t.Errorf("Expected AfterInit function not found in output:\n%s", output)
	}

	// Test lifecycle handler registration
	if !strings.Contains(output, "server.OnConnection(HandleConnection)") {
		t.Errorf("Expected connection handler registration not found in output:\n%s", output)
	}

	if !strings.Contains(output, "server.OnDisconnection(HandleDisconnect)") {
		t.Errorf("Expected disconnect handler registration not found in output:\n%s", output)
	}

	if !strings.Contains(output, "server.OnInit(AfterInit)") {
		t.Errorf("Expected init handler registration not found in output:\n%s", output)
	}

	// Test lifecycle handler comments
	if !strings.Contains(output, "// Handle new WebSocket connection") {
		t.Errorf("Expected connection handler comment not found in output:\n%s", output)
	}

	if !strings.Contains(output, "// Handle WebSocket disconnection") {
		t.Errorf("Expected disconnect handler comment not found in output:\n%s", output)
	}

	if !strings.Contains(output, "// Initialize WebSocket gateway") {
		t.Errorf("Expected init handler comment not found in output:\n%s", output)
	}
}

func TestWebSocketParameterDecorators(t *testing.T) {
	input := `package test

@WebSocketGateway(8080)
type Gateway struct {}

@SubscribeMessage("complex")
func ComplexHandler(
    @MessageBody() body interface{},
    @ConnectedSocket() client *WebSocketClient,
    @MessageAck() ack *AckCallback,
    @Headers() headers map[string]string,
    @Session() session *Session,
    @Rooms() rooms []string,
    @Namespace() ns string,
    @CurrentUser() user *User,
    @ClientIP() ip string,
    @EventName() event string
) {
    // Complex handler
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Test complex handler function generation with all parameter types
	expectedParams := []string{
		"body interface{}",
		"client *WebSocketClient", 
		"ack *AckCallback",
		"headers map[string]string",
		"session *Session",
		"rooms []string",
		"ns string",
		"user *User",
		"ip string",
		"event string",
	}

	for _, param := range expectedParams {
		if !strings.Contains(output, param) {
			t.Errorf("Expected parameter '%s' not found in output:\n%s", param, output)
		}
	}

	// Test complex handler registration
	if !strings.Contains(output, `server.OnMessage("complex", ComplexHandler)`) {
		t.Errorf("Expected complex handler registration not found in output:\n%s", output)
	}
}

func TestWebSocketMultipleGateways(t *testing.T) {
	input := `package test

@WebSocketGateway(8080)
type ChatGateway struct {}

@WebSocketGateway({port: 9000, namespace: "/game"})
type GameGateway struct {}

@SubscribeMessage("message")
func HandleMessage() {}

@SubscribeMessage("game_action")
func HandleGameAction() {}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Test multiple gateway generation
	if !strings.Contains(output, "type ChatGateway struct") {
		t.Errorf("Expected ChatGateway struct not found in output:\n%s", output)
	}

	if !strings.Contains(output, "type GameGateway struct") {
		t.Errorf("Expected GameGateway struct not found in output:\n%s", output)
	}

	// Test multiple setup functions
	if !strings.Contains(output, "func (ws *ChatGateway) SetupChatGateway()") {
		t.Errorf("Expected ChatGateway setup function not found in output:\n%s", output)
	}

	if !strings.Contains(output, "func (ws *GameGateway) SetupGameGateway()") {
		t.Errorf("Expected GameGateway setup function not found in output:\n%s", output)
	}

	// Test different configurations
	if !strings.Contains(output, "Port: 8080,") {
		t.Errorf("Expected port 8080 not found in output:\n%s", output)
	}

	if !strings.Contains(output, "Port:      9000,") {
		t.Errorf("Expected port 9000 not found in output:\n%s", output)
	}

	if !strings.Contains(output, `Namespace: "/game",`) {
		t.Errorf("Expected game namespace not found in output:\n%s", output)
	}

	// Test handlers registered in both gateways
	chatHandlers := strings.Count(output, `server.OnMessage("message", HandleMessage)`)
	gameHandlers := strings.Count(output, `server.OnMessage("game_action", HandleGameAction)`)
	
	if chatHandlers < 2 { // Should be registered in both gateways
		t.Errorf("Expected message handler to be registered in multiple gateways, found %d registrations", chatHandlers)
	}

	if gameHandlers < 2 { // Should be registered in both gateways
		t.Errorf("Expected game_action handler to be registered in multiple gateways, found %d registrations", gameHandlers)
	}
}

func TestWebSocketNoDuplicateRegistration(t *testing.T) {
	input := `package test

@WebSocketGateway(8080)
type Gateway1 struct {}

@WebSocketGateway(9000) 
type Gateway2 struct {}

@SubscribeMessage("test")
func TestHandler() {}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Each gateway should register the handler once in setup and once in register function
	// So we should see exactly 4 registrations total (2 per gateway: setup + register)
	registrations := strings.Count(output, `server.OnMessage("test", TestHandler)`)
	
	if registrations != 4 {
		t.Errorf("Expected exactly 4 handler registrations (2 per gateway: setup + register), found %d registrations", registrations)
	}
}

// Test helper functions
func TestWebSocketHelperFunctions(t *testing.T) {
	generator := NewCodeGenerator("test")
	
	// Test isWebSocketParameterDecorator
	testCases := []struct {
		decorator string
		expected  bool
	}{
		{"MessageBody", true},
		{"ConnectedSocket", true},
		{"MessageAck", true},
		{"Rooms", true},
		{"Body", false}, // HTTP decorator
		{"Query", false}, // HTTP decorator
		{"Invalid", false},
	}

	for _, tc := range testCases {
		result := generator.isWebSocketParameterDecorator(tc.decorator)
		if result != tc.expected {
			t.Errorf("isWebSocketParameterDecorator(%s) = %v, expected %v", tc.decorator, result, tc.expected)
		}
	}
}

// TestWebSocketLifecycleEnhancement tests enhanced WebSocket lifecycle method generation
func TestWebSocketLifecycleEnhancement(t *testing.T) {
	generator := NewCodeGenerator("test")
	
	// Test OnGatewayConnection handler generation
	t.Run("OnGatewayConnection", func(t *testing.T) {
		method := &MethodNode{
			Name:       "HandleConnection",
			ReturnType: "",
			Decorators: []*DecoratorNode{
				{Name: "OnGatewayConnection", Args: []DecoratorArg{}},
			},
			Params: []*ParameterNode{
				{
					Name: "client", 
					Type: "*WebSocketClient",
					Decorators: []*DecoratorNode{{Name: "ConnectedSocket"}},
				},
				{
					Name: "headers", 
					Type: "map[string]string",
					Decorators: []*DecoratorNode{{Name: "Headers"}},
				},
			},
		}
		
		generator.generateOnConnectionHandler(method)
		
		result := generator.generatedCode.String()
		
		// Check for essential connection handler elements
		expectedElements := []string{
			"// WebSocket connection established",
			"client := wsCtx.Client()",
			"headers := wsCtx.Headers()",
			"fmt.Printf(\"Client connected:",
			"// Authentication and authorization",
			"client.Join(\"global\")",
			"client.Broadcast().Emit(\"user_connected\"",
		}
		
		for _, element := range expectedElements {
			if !strings.Contains(result, element) {
				t.Errorf("Expected connection handler to contain: %s", element)
			}
		}
	})
	
	// Test OnGatewayDisconnect handler generation
	t.Run("OnGatewayDisconnect", func(t *testing.T) {
		method := &MethodNode{
			Name:       "HandleDisconnect",
			ReturnType: "",
			Decorators: []*DecoratorNode{
				{Name: "OnGatewayDisconnect", Args: []DecoratorArg{}},
			},
			Params: []*ParameterNode{
				{
					Name: "reason", 
					Type: "string",
					Decorators: []*DecoratorNode{{Name: "DisconnectReason"}},
				},
			},
		}
		
		generator.reset() // Reset generator state
		generator.generateOnDisconnectHandler(method)
		
		result := generator.generatedCode.String()
		
		// Check for essential disconnect handler elements
		expectedElements := []string{
			"// WebSocket connection terminated",
			"client := wsCtx.Client()",
			"reason := wsCtx.DisconnectReason()",
			"fmt.Printf(\"Client disconnected:",
			"client.LeaveAllRooms()",
			"client.Broadcast().Emit(\"user_disconnected\"",
			"// Cleanup resources",
		}
		
		for _, element := range expectedElements {
			if !strings.Contains(result, element) {
				t.Errorf("Expected disconnect handler to contain: %s", element)
			}
		}
	})
	
	// Test OnGatewayInit handler generation
	t.Run("OnGatewayInit", func(t *testing.T) {
		method := &MethodNode{
			Name:       "AfterInit",
			ReturnType: "",
			Decorators: []*DecoratorNode{
				{Name: "OnGatewayInit", Args: []DecoratorArg{}},
			},
			Params: []*ParameterNode{},
		}
		
		generator.reset() // Reset generator state
		generator.generateOnInitHandler(method)
		
		result := generator.generatedCode.String()
		
		// Check for essential init handler elements
		expectedElements := []string{
			"// WebSocket gateway initialization",
			"fmt.Println(\"Initializing WebSocket gateway...\")",
			"// Initialize gateway state",
			"// Connect to external services",
			"// Start background services",
			"// Load configuration",
			"fmt.Println(\"WebSocket gateway initialized successfully\")",
		}
		
		for _, element := range expectedElements {
			if !strings.Contains(result, element) {
				t.Errorf("Expected init handler to contain: %s", element)
			}
		}
	})
}

// TestWebSocketLifecycleParameterExtraction tests parameter extraction for lifecycle handlers
func TestWebSocketLifecycleParameterExtraction(t *testing.T) {
	generator := NewCodeGenerator("test")
	
	method := &MethodNode{
		Name: "TestLifecycleMethod",
		Params: []*ParameterNode{
			{
				Name: "client", 
				Type: "*WebSocketClient",
				Decorators: []*DecoratorNode{{Name: "ConnectedSocket"}},
			},
			{
				Name: "headers", 
				Type: "map[string]string",
				Decorators: []*DecoratorNode{{Name: "Headers"}},
			},
			{
				Name: "userIP", 
				Type: "string",
				Decorators: []*DecoratorNode{{Name: "ClientIP"}},
			},
			{
				Name: "user", 
				Type: "*User",
				Decorators: []*DecoratorNode{{Name: "CurrentUser"}},
			},
		},
	}
	
	generator.generateWebSocketLifecycleParameterExtraction(method)
	result := generator.generatedCode.String()
	
	// Check for parameter extraction
	expectedExtractions := []string{
		"client := wsCtx.Client()",
		"headers := wsCtx.Headers()",
		"userIP := wsCtx.ClientIP()",
		"user := wsCtx.User()",
	}
	
	for _, extraction := range expectedExtractions {
		if !strings.Contains(result, extraction) {
			t.Errorf("Expected parameter extraction to contain: %s", extraction)
		}
	}
}

// TestHasWebSocketParameterDecorators tests detection of WebSocket parameter decorators
func TestHasWebSocketParameterDecorators(t *testing.T) {
	generator := NewCodeGenerator("test")
	
	// Test method with WebSocket parameter decorators
	methodWithWS := &MethodNode{
		Params: []*ParameterNode{
			{
				Name: "client", 
				Type: "*WebSocketClient",
				Decorators: []*DecoratorNode{{Name: "ConnectedSocket"}},
			},
		},
	}
	
	if !generator.hasWebSocketParameterDecorators(methodWithWS) {
		t.Error("Expected method with WebSocket decorators to return true")
	}
	
	// Test method without WebSocket parameter decorators
	methodWithoutWS := &MethodNode{
		Params: []*ParameterNode{
			{
				Name: "data", 
				Type: "string",
				Decorators: []*DecoratorNode{{Name: "Body"}}, // HTTP decorator
			},
		},
	}
	
	if generator.hasWebSocketParameterDecorators(methodWithoutWS) {
		t.Error("Expected method without WebSocket decorators to return false")
	}
}

// TestWebSocketLifecycleIntegration tests full WebSocket lifecycle handler generation
func TestWebSocketLifecycleIntegration(t *testing.T) {
	generator := NewCodeGenerator("test")
	
	// Test complete lifecycle handler generation
	gateway := &WebSocketGatewayDeclaration{
		Name: "ChatGateway",
		Fields: []*FieldNode{},
		Methods: []*MethodNode{
			{
				Name:       "HandleConnection",
				ReturnType: "",
				Decorators: []*DecoratorNode{
					{Name: "OnGatewayConnection", Args: []DecoratorArg{}},
				},
				Params: []*ParameterNode{
					{
						Name: "client", 
						Type: "*WebSocketClient",
						Decorators: []*DecoratorNode{{Name: "ConnectedSocket"}},
					},
				},
			},
		},
		Decorators: []*DecoratorNode{
			{Name: "WebSocketGateway", Args: []DecoratorArg{
				{Value: 8080},
			}},
		},
	}
	
	method := gateway.Methods[0]
	err := generator.generateWebSocketLifecycleHandler(gateway, method)
	if err != nil {
		t.Errorf("Failed to generate lifecycle handler: %v", err)
	}
	
	result := generator.generatedCode.String()
	
	// Check for complete lifecycle handler structure
	expectedElements := []string{
		"func (ws *ChatGateway) HandleConnection(wsCtx *websocket.LifecycleContext)",
		"// Extract parameters from WebSocket lifecycle context",
		"client := wsCtx.Client()",
		"// WebSocket connection established",
		"client.Join(\"global\")",
		"return",
	}
	
	for _, element := range expectedElements {
		if !strings.Contains(result, element) {
			t.Errorf("Expected complete lifecycle handler to contain: %s\nActual output: %s", element, result)
		}
	}
}

// ===================== WebSocket Middleware Integration Tests =====================

// TestWebSocketMiddlewareIntegration tests WebSocket middleware generation
func TestWebSocketMiddlewareIntegration(t *testing.T) {
	input := `package test

@WebSocketGateway(8080)
@UseGuards("WSAuthGuard", "WSRoleGuard")
@UseInterceptors("WSLoggingInterceptor")
@UseFilters("WSGlobalExceptionFilter")
type SecureGateway struct {
    @Inject("authService")
    authService *AuthService
}

@SubscribeMessage("secure_message")
@UseGuards("WSPermissionGuard")
@UseInterceptors("WSValidationInterceptor")
@UsePipes("WSValidationPipe")
func HandleSecureMessage(
    @MessageBody() data *SecureMessage,
    @ConnectedSocket() client *WebSocketClient
) error {
    return nil
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Test middleware registration in setup function
	expectedMiddlewareRegistrations := []string{
		"server.UseGuard(\"WSAuthGuard\", ws.WSAuthGuard)",
		"server.UseGuard(\"WSRoleGuard\", ws.WSRoleGuard)",
		"server.UseGuard(\"WSPermissionGuard\", ws.WSPermissionGuard)",
		"server.UseInterceptor(\"WSLoggingInterceptor\", ws.WSLoggingInterceptor)",
		"server.UseInterceptor(\"WSValidationInterceptor\", ws.WSValidationInterceptor)",
		"server.UsePipe(\"WSValidationPipe\", ws.WSValidationPipe)",
		"server.UseFilter(\"WSGlobalExceptionFilter\", ws.WSGlobalExceptionFilter)",
	}

	for _, registration := range expectedMiddlewareRegistrations {
		if !strings.Contains(output, registration) {
			t.Errorf("Expected middleware registration not found: %s", registration)
		}
	}

	// Test guard methods generation
	expectedGuardMethods := []string{
		"func (ws *SecureGateway) WSAuthGuard(wsCtx *websocket.GuardContext) bool {",
		"func (ws *SecureGateway) WSRoleGuard(wsCtx *websocket.GuardContext) bool {",
		"func (ws *SecureGateway) WSPermissionGuard(wsCtx *websocket.GuardContext) bool {",
	}

	for _, method := range expectedGuardMethods {
		if !strings.Contains(output, method) {
			t.Errorf("Expected guard method not found: %s", method)
		}
	}

	// Test interceptor methods generation
	expectedInterceptorMethods := []string{
		"func (ws *SecureGateway) WSLoggingInterceptor(wsCtx *websocket.InterceptorContext) {",
		"func (ws *SecureGateway) WSValidationInterceptor(wsCtx *websocket.InterceptorContext) {",
	}

	for _, method := range expectedInterceptorMethods {
		if !strings.Contains(output, method) {
			t.Errorf("Expected interceptor method not found: %s", method)
		}
	}

	// Test pipe methods generation
	if !strings.Contains(output, "func (ws *SecureGateway) WSValidationPipe(wsCtx *websocket.PipeContext) interface{} {") {
		t.Errorf("Expected pipe method not found")
	}

	// Test filter methods generation
	if !strings.Contains(output, "func (ws *SecureGateway) WSGlobalExceptionFilter(wsCtx *websocket.FilterContext, err error) {") {
		t.Errorf("Expected filter method not found")
	}
}

// TestWebSocketMiddlewareDecoratorExtraction tests middleware decorator extraction
func TestWebSocketMiddlewareDecoratorExtraction(t *testing.T) {
	generator := NewCodeGenerator("test")
	
	// Test guard decorators extraction
	guardDecorators := []*DecoratorNode{
		{Name: "UseGuards", Args: []DecoratorArg{{Value: "WSAuthGuard"}, {Value: "WSRoleGuard"}}},
		{Name: "SubscribeMessage", Args: []DecoratorArg{{Value: "test_message"}}},
	}
	
	guards := generator.getWebSocketGuardDecorators(guardDecorators)
	expectedGuards := []string{"WSAuthGuard", "WSRoleGuard"}
	
	if len(guards) != len(expectedGuards) {
		t.Errorf("Expected %d guards, got %d", len(expectedGuards), len(guards))
	}
	
	for i, expectedGuard := range expectedGuards {
		if guards[i] != expectedGuard {
			t.Errorf("Expected guard '%s', got '%s'", expectedGuard, guards[i])
		}
	}
	
	// Test interceptor decorators extraction
	interceptorDecorators := []*DecoratorNode{
		{Name: "UseInterceptors", Args: []DecoratorArg{{Value: "WSLoggingInterceptor"}, {Value: "WSValidationInterceptor"}}},
	}
	
	interceptors := generator.getWebSocketInterceptorDecorators(interceptorDecorators)
	expectedInterceptors := []string{"WSLoggingInterceptor", "WSValidationInterceptor"}
	
	if len(interceptors) != len(expectedInterceptors) {
		t.Errorf("Expected %d interceptors, got %d", len(expectedInterceptors), len(interceptors))
	}
	
	for i, expectedInterceptor := range expectedInterceptors {
		if interceptors[i] != expectedInterceptor {
			t.Errorf("Expected interceptor '%s', got '%s'", expectedInterceptor, interceptors[i])
		}
	}
	
	// Test pipe decorators extraction
	pipeDecorators := []*DecoratorNode{
		{Name: "UsePipes", Args: []DecoratorArg{{Value: "WSValidationPipe"}, {Value: "WSParseIntPipe"}}},
	}
	
	pipes := generator.getWebSocketPipeDecorators(pipeDecorators)
	expectedPipes := []string{"WSValidationPipe", "WSParseIntPipe"}
	
	if len(pipes) != len(expectedPipes) {
		t.Errorf("Expected %d pipes, got %d", len(expectedPipes), len(pipes))
	}
	
	for i, expectedPipe := range expectedPipes {
		if pipes[i] != expectedPipe {
			t.Errorf("Expected pipe '%s', got '%s'", expectedPipe, pipes[i])
		}
	}
	
	// Test filter decorators extraction
	filterDecorators := []*DecoratorNode{
		{Name: "UseFilters", Args: []DecoratorArg{{Value: "WSGlobalExceptionFilter"}, {Value: "WSAuthExceptionFilter"}}},
	}
	
	filters := generator.getWebSocketFilterDecorators(filterDecorators)
	expectedFilters := []string{"WSGlobalExceptionFilter", "WSAuthExceptionFilter"}
	
	if len(filters) != len(expectedFilters) {
		t.Errorf("Expected %d filters, got %d", len(expectedFilters), len(filters))
	}
	
	for i, expectedFilter := range expectedFilters {
		if filters[i] != expectedFilter {
			t.Errorf("Expected filter '%s', got '%s'", expectedFilter, filters[i])
		}
	}
}