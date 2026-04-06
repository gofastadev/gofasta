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

func NewLocalStorage(cfg config.LocalStorageConfig) *LocalStorage {
	os.MkdirAll(cfg.Path, 0755)
	return &LocalStorage{basePath: cfg.Path}
}

func (s *LocalStorage) Upload(_ context.Context, path string, reader io.Reader, _ int64) error {
	fullPath := filepath.Join(s.basePath, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, reader)
	return err
}

func (s *LocalStorage) Download(_ context.Context, path string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(s.basePath, path))
}

func (s *LocalStorage) Delete(_ context.Context, path string) error {
	return os.Remove(filepath.Join(s.basePath, path))
}

func (s *LocalStorage) URL(path string) string {
	return filepath.Join(s.basePath, path)
}
