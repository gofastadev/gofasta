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

func wait() {
	time.Sleep(50 * time.Millisecond)
}
