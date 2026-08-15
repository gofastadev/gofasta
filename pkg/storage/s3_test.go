package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofastadev/gofasta/pkg/config"
	miniogo "github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
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

// TestS3Storage_ResolveBucket pins the per-call bucket contract: a named bucket
// wins, and the empty string falls back to the one configured at construction
// so single-bucket projects never have to name it.
func TestS3Storage_ResolveBucket(t *testing.T) {
	s := &S3Storage{bucket: "configured"}
	assert.Equal(t, "explicit", s.resolveBucket("explicit"))
	assert.Equal(t, "configured", s.resolveBucket(""))
}

func TestS3Storage_Upload_Download_RoundTrip(t *testing.T) {
	storage := sharedMinIO(t)
	bucket := newBucket(t, storage)
	ctx := context.Background()

	content := "hello, this is a round-trip test"
	err := storage.Upload(ctx, bucket, "testfile.txt", strings.NewReader(content), int64(len(content)), nil)
	require.NoError(t, err)

	reader, err := storage.Download(ctx, bucket, "testfile.txt")
	require.NoError(t, err)
	defer reader.Close()

	downloaded, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, string(downloaded))
}

// TestS3Storage_Upload_AppliesOptions covers the opts branch: content type,
// cache-control and user metadata must reach the object, since a browser
// fetching an object served with the wrong type downloads it instead of
// rendering it.
func TestS3Storage_Upload_AppliesOptions(t *testing.T) {
	storage := sharedMinIO(t)
	bucket := newBucket(t, storage)
	ctx := context.Background()

	content := "<h1>hi</h1>"
	err := storage.Upload(ctx, bucket, "page.html", strings.NewReader(content), int64(len(content)), &UploadOptions{
		ContentType:  "text/html",
		CacheControl: "max-age=3600",
		Metadata:     map[string]string{"Owner": "acme"},
	})
	require.NoError(t, err)

	info, err := storage.client.StatObject(ctx, bucket, "page.html", miniogo.StatObjectOptions{})
	require.NoError(t, err)
	assert.Equal(t, "text/html", info.ContentType)
	assert.Equal(t, "max-age=3600", info.Metadata.Get("Cache-Control"))
	assert.Equal(t, "acme", info.UserMetadata["Owner"])
}

func TestS3Storage_Delete(t *testing.T) {
	storage := sharedMinIO(t)
	bucket := newBucket(t, storage)
	ctx := context.Background()

	content := "file to be deleted"
	err := storage.Upload(ctx, bucket, "deleteme.txt", strings.NewReader(content), int64(len(content)), nil)
	require.NoError(t, err)

	err = storage.Delete(ctx, bucket, "deleteme.txt")
	require.NoError(t, err)

	reader, err := storage.Download(ctx, bucket, "deleteme.txt")
	require.NoError(t, err)
	defer reader.Close()

	// minio-go returns the error on Read, not on GetObject
	_, err = io.ReadAll(reader)
	require.Error(t, err, "expected error when reading a deleted object")
}

func TestS3Storage_Upload_SubdirectoryPath(t *testing.T) {
	storage := sharedMinIO(t)
	bucket := newBucket(t, storage)
	ctx := context.Background()

	content := "nested file content"
	path := "subdir/nested/file.txt"

	err := storage.Upload(ctx, bucket, path, strings.NewReader(content), int64(len(content)), nil)
	require.NoError(t, err)

	reader, err := storage.Download(ctx, bucket, path)
	require.NoError(t, err)
	defer reader.Close()

	downloaded, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, string(downloaded))
}

// TestS3Storage_Exists_ReportsPresence covers both answers of the existence
// check. A missing key is the answer (false, nil), not an error — that is what
// lets callers probe availability without transferring bytes.
func TestS3Storage_Exists_ReportsPresence(t *testing.T) {
	storage := sharedMinIO(t)
	bucket := newBucket(t, storage)
	ctx := context.Background()

	require.NoError(t, storage.Upload(ctx, bucket, "here.txt", strings.NewReader("x"), 1, nil))

	present, err := storage.Exists(ctx, bucket, "here.txt")
	require.NoError(t, err)
	assert.True(t, present)

	present, err = storage.Exists(ctx, bucket, "absent.txt")
	require.NoError(t, err, "a missing key must not be reported as a failure")
	assert.False(t, present)
}

// TestS3Storage_Exists_TransportFailureIsNotAbsence is the other half of that
// contract: anything which is not "key not found" has to propagate, otherwise a
// permissions or connectivity fault reads as a confident "no such object".
func TestS3Storage_Exists_TransportFailureIsNotAbsence(t *testing.T) {
	storage := sharedMinIO(t)
	ctx := context.Background()

	present, err := storage.Exists(ctx, "bucket-that-was-never-created", "any.txt")
	require.Error(t, err)
	assert.False(t, present)
	assert.Equal(t, "NoSuchBucket", miniogo.ToErrorResponse(err).Code)
}

