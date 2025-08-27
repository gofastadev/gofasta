package core

import (
	"go/token"
	"testing"
)

// TestWebSocketGatewayDeclaration tests the WebSocketGatewayDeclaration AST node
func TestWebSocketGatewayDeclaration(t *testing.T) {
	gateway := &WebSocketGatewayDeclaration{
		Name: "ChatGateway",
		Port: intPtr(8080),
		Namespace: stringPtr("/chat"),
		Config: map[string]interface{}{
			"cors": true,
			"transports": []string{"websocket", "polling"},
		},
		Position: token.Pos(1),
	}

	// Test String method
	if gateway.String() != "WebSocketGateway: ChatGateway" {
		t.Errorf("Expected 'WebSocketGateway: ChatGateway', got '%s'", gateway.String())
	}

	// Test Pos method
	if gateway.Pos() != token.Pos(1) {
		t.Errorf("Expected position 1, got %v", gateway.Pos())
	}

	// Test isDeclaration method (should not panic)
	gateway.isDeclaration()
}

// TestWebSocketDecoratorTypes tests the WebSocket decorator type constants
func TestWebSocketDecoratorTypes(t *testing.T) {
	testCases := []struct {
		name         string
		decoratorType DecoratorType
		expected     bool
	}{
		{"WebSocketGateway", WebSocketGatewayDecorator, true},
		{"SubscribeMessage", SubscribeMessageDecorator, true},
		{"OnGatewayConnection", OnGatewayConnectionDecorator, true},
		{"MessageBody", MessageBodyDecorator, true},
		{"ConnectedSocket", ConnectedSocketDecorator, true},
		{"WebSocketClient", WebSocketClientDecorator, true},
		{"Controller", ControllerDecorator, false}, // Non-WebSocket decorator
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsWebSocketDecorator(tc.decoratorType)
			if result != tc.expected {
				t.Errorf("IsWebSocketDecorator(%s) = %v, expected %v", tc.name, result, tc.expected)
			}
		})
	}
}

// TestWebSocketDecoratorTypeMap tests that WebSocket decorators are correctly mapped
func TestWebSocketDecoratorTypeMap(t *testing.T) {
	testCases := []struct {
		name         string
		expectedType DecoratorType
	}{
		{"WebSocketGateway", WebSocketGatewayDecorator},
		{"SubscribeMessage", SubscribeMessageDecorator},
		{"OnMessage", OnMessageDecorator},
		{"MessagePattern", MessagePatternDecorator},
		{"OnGatewayConnection", OnGatewayConnectionDecorator},
		{"OnGatewayDisconnect", OnGatewayDisconnectDecorator},
		{"OnGatewayInit", OnGatewayInitDecorator},
		{"MessageBody", MessageBodyDecorator},
		{"ConnectedSocket", ConnectedSocketDecorator},
		{"MessageAck", MessageAckDecorator},
		{"Rooms", RoomsDecorator},
		{"Namespace", NamespaceDecorator},
		{"CurrentUser", CurrentUserDecorator},
		{"ClientIP", ClientIPDecorator},
		{"DisconnectReason", DisconnectReasonDecorator},
		{"EventName", EventNameDecorator},
		{"RawMessage", RawMessageDecorator},
		{"Server", ServerDecorator},
		{"WebSocketClient", WebSocketClientDecorator},
		{"WebSocketTestClient", WebSocketTestClientDecorator},
		{"WebSocketIntegrationTest", WebSocketIntegrationTestDecorator},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualType := GetDecoratorType(tc.name)
			if actualType != tc.expectedType {
				t.Errorf("GetDecoratorType(%s) = %v, expected %v", tc.name, actualType, tc.expectedType)
			}
		})
	}
}

// TestWebSocketGatewayDecoratorHelper tests IsWebSocketGatewayDecorator helper function
func TestWebSocketGatewayDecoratorHelper(t *testing.T) {
	testCases := []struct {
		decoratorType DecoratorType
		expected     bool
	}{
		{WebSocketGatewayDecorator, true},
		{SubscribeMessageDecorator, false},
		{ControllerDecorator, false},
		{GetDecorator, false},
	}

	for _, tc := range testCases {
		result := IsWebSocketGatewayDecorator(tc.decoratorType)
		if result != tc.expected {
			t.Errorf("IsWebSocketGatewayDecorator(%v) = %v, expected %v", tc.decoratorType, result, tc.expected)
		}
	}
}

