// WebSocket validation tests for GoFasta transpiler
package parsing

import (
	"testing"
)

// === @ConnectedSocket() Parameter Decorator Tests ===

func TestConnectedSocketParameterValidation(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		expectError bool
		description string
	}{
		{
			name: "Valid @ConnectedSocket() with *WebSocketClient type",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@ConnectedSocket() client *WebSocketClient) {}
			`,
			expectError: false,
			description: "Should accept @ConnectedSocket() with *WebSocketClient type",
		},
		{
			name: "Invalid @ConnectedSocket() with wrong type - WebSocketClient",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")  
func HandleTest(@ConnectedSocket() client WebSocketClient) {}
			`,
			expectError: true,
			description: "Should reject @ConnectedSocket() with WebSocketClient type (must be *WebSocketClient)",
		},
		{
			name: "Invalid @ConnectedSocket() with wrong type - string",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@ConnectedSocket() client string) {}
			`,
			expectError: true,
			description: "Should reject @ConnectedSocket() with string type (must be *WebSocketClient)",
		},
		{
			name: "Invalid @ConnectedSocket() with wrong type - interface{}",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@ConnectedSocket() client interface{}) {}
			`,
			expectError: true,
			description: "Should reject @ConnectedSocket() with interface{} type (must be *WebSocketClient)",
		},
		{
			name: "Valid @ConnectedSocket() combined with @MessageBody()",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@MessageBody() data string, @ConnectedSocket() client *WebSocketClient) {}
			`,
			expectError: false,
			description: "Should accept @ConnectedSocket() combined with @MessageBody()",
		},
		{
			name: "Valid @ConnectedSocket() with multiple decorators",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(
	@MessageBody() data *TestData,
	@ConnectedSocket() client *WebSocketClient,
	@Headers() headers map[string]string
) {}

type TestData struct {
	Message string
}
			`,
			expectError: false,
			description: "Should accept @ConnectedSocket() with multiple other decorators",
		},
		{
			name: "Multiple @ConnectedSocket() parameters",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@ConnectedSocket() client1 *WebSocketClient, @ConnectedSocket() client2 *WebSocketClient) {}
			`,
			expectError: false,
			description: "Should accept multiple @ConnectedSocket() parameters",
		},
		{
			name: "Invalid @ConnectedSocket() with custom struct type",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@ConnectedSocket() client *MyClient) {}

type MyClient struct {
	ID string
}
			`,
			expectError: true,
			description: "Should reject @ConnectedSocket() with custom struct type (must be *WebSocketClient)",
		},
		{
			name: "Invalid @ConnectedSocket() with **WebSocketClient type",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@ConnectedSocket() client **WebSocketClient) {}
			`,
			expectError: true,
			description: "Should reject @ConnectedSocket() with **WebSocketClient type (must be *WebSocketClient)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseGofaFile(tt.code)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none. %s", tt.name, tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for %s, but got: %v. %s", tt.name, err, tt.description)
				}
			}
		})
	}
}

// === @MessageAck() Parameter Decorator Tests ===

func TestMessageAckParameterValidation(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		expectError bool
		description string
	}{
		{
			name: "Valid @MessageAck() with *AckCallback type",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@MessageAck() ack *AckCallback) {}
			`,
			expectError: false,
			description: "Should accept @MessageAck() with *AckCallback type",
		},
		{
			name: "Invalid @MessageAck() with wrong type - AckCallback",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")  
func HandleTest(@MessageAck() ack AckCallback) {}
			`,
			expectError: true,
			description: "Should reject @MessageAck() with AckCallback type (must be *AckCallback)",
		},
		{
			name: "Invalid @MessageAck() with wrong type - string",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@MessageAck() ack string) {}
			`,
			expectError: true,
			description: "Should reject @MessageAck() with string type (must be *AckCallback)",
		},
		{
			name: "Invalid @MessageAck() with wrong type - func()",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@MessageAck() ack func()) {}
			`,
			expectError: true,
			description: "Should reject @MessageAck() with func() type (must be *AckCallback)",
		},
		{
			name: "Valid @MessageAck() combined with @MessageBody()",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@MessageBody() data string, @MessageAck() ack *AckCallback) {}
			`,
			expectError: false,
			description: "Should accept @MessageAck() combined with @MessageBody()",
		},
		{
			name: "Valid @MessageAck() combined with @ConnectedSocket()",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@MessageAck() ack *AckCallback, @ConnectedSocket() client *WebSocketClient) {}
			`,
			expectError: false,
			description: "Should accept @MessageAck() combined with @ConnectedSocket()",
		},
		{
			name: "Valid @MessageAck() with multiple decorators",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(
	@MessageBody() data *TestData,
	@ConnectedSocket() client *WebSocketClient,
	@MessageAck() ack *AckCallback
) {}

type TestData struct {
	Message string
}
			`,
			expectError: false,
			description: "Should accept @MessageAck() with multiple other decorators",
		},
		{
			name: "Multiple @MessageAck() parameters",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@MessageAck() ack1 *AckCallback, @MessageAck() ack2 *AckCallback) {}
			`,
			expectError: false,
			description: "Should accept multiple @MessageAck() parameters",
		},
		{
			name: "Invalid @MessageAck() with interface type",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@MessageAck() ack interface{}) {}
			`,
			expectError: true,
			description: "Should reject @MessageAck() with interface{} type (must be *AckCallback)",
		},
		{
			name: "Invalid @MessageAck() with **AckCallback type",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@MessageAck() ack **AckCallback) {}
			`,
			expectError: true,
			description: "Should reject @MessageAck() with **AckCallback type (must be *AckCallback)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseGofaFile(tt.code)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none. %s", tt.name, tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for %s, but got: %v. %s", tt.name, err, tt.description)
				}
			}
		})
	}
}

// === @Rooms() Parameter Decorator Tests ===

func TestRoomsParameterValidation(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		expectError bool
		description string
	}{
		{
			name: "Valid @Rooms() with []string type",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@Rooms() rooms []string) {}
			`,
			expectError: false,
			description: "Should accept @Rooms() with []string type",
		},
		{
			name: "Invalid @Rooms() with wrong type - string",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")  
func HandleTest(@Rooms() room string) {}
			`,
			expectError: true,
			description: "Should reject @Rooms() with string type (must be []string)",
		},
		{
			name: "Invalid @Rooms() with wrong type - []int",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@Rooms() rooms []int) {}
			`,
			expectError: true,
			description: "Should reject @Rooms() with []int type (must be []string)",
		},
		{
			name: "Invalid @Rooms() with wrong type - *[]string",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@Rooms() rooms *[]string) {}
			`,
			expectError: true,
			description: "Should reject @Rooms() with *[]string type (must be []string)",
		},
		{
			name: "Valid @Rooms() combined with @MessageBody()",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@MessageBody() data string, @Rooms() rooms []string) {}
			`,
			expectError: false,
			description: "Should accept @Rooms() combined with @MessageBody()",
		},
		{
			name: "Valid @Rooms() combined with @ConnectedSocket()",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@Rooms() rooms []string, @ConnectedSocket() client *WebSocketClient) {}
			`,
			expectError: false,
			description: "Should accept @Rooms() combined with @ConnectedSocket()",
		},
		{
			name: "Valid @Rooms() with multiple decorators",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(
	@MessageBody() data *TestData,
	@Rooms() rooms []string,
	@ConnectedSocket() client *WebSocketClient,
	@MessageAck() ack *AckCallback
) {}

type TestData struct {
	Message string
}
			`,
			expectError: false,
			description: "Should accept @Rooms() with multiple other decorators",
		},
		{
			name: "Multiple @Rooms() parameters",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@Rooms() userRooms []string, @Rooms() adminRooms []string) {}
			`,
			expectError: false,
			description: "Should accept multiple @Rooms() parameters",
		},
		{
			name: "Invalid @Rooms() with custom struct type",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@Rooms() rooms *RoomList) {}

type RoomList struct {
	Rooms []string
}
			`,
			expectError: true,
			description: "Should reject @Rooms() with custom struct type (must be []string)",
		},
		{
			name: "Invalid @Rooms() with interface type",
			code: `
package test

@WebSocketGateway(3000)
type TestGateway struct {}

@SubscribeMessage("test")
func HandleTest(@Rooms() rooms interface{}) {}
			`,
			expectError: true,
			description: "Should reject @Rooms() with interface{} type (must be []string)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseGofaFile(tt.code)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none. %s", tt.name, tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for %s, but got: %v. %s", tt.name, err, tt.description)
				}
			}
		})
	}
}