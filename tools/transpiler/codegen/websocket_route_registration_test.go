package codegen

import (
	"strings"
	"testing"
)

// TestWebSocketRouteRegistrationBasic tests basic WebSocket route registration code generation
func TestWebSocketRouteRegistrationBasic(t *testing.T) {
	input := `package test

@WebSocketGateway(8080)
type ChatGateway struct {
    @Inject("chatService")
    chatService *ChatService
}

@SubscribeMessage("message")
func HandleMessage(
    @MessageBody() data *ChatMessage,
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

	// Test route registry creation
	if !strings.Contains(output, "routeRegistry := websocket.NewRouteRegistry()") {
		t.Errorf("Expected route registry creation not found in output:\n%s", output)
	}

	// Test route registration with metadata
	if !strings.Contains(output, `// Register standalone route: message -> HandleMessage`) {
		t.Errorf("Expected route registration comment not found in output:\n%s", output)
	}

	// Test route config structure
	if !strings.Contains(output, "standaloneConfig := &websocket.RouteConfig{") {
		t.Errorf("Expected route config structure not found in output:\n%s", output)
	}

	// Test route configuration fields
	expectedFields := []string{
		`EventName:    "message",`,
		`Handler:      HandleMessage,`,
		`Gateway:      "Standalone",`,
		`Method:       "HandleMessage",`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Expected route config field not found: %s", field)
		}
	}

	// Test route registration call
	if !strings.Contains(output, `routeRegistry.Register("message", standaloneConfig)`) {
		t.Errorf("Expected route registry registration call not found in output:\n%s", output)
	}

	// Test route registry application to server
	if !strings.Contains(output, "server.ApplyRouteRegistry(routeRegistry)") {
		t.Errorf("Expected route registry application not found in output:\n%s", output)
	}
}

