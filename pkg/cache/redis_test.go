package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMiniRedis(t *testing.T) (*miniredis.Miniredis, *RedisCache) {
	t.Helper()
	mr := miniredis.RunT(t)
	cfg := config.RedisConfig{
		Host: mr.Host(),
		Port: mr.Port(),
	}
	rc, err := NewRedisCache(cfg)
	require.NoError(t, err)
	return mr, rc
}

func TestNewRedisCache_Success(t *testing.T) {
	_, rc := setupMiniRedis(t)
	assert.NotNil(t, rc)
	assert.NotNil(t, rc.client)
}

func TestNewRedisCache_ConnectionFails(t *testing.T) {
	cfg := config.RedisConfig{
		Host: "localhost",
		Port: "59999",
	}
	rc, err := NewRedisCache(cfg)
	assert.Error(t, err)
	assert.Nil(t, rc)
	assert.Contains(t, err.Error(), "redis connection failed")
}

func TestRedisCache_Client(t *testing.T) {
	_, rc := setupMiniRedis(t)
	assert.NotNil(t, rc.Client())
}

func TestRedisCache_SetAndGet(t *testing.T) {
	_, rc := setupMiniRedis(t)
	ctx := context.Background()

	err := rc.Set(ctx, "key1", "value1", time.Minute)
	require.NoError(t, err)

	val, err := rc.Get(ctx, "key1")
	require.NoError(t, err)
	assert.Equal(t, "value1", val)
}

func TestRedisCache_Get_Miss(t *testing.T) {
	_, rc := setupMiniRedis(t)
	ctx := context.Background()

	_, err := rc.Get(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestRedisCache_Delete(t *testing.T) {
	_, rc := setupMiniRedis(t)
	ctx := context.Background()

	_ = rc.Set(ctx, "key1", "value1", time.Minute)
	err := rc.Delete(ctx, "key1")
	require.NoError(t, err)

	_, err = rc.Get(ctx, "key1")
	assert.Error(t, err)
}

func TestRedisCache_Flush(t *testing.T) {
	_, rc := setupMiniRedis(t)
	ctx := context.Background()

	_ = rc.Set(ctx, "key1", "v1", time.Minute)
	_ = rc.Set(ctx, "key2", "v2", time.Minute)

	err := rc.Flush(ctx)
	require.NoError(t, err)

	_, err = rc.Get(ctx, "key1")
	assert.Error(t, err)
	_, err = rc.Get(ctx, "key2")
	assert.Error(t, err)
}

func TestRedisCache_Ping(t *testing.T) {
	_, rc := setupMiniRedis(t)
	err := rc.Ping(context.Background())
	assert.NoError(t, err)
}

func TestRedisCache_SetWithTTL(t *testing.T) {
	mr, rc := setupMiniRedis(t)
	ctx := context.Background()

	err := rc.Set(ctx, "ttl-key", "ttl-value", 5*time.Second)
	require.NoError(t, err)

	// Fast-forward miniredis time
	mr.FastForward(6 * time.Second)

	_, err = rc.Get(ctx, "ttl-key")
	assert.Error(t, err)
}
