package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/gofastadev/gofasta/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestETagOf_Quotes — the returned value is always quoted so
// SetETag(w, "3") produces a valid `"3"` header value per RFC 7232.
func TestETagOf_Quotes(t *testing.T) {
	assert.Equal(t, `"3"`, ETagOf("3"))
	assert.Equal(t, `""`, ETagOf(""))
	assert.Equal(t, `"abc-def"`, ETagOf("abc-def"))
}

// TestSetETag_WritesHeader — the wrapper writes the formatted value
// to the supplied response writer's ETag header. Asserts the header
// shape downstream callers (gofasta scaffold controllers) depend on.
func TestSetETag_WritesHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	SetETag(rec, "7")
	assert.Equal(t, `"7"`, rec.Header().Get("ETag"))
}

// TestParseIfMatch_MissingReturnsPreconditionRequired — RFC 6585 §3
// 428 Precondition Required. The error carries the right AppError
// type so httputil.Handle maps it to 428 without controller code.
func TestParseIfMatch_MissingReturnsPreconditionRequired(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/x", nil)
	v, err := ParseIfMatch(req)
	assert.Equal(t, "", v)
	require.Error(t, err)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperrors.PreconditionRequired, appErr.Type)
}

// TestParseIfMatch_Quoted — the canonical RFC 7232 form: one
// quoted token. Returns the unquoted version.
func TestParseIfMatch_Quoted(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/x", nil)
	req.Header.Set("If-Match", `"3"`)
	v, err := ParseIfMatch(req)
	require.NoError(t, err)
	assert.Equal(t, "3", v)
}

// TestParseIfMatch_Weak — `W/"3"` is the weak validator form; we
// strip the W/ prefix so the same callers handle both. Strong vs
// weak comparison is the server's responsibility downstream — for
// record-version use we already know it's exact equality.
func TestParseIfMatch_Weak(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/x", nil)
	req.Header.Set("If-Match", `W/"3"`)
	v, err := ParseIfMatch(req)
	require.NoError(t, err)
	assert.Equal(t, "3", v)
}

// TestParseIfMatch_Wildcard — `*` means "match any current
// representation" per RFC 7232 §3.1. We surface it verbatim so the
// caller can decide whether to honor the precondition unconditionally
// or refuse on safety grounds.
func TestParseIfMatch_Wildcard(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/x", nil)
	req.Header.Set("If-Match", "*")
	v, err := ParseIfMatch(req)
	require.NoError(t, err)
	assert.Equal(t, "*", v)
}

// TestParseIfMatch_MultipleETags — a comma-separated list is legal;
// we take the first value. Servers needing every value should call
// ParseIfMatch then handle the raw header themselves.
func TestParseIfMatch_MultipleETags(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/x", nil)
	req.Header.Set("If-Match", `"3", "4", "5"`)
	v, err := ParseIfMatch(req)
	require.NoError(t, err)
	assert.Equal(t, "3", v)
}

// TestParseIfMatch_Unquoted — non-conformant clients sometimes send
// the bare token. Accepted for resilience so a misconfigured curl
// invocation doesn't return 428 when the intent is clear.
func TestParseIfMatch_Unquoted(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/x", nil)
	req.Header.Set("If-Match", "3")
	v, err := ParseIfMatch(req)
	require.NoError(t, err)
	assert.Equal(t, "3", v)
}
