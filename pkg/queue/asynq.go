package queue

import (
	"context"
	"fmt"

	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/hibiken/asynq"
)

// AsynqQueue implements QueueService using hibiken/asynq (Redis-based).
type AsynqQueue struct {
	client *asynq.Client
	server *asynq.Server
	mux    *asynq.ServeMux
}

// NewAsynqQueue constructs an asynq-backed queue from the given config.
func NewAsynqQueue(cfg *config.QueueConfig) (*AsynqQueue, error) {
	redisOpt := asynq.RedisClientOpt{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	client := asynq.NewClient(redisOpt)

	queues := make(map[string]int)
	for k, v := range cfg.Queues {
		queues[k] = v
	}
	if len(queues) == 0 {
		queues["default"] = 1
	}

	server := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: cfg.Concurrency,
		Queues:      queues,
	})

	return &AsynqQueue{
		client: client,
		server: server,
		mux:    asynq.NewServeMux(),
	}, nil
}

// Enqueue pushes a task onto the queue.
func (q *AsynqQueue) Enqueue(ctx context.Context, taskName string, payload []byte, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	task := asynq.NewTask(taskName, payload, opts...)
	return q.client.EnqueueContext(ctx, task)
}

// RegisterHandler registers a task handler under the given pattern.
func (q *AsynqQueue) RegisterHandler(pattern string, handler asynq.Handler) {
	q.mux.Handle(pattern, handler)
}

// Start begins processing tasks. Blocks until Shutdown is called.
func (q *AsynqQueue) Start() error {
	return q.server.Start(q.mux)
}

// Shutdown gracefully stops the server and closes the client.
func (q *AsynqQueue) Shutdown() {
	q.server.Shutdown()
	_ = q.client.Close()
}
