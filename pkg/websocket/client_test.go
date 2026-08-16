package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ws "github.com/gorilla/websocket"
)

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

func TestServeWS_WriteMessageError(t *testing.T) {
	h := newTestHub(t)

	// Use a tiny writeWait so SetWriteDeadline expires immediately,
	// causing WriteMessage(TextMessage) to fail deterministically.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWS(h, w, r, WithWriteWait(time.Nanosecond))
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := ws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	wait()

	if h.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", h.ClientCount())
	}

	// Broadcast a message — hub puts it in client.send, writePump reads it,
	// sets an already-expired write deadline, then WriteMessage fails.
	h.Broadcast(Message{Event: "test", Payload: []byte("should fail")})

	// Wait for writePump to detect the error and for readPump to unregister
	time.Sleep(300 * time.Millisecond)

	if h.ClientCount() != 0 {
		t.Errorf("expected 0 clients after write error, got %d", h.ClientCount())
	}
}

func TestServeWS_PingError(t *testing.T) {
	h := newTestHub(t)

	// Ping period must comfortably exceed wait() (50ms) so the
	// initial "client registered" assertion runs BEFORE the first
	// ping fires; otherwise CI's contended scheduler can fire the
	// ping ticker first, unregister the client, and we'd see
	// ClientCount==0 instead of 1 at the first check. 200ms is the
	// shortest period that's been stable across both local runs and
	// the GitHub Actions Linux runner.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWS(h, w, r,
			WithPingPeriod(200*time.Millisecond),
			WithWriteWait(time.Nanosecond),
		)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := ws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	wait()

	if h.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", h.ClientCount())
	}

	// Wait long enough for the ping ticker to fire — PingPeriod is
	// 200ms, writeWait is 1ns so the ping write fails on the first
	// tick. 500ms (≈ 2.5× the period) is enough headroom even on a
	// slow runner.
	time.Sleep(500 * time.Millisecond)

	if h.ClientCount() != 0 {
		t.Errorf("expected 0 clients after ping error, got %d", h.ClientCount())
	}
}
