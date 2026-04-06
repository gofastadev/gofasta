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

func (s *S3Storage) Upload(ctx context.Context, path string, reader io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, s.bucket, path, reader, size, minio.PutObjectOptions{})
	return err
}

func (s *S3Storage) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	return s.client.GetObject(ctx, s.bucket, path, minio.GetObjectOptions{})
}

func (s *S3Storage) Delete(ctx context.Context, path string) error {
	return s.client.RemoveObject(ctx, s.bucket, path, minio.RemoveObjectOptions{})
}

func (s *S3Storage) URL(path string) string {
	return fmt.Sprintf("https://%s/%s/%s", s.client.EndpointURL().Host, s.bucket, path)
}
