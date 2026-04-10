package queue

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/hibiken/asynq"
)

func TestNewQueueService_Disabled(t *testing.T) {
	cfg := &config.QueueConfig{Enabled: false}
	svc, err := NewQueueService(cfg, slog.Default())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc != nil {
		t.Error("expected nil service when disabled")
	}
}

func TestNewQueueService_Enabled(t *testing.T) {
	cfg := &config.QueueConfig{
		Enabled:     true,
		Concurrency: 5,
		Redis: config.QueueRedisConfig{
			Host: "localhost",
			Port: "6379",
		},
	}
	svc, err := NewQueueService(cfg, slog.Default())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service when enabled")
	}
}

func TestNewAsynqQueue(t *testing.T) {
	cfg := &config.QueueConfig{
		Enabled:     true,
		Concurrency: 3,
		Queues:      map[string]int{"default": 1, "critical": 5},
		Redis: config.QueueRedisConfig{
			Host: "localhost",
			Port: "6379",
		},
	}
	q, err := NewAsynqQueue(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q == nil {
		t.Fatal("expected non-nil AsynqQueue")
	}
	if q.client == nil {
		t.Error("expected non-nil client")
	}
	if q.server == nil {
		t.Error("expected non-nil server")
	}
	if q.mux == nil {
		t.Error("expected non-nil mux")
	}
}

func TestAsynqQueue_RegisterHandler(t *testing.T) {
	cfg := &config.QueueConfig{
		Enabled:     true,
		Concurrency: 1,
		Redis: config.QueueRedisConfig{
			Host: "localhost",
			Port: "6379",
		},
	}
	q, err := NewAsynqQueue(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// RegisterHandler should not panic
	handler := asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
		return nil
	})
	q.RegisterHandler("test:task", handler)
}

func TestAsynqQueue_Enqueue(t *testing.T) {
	mr := miniredis.RunT(t)

	cfg := &config.QueueConfig{
		Enabled:     true,
		Concurrency: 1,
		Redis: config.QueueRedisConfig{
			Host: mr.Host(),
			Port: mr.Port(),
		},
	}
	q, err := NewAsynqQueue(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := q.Enqueue(context.Background(), "test:email", []byte(`{"to":"test@example.com"}`))
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil task info")
	}
}

func TestAsynqQueue_Start(t *testing.T) {
	mr := miniredis.RunT(t)

	cfg := &config.QueueConfig{
		Enabled:     true,
		Concurrency: 1,
		Redis: config.QueueRedisConfig{
			Host: mr.Host(),
			Port: mr.Port(),
		},
	}
	q, err := NewAsynqQueue(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Register a handler (asynq requires at least one)
	q.RegisterHandler("test:task", asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
		return nil
	}))

	errCh := make(chan error, 1)
	go func() {
		errCh <- q.Start()
	}()

	// Give the server time to start
	time.Sleep(200 * time.Millisecond)

	// Shutdown should cause Start to return
	q.Shutdown()

	select {
	case startErr := <-errCh:
		// Start may return nil or an error after Shutdown; either way the code path is covered
		_ = startErr
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after Shutdown")
	}
}

func TestAsynqQueue_Shutdown(t *testing.T) {
	mr := miniredis.RunT(t)

	cfg := &config.QueueConfig{
		Enabled:     true,
		Concurrency: 1,
		Redis: config.QueueRedisConfig{
			Host: mr.Host(),
			Port: mr.Port(),
		},
	}
	q, err := NewAsynqQueue(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Shutdown should not panic
	q.Shutdown()
}
