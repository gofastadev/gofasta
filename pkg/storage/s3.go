package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Storage stores files in S3-compatible storage (AWS S3, GCS, MinIO, DigitalOcean Spaces).
type S3Storage struct {
	client *minio.Client
	bucket string
}

// NewS3Storage creates a minio-client-backed S3Storage from the given config.
func NewS3Storage(cfg config.S3Config) (*S3Storage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("s3 connection failed: %w", err)
	}
	return &S3Storage{client: client, bucket: cfg.Bucket}, nil
}

// Upload writes size bytes from reader to the object at path.
// resolveBucket returns the bucket to operate on: the caller's choice, or the
// one configured at construction when they pass the empty string.
func (s *S3Storage) resolveBucket(bucket string) string {
	if bucket != "" {
		return bucket
	}
	return s.bucket
}

// Upload writes size bytes from reader to bucket/path.
func (s *S3Storage) Upload(ctx context.Context, bucket, path string, reader io.Reader, size int64, opts *UploadOptions) error {
	putOpts := minio.PutObjectOptions{}
	if opts != nil {
		putOpts.ContentType = opts.ContentType
		putOpts.CacheControl = opts.CacheControl
		putOpts.UserMetadata = opts.Metadata
	}
	_, err := s.client.PutObject(ctx, s.resolveBucket(bucket), path, reader, size, putOpts)
	return err
}

// Download opens the object at bucket/path.
func (s *S3Storage) Download(ctx context.Context, bucket, path string) (io.ReadCloser, error) {
	return s.client.GetObject(ctx, s.resolveBucket(bucket), path, minio.GetObjectOptions{})
}

// Delete removes the object at bucket/path.
func (s *S3Storage) Delete(ctx context.Context, bucket, path string) error {
	return s.client.RemoveObject(ctx, s.resolveBucket(bucket), path, minio.RemoveObjectOptions{})
}

// URL returns the HTTPS URL of the object at bucket/path.
func (s *S3Storage) URL(bucket, path string) string {
	return fmt.Sprintf("https://%s/%s/%s", s.client.EndpointURL().Host, s.resolveBucket(bucket), path)
}

// Exists reports whether an object is present at bucket/path.
//
// StatObject rather than a ranged GetObject: it costs one HEAD and transfers
// no payload, which is what makes an existence check affordable for large
// media. A "key not found" response is the answer (false, nil), not an error —
// only transport or permission failures propagate.
func (s *S3Storage) Exists(ctx context.Context, bucket, path string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.resolveBucket(bucket), path, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Copy duplicates an object server-side, possibly across buckets, without
// routing the bytes through this process.
func (s *S3Storage) Copy(ctx context.Context, srcBucket, srcPath, dstBucket, dstPath string) error {
	_, err := s.client.CopyObject(ctx,
		minio.CopyDestOptions{Bucket: s.resolveBucket(dstBucket), Object: dstPath},
		minio.CopySrcOptions{Bucket: s.resolveBucket(srcBucket), Object: srcPath},
	)
	return err
}

// PresignedURL returns a time-limited URL granting direct GET access.
func (s *S3Storage) PresignedURL(ctx context.Context, bucket, path string, expiry time.Duration) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, s.resolveBucket(bucket), path, expiry, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// List returns the objects under prefix, recursively.
func (s *S3Storage) List(ctx context.Context, bucket, prefix string) ([]ObjectInfo, error) {
	var out []ObjectInfo
	for obj := range s.client.ListObjects(ctx, s.resolveBucket(bucket), minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}) {
		if obj.Err != nil {
			return nil, obj.Err
		}
		out = append(out, ObjectInfo{
			Path:         obj.Key,
			Size:         obj.Size,
			LastModified: obj.LastModified,
			ContentType:  obj.ContentType,
		})
	}
	return out, nil
}

// EnsureBucket creates the bucket when absent.
func (s *S3Storage) EnsureBucket(ctx context.Context, bucket string) error {
	name := s.resolveBucket(bucket)
	exists, err := s.client.BucketExists(ctx, name)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.client.MakeBucket(ctx, name, minio.MakeBucketOptions{})
}
