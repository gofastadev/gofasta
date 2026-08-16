// Package httpcontext carries request-scoped values that the handler chain
// needs but a handler signature cannot pass.
//
// Two things live here. The request and its response writer, because code
// called far from the handler — a GraphQL resolver setting a cookie, an audit
// record wanting the client IP — has the context and nothing else. And the
// caller's preferred language, so anything rendering text can render it in a
// language the caller reads.
//
// Keys are unexported struct types. A context key of a named string type still
// collides with another package's key of the same underlying string, and the
// collision is silent: one package reads a value another wrote, of a type it
// did not expect, and the type assertion quietly yields the zero value.
package httpcontext

import (
	"context"
	"net/http"
	"strings"
)

type (
	requestKey        struct{}
	responseWriterKey struct{}
	languageKey       struct{}
)

// WithRequest returns a context carrying r.
func WithRequest(ctx context.Context, r *http.Request) context.Context {
	if r == nil {
		return ctx
	}
	return context.WithValue(ctx, requestKey{}, r)
}

// Request returns the request this context was built from, or nil when the
// middleware did not run — a background job, a scheduler, a test.
//
// Callers must handle nil. An audit record built from a nil request has no
// client IP and no user agent, which is worse than it sounds: those are the
// two fields a security investigation starts from.
func Request(ctx context.Context) *http.Request {
	r, _ := ctx.Value(requestKey{}).(*http.Request)
	return r
}

// WithResponseWriter returns a context carrying w.
func WithResponseWriter(ctx context.Context, w http.ResponseWriter) context.Context {
	if w == nil {
		return ctx
	}
	return context.WithValue(ctx, responseWriterKey{}, w)
}

// ResponseWriter returns the response writer for this request, or nil.
//
// The reason to reach for it is almost always a cookie: a GraphQL resolver has
// no writer of its own, and a refresh-token flow has to set one.
func ResponseWriter(ctx context.Context) http.ResponseWriter {
	w, _ := ctx.Value(responseWriterKey{}).(http.ResponseWriter)
	return w
}

// Middleware puts the request and its response writer into the context.
//
// Install it before anything that reads them. It is separate from
// LanguageMiddleware because a service may want one without the other.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := WithRequest(r.Context(), r)
		ctx = WithResponseWriter(ctx, w)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// WithLanguage returns a context carrying lang.
//
// Exported for the paths that know a language but have no HTTP request: a
// scheduled job acting on a known user, a queue consumer, a test.
func WithLanguage(ctx context.Context, lang string) context.Context {
	if lang == "" {
		return ctx
	}
	return context.WithValue(ctx, languageKey{}, lang)
}

// Language returns the language resolved for this request, or "" when the
// middleware did not run. Callers treat "" as "use the default", which is what
// a translation library does with an unrecognized tag.
func Language(ctx context.Context) string {
	lang, _ := ctx.Value(languageKey{}).(string)
	return lang
}

// LanguageMiddleware puts the caller's preferred language into the context.
//
// The source is Accept-Language, which the browser sets from the user's own
// preference. A stored per-user locale is the other candidate, but reading it
// costs a lookup per request to learn something the request already carries —
// and the header is the more current of the two when someone switches language
// in the UI.
//
// A request without the header gets no value and falls back to the service's
// default language.
func LanguageMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if lang := LanguageFromRequest(r); lang != "" {
			r = r.WithContext(WithLanguage(r.Context(), lang))
		}
		next.ServeHTTP(w, r)
	})
}

// LanguageFromRequest returns the most-preferred language tag from
// Accept-Language, or "" when the header is absent or unusable.
//
// This does not do full RFC 4647 quality-value negotiation. Browsers and
// frontend i18n libraries send the preferred tag first, so the first entry is
// the answer; ranking the rest would be machinery serving a case that does not
// arise. The region subtag is kept ("pt-BR" stays "pt-BR") because a
// translation library matches it against the catalog itself and degrades to
// the base language on its own.
func LanguageFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	accept := r.Header.Get("Accept-Language")
	if accept == "" {
		return ""
	}
	first, _, _ := strings.Cut(accept, ",")
	tag, _, _ := strings.Cut(first, ";")
	tag = strings.TrimSpace(tag)
	if tag == "" || tag == "*" {
		return ""
	}
	return tag
}
