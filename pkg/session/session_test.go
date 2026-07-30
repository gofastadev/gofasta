package session

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/sessions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key-32-bytes-long!!"
const testSessionName = "test_session"

func TestNewCookieStore(t *testing.T) {
	s := NewCookieStore(testSecret, testSessionName)
	assert.NotNil(t, s)
	assert.NotNil(t, s.store)
	assert.Equal(t, testSessionName, s.name)
}

func TestNewFilesystemStore(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewFilesystemStore(tmpDir, testSecret, testSessionName)
	assert.NotNil(t, s)
	assert.NotNil(t, s.store)
	assert.Equal(t, testSessionName, s.name)
}

func TestStore_GetSession(t *testing.T) {
	s := NewCookieStore(testSecret, testSessionName)
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	session, err := s.Get(r)
	require.NoError(t, err)
	assert.NotNil(t, session)
	assert.True(t, session.IsNew)
}

func TestStore_Save(t *testing.T) {
	s := NewCookieStore(testSecret, testSessionName)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	session, err := s.Get(r)
	require.NoError(t, err)

	session.Values["foo"] = "bar"
	err = s.Save(r, w, session)
	require.NoError(t, err)

	// Check that Set-Cookie header was written
	cookies := w.Result().Cookies()
	assert.NotEmpty(t, cookies)
}

func TestStore_SetValueAndGetValue(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value interface{}
	}{
		{
			name:  "string value",
			key:   "username",
			value: "john",
		},
		{
			name:  "integer value",
			key:   "count",
			value: 42,
		},
		{
			name:  "boolean value",
			key:   "active",
			value: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewCookieStore(testSecret, testSessionName)
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()

			err := s.SetValue(r, w, tt.key, tt.value)
			require.NoError(t, err)

			// Create a new request with the cookies from the response
			r2 := httptest.NewRequest(http.MethodGet, "/", nil)
			for _, c := range w.Result().Cookies() {
				r2.AddCookie(c)
			}

			val, err := s.GetValue(r2, tt.key)
			require.NoError(t, err)
			assert.Equal(t, tt.value, val)
		})
	}
}

func TestStore_GetValue_NonExistentKey(t *testing.T) {
	s := NewCookieStore(testSecret, testSessionName)
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	val, err := s.GetValue(r, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, val)
}

func TestStore_Destroy(t *testing.T) {
	s := NewCookieStore(testSecret, testSessionName)

	// Set a value first
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	err := s.SetValue(r, w, "key", "value")
	require.NoError(t, err)

	// Destroy the session
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range w.Result().Cookies() {
		r2.AddCookie(c)
	}
	w2 := httptest.NewRecorder()

	err = s.Destroy(r2, w2)
	require.NoError(t, err)

	// Verify MaxAge is -1 on the cookie (session deletion)
	cookies := w2.Result().Cookies()
	assert.NotEmpty(t, cookies)
	found := false
	for _, c := range cookies {
		if c.Name == testSessionName {
			assert.True(t, c.MaxAge < 0, "MaxAge should be negative to delete cookie")
			found = true
		}
	}
	assert.True(t, found, "session cookie should be present in response")
}

func TestStore_SetValue_ErrorOnBadSession(t *testing.T) {
	// Use a store with one secret, then send a cookie encoded with a different secret
	s1 := NewCookieStore("secret-key-32-bytes-long-aaaaaa", testSessionName)
	s2 := NewCookieStore("different-secret-32-bytes-bbbbb!", testSessionName)

	// Set a value with s1
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	_ = s1.SetValue(r, w, "key", "value")

	// Try to SetValue with s2 (different secret) - Get returns error
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range w.Result().Cookies() {
		r2.AddCookie(c)
	}
	w2 := httptest.NewRecorder()
	err := s2.SetValue(r2, w2, "key", "newvalue")
	assert.Error(t, err)
}

func TestStore_GetValue_ErrorOnBadSession(t *testing.T) {
	s1 := NewCookieStore("secret-key-32-bytes-long-aaaaaa", testSessionName)
	s2 := NewCookieStore("different-secret-32-bytes-bbbbb!", testSessionName)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	_ = s1.SetValue(r, w, "key", "value")

	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range w.Result().Cookies() {
		r2.AddCookie(c)
	}
	val, err := s2.GetValue(r2, "key")
	assert.Error(t, err)
	assert.Nil(t, val)
}

