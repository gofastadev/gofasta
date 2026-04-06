package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/healtronlabs/gofasta/configs"
	"github.com/redis/go-redis/v9"
)

// RedisCache implements CacheService using Redis.
type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(cfg configs.RedisConfig) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}
	return &RedisCache{client: client}, nil
}

// Client returns the underlying redis client for reuse (rate limiter, queue, etc.)
func (r *RedisCache) Client() *redis.Client {
	return r.client
}

func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *RedisCache) Flush(ctx context.Context) error {
	return r.client.FlushDB(ctx).Err()
}

func (r *RedisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
