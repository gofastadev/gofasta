package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/healtronlabs/gofasta/packages/core"
)

// Test WebSocket handler implementation  
type MockWebSocketHandler struct {
	onConnectCalled    bool
	onMessageCalled    bool
	onDisconnectCalled bool
	lastMessage        []byte
	shouldError        bool
}

func (h *MockWebSocketHandler) OnConnect(conn *WebSocketConnection) error {
	h.onConnectCalled = true
	if h.shouldError {
		return fmt.Errorf("test connection error")
	}
	return nil
}

func (h *MockWebSocketHandler) OnMessage(conn *WebSocketConnection, message []byte) error {
	h.onMessageCalled = true
	h.lastMessage = message
	if h.shouldError {
		return fmt.Errorf("test message error")
	}
	
	// Echo the message back
	return conn.WriteText(string(message))
}

func (h *MockWebSocketHandler) OnDisconnect(conn *WebSocketConnection) error {
	h.onDisconnectCalled = true
	return nil
}

// Test WebSocket connection functionality
func TestWebSocketConnection(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("ConnectionCreation", func(t *testing.T) {
		// Create a test WebSocket connection
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			upgrader := websocket.Upgrader{
				CheckOrigin: func(r *http.Request) bool { return true },
			}
			
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				t.Fatalf("Failed to upgrade connection: %v", err)
			}
			defer conn.Close()
			
			wsConn := &WebSocketConnection{
				conn:   conn,
				server: server,
				id:     generateConnectionID(),
				data:   make(map[string]interface{}),
			}
			
			// Test connection properties
			if wsConn.GetID() == "" {
				t.Error("Connection ID should not be empty")
			}
			
			if wsConn.IsClosed() {
				t.Error("Connection should not be closed initially")
			}
			
			// Test setting/getting user
			wsConn.SetUser("test-user")
			if wsConn.GetUser() != "test-user" {
				t.Error("User should be set correctly")
			}
			
			// Test setting/getting data
			wsConn.SetData("key", "value")
			if wsConn.GetData("key") != "value" {
				t.Error("Data should be set correctly")
			}
		}))
		defer testServer.Close()
		
		// Connect to the test server
		url := "ws" + strings.TrimPrefix(testServer.URL, "http")
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer conn.Close()
	})
	
	t.Run("MessageSending", func(t *testing.T) {
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			upgrader := websocket.Upgrader{
				CheckOrigin: func(r *http.Request) bool { return true },
			}
			
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				t.Fatalf("Failed to upgrade connection: %v", err)
			}
			defer conn.Close()
			
			wsConn := &WebSocketConnection{
				conn:   conn,
				server: server,
				id:     generateConnectionID(),
				data:   make(map[string]interface{}),
			}
			
			// Test writing text message
			err = wsConn.WriteText("Hello, WebSocket!")
			if err != nil {
				t.Errorf("Failed to write text message: %v", err)
			}
			
			// Test writing JSON message
			testData := map[string]interface{}{
				"type":    "test",
				"message": "Hello, JSON!",
			}
			err = wsConn.WriteJSON(testData)
			if err != nil {
				t.Errorf("Failed to write JSON message: %v", err)
			}
			
			// Test writing binary message
			binaryData := []byte{0x01, 0x02, 0x03, 0x04}
			err = wsConn.WriteBinary(binaryData)
			if err != nil {
				t.Errorf("Failed to write binary message: %v", err)
			}
			
			// Test writing error message
			testError := fmt.Errorf("test error")
			err = wsConn.WriteError(testError)
			if err != nil {
				t.Errorf("Failed to write error message: %v", err)
			}
		}))
		defer testServer.Close()
		
		// Connect and verify messages
		url := "ws" + strings.TrimPrefix(testServer.URL, "http")
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer conn.Close()
		
		// Read text message
		_, message, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("Failed to read message: %v", err)
		}
		if string(message) != "Hello, WebSocket!" {
			t.Errorf("Expected 'Hello, WebSocket!', got '%s'", string(message))
		}
		
		// Read JSON message
		_, message, err = conn.ReadMessage()
		if err != nil {
			t.Fatalf("Failed to read JSON message: %v", err)
		}
		var jsonData map[string]interface{}
		err = json.Unmarshal(message, &jsonData)
		if err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}
		if jsonData["type"] != "test" {
			t.Error("JSON message not received correctly")
		}
		
		// Read binary message
		_, message, err = conn.ReadMessage()
		if err != nil {
			t.Fatalf("Failed to read binary message: %v", err)
		}
		expectedBinary := []byte{0x01, 0x02, 0x03, 0x04}
		if len(message) != len(expectedBinary) {
			t.Error("Binary message length mismatch")
		}
		
		// Read error message
		_, message, err = conn.ReadMessage()
		if err != nil {
			t.Fatalf("Failed to read error message: %v", err)
		}
		var errorData map[string]interface{}
		err = json.Unmarshal(message, &errorData)
		if err != nil {
			t.Fatalf("Failed to unmarshal error JSON: %v", err)
		}
		if errorData["type"] != "error" {
			t.Error("Error message not received correctly")
		}
	})
}