func TestStore_Destroy_ErrorOnBadSession(t *testing.T) {
	s1 := NewCookieStore("secret-key-32-bytes-long-aaaaaa", testSessionName)
	s2 := NewCookieStore("different-secret-32-bytes-bbbbb!", testSessionName)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	_ = s1.SetValue(r, w, "key", "value")

	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range w.Result().Cookies() {
		r2.AddCookie(c)
	}
	w2 := httptest.NewRecorder()
	err := s2.Destroy(r2, w2)
	assert.Error(t, err)
}

func TestStore_Destroy_NoExistingSession(t *testing.T) {
	s := NewCookieStore(testSecret, testSessionName)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	// Destroy a session that was never created - should still work
	err := s.Destroy(r, w)
	require.NoError(t, err)
}

func TestStore_FilesystemStore_SetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewFilesystemStore(tmpDir, testSecret, testSessionName)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	err := s.SetValue(r, w, "fs_key", "fs_value")
	require.NoError(t, err)

	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range w.Result().Cookies() {
		r2.AddCookie(c)
	}

	val, err := s.GetValue(r2, "fs_key")
	require.NoError(t, err)
	assert.Equal(t, "fs_value", val)
}

// TestNewCookieStore_DefaultOptions — with no explicit options the safe
// defaults must land on the underlying gorilla store (gorilla v1.2.x
// itself defaults to HTTPOnly=false / SameSite unset).
func TestNewCookieStore_DefaultOptions(t *testing.T) {
	s := NewCookieStore("secret-0123456789abcdef0123456789ab", "app_session")
	cs, ok := s.store.(*sessions.CookieStore)
	if !ok {
		t.Fatal("expected *sessions.CookieStore")
	}
	if !cs.Options.HttpOnly {
		t.Error("expected HTTPOnly true by default")
	}
	if cs.Options.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected SameSite Lax, got %v", cs.Options.SameSite)
	}
	if cs.Options.Path != "/" {
		t.Errorf("expected Path /, got %q", cs.Options.Path)
	}
	if cs.Options.MaxAge != 30*24*60*60 {
		t.Errorf("expected 30d MaxAge, got %d", cs.Options.MaxAge)
	}
	if cs.Options.Secure {
		t.Error("expected Secure false by default (local dev over HTTP)")
	}
}

// TestNewCookieStore_ExplicitOptions — caller options win; zero-valued
// SameSite/MaxAge/Path backfill from defaults.
func TestNewCookieStore_ExplicitOptions(t *testing.T) {
	s := NewCookieStore("secret-0123456789abcdef0123456789ab", "app_session", Options{
		Secure:   true,
		HTTPOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   3600,
		Path:     "/app",
	})
	cs := s.store.(*sessions.CookieStore)
	if !cs.Options.Secure || cs.Options.SameSite != http.SameSiteStrictMode ||
		cs.Options.MaxAge != 3600 || cs.Options.Path != "/app" {
		t.Errorf("explicit options not applied: %+v", cs.Options)
	}

	partial := NewCookieStore("secret-0123456789abcdef0123456789ab", "s", Options{HTTPOnly: true})
	pcs := partial.store.(*sessions.CookieStore)
	if pcs.Options.SameSite != http.SameSiteLaxMode || pcs.Options.Path != "/" || pcs.Options.MaxAge == 0 {
		t.Errorf("zero-value backfill missing: %+v", pcs.Options)
	}
}

// TestNewFilesystemStore_OptionsApplied — same contract for the
// filesystem store.
func TestNewFilesystemStore_OptionsApplied(t *testing.T) {
	s := NewFilesystemStore(t.TempDir(), "secret-0123456789abcdef0123456789ab", "s", Options{
		Secure: true, HTTPOnly: true,
	})
	fs, ok := s.store.(*sessions.FilesystemStore)
	if !ok {
		t.Fatal("expected *sessions.FilesystemStore")
	}
	if !fs.Options.Secure || !fs.Options.HttpOnly {
		t.Errorf("options not applied: %+v", fs.Options)
	}
}
