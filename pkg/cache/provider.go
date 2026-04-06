package cache

import (
	"fmt"
	"log/slog"

	"github.com/healtronlabs/gofasta/configs"
)

// NewCacheService creates the appropriate cache backend from config.
func NewCacheService(cfg *configs.CacheConfig, logger *slog.Logger) (CacheService, error) {
	switch cfg.Driver {
	case "redis":
		logger.Info("initializing Redis cache", "host", cfg.Redis.Host, "port", cfg.Redis.Port)
		return NewRedisCache(cfg.Redis)
	case "memory", "":
		logger.Info("initializing in-memory cache")
		return NewMemoryCache(), nil
	default:
		return nil, fmt.Errorf("unsupported cache driver: %q (supported: memory, redis)", cfg.Driver)
	}
}
