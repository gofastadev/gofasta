package session

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