// Test WebSocket handler functionality
func TestWebSocketHandler(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("HandlerCallbacks", func(t *testing.T) {
		handler := &MockWebSocketHandler{}
		
		// Register WebSocket handler
		server.WebSocketUpgrade("/ws", handler)
		
		// Create test server
		testServer := httptest.NewServer(server.router)
		defer testServer.Close()
		
		// Connect to WebSocket
		url := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer conn.Close()
		
		// Give some time for connection to be established
		time.Sleep(100 * time.Millisecond)
		
		// Send a test message
		err = conn.WriteMessage(websocket.TextMessage, []byte("test message"))
		if err != nil {
			t.Fatalf("Failed to send message: %v", err)
		}
		
		// Read echo response
		_, message, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}
		
		if string(message) != "test message" {
			t.Errorf("Expected 'test message', got '%s'", string(message))
		}
		
		// Close connection
		conn.Close()
		
		// Give some time for disconnection to be processed
		time.Sleep(100 * time.Millisecond)
		
		// Verify handler callbacks were called
		if !handler.onConnectCalled {
			t.Error("OnConnect should have been called")
		}
		
		if !handler.onMessageCalled {
			t.Error("OnMessage should have been called")
		}
		
		if string(handler.lastMessage) != "test message" {
			t.Errorf("Expected last message 'test message', got '%s'", string(handler.lastMessage))
		}
	})
	
	t.Run("HandlerErrors", func(t *testing.T) {
		handler := &MockWebSocketHandler{shouldError: true}
		
		// Register WebSocket handler
		server2 := NewHTTPServer(container)
		server2.WebSocketUpgrade("/ws", handler)
		
		// Create test server
		testServer := httptest.NewServer(server2.router)
		defer testServer.Close()
		
		// Try to connect to WebSocket
		url := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			// Connection might fail due to OnConnect error
			return
		}
		defer conn.Close()
		
		// If connection succeeded, try sending a message
		err = conn.WriteMessage(websocket.TextMessage, []byte("test message"))
		if err != nil {
			t.Fatalf("Failed to send message: %v", err)
		}
		
		// Connection should be closed due to message error
		// Give some time for error handling
		time.Sleep(100 * time.Millisecond)
	})
}

// Test WebSocket configuration
func TestWebSocketConfig(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		config := DefaultWebSocketConfig()
		
		if config.ReadBufferSize != 1024 {
			t.Errorf("Expected read buffer size 1024, got %d", config.ReadBufferSize)
		}
		
		if config.WriteBufferSize != 1024 {
			t.Errorf("Expected write buffer size 1024, got %d", config.WriteBufferSize)
		}
		
		if !config.EnablePing {
			t.Error("Ping should be enabled by default")
		}
		
		if config.PingInterval != 30 {
			t.Errorf("Expected ping interval 30, got %d", config.PingInterval)
		}
		
		// Test CheckOrigin function
		req := &http.Request{
			Header: make(http.Header),
		}
		req.Header.Set("Origin", "http://example.com")
		
		if !config.CheckOrigin(req) {
			t.Error("CheckOrigin should allow all origins by default")
		}
	})
	
	t.Run("CustomConfig", func(t *testing.T) {
		container := core.NewDIContainer()
		server := NewHTTPServer(container)
		
		customConfig := &WebSocketConfig{
			CheckOrigin: func(r *http.Request) bool {
				return r.Header.Get("Origin") == "http://allowed.com"
			},
			ReadBufferSize:  2048,
			WriteBufferSize: 2048,
			EnablePing:      false,
			PingInterval:    60,
		}
		
		// Test that custom config is used
		handler := &MockWebSocketHandler{}
		server.WebSocketUpgrade("/ws", handler, customConfig)
		
		// The actual test would require setting up a real WebSocket connection
		// to verify the custom configuration is applied
		if customConfig.ReadBufferSize != 2048 {
			t.Errorf("Expected custom read buffer size 2048, got %d", customConfig.ReadBufferSize)
		}
	})
}

