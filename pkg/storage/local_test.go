package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// The tests below drive the filesystem failure branches. They induce errors by
// path collision — putting a regular file where a directory has to be, or a
// directory where a file has to be — which yields a genuine ENOTDIR/EISDIR from
// the kernel deterministically, and behaves the same whether the suite runs as
// root or not. Permission-based tests would silently stop erroring under root,
// which is how CI containers usually run.

// blockerFile writes a regular file inside the storage root and returns the
// bucket-relative name callers should try to descend through.
func blockerFile(t *testing.T, s *LocalStorage) string {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(s.basePath, "blocker"), []byte("x"), 0o644))
	return "blocker"
}

// TestLocalStorage_Exists_StatFailureIsNotAbsence pins the distinction the
// method exists to make: only "file not there" answers false, while a real
// stat fault propagates instead of being mistaken for absence.
func TestLocalStorage_Exists_StatFailureIsNotAbsence(t *testing.T) {
	s := newTestStorage(t)
	blocker := blockerFile(t, s)

	present, err := s.Exists(context.Background(), "", filepath.Join(blocker, "child.txt"))
	require.Error(t, err)
	assert.False(t, present)
	assert.False(t, os.IsNotExist(err), "a non-ENOENT stat failure must not read as absence")
}

func TestLocalStorage_Copy_MissingSourceErrors(t *testing.T) {
	s := newTestStorage(t)

	err := s.Copy(context.Background(), "", "nope.txt", "", "dst.txt")
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

// TestLocalStorage_Copy_UncreatableDestinationParentErrors covers the MkdirAll
// branch: the destination's parent cannot be created because a regular file
// already occupies that path.
func TestLocalStorage_Copy_UncreatableDestinationParentErrors(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()
	require.NoError(t, s.Upload(ctx, "", "src.txt", strings.NewReader("payload"), 7, nil))
	blocker := blockerFile(t, s)

	err := s.Copy(ctx, "", "src.txt", "", filepath.Join(blocker, "child.txt"))
	require.Error(t, err)
	assert.False(t, os.IsNotExist(err))
}

// TestLocalStorage_Copy_DestinationIsADirectoryErrors covers the os.Create
// branch, which is reached only when the parent directory resolves fine and
// the final component is itself a directory.
func TestLocalStorage_Copy_DestinationIsADirectoryErrors(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()
	require.NoError(t, s.Upload(ctx, "", "src.txt", strings.NewReader("payload"), 7, nil))
	require.NoError(t, os.MkdirAll(filepath.Join(s.basePath, "adir"), 0o755))

	err := s.Copy(ctx, "", "src.txt", "", "adir")
	require.Error(t, err)
}

// TestLocalStorage_PresignedURL_ReturnsPlainURL documents the backend's
// deliberate weakening of the interface contract: local disk has nothing to
// sign against, so expiry is ignored and the ordinary URL comes back.
func TestLocalStorage_PresignedURL_ReturnsPlainURL(t *testing.T) {
	s := newTestStorage(t)

	got, err := s.PresignedURL(context.Background(), "bkt", "a/b.txt", time.Minute)
	require.NoError(t, err)
	assert.Equal(t, s.URL("bkt", "a/b.txt"), got)

	// Expiry genuinely makes no difference — worth pinning, because a caller
	// relying on it as a security control would be silently unprotected.
	withoutExpiry, err := s.PresignedURL(context.Background(), "bkt", "a/b.txt", 0)
	require.NoError(t, err)
	assert.Equal(t, got, withoutExpiry)
}

// TestLocalStorage_List_MissingPrefixIsEmptyNotError matches how object stores
// answer for a prefix with no keys.
func TestLocalStorage_List_MissingPrefixIsEmptyNotError(t *testing.T) {
	s := newTestStorage(t)

	got, err := s.List(context.Background(), "bucket-never-created", "no/such/prefix")
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestLocalStorage_List_EntryStatFailurePropagates covers the per-entry Info()
// branch, where an entry can be listed but not described.
//
// This one needs permissions rather than a path collision: ReadDir only needs
// read on the directory, while Info() re-stats each entry and needs execute on
// it. A directory set to r-- therefore enumerates its children and then fails
// on every one of them. The alternative — removing the file between the
// directory read and the stat — is a race, not a test.
//
// Skipped under root, which bypasses permission checks altogether. CI runs as
// an unprivileged user, so the branch is still exercised there.
func TestLocalStorage_List_EntryStatFailurePropagates(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses the permission check this test depends on")
	}

	s := newTestStorage(t)
	ctx := context.Background()
	require.NoError(t, s.Upload(ctx, "", "locked/file.txt", strings.NewReader("x"), 1, nil))

	dir := filepath.Join(s.basePath, "locked")
	require.NoError(t, os.Chmod(dir, 0o444))
	// Restore before TempDir's own cleanup, which cannot remove a directory it
	// is not allowed to traverse.
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	got, err := s.List(ctx, "", "locked")
	require.Error(t, err)
	assert.Nil(t, got, "a listing that cannot describe its entries must not return partial results")
}

// TestLocalStorage_List_WalkFailurePropagates is the counterpart: a walk error
// that is not "missing prefix" is a real fault and must reach the caller rather
// than be flattened into an empty listing.
func TestLocalStorage_List_WalkFailurePropagates(t *testing.T) {
	s := newTestStorage(t)
	blocker := blockerFile(t, s)

	got, err := s.List(context.Background(), "", filepath.Join(blocker, "child"))
	require.Error(t, err)
	assert.False(t, os.IsNotExist(err), "ENOTDIR is a fault, not an empty listing")
	assert.Nil(t, got)
}

// TestLocalStorage_PerCallBucketIsolatesObjects is the point of the per-call
// bucket parameter: one service instance must be able to address several
// buckets, so the same path in two buckets is two distinct objects.
func TestLocalStorage_PerCallBucketIsolatesObjects(t *testing.T) {
	s := NewLocalStorage(config.LocalStorageConfig{Path: t.TempDir()})
	ctx := context.Background()

	require.NoError(t, s.Upload(ctx, "public", "logo.png", strings.NewReader("public-bytes"), 12, nil))
	require.NoError(t, s.Upload(ctx, "private", "logo.png", strings.NewReader("private-bytes"), 13, nil))

	inPublic, err := s.Exists(ctx, "public", "logo.png")
	require.NoError(t, err)
	assert.True(t, inPublic)

	// Deleting from one bucket must leave the other intact.
	require.NoError(t, s.Delete(ctx, "public", "logo.png"))

	inPublic, err = s.Exists(ctx, "public", "logo.png")
	require.NoError(t, err)
	assert.False(t, inPublic)

	inPrivate, err := s.Exists(ctx, "private", "logo.png")
	require.NoError(t, err)
	assert.True(t, inPrivate, "deleting from one bucket must not affect another")
}

// TestLocalStorage_EmptyBucketUsesConfiguredDefault keeps single-bucket
// projects working without naming a bucket anywhere.
func TestLocalStorage_EmptyBucketUsesConfiguredDefault(t *testing.T) {
	s := NewLocalStorage(config.LocalStorageConfig{Path: t.TempDir()})
	ctx := context.Background()

	require.NoError(t, s.Upload(ctx, "", "file.txt", strings.NewReader("data"), 4, nil))

	present, err := s.Exists(ctx, "", "file.txt")
	require.NoError(t, err)
	assert.True(t, present)
}

func TestLocalStorage_CopyAcrossBuckets(t *testing.T) {
	s := NewLocalStorage(config.LocalStorageConfig{Path: t.TempDir()})
	ctx := context.Background()
	require.NoError(t, s.Upload(ctx, "src", "a/one.txt", strings.NewReader("payload"), 7, nil))

	require.NoError(t, s.Copy(ctx, "src", "a/one.txt", "dst", "b/two.txt"))

	present, err := s.Exists(ctx, "dst", "b/two.txt")
	require.NoError(t, err)
	assert.True(t, present)

	// The source must survive a copy.
	present, err = s.Exists(ctx, "src", "a/one.txt")
	require.NoError(t, err)
	assert.True(t, present)
}

func TestLocalStorage_ListIsScopedToBucket(t *testing.T) {
	s := NewLocalStorage(config.LocalStorageConfig{Path: t.TempDir()})
	ctx := context.Background()
	require.NoError(t, s.Upload(ctx, "b1", "x/one.txt", strings.NewReader("1"), 1, nil))
	require.NoError(t, s.Upload(ctx, "b1", "x/y/two.txt", strings.NewReader("2"), 1, nil))
	require.NoError(t, s.Upload(ctx, "b2", "x/three.txt", strings.NewReader("3"), 1, nil))

	got, err := s.List(ctx, "b1", "x")
	require.NoError(t, err)

	paths := make([]string, 0, len(got))
	for _, o := range got {
		paths = append(paths, o.Path)
		assert.NotZero(t, o.Size, "ObjectInfo should carry the size")
	}
	assert.ElementsMatch(t, []string{"x/one.txt", "x/y/two.txt"}, paths)
}

func TestLocalStorage_EnsureBucketIsIdempotent(t *testing.T) {
	s := NewLocalStorage(config.LocalStorageConfig{Path: t.TempDir()})
	ctx := context.Background()
	require.NoError(t, s.EnsureBucket(ctx, "fresh"))
	assert.NoError(t, s.EnsureBucket(ctx, "fresh"), "creating an existing bucket must be a no-op")
}
