package websocket

import (
	"log/slog"
	"sync"
)

// Hub manages WebSocket connections and rooms.
// Register clients, broadcast messages, and manage rooms.
type Hub struct {
	clients    map[*Client]bool
	rooms      map[string]map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	logger     *slog.Logger
}

// Message represents a WebSocket message with optional room targeting.
type Message struct {
	Room    string `json:"room,omitempty"`
	Event   string `json:"event"`
	Payload []byte `json:"payload"`
}

// NewHub creates a new WebSocket hub.
func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]map[*Client]bool),
		broadcast:  make(chan Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
	}
}

// Run starts the hub's event loop. Call this in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Debug("websocket client connected", "id", client.ID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				// Remove from all rooms
				for room, members := range h.rooms {
					delete(members, client)
					if len(members) == 0 {
						delete(h.rooms, room)
					}
				}
			}
			h.mu.Unlock()
			h.logger.Debug("websocket client disconnected", "id", client.ID)

		case msg := <-h.broadcast:
			h.mu.RLock()
			if msg.Room != "" {
				// Send to room members only
				if members, ok := h.rooms[msg.Room]; ok {
					for client := range members {
						select {
						case client.send <- msg.Payload:
						default:
							close(client.send)
							delete(members, client)
						}
					}
				}
			} else {
				// Broadcast to all
				for client := range h.clients {
					select {
					case client.send <- msg.Payload:
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients or to a specific room.
func (h *Hub) Broadcast(msg Message) {
	h.broadcast <- msg
}

// JoinRoom adds a client to a room.
func (h *Hub) JoinRoom(client *Client, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*Client]bool)
	}
	h.rooms[room][client] = true
}

// LeaveRoom removes a client from a room.
func (h *Hub) LeaveRoom(client *Client, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if members, ok := h.rooms[room]; ok {
		delete(members, client)
		if len(members) == 0 {
			delete(h.rooms, room)
		}
	}
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
