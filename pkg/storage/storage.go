package storage

import (
	"context"
	"io"
)

// StorageService is the interface for file storage backends.
type StorageService interface {
	Upload(ctx context.Context, path string, reader io.Reader, size int64) error
	Download(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	URL(path string) string
}
