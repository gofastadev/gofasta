package queue

import (
	"context"
	"fmt"

	"github.com/healtronlabs/gofasta/configs"
	"github.com/hibiken/asynq"
)

// AsynqQueue implements QueueService using hibiken/asynq (Redis-based).
type AsynqQueue struct {
	client *asynq.Client
	server *asynq.Server
	mux    *asynq.ServeMux
}

func NewAsynqQueue(cfg *configs.QueueConfig) (*AsynqQueue, error) {
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

func (q *AsynqQueue) Enqueue(ctx context.Context, taskName string, payload []byte, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	task := asynq.NewTask(taskName, payload, opts...)
	return q.client.EnqueueContext(ctx, task)
}

func (q *AsynqQueue) RegisterHandler(pattern string, handler asynq.Handler) {
	q.mux.Handle(pattern, handler)
}

func (q *AsynqQueue) Start() error {
	return q.server.Start(q.mux)
}

func (q *AsynqQueue) Shutdown() {
	q.server.Shutdown()
	q.client.Close()
}
