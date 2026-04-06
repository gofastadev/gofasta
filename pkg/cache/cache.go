package cache

import (
	"context"
	"time"
)

// CacheService is the interface for all cache backends.
type CacheService interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Flush(ctx context.Context) error
	Ping(ctx context.Context) error
}
