package http

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketConnection represents a WebSocket connection
type WebSocketConnection struct {
	conn      *websocket.Conn
	server    *HTTPServer
	id        string
	user      interface{}
	data      map[string]interface{}
	mutex     sync.RWMutex
	closed    bool
	onMessage func(*WebSocketConnection, []byte)
	onClose   func(*WebSocketConnection)
	onError   func(*WebSocketConnection, error)
}

// WebSocketHandler represents a WebSocket handler
type WebSocketHandler interface {
	OnConnect(conn *WebSocketConnection) error
	OnMessage(conn *WebSocketConnection, message []byte) error
	OnDisconnect(conn *WebSocketConnection) error
}

// WebSocketConfig holds WebSocket configuration
type WebSocketConfig struct {
	CheckOrigin     func(*http.Request) bool
	ReadBufferSize  int
	WriteBufferSize int
	EnablePing      bool
	PingInterval    int // seconds
}

// DefaultWebSocketConfig returns default WebSocket configuration
func DefaultWebSocketConfig() *WebSocketConfig {
	return &WebSocketConfig{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins by default
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		EnablePing:      true,
		PingInterval:    30,
	}
}

// WebSocketUpgrade upgrades an HTTP connection to WebSocket
func (s *HTTPServer) WebSocketUpgrade(path string, handler WebSocketHandler, config ...*WebSocketConfig) {
	cfg := DefaultWebSocketConfig()
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	}

	// Configure upgrader
	upgrader := websocket.Upgrader{
		CheckOrigin:     cfg.CheckOrigin,
		ReadBufferSize:  cfg.ReadBufferSize,
		WriteBufferSize: cfg.WriteBufferSize,
	}

	s.router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		// Upgrade the connection
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, fmt.Sprintf("WebSocket upgrade failed: %v", err), http.StatusBadRequest)
			return
		}

		// Create WebSocket connection
		wsConn := &WebSocketConnection{
			conn:   conn,
			server: s,
			id:     generateConnectionID(),
			data:   make(map[string]interface{}),
		}

		// Handle connection
		s.handleWebSocketConnection(wsConn, handler)
	})
}

// handleWebSocketConnection handles a WebSocket connection
func (s *HTTPServer) handleWebSocketConnection(conn *WebSocketConnection, handler WebSocketHandler) {
	defer func() {
		conn.Close()
		if handler != nil {
			handler.OnDisconnect(conn)
		}
	}()

	// Call OnConnect
	if handler != nil {
		if err := handler.OnConnect(conn); err != nil {
			conn.WriteError(err)
			return
		}
	}

	// Start message loop
	for {
		_, message, err := conn.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				if handler != nil && conn.onError != nil {
					conn.onError(conn, err)
				}
			}
			break
		}

		// Handle message
		if handler != nil {
			if err := handler.OnMessage(conn, message); err != nil {
				conn.WriteError(err)
				break
			}
		}

		// Call custom onMessage handler
		if conn.onMessage != nil {
			conn.onMessage(conn, message)
		}
	}
}

// WriteMessage writes a message to the WebSocket connection
func (conn *WebSocketConnection) WriteMessage(messageType int, data []byte) error {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	if conn.closed {
		return fmt.Errorf("connection is closed")
	}

	return conn.conn.WriteMessage(messageType, data)
}

// WriteText writes a text message to the WebSocket connection
func (conn *WebSocketConnection) WriteText(text string) error {
	return conn.WriteMessage(websocket.TextMessage, []byte(text))
}

// WriteBinary writes a binary message to the WebSocket connection
func (conn *WebSocketConnection) WriteBinary(data []byte) error {
	return conn.WriteMessage(websocket.BinaryMessage, data)
}

// WriteJSON writes a JSON message to the WebSocket connection
func (conn *WebSocketConnection) WriteJSON(v interface{}) error {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	if conn.closed {
		return fmt.Errorf("connection is closed")
	}

	return conn.conn.WriteJSON(v)
}

// WriteError writes an error message to the WebSocket connection
func (conn *WebSocketConnection) WriteError(err error) error {
	errorMessage := map[string]interface{}{
		"type":    "error",
		"message": err.Error(),
	}
	return conn.WriteJSON(errorMessage)
}

// Close closes the WebSocket connection
func (conn *WebSocketConnection) Close() error {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	if conn.closed {
		return nil
	}

	conn.closed = true

	// Call custom onClose handler
	if conn.onClose != nil {
		conn.onClose(conn)
	}

	return conn.conn.Close()
}

