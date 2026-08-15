package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCORS_EchoesMatchingOriginNotAJoinedList(t *testing.T) {
	// Access-Control-Allow-Origin takes ONE origin. A previous implementation
	// joined the configured origins with ", ", which browsers reject — every
	// project with more than one origin had CORS failing on every request.
	origins := []string{"https://example.com", "https://other.com"}
	handler := CORS(origins)(noopHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://other.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "https://other.com", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.NotContains(t, rec.Header().Get("Access-Control-Allow-Origin"), ",")
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
	assert.Contains(t, rec.Header().Get("Vary"), "Origin")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCORS_ExposesHeadersClientsNeed(t *testing.T) {
	handler := CORS([]string{"https://example.com"})(noopHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Cookie-based refresh flows are invisible to JS without Set-Cookie here.
	assert.Contains(t, rec.Header().Get("Access-Control-Expose-Headers"), "Set-Cookie")
	// Apollo Client sends this on every non-simple request; omitting it from
	// the allow-list fails the preflight for every GraphQL call.
	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "Apollo-Require-Preflight")
}

func TestCORS_DisallowedOriginGetsNoCORSHeaders(t *testing.T) {
	handler := CORS([]string{"https://example.com"})(noopHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_SuffixLookalikeOriginIsRejected(t *testing.T) {
	// Matching must be exact: a suffix match on "example.com" would also
	// accept "evil-example.com".
	handler := CORS([]string{"https://example.com"})(noopHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil-example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_WildcardOmitsCredentials(t *testing.T) {
	// "*" with Allow-Credentials: true is rejected by browsers.
	handler := CORS([]string{"*"})(noopHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://anything.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORS_PreflightReturnsNoContent(t *testing.T) {
	handler := CORS([]string{"*"})(noopHandler)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, "/api/resource", nil))

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestCORS_NonPreflightCallsNext(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := CORS([]string{"*"})(inner)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/resource", nil))

	assert.True(t, called)
}

func TestCORS_RequestWithoutOriginIsStillServed(t *testing.T) {
	// Non-browser clients send no Origin; they must not be blocked.
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })
	handler := CORS([]string{"https://example.com"})(inner)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.True(t, called)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_EmptyOrigins(t *testing.T) {
	handler := CORS([]string{})(noopHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "", rec.Header().Get("Access-Control-Allow-Origin"))
}

// TestCORS_BlankEntriesAreDiscarded covers the skip branch in the origin
// parser. Origin lists usually arrive from an env var, so trailing separators
// and padded entries are routine; a blank surviving into the allow-list would
// be an origin that matches nothing useful while looking configured.
func TestCORS_BlankEntriesAreDiscarded(t *testing.T) {
	handler := CORS([]string{"", "   ", "https://app.example.com"})(noopHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "https://app.example.com", rec.Header().Get("Access-Control-Allow-Origin"),
		"the one real entry must still be honored")

	// A blank entry must not have become a wildcard or an empty-origin match.
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "", rec.Header().Get("Access-Control-Allow-Origin"))
}
