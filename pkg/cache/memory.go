package cache

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryCache implements CacheService using an in-memory map with TTL.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]memoryItem
}

type memoryItem struct {
	value     string
	expiresAt time.Time
}

// NewMemoryCache returns an empty in-memory cache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{items: make(map[string]memoryItem)}
}

// Get returns the value cached under key, or an error on miss/expired entry.
func (m *MemoryCache) Get(_ context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.items[key]
	if !ok || (!item.expiresAt.IsZero() && time.Now().After(item.expiresAt)) {
		return "", fmt.Errorf("cache miss: %s", key)
	}
	return item.value, nil
}

// Set stores value under key, expiring after ttl (0 = no expiry).
func (m *MemoryCache) Set(_ context.Context, key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var expires time.Time
	if ttl > 0 {
		expires = time.Now().Add(ttl)
	}
	m.items[key] = memoryItem{value: fmt.Sprintf("%v", value), expiresAt: expires}
	return nil
}

// Delete removes the entry under key if present.
func (m *MemoryCache) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, key)
	return nil
}

// Flush removes every entry from the cache.
func (m *MemoryCache) Flush(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = make(map[string]memoryItem)
	return nil
}

// Ping always succeeds for the in-memory cache.
func (m *MemoryCache) Ping(_ context.Context) error {
	return nil
}

// purgeExpired removes all expired items from the cache.
func (m *MemoryCache) purgeExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for k, v := range m.items {
		if !v.expiresAt.IsZero() && now.After(v.expiresAt) {
			delete(m.items, k)
		}
	}
}
