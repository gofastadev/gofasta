package session

import (
	"net/http"

	"github.com/gorilla/sessions"
)

// Store wraps gorilla/sessions for server-side session management.
type Store struct {
	store sessions.Store
	name  string
}

// NewCookieStore creates a cookie-based session store.
// Secret should be 32 or 64 bytes for HMAC signing.
func NewCookieStore(secret, sessionName string) *Store {
	return &Store{
		store: sessions.NewCookieStore([]byte(secret)),
		name:  sessionName,
	}
}

// NewFilesystemStore creates a filesystem-based session store.
func NewFilesystemStore(path, secret, sessionName string) *Store {
	return &Store{
		store: sessions.NewFilesystemStore(path, []byte(secret)),
		name:  sessionName,
	}
}

// Get returns the session for the current request.
func (s *Store) Get(r *http.Request) (*sessions.Session, error) {
	return s.store.Get(r, s.name)
}

// Save persists the session to the response.
func (s *Store) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	return s.store.Save(r, w, session)
}

// SetValue sets a value in the session.
func (s *Store) SetValue(r *http.Request, w http.ResponseWriter, key string, value interface{}) error {
	session, err := s.Get(r)
	if err != nil {
		return err
	}
	session.Values[key] = value
	return s.Save(r, w, session)
}

// GetValue retrieves a value from the session.
func (s *Store) GetValue(r *http.Request, key string) (interface{}, error) {
	session, err := s.Get(r)
	if err != nil {
		return nil, err
	}
	return session.Values[key], nil
}

// Destroy removes all session data.
func (s *Store) Destroy(r *http.Request, w http.ResponseWriter) error {
	session, err := s.Get(r)
	if err != nil {
		return err
	}
	session.Options.MaxAge = -1
	return s.Save(r, w, session)
}
