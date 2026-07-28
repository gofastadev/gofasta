package storage

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gofastadev/gofasta/pkg/config"
	miniogo "github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcminio "github.com/testcontainers/testcontainers-go/modules/minio"
)

// minioImage pins the MinIO release the S3 integration tests run against.
const minioImage = "minio/minio:RELEASE.2024-01-16T16-07-38Z"

// defaultTestBucket backs the calls that pass "" as the bucket, which is how a
// single-bucket project addresses storage.
const defaultTestBucket = "default-bucket"

// Every S3 test in this package shares one MinIO container. They need separate
// buckets, not separate servers, and a container per test cost roughly eight
// seconds each.
//
// Startup is lazy rather than eager in TestMain on purpose: the local-disk and
// provider tests in this package need no Docker at all, and starting the
// container from TestMain would make the whole package undiagnosably fail on a
// machine without a daemon. Only a test that asks for MinIO pays for it.
var (
	minioOnce      sync.Once
	minioStorage   *S3Storage
	minioStartErr  error
	minioTerminate func()
)

func startSharedMinIO() {
	ctx := context.Background()

	container, err := tcminio.Run(ctx, minioImage,
		testcontainers.WithEnv(map[string]string{
			"MINIO_ROOT_USER":     "minioadmin",
			"MINIO_ROOT_PASSWORD": "minioadmin",
		}),
	)
	if err != nil {
		minioStartErr = fmt.Errorf("starting minio container: %w", err)
		return
	}
	minioTerminate = func() { _ = testcontainers.TerminateContainer(container) }

	endpoint, err := container.ConnectionString(ctx)
	if err != nil {
		minioStartErr = fmt.Errorf("reading minio connection string: %w", err)
		return
	}

	s, err := NewS3Storage(config.S3Config{
		Endpoint:  endpoint,
		Bucket:    defaultTestBucket,
		AccessKey: container.Username,
		SecretKey: container.Password,
		Region:    "us-east-1",
		UseSSL:    false,
	})
	if err != nil {
		minioStartErr = fmt.Errorf("building s3 storage: %w", err)
		return
	}
	if err := s.client.MakeBucket(ctx, defaultTestBucket, miniogo.MakeBucketOptions{}); err != nil {
		minioStartErr = fmt.Errorf("creating %s: %w", defaultTestBucket, err)
		return
	}
	minioStorage = s
}

// sharedMinIO returns the package-wide S3Storage, starting the container on
// first use.
func sharedMinIO(t *testing.T) *S3Storage {
	t.Helper()
	requireIntegration(t)
	minioOnce.Do(startSharedMinIO)
	require.NoError(t, minioStartErr)
	return minioStorage
}

// bucketSeq names buckets uniquely so tests sharing one container cannot see
// each other's objects.
var bucketSeq atomic.Uint64

// newBucket creates an empty bucket for the calling test's exclusive use.
func newBucket(t *testing.T, s *S3Storage) string {
	t.Helper()
	name := fmt.Sprintf("bucket-%d", bucketSeq.Add(1))
	require.NoError(t, s.client.MakeBucket(context.Background(), name, miniogo.MakeBucketOptions{}))
	return name
}

// requireIntegration gates the tests that need a container, matching the
// short-mode convention the rest of the suite uses.
func requireIntegration(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	// m.Run has already written the coverage profile, so exiting here is safe.
	if minioTerminate != nil {
		minioTerminate()
	}
	os.Exit(code)
}
