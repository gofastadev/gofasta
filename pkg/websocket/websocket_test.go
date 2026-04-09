package websocket

import (
	"log/slog"
	"os"
	"testing"
	"time"
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
