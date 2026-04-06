package queue

import (
	"log/slog"

	"github.com/gofastadev/gofasta/configs"
)

// NewQueueService creates the task queue from config. Returns nil if disabled.
func NewQueueService(cfg *configs.QueueConfig, logger *slog.Logger) (QueueService, error) {
	if !cfg.Enabled {
		logger.Info("task queue disabled")
		return nil, nil
	}
	logger.Info("initializing async task queue", "concurrency", cfg.Concurrency)
	return NewAsynqQueue(cfg)
}
