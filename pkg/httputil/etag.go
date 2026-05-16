package httputil

import (
	"net/http"
	"strings"

	apperrors "github.com/gofastadev/gofasta/pkg/errors"
)

// ETagOf formats a record version (or any opaque revision token) as a
// strong ETag value: `"<v>"`. Use the result as the value of an ETag
// response header so callers can supply it back as If-Match on a
// follow-up PUT/PATCH/DELETE.
//
//	w.Header().Set("ETag", httputil.ETagOf("3"))   // ETag: "3"
//
// Strong validators (no W/ prefix) are correct here because the body
// is byte-identical for the same record version — there is no
// transformation that would justify weak comparison.
func ETagOf(version string) string {
	return `"` + version + `"`
}

// SetETag is a convenience wrapper around w.Header().Set("ETag", ...)
// + ETagOf. Slightly shorter at the call site:
//
//	httputil.SetETag(w, "3")
func SetETag(w http.ResponseWriter, version string) {
	w.Header().Set("ETag", ETagOf(version))
}

// ParseIfMatch returns the unquoted If-Match value from r. The value
// is what the server originally emitted as the resource's ETag — the
// caller compares it against current server state.
//
// Behavior:
//   - missing header                       → ("", apperrors.PreconditionRequired)
//   - present but only "*" (match-any)     → ("*", nil) — RFC 7232 §3.1
//   - present with one quoted token        → unquoted value, nil
//   - present with multiple comma-sep vals → first token unquoted, nil
//   - W/"<v>" (weak validator)             → "<v>", nil — accepted
//
// The PreconditionRequired error carries an HTTP 428 mapping so an
// AppError-aware middleware (httputil.Handle) translates it without
// the controller needing to set a status code itself.
func ParseIfMatch(r *http.Request) (string, error) {
	raw := r.Header.Get("If-Match")
	if raw == "" {
		return "", apperrors.NewPreconditionRequired(
			"If-Match header is required for this operation", nil)
	}
	// Take the first comma-separated value — multiple ETags are legal
	// per RFC 7232 §3.1 but the common case is one. Servers that need
	// to honor multiple should call this and then split themselves.
	if idx := strings.IndexByte(raw, ','); idx >= 0 {
		raw = raw[:idx]
	}
	raw = strings.TrimSpace(raw)
	if raw == "*" {
		return "*", nil
	}
	// Strip the optional weak prefix and surrounding quotes. Tokens
	// without quotes are non-conformant per the grammar but accepted
	// here for resilience against hand-rolled clients.
	raw = strings.TrimPrefix(raw, "W/")
	raw = strings.TrimPrefix(raw, `"`)
	raw = strings.TrimSuffix(raw, `"`)
	return raw, nil
}
