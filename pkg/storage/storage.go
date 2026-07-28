package storage

import (
	"context"
	"io"
	"time"
)

// ObjectInfo describes a stored object as returned by List.
type ObjectInfo struct {
	Path         string
	Size         int64
	LastModified time.Time
	ContentType  string
}

// UploadOptions carries per-object settings applied at write time.
//
// A nil *UploadOptions is valid and means "no options" — backends then let the
// store pick its own defaults, which for S3 means sniffing the content type.
type UploadOptions struct {
	// ContentType is stored with the object and returned on download. Worth
	// setting explicitly: a browser fetching an object served with the wrong
	// type will download it instead of rendering it.
	ContentType string

	// CacheControl is the Cache-Control header the store returns to clients.
	CacheControl string

	// Metadata is arbitrary user metadata stored alongside the object.
	Metadata map[string]string
}

// StorageService is the interface for file storage backends.
//
// Every method takes the bucket per call rather than binding one at
// construction, because a single service commonly writes to several — public
// assets, private uploads, and generated artifacts often need different
// lifecycle and access policies, and a per-instance bucket would force a
// separate service (and a separate connection pool) for each. Passing the
// empty string selects the bucket configured at construction, so single-bucket
// projects never have to name it.
//
//nolint:revive // name kept for public-API stability; rename is a breaking change.
type StorageService interface {
	// Upload writes size bytes from reader to bucket/path. opts may be nil.
	Upload(ctx context.Context, bucket, path string, reader io.Reader, size int64, opts *UploadOptions) error

	// Download opens the object at bucket/path. The caller closes the reader.
	Download(ctx context.Context, bucket, path string) (io.ReadCloser, error)

	// Delete removes the object at bucket/path.
	//
	// Backends differ on a missing object: the S3 backend treats DELETE as
	// idempotent (object stores do), while the local backend returns an
	// error. Callers that need one behavior should check Exists first.
	Delete(ctx context.Context, bucket, path string) error

	// URL returns the object's canonical address. It does not check existence
	// and grants no access of its own; a private bucket needs PresignedURL.
	URL(bucket, path string) string

	// Exists reports whether an object is present. Distinguished from a failed
	// Download so callers can check availability without transferring bytes.
	Exists(ctx context.Context, bucket, path string) (bool, error)

	// Copy duplicates an object, possibly across buckets. Implementations must
	// avoid round-tripping the bytes through the caller.
	Copy(ctx context.Context, srcBucket, srcPath, dstBucket, dstPath string) error

	// PresignedURL returns a time-limited URL granting direct access without
	// the caller proxying the bytes. Backends with no notion of signing return
	// their ordinary URL and ignore expiry.
	PresignedURL(ctx context.Context, bucket, path string, expiry time.Duration) (string, error)

	// List returns the objects under prefix, recursively. The prefix is
	// matched literally, so a caller wanting a directory should include the
	// trailing separator.
	List(ctx context.Context, bucket, prefix string) ([]ObjectInfo, error)

	// EnsureBucket creates the bucket if it does not exist, and is a no-op if
	// it does. Safe to call on every write path.
	EnsureBucket(ctx context.Context, bucket string) error
}
