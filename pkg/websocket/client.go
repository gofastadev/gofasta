package websocket

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	ws "github.com/gorilla/websocket"
)

const (
	writeWait         = 10 * time.Second
	pongWait          = 60 * time.Second
	defaultPingPeriod = (pongWait * 9) / 10
	maxMsgSize        = 8192
)

var upgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Client represents a single WebSocket connection.
type Client struct {
	ID         string
	hub        *Hub
	conn       *ws.Conn
	send       chan []byte
	logger     *slog.Logger
	pingPeriod time.Duration
}

// ClientOption is a function that configures a Client.
type ClientOption func(*Client)

// WithPingPeriod sets a custom ping period for the client.
func WithPingPeriod(d time.Duration) ClientOption {
	return func(c *Client) {
		c.pingPeriod = d
	}
}

// ServeWS upgrades an HTTP connection to WebSocket and registers the client with the hub.
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request, opts ...ClientOption) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		hub.logger.Error("websocket upgrade failed", "error", err)
		return
	}
	client := &Client{
		ID:         uuid.New().String(),
		hub:        hub,
		conn:       conn,
		send:       make(chan []byte, 256),
		logger:     hub.logger,
		pingPeriod: defaultPingPeriod,
	}
	for _, opt := range opts {
		opt(client)
	}
	hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMsgSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		c.hub.broadcast <- Message{Payload: message}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(c.pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(ws.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(ws.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(ws.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
