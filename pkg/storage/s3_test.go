package storage

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/gofastadev/gofasta/pkg/config"
	miniogo "github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcminio "github.com/testcontainers/testcontainers-go/modules/minio"
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

	url := s.URL("", "path/to/file.txt")
	assert.Contains(t, url, "my-bucket")
	assert.Contains(t, url, "path/to/file.txt")
}

// setupMinIO starts a MinIO testcontainer and returns a configured S3Storage
// with a pre-created test bucket. The container is terminated when the test ends.
func setupMinIO(t *testing.T) *S3Storage {
	t.Helper()
	ctx := context.Background()

	container, err := tcminio.Run(ctx,
		"minio/minio:RELEASE.2024-01-16T16-07-38Z",
		testcontainers.WithEnv(map[string]string{
			"MINIO_ROOT_USER":     "minioadmin",
			"MINIO_ROOT_PASSWORD": "minioadmin",
		}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(container) })

	endpoint, err := container.ConnectionString(ctx)
	require.NoError(t, err)

	cfg := config.S3Config{
		Endpoint:  endpoint,
		Bucket:    "test-bucket",
		AccessKey: container.Username,
		SecretKey: container.Password,
		Region:    "us-east-1",
		UseSSL:    false,
	}

	storage, err := NewS3Storage(cfg)
	require.NoError(t, err)

	err = storage.client.MakeBucket(ctx, "test-bucket", miniogo.MakeBucketOptions{})
	require.NoError(t, err)

	return storage
}

func TestS3Storage_Upload_Download_RoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupMinIO(t)
	ctx := context.Background()

	content := "hello, this is a round-trip test"
	err := storage.Upload(ctx, "", "testfile.txt", strings.NewReader(content), int64(len(content)), nil)
	require.NoError(t, err)

	reader, err := storage.Download(ctx, "", "testfile.txt")
	require.NoError(t, err)
	defer reader.Close()

	downloaded, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, string(downloaded))
}

func TestS3Storage_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupMinIO(t)
	ctx := context.Background()

	content := "file to be deleted"
	err := storage.Upload(ctx, "", "deleteme.txt", strings.NewReader(content), int64(len(content)), nil)
	require.NoError(t, err)

	err = storage.Delete(ctx, "", "deleteme.txt")
	require.NoError(t, err)

	reader, err := storage.Download(ctx, "", "deleteme.txt")
	require.NoError(t, err)
	defer reader.Close()

	// minio-go returns the error on Read, not on GetObject
	_, err = io.ReadAll(reader)
	require.Error(t, err, "expected error when reading a deleted object")
}

func TestS3Storage_Upload_SubdirectoryPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupMinIO(t)
	ctx := context.Background()

	content := "nested file content"
	path := "subdir/nested/file.txt"

	err := storage.Upload(ctx, "", path, strings.NewReader(content), int64(len(content)), nil)
	require.NoError(t, err)

	reader, err := storage.Download(ctx, "", path)
	require.NoError(t, err)
	defer reader.Close()

	downloaded, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, string(downloaded))
}

func TestNewS3Storage_EmptyEndpoint(t *testing.T) {
	cfg := config.S3Config{
		Endpoint:  "",
		Bucket:    "test",
		AccessKey: "key",
		SecretKey: "secret",
	}
	storage, err := NewS3Storage(cfg)
	assert.Error(t, err)
	assert.Nil(t, storage)
	assert.Contains(t, err.Error(), "s3 connection failed")
}

func TestNewS3Storage_ConnectionError(t *testing.T) {
	cfg := config.S3Config{
		Endpoint:  "invalid.endpoint.that.will.not.resolve.test:9000",
		Bucket:    "test-bucket",
		AccessKey: "key",
		SecretKey: "secret",
		Region:    "us-east-1",
		UseSSL:    false,
	}

	// NewS3Storage only creates a client; it does not connect immediately.
	// The error surfaces on the first operation.
	storage, err := NewS3Storage(cfg)
	if err != nil {
		// If the constructor itself returns an error, that satisfies the test.
		return
	}
	require.NotNil(t, storage)

	ctx := context.Background()
	err = storage.Upload(ctx, "", "file.txt", strings.NewReader("data"), 4, nil)
	assert.Error(t, err, "expected error when uploading to an invalid endpoint")
}
