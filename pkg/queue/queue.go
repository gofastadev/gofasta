package queue

import (
	"context"

	"github.com/hibiken/asynq"
)

// QueueService is the interface for async task processing.
type QueueService interface {
	Enqueue(ctx context.Context, taskName string, payload []byte, opts ...asynq.Option) (*asynq.TaskInfo, error)
	RegisterHandler(pattern string, handler asynq.Handler)
	Start() error
	Shutdown()
}
