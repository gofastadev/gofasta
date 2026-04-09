package queue

import (
	"context"
	"log/slog"
	"testing"

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