// TestWebSocketMessageDecoratorHelper tests IsWebSocketMessageDecorator helper function
func TestWebSocketMessageDecoratorHelper(t *testing.T) {
	testCases := []struct {
		decoratorType DecoratorType
		expected     bool
	}{
		{SubscribeMessageDecorator, true},
		{OnMessageDecorator, true},
		{MessagePatternDecorator, true},
		{OnGatewayConnectionDecorator, false},
		{MessageBodyDecorator, false},
		{WebSocketGatewayDecorator, false},
	}

	for _, tc := range testCases {
		result := IsWebSocketMessageDecorator(tc.decoratorType)
		if result != tc.expected {
			t.Errorf("IsWebSocketMessageDecorator(%v) = %v, expected %v", tc.decoratorType, result, tc.expected)
		}
	}
}

// TestWebSocketLifecycleDecoratorHelper tests IsWebSocketLifecycleDecorator helper function
func TestWebSocketLifecycleDecoratorHelper(t *testing.T) {
	testCases := []struct {
		decoratorType DecoratorType
		expected     bool
	}{
		{OnGatewayConnectionDecorator, true},
		{OnGatewayDisconnectDecorator, true},
		{OnGatewayInitDecorator, true},
		{SubscribeMessageDecorator, false},
		{MessageBodyDecorator, false},
	}

	for _, tc := range testCases {
		result := IsWebSocketLifecycleDecorator(tc.decoratorType)
		if result != tc.expected {
			t.Errorf("IsWebSocketLifecycleDecorator(%v) = %v, expected %v", tc.decoratorType, result, tc.expected)
		}
	}
}

// TestWebSocketParameterDecoratorHelper tests IsWebSocketParameterDecorator helper function
func TestWebSocketParameterDecoratorHelper(t *testing.T) {
	testCases := []struct {
		decoratorType DecoratorType
		expected     bool
	}{
		{MessageBodyDecorator, true},
		{ConnectedSocketDecorator, true},
		{MessageAckDecorator, true},
		{RoomsDecorator, true},
		{NamespaceDecorator, true},
		{CurrentUserDecorator, true},
		{ClientIPDecorator, true},
		{DisconnectReasonDecorator, true},
		{EventNameDecorator, true},
		{RawMessageDecorator, true},
		{ServerDecorator, true},
		{OnGatewayConnectionDecorator, false},
		{SubscribeMessageDecorator, false},
		{WebSocketClientDecorator, false},
	}

	for _, tc := range testCases {
		result := IsWebSocketParameterDecorator(tc.decoratorType)
		if result != tc.expected {
			t.Errorf("IsWebSocketParameterDecorator(%v) = %v, expected %v", tc.decoratorType, result, tc.expected)
		}
	}
}

// TestWebSocketClientDecoratorHelper tests IsWebSocketClientDecorator helper function
func TestWebSocketClientDecoratorHelper(t *testing.T) {
	testCases := []struct {
		decoratorType DecoratorType
		expected     bool
	}{
		{WebSocketClientDecorator, true},
		{WebSocketTestClientDecorator, true},
		{WebSocketIntegrationTestDecorator, true},
		{WebSocketGatewayDecorator, false},
		{MessageBodyDecorator, false},
	}

	for _, tc := range testCases {
		result := IsWebSocketClientDecorator(tc.decoratorType)
		if result != tc.expected {
			t.Errorf("IsWebSocketClientDecorator(%v) = %v, expected %v", tc.decoratorType, result, tc.expected)
		}
	}
}

// TestWebSocketGatewayASTWalking tests that WebSocket gateway declarations are properly walked
func TestWebSocketGatewayASTWalking(t *testing.T) {
	// Create a test WebSocket gateway with decorators and methods
	gateway := &WebSocketGatewayDeclaration{
		Name: "TestGateway",
		Decorators: []*DecoratorNode{
			{
				Name:     "WebSocketGateway",
				Position: token.Pos(1),
			},
		},
		Fields: []*FieldNode{
			{
				Name:     "service",
				Type:     "*ChatService",
				Position: token.Pos(2),
			},
		},
		Methods: []*MethodNode{
			{
				Name:     "HandleMessage",
				Position: token.Pos(3),
			},
		},
		Position: token.Pos(0),
	}

	// Create a visitor to count nodes
	visitor := &nodeCounter{}
	Walk(visitor, gateway)

	// Should have visited: gateway + decorator + field + method = 4 nodes
	expectedCount := 4
	if visitor.count != expectedCount {
		t.Errorf("Expected to visit %d nodes, but visited %d", expectedCount, visitor.count)
	}
}

// Helper type for testing AST walking
type nodeCounter struct {
	count int
}

func (nc *nodeCounter) Visit(node GofaASTNode) Visitor {
	nc.count++
	return nc
}

// Helper functions for creating pointers
func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}