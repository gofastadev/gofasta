package storage

import (
	"testing"

	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewS3Storage_Success(t *testing.T) {
	cfg := config.S3Config{
		Endpoint:  "play.min.io",
		Bucket:    "test-bucket",
		AccessKey: "test-key",
		SecretKey: "test-secret",
		Region:    "us-east-1",
		UseSSL:    true,
	}
	s, err := NewS3Storage(cfg)
	require.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, "test-bucket", s.bucket)
	assert.NotNil(t, s.client)
}

func TestS3Storage_URL(t *testing.T) {
	cfg := config.S3Config{
		Endpoint:  "play.min.io",
		Bucket:    "my-bucket",
		AccessKey: "key",
		SecretKey: "secret",
		UseSSL:    true,
	}
	s, err := NewS3Storage(cfg)
	require.NoError(t, err)

	url := s.URL("path/to/file.txt")
	assert.Contains(t, url, "my-bucket")
	assert.Contains(t, url, "path/to/file.txt")
}