func TestS3Storage_Copy_AcrossBuckets(t *testing.T) {
	storage := sharedMinIO(t)
	src := newBucket(t, storage)
	dst := newBucket(t, storage)
	ctx := context.Background()

	content := "payload to duplicate"
	require.NoError(t, storage.Upload(ctx, src, "a/one.txt", strings.NewReader(content), int64(len(content)), nil))

	require.NoError(t, storage.Copy(ctx, src, "a/one.txt", dst, "b/two.txt"))

	reader, err := storage.Download(ctx, dst, "b/two.txt")
	require.NoError(t, err)
	defer reader.Close()
	got, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, string(got))

	// The source must survive a copy.
	present, err := storage.Exists(ctx, src, "a/one.txt")
	require.NoError(t, err)
	assert.True(t, present)
}

func TestS3Storage_Copy_MissingSourceErrors(t *testing.T) {
	storage := sharedMinIO(t)
	bucket := newBucket(t, storage)

	err := storage.Copy(context.Background(), bucket, "nope.txt", bucket, "dst.txt")
	require.Error(t, err)
}

func TestS3Storage_PresignedURL_GrantsDirectAccess(t *testing.T) {
	storage := sharedMinIO(t)
	bucket := newBucket(t, storage)
	ctx := context.Background()

	require.NoError(t, storage.Upload(ctx, bucket, "signed.txt", strings.NewReader("secret"), 6, nil))

	got, err := storage.PresignedURL(ctx, bucket, "signed.txt", 15*time.Minute)
	require.NoError(t, err)
	assert.Contains(t, got, bucket)
	assert.Contains(t, got, "signed.txt")
	assert.Contains(t, got, "X-Amz-Signature", "a presigned URL must carry its signature")
}

// TestS3Storage_PresignedURL_RejectsOutOfRangeExpiry covers the error return:
// S3 signatures are only valid between one second and seven days, and the
// client refuses to mint a URL outside that window.
func TestS3Storage_PresignedURL_RejectsOutOfRangeExpiry(t *testing.T) {
	storage := sharedMinIO(t)
	bucket := newBucket(t, storage)

	got, err := storage.PresignedURL(context.Background(), bucket, "signed.txt", 0)
	require.Error(t, err)
	assert.Empty(t, got)
}

func TestS3Storage_List_ReturnsObjectsUnderPrefix(t *testing.T) {
	storage := sharedMinIO(t)
	bucket := newBucket(t, storage)
	ctx := context.Background()

	require.NoError(t, storage.Upload(ctx, bucket, "x/one.txt", strings.NewReader("1"), 1, nil))
	require.NoError(t, storage.Upload(ctx, bucket, "x/y/two.txt", strings.NewReader("22"), 2, nil))
	require.NoError(t, storage.Upload(ctx, bucket, "z/three.txt", strings.NewReader("333"), 3, nil))

	got, err := storage.List(ctx, bucket, "x")
	require.NoError(t, err)

	paths := make([]string, 0, len(got))
	for _, o := range got {
		paths = append(paths, o.Path)
		assert.NotZero(t, o.Size, "ObjectInfo should carry the size")
		assert.False(t, o.LastModified.IsZero(), "ObjectInfo should carry the modification time")
	}
	assert.ElementsMatch(t, []string{"x/one.txt", "x/y/two.txt"}, paths)
}

// TestS3Storage_List_PropagatesIterationError covers the obj.Err branch: the
// listing channel reports failures per item, so a fault mid-iteration must
// abort rather than return a silently truncated listing.
func TestS3Storage_List_PropagatesIterationError(t *testing.T) {
	storage := sharedMinIO(t)

	got, err := storage.List(context.Background(), "bucket-that-was-never-created", "")
	require.Error(t, err)
	assert.Nil(t, got, "a failed listing must not return partial results")
}

func TestS3Storage_EnsureBucket_CreatesThenIsIdempotent(t *testing.T) {
	storage := sharedMinIO(t)
	ctx := context.Background()
	name := "ensured-bucket-1"

	require.NoError(t, storage.EnsureBucket(ctx, name))

	exists, err := storage.client.BucketExists(ctx, name)
	require.NoError(t, err)
	assert.True(t, exists)

	assert.NoError(t, storage.EnsureBucket(ctx, name), "creating an existing bucket must be a no-op")
}

// TestS3Storage_EnsureBucket_InvalidNameErrors covers the existence-check
// failure branch. The client validates bucket names before any request, so an
// illegal name fails here rather than reaching the server.
func TestS3Storage_EnsureBucket_InvalidNameErrors(t *testing.T) {
	storage := sharedMinIO(t)

	err := storage.EnsureBucket(context.Background(), "A")
	require.Error(t, err)
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
