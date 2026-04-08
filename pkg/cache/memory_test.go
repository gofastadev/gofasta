package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryCache(t *testing.T) {
	c := NewMemoryCache()
	assert.NotNil(t, c)
	assert.NotNil(t, c.items)
}

func TestMemoryCache_SetAndGet(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		key       string
		value     interface{}
		ttl       time.Duration
		expected  string
		expectErr bool
	}{
		{
			name:     "set and get string value",
			key:      "key1",
			value:    "value1",
			ttl:      time.Minute,
			expected: "value1",
		},
		{
			name:     "set and get integer value",
			key:      "key2",
			value:    42,
			ttl:      time.Minute,
			expected: "42",
		},
		{
			name:     "set with zero TTL (no expiration)",
			key:      "key3",
			value:    "persistent",
			ttl:      0,
			expected: "persistent",
		},
		{
			name:      "get non-existent key",
			key:       "nonexistent",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewMemoryCache()
			if !tt.expectErr {
				err := c.Set(ctx, tt.key, tt.value, tt.ttl)
				require.NoError(t, err)
			}

			val, err := c.Get(ctx, tt.key)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "cache miss")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, val)
			}
		})
	}
}

func TestMemoryCache_TTLExpiration(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCache()

	err := c.Set(ctx, "expiring", "value", 50*time.Millisecond)
	require.NoError(t, err)

	// Should be available immediately
	val, err := c.Get(ctx, "expiring")
	require.NoError(t, err)
	assert.Equal(t, "value", val)

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Should now be a cache miss
	_, err = c.Get(ctx, "expiring")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache miss")
}

func TestMemoryCache_Delete(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCache()

	err := c.Set(ctx, "toDelete", "value", time.Minute)
	require.NoError(t, err)

	err = c.Delete(ctx, "toDelete")
	require.NoError(t, err)

	_, err = c.Get(ctx, "toDelete")
	assert.Error(t, err)

	// Deleting non-existent key should not error
	err = c.Delete(ctx, "nonexistent")
	assert.NoError(t, err)
}

func TestMemoryCache_Flush(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCache()

	_ = c.Set(ctx, "key1", "val1", time.Minute)
	_ = c.Set(ctx, "key2", "val2", time.Minute)

	err := c.Flush(ctx)
	require.NoError(t, err)

	_, err = c.Get(ctx, "key1")
	assert.Error(t, err)

	_, err = c.Get(ctx, "key2")
	assert.Error(t, err)
}

func TestMemoryCache_Ping(t *testing.T) {
	c := NewMemoryCache()
	err := c.Ping(context.Background())
	assert.NoError(t, err)
}

func TestMemoryCache_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCache()

	var wg sync.WaitGroup
	const goroutines = 100

	// Concurrent writes
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key" + string(rune('0'+n%10))
			_ = c.Set(ctx, key, n, time.Minute)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key" + string(rune('0'+n%10))
			_, _ = c.Get(ctx, key)
		}(i)
	}

	// Concurrent deletes
	for i := 0; i < goroutines/2; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key" + string(rune('0'+n%10))
			_ = c.Delete(ctx, key)
		}(i)
	}

	wg.Wait()
	// If we reach here without a race condition panic, the test passes
}

func TestMemoryCache_OverwriteExistingKey(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCache()

	err := c.Set(ctx, "key", "original", time.Minute)
	require.NoError(t, err)

	err = c.Set(ctx, "key", "updated", time.Minute)
	require.NoError(t, err)

	val, err := c.Get(ctx, "key")
	require.NoError(t, err)
	assert.Equal(t, "updated", val)
}