// GetID returns the connection ID
func (conn *WebSocketConnection) GetID() string {
	return conn.id
}

// GetUser returns the user associated with the connection
func (conn *WebSocketConnection) GetUser() interface{} {
	conn.mutex.RLock()
	defer conn.mutex.RUnlock()
	return conn.user
}

// SetUser sets the user associated with the connection
func (conn *WebSocketConnection) SetUser(user interface{}) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	conn.user = user
}

// GetData returns connection data
func (conn *WebSocketConnection) GetData(key string) interface{} {
	conn.mutex.RLock()
	defer conn.mutex.RUnlock()
	return conn.data[key]
}

// SetData sets connection data
func (conn *WebSocketConnection) SetData(key string, value interface{}) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	conn.data[key] = value
}

// OnMessage sets the message handler
func (conn *WebSocketConnection) OnMessage(handler func(*WebSocketConnection, []byte)) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	conn.onMessage = handler
}

// OnClose sets the close handler
func (conn *WebSocketConnection) OnClose(handler func(*WebSocketConnection)) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	conn.onClose = handler
}

// OnError sets the error handler
func (conn *WebSocketConnection) OnError(handler func(*WebSocketConnection, error)) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	conn.onError = handler
}

// IsClosed returns whether the connection is closed
func (conn *WebSocketConnection) IsClosed() bool {
	conn.mutex.RLock()
	defer conn.mutex.RUnlock()
	return conn.closed
}

// generateConnectionID generates a unique connection ID
func generateConnectionID() string {
	return fmt.Sprintf("ws_%d", time.Now().UnixNano())
}

// WebSocketGateway represents a WebSocket gateway
type WebSocketGateway struct {
	server      *HTTPServer
	connections map[string]*WebSocketConnection
	mutex       sync.RWMutex
}

// NewWebSocketGateway creates a new WebSocket gateway
func NewWebSocketGateway(server *HTTPServer) *WebSocketGateway {
	return &WebSocketGateway{
		server:      server,
		connections: make(map[string]*WebSocketConnection),
	}
}

// AddConnection adds a connection to the gateway
func (gw *WebSocketGateway) AddConnection(conn *WebSocketConnection) {
	gw.mutex.Lock()
	defer gw.mutex.Unlock()
	gw.connections[conn.GetID()] = conn

	// Set close handler to remove from gateway
	conn.OnClose(func(c *WebSocketConnection) {
		gw.RemoveConnection(c.GetID())
	})
}

// RemoveConnection removes a connection from the gateway
func (gw *WebSocketGateway) RemoveConnection(id string) {
	gw.mutex.Lock()
	defer gw.mutex.Unlock()
	delete(gw.connections, id)
}

// GetConnection gets a connection by ID
func (gw *WebSocketGateway) GetConnection(id string) *WebSocketConnection {
	gw.mutex.RLock()
	defer gw.mutex.RUnlock()
	return gw.connections[id]
}

// GetConnections returns all connections
func (gw *WebSocketGateway) GetConnections() map[string]*WebSocketConnection {
	gw.mutex.RLock()
	defer gw.mutex.RUnlock()

	// Return a copy to avoid race conditions
	connections := make(map[string]*WebSocketConnection)
	for id, conn := range gw.connections {
		connections[id] = conn
	}
	return connections
}

// BroadcastText broadcasts a text message to all connections
func (gw *WebSocketGateway) BroadcastText(message string) {
	gw.mutex.RLock()
	defer gw.mutex.RUnlock()

	for _, conn := range gw.connections {
		if !conn.IsClosed() {
			conn.WriteText(message)
		}
	}
}

// BroadcastJSON broadcasts a JSON message to all connections
func (gw *WebSocketGateway) BroadcastJSON(data interface{}) {
	gw.mutex.RLock()
	defer gw.mutex.RUnlock()

	for _, conn := range gw.connections {
		if !conn.IsClosed() {
			conn.WriteJSON(data)
		}
	}
}

// BroadcastToUser broadcasts a message to a specific user
func (gw *WebSocketGateway) BroadcastToUser(userID interface{}, data interface{}) {
	gw.mutex.RLock()
	defer gw.mutex.RUnlock()

	for _, conn := range gw.connections {
		if !conn.IsClosed() && conn.GetUser() == userID {
			conn.WriteJSON(data)
		}
	}
}

// GetConnectionCount returns the number of active connections
func (gw *WebSocketGateway) GetConnectionCount() int {
	gw.mutex.RLock()
	defer gw.mutex.RUnlock()
	return len(gw.connections)
}