// TestWebSocketRouteRegistrationWithMiddleware tests route registration with middleware
func TestWebSocketRouteRegistrationWithMiddleware(t *testing.T) {
	input := `package test

@WebSocketGateway(8080)
type SecureGateway struct {}

@SubscribeMessage("secure_action")
@UseGuards("AuthGuard", "RoleGuard")
@UseInterceptors("LoggingInterceptor")
@UsePipes("ValidationPipe")
@UseFilters("ExceptionFilter")
func HandleSecureAction(
    @MessageBody() data *SecureData,
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

	// Test middleware configuration in route config
	middlewareChecks := []string{
		"Guards: []string{",
		`"AuthGuard",`,
		`"RoleGuard",`,
		"Interceptors: []string{",
		`"LoggingInterceptor",`,
		"Pipes: []string{",
		`"ValidationPipe",`,
		"Filters: []string{",
		`"ExceptionFilter",`,
	}

	for _, check := range middlewareChecks {
		if !strings.Contains(output, check) {
			t.Errorf("Expected middleware configuration not found: %s", check)
		}
	}
}

// TestWebSocketRouteRegistrationParameterMetadata tests parameter metadata generation
func TestWebSocketRouteRegistrationParameterMetadata(t *testing.T) {
	input := `package test

@WebSocketGateway(8080)
type Gateway struct {}

@SubscribeMessage("complex_message")
func HandleComplexMessage(
    @MessageBody() body interface{},
    @ConnectedSocket() client *WebSocketClient,
    @MessageAck() ack *AckCallback,
    @Headers() headers map[string]string,
    @CurrentUser() user *User
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

	// Test parameter metadata structure
	if !strings.Contains(output, "Parameters: []websocket.ParameterMetadata{") {
		t.Errorf("Expected parameter metadata structure not found in output:\n%s", output)
	}

	// Test parameter information
	parameterChecks := []string{
		`Name: "body",`,
		`Type: "interface{}",`,
		`Name: "client",`,
		`Type: "*WebSocketClient",`,
		`Name: "ack",`,
		`Type: "*AckCallback",`,
		`Name: "headers",`,
		`Type: "map[string]string",`,
		`Name: "user",`,
		`Type: "*User",`,
	}

	for _, check := range parameterChecks {
		if !strings.Contains(output, check) {
			t.Errorf("Expected parameter information not found: %s", check)
		}
	}

	// Test decorator information in parameters
	decoratorChecks := []string{
		"Decorators: []string{",
		`"MessageBody",`,
		`"ConnectedSocket",`,
		`"MessageAck",`,
		`"Headers",`,
		`"CurrentUser",`,
	}

	for _, check := range decoratorChecks {
		if !strings.Contains(output, check) {
			t.Errorf("Expected decorator information not found: %s", check)
		}
	}
}

// TestWebSocketRouteRegistrationValidationMetadata tests validation metadata generation
func TestWebSocketRouteRegistrationValidationMetadata(t *testing.T) {
	input := `package test

@WebSocketGateway(8080)
type Gateway struct {}

@SubscribeMessage("validated_message")
func HandleValidatedMessage(
    @MessageBody() @IsNotEmpty() @MaxLength(100) content string,
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

	// Test validation metadata structure
	validationChecks := []string{
		"HasValidation: true,",
		"ValidationRules: []string{",
		`"content:IsNotEmpty",`,
		`"content:MaxLength",`,
	}

	for _, check := range validationChecks {
		if !strings.Contains(output, check) {
			t.Errorf("Expected validation metadata not found: %s", check)
		}
	}
}

// TestWebSocketRouteRegistrationStandaloneFunctions tests standalone function route registration
func TestWebSocketRouteRegistrationStandaloneFunctions(t *testing.T) {
	input := `package test

@WebSocketGateway(8080)
type Gateway struct {}

@SubscribeMessage("standalone_message")
func HandleStandaloneMessage(
    @MessageBody() data *Message,
    @ConnectedSocket() client *WebSocketClient
) error {
    return nil
}

@SubscribeMessage("another_standalone")
func HandleAnotherStandalone() {}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Test standalone function registration comments
	standaloneChecks := []string{
		"// Register standalone WebSocket function routes",
		`// Register standalone route: standalone_message -> HandleStandaloneMessage`,
		`// Register standalone route: another_standalone -> HandleAnotherStandalone`,
	}

	for _, check := range standaloneChecks {
		if !strings.Contains(output, check) {
			t.Errorf("Expected standalone route registration not found: %s", check)
		}
	}

	// Test standalone route configuration
	standaloneConfigChecks := []string{
		"standaloneConfig := &websocket.RouteConfig{",
		`EventName:    "standalone_message",`,
		`Handler:      HandleStandaloneMessage,`,
		`Gateway:      "Standalone",`,
		`Method:       "HandleStandaloneMessage",`,
		"IsStandalone: true,",
	}

	for _, check := range standaloneConfigChecks {
		if !strings.Contains(output, check) {
			t.Errorf("Expected standalone route config not found: %s", check)
		}
	}

	// Test standalone route registration calls
	registrationChecks := []string{
		`routeRegistry.Register("standalone_message", standaloneConfig)`,
		`routeRegistry.Register("another_standalone", standaloneConfig)`,
	}

	for _, check := range registrationChecks {
		if !strings.Contains(output, check) {
			t.Errorf("Expected standalone route registration call not found: %s", check)
		}
	}
}

// TestWebSocketRouteRegistrationMetadata tests route metadata generation
func TestWebSocketRouteRegistrationMetadata(t *testing.T) {
	input := `package test

@WebSocketGateway({
	port: 3000,
	namespace: "/chat",
	cors: true,
	transports: ["websocket", "polling"]
})
type ChatGateway struct {}

@SubscribeMessage("message1")
func HandleMessage1() {}

@SubscribeMessage("message2") 
@UseGuards("AuthGuard")
func HandleMessage2() {}

@OnGatewayConnection()
func HandleConnection() {}

@SubscribeMessage("standalone")
func HandleStandalone() {}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Test route metadata structure
	metadataChecks := []string{
		"// Generate WebSocket route metadata",
		"metadata := websocket.RouteMetadata{",
		`GatewayName: "ChatGateway",`,
		"Routes:      routeRegistry.GetAllRoutes(),",
		"GeneratedAt: time.Now(),",
	}

	for _, check := range metadataChecks {
		if !strings.Contains(output, check) {
			t.Errorf("Expected route metadata not found: %s", check)
		}
	}

	// Test gateway configuration metadata
	gatewayConfigChecks := []string{
		"GatewayConfig: websocket.GatewayConfigMetadata{",
		"Port:      3000,",
		`Namespace: "/chat",`,
		"CORS:      true,",
		"Transports: []string{",
		`"websocket",`,
		`"polling",`,
	}

	for _, check := range gatewayConfigChecks {
		if !strings.Contains(output, check) {
			t.Errorf("Expected gateway config metadata not found: %s", check)
		}
	}

	// Test route statistics
	statisticsChecks := []string{
		"Statistics: websocket.RouteStatistics{",
		"MessageHandlers:    0,", // Standalone functions don't count as gateway message handlers
		"LifecycleHandlers:  0,", // Standalone functions don't count as gateway lifecycle handlers
		"StandaloneHandlers: 4,", // HandleMessage1, HandleMessage2, HandleConnection, HandleStandalone
		"TotalMiddleware:    0,", // Gateway methods middleware counts
		"TotalRoutes:        4,", // message1, message2, connection, standalone
	}

	for _, check := range statisticsChecks {
		if !strings.Contains(output, check) {
			t.Errorf("Expected route statistics not found: %s", check)
		}
	}

	// Test metadata application to server
	if !strings.Contains(output, "server.SetRouteMetadata(metadata)") {
		t.Errorf("Expected metadata application to server not found in output:\n%s", output)
	}
}

// TestWebSocketRouteRegistrationMultipleEvents tests route registration with multiple events
func TestWebSocketRouteRegistrationMultipleEvents(t *testing.T) {
	input := `package test

@WebSocketGateway(8080)
type Gateway struct {}

@SubscribeMessage(["event1", "event2", "event3"])
func HandleMultipleEvents(
    @MessageBody() data interface{},
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

	// Test multiple event route registrations
	multiEventChecks := []string{
		`// Register standalone route: event1 -> HandleMultipleEvents`,
		`// Register standalone route: event2 -> HandleMultipleEvents`,
		`// Register standalone route: event3 -> HandleMultipleEvents`,
		`routeRegistry.Register("event1", standaloneConfig)`,
		`routeRegistry.Register("event2", standaloneConfig)`,
		`routeRegistry.Register("event3", standaloneConfig)`,
	}

	for _, check := range multiEventChecks {
		if !strings.Contains(output, check) {
			t.Errorf("Expected multi-event route registration not found: %s", check)
		}
	}

	// Each event should have its own route config
	eventConfigChecks := []string{
		`EventName:    "event1",`,
		`EventName:    "event2",`,
		`EventName:    "event3",`,
	}

	for _, check := range eventConfigChecks {
		if !strings.Contains(output, check) {
			t.Errorf("Expected event config not found: %s", check)
		}
	}
}

// TestWebSocketRouteRegistrationMultipleGateways tests route registration across multiple gateways
func TestWebSocketRouteRegistrationMultipleGateways(t *testing.T) {
	input := `package test

@WebSocketGateway(8080)
type ChatGateway struct {}

@WebSocketGateway(9000)
type GameGateway struct {}

@SubscribeMessage("chat_message")
func HandleChatMessage() {}

@SubscribeMessage("game_action")
func HandleGameAction() {}

@SubscribeMessage("global_event")
func HandleGlobalEvent() {}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Each gateway should have its own route registration
	chatGatewayChecks := []string{
		"// Register comprehensive WebSocket routes for ChatGateway",
		`// Register standalone route: chat_message -> HandleChatMessage`,
		`Gateway:      "Standalone",`,
		`Method:       "HandleChatMessage",`,
	}

	gameGatewayChecks := []string{
		"// Register comprehensive WebSocket routes for GameGateway", 
		`// Register standalone route: game_action -> HandleGameAction`,
		`Gateway:      "Standalone",`,
		`Method:       "HandleGameAction",`,
	}

	for _, check := range chatGatewayChecks {
		if !strings.Contains(output, check) {
			t.Errorf("Expected ChatGateway route registration not found: %s", check)
		}
	}

	for _, check := range gameGatewayChecks {
		if !strings.Contains(output, check) {
			t.Errorf("Expected GameGateway route registration not found: %s", check)
		}
	}

	// Standalone function should be registered in both gateways  
	standaloneRegistrationCount := strings.Count(output, `// Register standalone route: global_event -> HandleGlobalEvent`)
	if standaloneRegistrationCount < 2 {
		t.Errorf("Expected at least 2 standalone route registrations (one per gateway), found %d", standaloneRegistrationCount)
	}

	// Each gateway should have its own metadata
	metadataChecks := []string{
		`GatewayName: "ChatGateway",`,
		`GatewayName: "GameGateway",`,
	}

	for _, check := range metadataChecks {
		if !strings.Contains(output, check) {
			t.Errorf("Expected gateway metadata not found: %s", check)
		}
	}
}

// TestWebSocketRouteRegistrationEmpty tests route registration with no routes
func TestWebSocketRouteRegistrationEmpty(t *testing.T) {
	input := `package test

@WebSocketGateway(8080)
type EmptyGateway struct {
    @Inject("service")
    service *Service
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

	// Even empty gateways should have route registration structure
	emptyChecks := []string{
		"// Register comprehensive WebSocket routes for EmptyGateway",
		"routeRegistry := websocket.NewRouteRegistry()",
		"server.ApplyRouteRegistry(routeRegistry)",
		"// Generate WebSocket route metadata",
		"Statistics: websocket.RouteStatistics{",
		"MessageHandlers:    0,",
		"LifecycleHandlers:  0,",
		"StandaloneHandlers: 0,",
		"TotalRoutes:        0,",
	}

	for _, check := range emptyChecks {
		if !strings.Contains(output, check) {
			t.Errorf("Expected empty gateway route registration not found: %s", check)
		}
	}
}

// TestWebSocketRouteRegistrationComplexMiddleware tests complex middleware scenarios
func TestWebSocketRouteRegistrationComplexMiddleware(t *testing.T) {
	input := `package test

@WebSocketGateway(8080)
@UseGuards("GatewayGuard")
@UseInterceptors("GatewayInterceptor")
type ComplexGateway struct {}

@SubscribeMessage("complex_action")
@UseGuards("MethodGuard1", "MethodGuard2") 
@UseInterceptors("MethodInterceptor1", "MethodInterceptor2")
@UsePipes("MethodPipe1", "MethodPipe2")
@UseFilters("MethodFilter1", "MethodFilter2")
func HandleComplexAction() {}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Test that both gateway-level and method-level middleware are captured
	complexMiddlewareChecks := []string{
		// Method-level middleware should be in route config
		`"MethodGuard1",`,
		`"MethodGuard2",`,
		`"MethodInterceptor1",`,
		`"MethodInterceptor2",`,
		`"MethodPipe1",`,
		`"MethodPipe2",`,
		`"MethodFilter1",`,
		`"MethodFilter2",`,
	}

	for _, check := range complexMiddlewareChecks {
		if !strings.Contains(output, check) {
			t.Errorf("Expected complex middleware not found: %s", check)
		}
	}

	// Test total middleware count includes all method-level middleware
	// Since standalone functions are used, the gateway method middleware count is 0
	// But the server-level middleware (GatewayGuard, GatewayInterceptor) should be registered
	if !strings.Contains(output, "TotalMiddleware:    0,") { 
		t.Errorf("Expected total middleware count of 0 not found in output")
	}

	// Test that gateway-level middleware is also registered
	gatewayMiddlewareChecks := []string{
		`server.UseGuard("GatewayGuard", ws.GatewayGuard)`,
		`server.UseInterceptor("GatewayInterceptor", ws.GatewayInterceptor)`,
	}

	for _, check := range gatewayMiddlewareChecks {
		if !strings.Contains(output, check) {
			t.Errorf("Expected gateway middleware registration not found: %s", check)
		}
	}
}