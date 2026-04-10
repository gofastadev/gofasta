package storage

import (
	"context"
	"fmt"
	"io"

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
func (s *S3Storage) Upload(ctx context.Context, path string, reader io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, s.bucket, path, reader, size, minio.PutObjectOptions{})
	return err
}

// Download returns a reader over the object at path.
func (s *S3Storage) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	return s.client.GetObject(ctx, s.bucket, path, minio.GetObjectOptions{})
}

// Delete removes the object at path.
func (s *S3Storage) Delete(ctx context.Context, path string) error {
	return s.client.RemoveObject(ctx, s.bucket, path, minio.RemoveObjectOptions{})
}

// URL returns the HTTPS URL of the object at path.
func (s *S3Storage) URL(path string) string {
	return fmt.Sprintf("https://%s/%s/%s", s.client.EndpointURL().Host, s.bucket, path)
}
