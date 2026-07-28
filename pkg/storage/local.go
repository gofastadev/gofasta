package storage

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

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

// bucketDir maps a bucket onto a subdirectory of the storage root, so the
// local backend can serve multi-bucket callers the same way the S3 one does.
// An empty bucket means the root itself, which is what a single-bucket project
// gets without naming anything.
func (s *LocalStorage) bucketDir(bucket string) string {
	if bucket == "" {
		return s.basePath
	}
	return filepath.Join(s.basePath, bucket)
}

// full resolves a bucket-and-path pair to an absolute filesystem path.
func (s *LocalStorage) full(bucket, path string) string {
	return filepath.Join(s.bucketDir(bucket), path)
}

// Upload writes the contents of reader to bucket/path.
//
// opts is accepted for interface parity and ignored: a filesystem has nowhere
// to record a content type, cache-control value, or user metadata. Callers
// depending on those must use an object-store backend.
func (s *LocalStorage) Upload(_ context.Context, bucket, path string, reader io.Reader, _ int64, _ *UploadOptions) error {
	fullPath := s.full(bucket, path)
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

// Download opens the file at bucket/path.
func (s *LocalStorage) Download(_ context.Context, bucket, path string) (io.ReadCloser, error) {
	return os.Open(s.full(bucket, path))
}

// Delete removes the file at bucket/path.
//
// Deleting a missing file returns an error, matching this backend's existing
// contract. Note that the S3 backend does not — object stores treat DELETE as
// idempotent. That divergence predates the per-call bucket parameter and is
// left alone here rather than changed as a side effect.
func (s *LocalStorage) Delete(_ context.Context, bucket, path string) error {
	return os.Remove(s.full(bucket, path))
}

// URL returns the filesystem path under which the file is served.
//
// Returned as a plain joined path, not a rooted URL: the base path is usually
// absolute already, and prefixing it would produce a doubled separator.
// Serving these files over HTTP is the caller's concern — it knows the mount
// point, this package does not.
func (s *LocalStorage) URL(bucket, path string) string {
	return filepath.Join(s.bucketDir(bucket), path)
}

// Exists reports whether a file is present. A missing file is the answer
// (false, nil); anything else — a permission failure, a broken mount —
// propagates so it is not mistaken for absence.
func (s *LocalStorage) Exists(_ context.Context, bucket, path string) (bool, error) {
	if _, err := os.Stat(s.full(bucket, path)); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Copy duplicates a file, creating the destination's parent directories.
func (s *LocalStorage) Copy(_ context.Context, srcBucket, srcPath, dstBucket, dstPath string) error {
	src, err := os.Open(s.full(srcBucket, srcPath))
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	dstFull := s.full(dstBucket, dstPath)
	if err := os.MkdirAll(filepath.Dir(dstFull), 0o755); err != nil {
		return err
	}
	dst, err := os.Create(dstFull)
	if err != nil {
		return err
	}
	defer func() { _ = dst.Close() }()

	_, err = io.Copy(dst, src)
	return err
}

// PresignedURL returns the ordinary URL for the object.
//
// Local disk has nothing to sign against, so expiry is ignored and the URL
// carries no time limit. Callers depending on expiry as a security control
// must not use this backend for that data.
func (s *LocalStorage) PresignedURL(_ context.Context, bucket, path string, _ time.Duration) (string, error) {
	return s.URL(bucket, path), nil
}

// List returns the objects under prefix, walking recursively. Paths are
// relative to the bucket so they round-trip into the other methods unchanged.
func (s *LocalStorage) List(_ context.Context, bucket, prefix string) ([]ObjectInfo, error) {
	root := s.bucketDir(bucket)
	var out []ObjectInfo
	err := filepath.WalkDir(filepath.Join(root, prefix), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			// A missing prefix is an empty listing, not a failure — it matches
			// how object stores answer for a prefix with no keys.
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		// Rel's error is discarded because it has no reachable failure mode
		// here: p is a path the walk itself derived from root, so the two
		// always share a form (both absolute or both relative) and sit on one
		// volume — the only conditions under which Rel gives up.
		rel, _ := filepath.Rel(root, p)
		info, infoErr := d.Info()
		if infoErr != nil {
			return infoErr
		}
		out = append(out, ObjectInfo{
			Path:         filepath.ToSlash(rel),
			Size:         info.Size(),
			LastModified: info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// EnsureBucket creates the bucket's directory when absent.
func (s *LocalStorage) EnsureBucket(_ context.Context, bucket string) error {
	return os.MkdirAll(s.bucketDir(bucket), 0o755)
}
