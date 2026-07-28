package storage

import (
	"context"
	"strings"
	"testing"

	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
