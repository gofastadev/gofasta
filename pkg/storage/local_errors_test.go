package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
