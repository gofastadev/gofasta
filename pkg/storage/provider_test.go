package storage

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

func TestNewStorageService_Local(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.StorageConfig{
		Driver: "local",
		Local:  config.LocalStorageConfig{Path: tmpDir},
	}
	svc, err := NewStorageService(cfg, testLogger())
	require.NoError(t, err)
	assert.NotNil(t, svc)
	_, ok := svc.(*LocalStorage)
	assert.True(t, ok)
}

func TestNewStorageService_LocalEmptyDriver(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.StorageConfig{
		Driver: "",
		Local:  config.LocalStorageConfig{Path: tmpDir},
	}
	svc, err := NewStorageService(cfg, testLogger())
	require.NoError(t, err)
	assert.NotNil(t, svc)
}

func TestNewStorageService_S3(t *testing.T) {
	cfg := &config.StorageConfig{
		Driver: "s3",
		S3: config.S3Config{
			Endpoint:  "play.min.io",
			Bucket:    "test-bucket",
			AccessKey: "test-key",
			SecretKey: "test-secret",
			Region:    "us-east-1",
			UseSSL:    true,
		},
	}
	svc, err := NewStorageService(cfg, testLogger())
	require.NoError(t, err)
	assert.NotNil(t, svc)
	_, ok := svc.(*S3Storage)
	assert.True(t, ok)
}

func TestNewStorageService_UnsupportedDriver(t *testing.T) {
	cfg := &config.StorageConfig{Driver: "gcs"}
	svc, err := NewStorageService(cfg, testLogger())
	require.Error(t, err)
	assert.Nil(t, svc)
	assert.Contains(t, err.Error(), "unsupported storage driver")
}
