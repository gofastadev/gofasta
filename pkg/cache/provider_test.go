package cache

import (
	"log/slog"
	"os"
	"testing"

	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNewCacheService_Memory(t *testing.T) {
	cfg := &config.CacheConfig{Driver: "memory"}
	svc, err := NewCacheService(cfg, testLogger())
	require.NoError(t, err)
	assert.NotNil(t, svc)
	_, ok := svc.(*MemoryCache)
	assert.True(t, ok)
}

func TestNewCacheService_MemoryEmptyDriver(t *testing.T) {
	cfg := &config.CacheConfig{Driver: ""}
	svc, err := NewCacheService(cfg, testLogger())
	require.NoError(t, err)
	assert.NotNil(t, svc)
}

func TestNewCacheService_UnsupportedDriver(t *testing.T) {
	cfg := &config.CacheConfig{Driver: "memcached"}
	svc, err := NewCacheService(cfg, testLogger())
	require.Error(t, err)
	assert.Nil(t, svc)
	assert.Contains(t, err.Error(), "unsupported cache driver")
}

func TestNewCacheService_RedisConnectionFails(t *testing.T) {
	cfg := &config.CacheConfig{
		Driver: "redis",
		Redis: config.RedisConfig{
			Host: "localhost",
			Port: "59999", // unreachable port
		},
	}
	svc, err := NewCacheService(cfg, testLogger())
	assert.Error(t, err)
	assert.Nil(t, svc)
}
