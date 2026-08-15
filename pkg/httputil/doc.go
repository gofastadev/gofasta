// Package httputil provides three helpers for HTTP handlers: Bind() parses
// and validates request bodies, Handle() wraps handler functions that return
// errors into standard http.Handler, and OK()/Created()/JSON() write JSON
// responses.
package httputil
