package http

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/healtronlabs/gofasta/packages/core"
)

// WebSocketUpgrader configures the websocket upgrader
var WebSocketUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	ID     string
	Conn   *websocket.Conn
	Send   chan []byte
	Hub    *WebSocketHub
	mutex  sync.Mutex
}

// WebSocketHub manages WebSocket connections
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan []byte
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	mutex      sync.RWMutex
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
	}
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mutex.Unlock()

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					delete(h.clients, client)
					close(client.Send)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *WebSocketHub) Broadcast(message []byte) {
	h.broadcast <- message
}

// GetClients returns all connected clients
func (h *WebSocketHub) GetClients() []*WebSocketClient {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	
	clients := make([]*WebSocketClient, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	return clients
}

// ReadPump handles reading messages from the WebSocket connection
func (c *WebSocketClient) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		
		// Process message (this would be enhanced with proper message handling)
		c.Hub.broadcast <- message
	}
}

// WritePump handles writing messages to the WebSocket connection
func (c *WebSocketClient) WritePump() {
	defer c.Conn.Close()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.mutex.Lock()
			err := c.Conn.WriteMessage(websocket.TextMessage, message)
			c.mutex.Unlock()
			
			if err != nil {
				return
			}
		}
	}
}

// WebSocketGateway represents a WebSocket gateway
type WebSocketGateway struct {
	Hub       *WebSocketHub
	Namespace string
	Port      int
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

// WebSocketResponse represents a WebSocket response
type WebSocketResponse struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
	Error string      `json:"error,omitempty"`
}

// HandleWebSocket handles WebSocket connections
func (s *HTTPServer) HandleWebSocket(w http.ResponseWriter, r *http.Request, gateway *WebSocketGateway) {
	conn, err := WebSocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &WebSocketClient{
		ID:   generateClientID(), // Would implement proper ID generation
		Conn: conn,
		Send: make(chan []byte, 256),
		Hub:  gateway.Hub,
	}

	gateway.Hub.register <- client

	go client.WritePump()
	go client.ReadPump()
}

// generateClientID generates a unique client ID
func generateClientID() string {
	// Simplified implementation - would use proper UUID generation
	return "client_" + "random_id"
}

// SubscribeMessage decorator interface
type SubscribeMessage struct {
	Event string
}

// ConnectedSocket parameter decorator
type ConnectedSocket struct{}

// MessageBody parameter decorator
type MessageBody struct{}

// WebSocketDecorator represents WebSocket gateway metadata
type WebSocketDecorator struct {
	Port      int
	Namespace string
}

// EmitToAll emits a message to all connected clients
func (gateway *WebSocketGateway) EmitToAll(event string, data interface{}) error {
	message := WebSocketResponse{
		Event: event,
		Data:  data,
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	gateway.Hub.Broadcast(messageBytes)
	return nil
}

// EmitToClient emits a message to a specific client
func (gateway *WebSocketGateway) EmitToClient(clientID string, event string, data interface{}) error {
	message := WebSocketResponse{
		Event: event,
		Data:  data,
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Find client by ID
	clients := gateway.Hub.GetClients()
	for _, client := range clients {
		if client.ID == clientID {
			select {
			case client.Send <- messageBytes:
				return nil
			default:
				return core.NewInternalServerException("Failed to send message to client", nil)
			}
		}
	}

	return core.NewNotFoundException("Client not found")
}