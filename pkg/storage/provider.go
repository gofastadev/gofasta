package storage

import (
	"fmt"
	"log/slog"

	"github.com/healtronlabs/gofasta/configs"
)

// NewStorageService creates the appropriate storage backend from config.
func NewStorageService(cfg *configs.StorageConfig, logger *slog.Logger) (StorageService, error) {
	switch cfg.Driver {
	case "s3":
		logger.Info("initializing S3 storage", "endpoint", cfg.S3.Endpoint, "bucket", cfg.S3.Bucket)
		return NewS3Storage(cfg.S3)
	case "local", "":
		logger.Info("initializing local storage", "path", cfg.Local.Path)
		return NewLocalStorage(cfg.Local), nil
	default:
		return nil, fmt.Errorf("unsupported storage driver: %q (supported: local, s3)", cfg.Driver)
	}
}
