package config

import "testing"

func TestRedisFromEnv(t *testing.T) {
	t.Run("host and port split", func(t *testing.T) {
		t.Setenv("REDIS_URL", "redis_container:6380")
		t.Setenv("REDIS_PASSWORD", "s3cret")
		t.Setenv("REDIS_DB", "3")

		got := RedisFromEnv()
		if got.Host != "redis_container" || got.Port != "6380" {
			t.Errorf("host/port = %q/%q", got.Host, got.Port)
		}
		if got.Password != "s3cret" || got.DB != 3 {
			t.Errorf("password/db = %q/%d", got.Password, got.DB)
		}
	})

	t.Run("missing port defaults to 6379", func(t *testing.T) {
		t.Setenv("REDIS_URL", "redis_container")
		if got := RedisFromEnv(); got.Port != "6379" {
			t.Errorf("port = %q, want 6379", got.Port)
		}
	})

	t.Run("unparseable db is zero", func(t *testing.T) {
		t.Setenv("REDIS_URL", "h:1")
		t.Setenv("REDIS_DB", "not-a-number")
		if got := RedisFromEnv(); got.DB != 0 {
			t.Errorf("db = %d, want 0", got.DB)
		}
	})
}
