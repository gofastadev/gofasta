package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStorage(t *testing.T) *LocalStorage {
	t.Helper()
	tmpDir := t.TempDir()
	cfg := config.LocalStorageConfig{Path: tmpDir}
	return NewLocalStorage(cfg)
}

func TestNewLocalStorage(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "uploads")

	s := NewLocalStorage(config.LocalStorageConfig{Path: storagePath})
	assert.NotNil(t, s)
	assert.Equal(t, storagePath, s.basePath)

	// Directory should have been created
	info, err := os.Stat(storagePath)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestLocalStorage_Upload(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		content string
	}{
		{
			name:    "simple file",
			path:    "test.txt",
			content: "hello world",
		},
		{
			name:    "file in subdirectory",
			path:    "subdir/nested/file.txt",
			content: "nested content",
		},
		{
			name:    "empty file",
			path:    "empty.txt",
			content: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStorage(t)
			ctx := context.Background()

			reader := strings.NewReader(tt.content)
			err := s.Upload(ctx, "", tt.path, reader, int64(len(tt.content)), nil)
			require.NoError(t, err)

			// Verify file exists on disk
			fullPath := filepath.Join(s.basePath, tt.path)
			data, err := os.ReadFile(fullPath)
			require.NoError(t, err)
			assert.Equal(t, tt.content, string(data))
		})
	}
}

func TestLocalStorage_Upload_ReadOnlyPath(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a read-only directory so MkdirAll for subdirectories fails
	roDir := filepath.Join(tmpDir, "readonly")
	os.Mkdir(roDir, 0555)

	s := &LocalStorage{basePath: roDir}
	err := s.Upload(context.Background(), "", "subdir/file.txt", strings.NewReader("data"), 4, nil)
	assert.Error(t, err, "MkdirAll should fail on read-only dir")
}

func TestLocalStorage_Upload_FileIsDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewLocalStorage(config.LocalStorageConfig{Path: tmpDir})
	// Create a directory where we'll try to create a file
	os.MkdirAll(filepath.Join(tmpDir, "file.txt"), 0755)
	err := s.Upload(context.Background(), "", "file.txt", strings.NewReader("data"), 4, nil)
	assert.Error(t, err)
}

func TestLocalStorage_Download(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	// Upload first
	content := "download me"
	err := s.Upload(ctx, "", "dl.txt", strings.NewReader(content), int64(len(content)), nil)
	require.NoError(t, err)

	// Download
	rc, err := s.Download(ctx, "", "dl.txt")
	require.NoError(t, err)
	defer rc.Close()

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestLocalStorage_Download_NonExistent(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	_, err := s.Download(ctx, "", "nonexistent.txt")
	assert.Error(t, err)
}

func TestLocalStorage_Delete(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	// Upload a file
	err := s.Upload(ctx, "", "to_delete.txt", strings.NewReader("data"), 4, nil)
	require.NoError(t, err)

	// Delete it
	err = s.Delete(ctx, "", "to_delete.txt")
	require.NoError(t, err)

	// Verify it's gone
	_, err = os.Stat(filepath.Join(s.basePath, "to_delete.txt"))
	assert.True(t, os.IsNotExist(err))
}

func TestLocalStorage_Delete_NonExistent(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	err := s.Delete(ctx, "", "nonexistent.txt")
	assert.Error(t, err)
}

func TestLocalStorage_URL(t *testing.T) {
	s := newTestStorage(t)

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple path",
			path:     "file.txt",
			expected: filepath.Join(s.basePath, "file.txt"),
		},
		{
			name:     "nested path",
			path:     "a/b/c.txt",
			expected: filepath.Join(s.basePath, "a/b/c.txt"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := s.URL("", tt.path)
			assert.Equal(t, tt.expected, url)
		})
	}
}
