package codegen

import (
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