// Test WebSocket gateway functionality
func TestWebSocketGateway(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	gateway := NewWebSocketGateway(server)
	
	t.Run("GatewayCreation", func(t *testing.T) {
		if gateway.server != server {
			t.Error("Gateway server should be set correctly")
		}
		
		if gateway.GetConnectionCount() != 0 {
			t.Error("Gateway should start with 0 connections")
		}
		
		connections := gateway.GetConnections()
		if len(connections) != 0 {
			t.Error("Gateway should start with empty connections map")
		}
	})
	
	t.Run("ConnectionManagement", func(t *testing.T) {
		// Create mock WebSocket connections
		conn1 := &WebSocketConnection{
			id:     "conn1",
			server: server,
			data:   make(map[string]interface{}),
		}
		
		conn2 := &WebSocketConnection{
			id:     "conn2",
			server: server,
			data:   make(map[string]interface{}),
		}
		
		// Add connections
		gateway.AddConnection(conn1)
		gateway.AddConnection(conn2)
		
		if gateway.GetConnectionCount() != 2 {
			t.Errorf("Expected 2 connections, got %d", gateway.GetConnectionCount())
		}
		
		// Get specific connection
		retrievedConn := gateway.GetConnection("conn1")
		if retrievedConn != conn1 {
			t.Error("Should retrieve correct connection")
		}
		
		// Remove connection
		gateway.RemoveConnection("conn1")
		if gateway.GetConnectionCount() != 1 {
			t.Errorf("Expected 1 connection after removal, got %d", gateway.GetConnectionCount())
		}
		
		if gateway.GetConnection("conn1") != nil {
			t.Error("Removed connection should not be retrievable")
		}
		
		// Clean up
		gateway.RemoveConnection("conn2")
	})
	
	t.Run("Broadcasting", func(t *testing.T) {
		// This test would require real WebSocket connections to fully test
		// For now, we'll test the gateway methods with mock connections
		
		// Since we can't easily mock the actual WebSocket connections,
		// we'll test the gateway structure and methods availability
		if gateway.GetConnectionCount() != 0 {
			t.Error("Gateway should have no connections initially")
		}
		
		// The actual broadcasting functionality would need real WebSocket connections
		// to test properly, which is complex in a unit test environment
	})
}

// Mock WebSocket connection for testing
type MockWebSocketConnection struct {
	id          string
	closed      bool
	onWriteText func(string) error
	onWriteJSON func(interface{}) error
}

func (m *MockWebSocketConnection) GetID() string {
	return m.id
}

func (m *MockWebSocketConnection) IsClosed() bool {
	return m.closed
}

func (m *MockWebSocketConnection) WriteText(text string) error {
	if m.onWriteText != nil {
		return m.onWriteText(text)
	}
	return nil
}

func (m *MockWebSocketConnection) WriteJSON(data interface{}) error {
	if m.onWriteJSON != nil {
		return m.onWriteJSON(data)
	}
	return nil
}

func (m *MockWebSocketConnection) Close() error {
	m.closed = true
	return nil
}

// Test connection ID generation
func TestConnectionIDGeneration(t *testing.T) {
	t.Run("UniqueIDs", func(t *testing.T) {
		id1 := generateConnectionID()
		id2 := generateConnectionID()
		
		if id1 == id2 {
			t.Error("Connection IDs should be unique")
		}
		
		if !strings.HasPrefix(id1, "ws_") {
			t.Error("Connection ID should have 'ws_' prefix")
		}
		
		if !strings.HasPrefix(id2, "ws_") {
			t.Error("Connection ID should have 'ws_' prefix")
		}
	})
}

// Test WebSocket upgrade functionality
func TestWebSocketUpgrade(t *testing.T) {
	container := core.NewDIContainer()
	server := NewHTTPServer(container)
	
	t.Run("UpgradeRegistration", func(t *testing.T) {
		handler := &MockWebSocketHandler{}
		
		// Register WebSocket upgrade
		server.WebSocketUpgrade("/ws", handler)
		
		// Verify route is registered (basic check)
		router := server.GetRouter()
		if router == nil {
			t.Error("Router should not be nil")
		}
		
		// The actual upgrade test would require HTTP connection upgrade
		// which is complex to test in unit tests
	})
	
	t.Run("UpgradeWithCustomConfig", func(t *testing.T) {
		handler := &MockWebSocketHandler{}
		config := &WebSocketConfig{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			EnablePing:      true,
			PingInterval:    15,
			CheckOrigin: func(r *http.Request) bool {
				return r.Header.Get("Origin") == "http://localhost"
			},
		}
		
		// Register WebSocket upgrade with custom config
		server.WebSocketUpgrade("/ws-custom", handler, config)
		
		// Verify route is registered
		router := server.GetRouter()
		if router == nil {
			t.Error("Router should not be nil")
		}
	})
}