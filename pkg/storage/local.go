package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/gofastadev/gofasta/pkg/config"
)

// LocalStorage stores files on the local filesystem.
type LocalStorage struct {
	basePath string
}

// NewLocalStorage returns a LocalStorage rooted at cfg.Path, creating the
// directory if it does not exist.
func NewLocalStorage(cfg config.LocalStorageConfig) *LocalStorage {
	_ = os.MkdirAll(cfg.Path, 0o755)
	return &LocalStorage{basePath: cfg.Path}
}

// Upload writes the contents of reader to the file at path.
func (s *LocalStorage) Upload(_ context.Context, path string, reader io.Reader, _ int64) error {
	fullPath := filepath.Join(s.basePath, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = io.Copy(f, reader)
	return err
}

// Download opens and returns the file at path.
func (s *LocalStorage) Download(_ context.Context, path string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(s.basePath, path))
}

// Delete removes the file at path.
func (s *LocalStorage) Delete(_ context.Context, path string) error {
	return os.Remove(filepath.Join(s.basePath, path))
}

// URL returns the filesystem URL (joined path) for path.
func (s *LocalStorage) URL(path string) string {
	return filepath.Join(s.basePath, path)
}
