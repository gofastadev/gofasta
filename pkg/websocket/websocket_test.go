package websocket

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	ws "github.com/gorilla/websocket"
)

var testLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

func newTestHub(t *testing.T) *Hub {
	t.Helper()
	h := NewHub(testLogger)
	go h.Run()
	return h
}

func newTestClient(id string, hub *Hub) *Client {
	return &Client{
		ID:     id,
		hub:    hub,
		send:   make(chan []byte, 256),
		logger: testLogger,
	}
}

func wait() {
	time.Sleep(50 * time.Millisecond)
}

func TestNewHub(t *testing.T) {
	h := NewHub(testLogger)
	if h == nil {
		t.Fatal("expected non-nil hub")
	}
	if h.clients == nil {
		t.Error("expected clients map initialized")
	}
	if h.rooms == nil {
		t.Error("expected rooms map initialized")
	}
	if h.broadcast == nil {
		t.Error("expected broadcast channel initialized")
	}
	if h.register == nil {
		t.Error("expected register channel initialized")
	}
	if h.unregister == nil {
		t.Error("expected unregister channel initialized")
	}
}

func TestHub_RegisterClient(t *testing.T) {
	h := newTestHub(t)
	c := newTestClient("test-1", h)

	h.register <- c
	wait()

	if h.ClientCount() != 1 {
		t.Errorf("expected 1 client, got %d", h.ClientCount())
	}
}

func TestHub_UnregisterClient(t *testing.T) {
	h := newTestHub(t)
	c := newTestClient("test-1", h)

	h.register <- c
	wait()

	h.unregister <- c
	wait()

	if h.ClientCount() != 0 {
		t.Errorf("expected 0 clients, got %d", h.ClientCount())
	}
}

func TestHub_JoinRoom(t *testing.T) {
	h := newTestHub(t)
	c := newTestClient("test-1", h)

	h.register <- c
	wait()

	h.JoinRoom(c, "room-a")

	h.mu.RLock()
	members, ok := h.rooms["room-a"]
	h.mu.RUnlock()

	if !ok {
		t.Fatal("expected room 'room-a' to exist")
	}
	if !members[c] {
		t.Error("expected client to be in room")
	}
}

func TestHub_LeaveRoom(t *testing.T) {
	h := newTestHub(t)
	c := newTestClient("test-1", h)

	h.register <- c
	wait()

	h.JoinRoom(c, "room-a")
	h.LeaveRoom(c, "room-a")

	h.mu.RLock()
	_, ok := h.rooms["room-a"]
	h.mu.RUnlock()

	if ok {
		t.Error("expected room 'room-a' to be removed after last member leaves")
	}
}

func TestHub_Broadcast_AllClients(t *testing.T) {
	h := newTestHub(t)
	c1 := newTestClient("test-1", h)
	c2 := newTestClient("test-2", h)

	h.register <- c1
	h.register <- c2
	wait()

	h.Broadcast(Message{Event: "ping", Payload: []byte("hello")})
	wait()

	select {
	case msg := <-c1.send:
		if string(msg) != "hello" {
			t.Errorf("c1: expected 'hello', got %q", string(msg))
		}
	default:
		t.Error("c1: expected to receive broadcast message")
	}

	select {
	case msg := <-c2.send:
		if string(msg) != "hello" {
			t.Errorf("c2: expected 'hello', got %q", string(msg))
		}
	default:
		t.Error("c2: expected to receive broadcast message")
	}
}

func TestHub_Broadcast_Room(t *testing.T) {
	h := newTestHub(t)
	c1 := newTestClient("test-1", h)
	c2 := newTestClient("test-2", h)

	h.register <- c1
	h.register <- c2
	wait()

	h.JoinRoom(c1, "room-a")

	h.Broadcast(Message{Room: "room-a", Event: "ping", Payload: []byte("room-msg")})
	wait()

	select {
	case msg := <-c1.send:
		if string(msg) != "room-msg" {
			t.Errorf("c1: expected 'room-msg', got %q", string(msg))
		}
	default:
		t.Error("c1: expected to receive room broadcast")
	}

	select {
	case <-c2.send:
		t.Error("c2: should not receive room broadcast (not a member)")
	default:
		// expected: c2 is not in the room
	}
}

func TestHub_ClientCount(t *testing.T) {
	h := newTestHub(t)

	if h.ClientCount() != 0 {
		t.Errorf("expected 0, got %d", h.ClientCount())
	}

	c1 := newTestClient("test-1", h)
	c2 := newTestClient("test-2", h)

	h.register <- c1
	h.register <- c2
	wait()

	if h.ClientCount() != 2 {
		t.Errorf("expected 2, got %d", h.ClientCount())
	}
}

func TestServeWS(t *testing.T) {
	h := newTestHub(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWS(h, w, r)
	}))
	defer server.Close()

	// Connect via WebSocket
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := ws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	wait()

	if h.ClientCount() != 1 {
		t.Errorf("expected 1 client after WS connect, got %d", h.ClientCount())
	}
}

