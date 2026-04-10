package queue

import (
	"context"

	"github.com/hibiken/asynq"
)

// QueueService is the interface for async task processing.
//
//nolint:revive // name kept for public-API stability; rename is a breaking change.
type QueueService interface {
	Enqueue(ctx context.Context, taskName string, payload []byte, opts ...asynq.Option) (*asynq.TaskInfo, error)
	RegisterHandler(pattern string, handler asynq.Handler)
	Start() error
	Shutdown()
}
