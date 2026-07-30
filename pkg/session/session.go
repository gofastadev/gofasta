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

// Options controls the session cookie's attributes. The wrapped
// gorilla/sessions store is unexported, so this is the only way callers
// can set them — gorilla v1.2.x's own defaults are HTTPOnly=false,
// Secure=false, SameSite unset, which is not an acceptable posture for
// a session cookie.
type Options struct {
	// Secure marks the cookie HTTPS-only. Enable in production; the
	// default is false so plain-HTTP local development keeps working.
	Secure bool
	// HTTPOnly hides the cookie from JavaScript. Default true.
	HTTPOnly bool
	// SameSite controls cross-site sending. Default http.SameSiteLaxMode.
	SameSite http.SameSite
	// MaxAge is the cookie lifetime in seconds. Default 30 days.
	MaxAge int
	// Path scopes the cookie. Default "/".
	Path string
}

// DefaultOptions returns the safe defaults applied when a constructor
// is called without explicit options.
func DefaultOptions() Options {
	return Options{
		Secure:   false,
		HTTPOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30 * 24 * 60 * 60,
		Path:     "/",
	}
}

// gorillaOptions converts Options, filling zero SameSite/MaxAge/Path
// with the defaults so a partially-populated Options stays sane.
func (o Options) gorillaOptions() *sessions.Options {
	def := DefaultOptions()
	if o.SameSite == 0 {
		o.SameSite = def.SameSite
	}
	if o.MaxAge == 0 {
		o.MaxAge = def.MaxAge
	}
	if o.Path == "" {
		o.Path = def.Path
	}
	return &sessions.Options{
		Path:     o.Path,
		MaxAge:   o.MaxAge,
		Secure:   o.Secure,
		HttpOnly: o.HTTPOnly,
		SameSite: o.SameSite,
	}
}

// resolveOptions picks the caller's options (variadic keeps existing
// call sites source-compatible) or the safe defaults.
func resolveOptions(opts []Options) *sessions.Options {
	if len(opts) > 0 {
		return opts[0].gorillaOptions()
	}
	return DefaultOptions().gorillaOptions()
}

// NewCookieStore creates a cookie-based session store.
// Secret should be 32 or 64 bytes for HMAC signing.
// With no opts, DefaultOptions applies (HTTPOnly, SameSite=Lax).
func NewCookieStore(secret, sessionName string, opts ...Options) *Store {
	cs := sessions.NewCookieStore([]byte(secret))
	cs.Options = resolveOptions(opts)
	return &Store{
		store: cs,
		name:  sessionName,
	}
}

// NewFilesystemStore creates a filesystem-based session store.
// With no opts, DefaultOptions applies (HTTPOnly, SameSite=Lax).
func NewFilesystemStore(path, secret, sessionName string, opts ...Options) *Store {
	fs := sessions.NewFilesystemStore(path, []byte(secret))
	fs.Options = resolveOptions(opts)
	return &Store{
		store: fs,
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