func TestServeWS_ReadWrite(t *testing.T) {
	h := newTestHub(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWS(h, w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect client 1
	conn1, _, err := ws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial c1 failed: %v", err)
	}
	defer conn1.Close()

	// Connect client 2
	conn2, _, err := ws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial c2 failed: %v", err)
	}
	defer conn2.Close()

	wait()

	// Client 1 sends a message, which readPump broadcasts to all
	err = conn1.WriteMessage(ws.TextMessage, []byte("hello from c1"))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Client 2 should receive the broadcast
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn2.ReadMessage()
	if err != nil {
		t.Fatalf("read from c2 failed: %v", err)
	}
	if string(msg) != "hello from c1" {
		t.Errorf("expected 'hello from c1', got %q", string(msg))
	}

	// Client 1 should also receive its own broadcast
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err = conn1.ReadMessage()
	if err != nil {
		t.Fatalf("read from c1 failed: %v", err)
	}
	if string(msg) != "hello from c1" {
		t.Errorf("expected 'hello from c1', got %q", string(msg))
	}
}

func TestServeWS_UpgradeError(t *testing.T) {
	h := newTestHub(t)
	rec := httptest.NewRecorder()
	// Send a plain HTTP request (not a WebSocket upgrade) — upgrade should fail gracefully
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	ServeWS(h, rec, req)
	wait()

	if h.ClientCount() != 0 {
		t.Errorf("expected 0 clients after failed upgrade, got %d", h.ClientCount())
	}
}

func TestHub_Broadcast_FullBuffer(t *testing.T) {
	h := newTestHub(t)

	// Create a client with a tiny send buffer (size 1)
	c := &Client{
		ID:     "slow",
		hub:    h,
		send:   make(chan []byte, 1),
		logger: testLogger,
	}

	h.register <- c
	wait()

	// Fill the buffer
	c.send <- []byte("filler")

	// Broadcast — the default case should fire (channel full), closing send
	h.Broadcast(Message{Event: "test", Payload: []byte("overflow")})
	wait()

	// Drain filler if present, then channel should be closed
	_, ok := <-c.send
	if ok {
		_, ok = <-c.send
	}
	if ok {
		t.Error("send channel should be closed after buffer overflow")
	}
}

func TestHub_Broadcast_RoomFullBuffer(t *testing.T) {
	h := newTestHub(t)

	c := &Client{
		ID:     "slow-room",
		hub:    h,
		send:   make(chan []byte, 1),
		logger: testLogger,
	}

	h.register <- c
	wait()
	h.JoinRoom(c, "room-a")

	// Fill the buffer
	c.send <- []byte("filler")

	// Broadcast to the room — default case should fire
	h.Broadcast(Message{Room: "room-a", Event: "test", Payload: []byte("overflow")})
	wait()

	_, ok := <-c.send
	if ok {
		_, ok = <-c.send
	}
	if ok {
		t.Error("send channel should be closed after room buffer overflow")
	}
}

func TestServeWS_ClientDisconnect(t *testing.T) {
	h := newTestHub(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWS(h, w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := ws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	wait()

	if h.ClientCount() != 1 {
		t.Errorf("expected 1 client, got %d", h.ClientCount())
	}

	// Close the client connection — triggers readPump exit → unregister → writePump exit
	conn.Close()

	// Wait for the hub to process the unregister
	time.Sleep(200 * time.Millisecond)
	if h.ClientCount() != 0 {
		t.Errorf("expected 0 clients after disconnect, got %d", h.ClientCount())
	}
}

func TestServeWS_PongHandler(t *testing.T) {
	h := newTestHub(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWS(h, w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := ws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	wait()

	// Send a pong frame to the server — this triggers the server's pong handler
	// which extends the read deadline
	err = conn.WriteMessage(ws.PongMessage, []byte(""))
	if err != nil {
		t.Fatalf("write pong: %v", err)
	}
	wait()

	// If the pong handler works, the read deadline was extended and the client is still connected
	if h.ClientCount() != 1 {
		t.Errorf("expected 1 client after pong, got %d", h.ClientCount())
	}
}

func TestHub_UnregisterClient_RemovesFromRooms(t *testing.T) {
	h := newTestHub(t)
	c := newTestClient("test-1", h)

	h.register <- c
	wait()

	h.JoinRoom(c, "room-a")
	h.JoinRoom(c, "room-b")

	h.unregister <- c
	wait()

	h.mu.RLock()
	_, aExists := h.rooms["room-a"]
	_, bExists := h.rooms["room-b"]
	h.mu.RUnlock()

	if aExists {
		t.Error("expected room-a to be removed")
	}
	if bExists {
		t.Error("expected room-b to be removed")
	}
}
