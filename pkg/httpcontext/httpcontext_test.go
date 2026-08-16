package httpcontext

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequest_AbsentWhenMiddlewareDidNotRun(t *testing.T) {
	// The case that matters: an audit record built from this context has no
	// client IP and no user agent. Callers must be able to detect it.
	if got := Request(context.Background()); got != nil {
		t.Errorf("Request() = %v, want nil", got)
	}
	if got := ResponseWriter(context.Background()); got != nil {
		t.Errorf("ResponseWriter() = %v, want nil", got)
	}
	if got := Language(context.Background()); got != "" {
		t.Errorf("Language() = %q, want empty so callers fall back", got)
	}
}

func TestMiddleware_CarriesRequestAndWriter(t *testing.T) {
	var gotReq *http.Request
	var gotWriter http.ResponseWriter

	handler := Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotReq = Request(r.Context())
		gotWriter = ResponseWriter(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = "203.0.113.7:5555"
	req.Header.Set("User-Agent", "probe/1.0")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if gotReq == nil {
		t.Fatal("request not carried")
	}
	// Asserted through the fields an audit record actually reads.
	if gotReq.RemoteAddr != "203.0.113.7:5555" {
		t.Errorf("RemoteAddr = %q", gotReq.RemoteAddr)
	}
	if gotReq.UserAgent() != "probe/1.0" {
		t.Errorf("UserAgent = %q", gotReq.UserAgent())
	}
	if gotWriter == nil {
		t.Error("response writer not carried")
	}
}

func TestWithRequest_IgnoresNil(t *testing.T) {
	// Storing a nil request would make Request() return a non-nil interface
	// holding a nil pointer, which a `!= nil` check passes and a field access
	// then panics on.
	ctx := WithRequest(context.Background(), nil)
	if ctx != context.Background() {
		t.Error("WithRequest(ctx, nil) should return the context unchanged")
	}
	if Request(ctx) != nil {
		t.Error("Request() returned a value after a nil write")
	}
}

func TestWithResponseWriter_IgnoresNil(t *testing.T) {
	ctx := WithResponseWriter(context.Background(), nil)
	if ctx != context.Background() {
		t.Error("WithResponseWriter(ctx, nil) should return the context unchanged")
	}
}

func TestKeysDoNotCollideWithNamedStringKeys(t *testing.T) {
	// The reason the keys are unexported struct types. A named string type
	// with the same underlying value is a different key, so a project's own
	// "request" key cannot be read as this one — nor silently overwrite it.
	type projectKey string

	ctx := WithRequest(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil))
	ctx = context.WithValue(ctx, projectKey("request"), "not a request")

	if Request(ctx) == nil {
		t.Error("a same-named string key displaced the typed key")
	}
	if got, _ := ctx.Value(projectKey("request")).(string); got != "not a request" {
		t.Error("the project's own key was displaced")
	}
}

func TestWithLanguage_RoundTripsAndIgnoresEmpty(t *testing.T) {
	if got := Language(WithLanguage(context.Background(), "sw")); got != "sw" {
		t.Errorf("Language() = %q, want sw", got)
	}

	// An empty tag must not be stored: a stored empty value is
	// indistinguishable from a missing header to anything checking presence.
	ctx := WithLanguage(context.Background(), "")
	if ctx != context.Background() {
		t.Error("WithLanguage(ctx, \"\") should return the context unchanged")
	}
}

func TestLanguageFromRequest(t *testing.T) {
	tests := []struct {
		name, header, want string
	}{
		{"absent header", "", ""},
		{"single tag", "fr", "fr"},
		{"first of several", "fr,en;q=0.8", "fr"},
		{"quality on the first entry", "de;q=0.9,en;q=0.8", "de"},
		{"region subtag kept for the catalog to match", "pt-BR,pt;q=0.9", "pt-BR"},
		{"surrounding whitespace", "  el  , en", "el"},
		{"wildcard means no preference", "*", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/graphql", nil)
			if tt.header != "" {
				r.Header.Set("Accept-Language", tt.header)
			}
			if got := LanguageFromRequest(r); got != tt.want {
				t.Errorf("LanguageFromRequest(%q) = %q, want %q", tt.header, got, tt.want)
			}
		})
	}
}

func TestLanguageFromRequest_NilRequest(t *testing.T) {
	if got := LanguageFromRequest(nil); got != "" {
		t.Errorf("LanguageFromRequest(nil) = %q, want empty", got)
	}
}

func TestLanguageMiddleware(t *testing.T) {
	var seen string
	handler := LanguageMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = Language(r.Context())
	}))

	r := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	r.Header.Set("Accept-Language", "nl,en;q=0.7")
	handler.ServeHTTP(httptest.NewRecorder(), r)
	if seen != "nl" {
		t.Errorf("context language = %q, want nl", seen)
	}

	seen = "unset"
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/graphql", nil))
	if seen != "" {
		t.Errorf("context language without the header = %q, want empty", seen)
	}
}

func TestMiddlewaresCompose(t *testing.T) {
	// Installed together, each still sees what the other wrote.
	var req *http.Request
	var lang string

	handler := Middleware(LanguageMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		req = Request(r.Context())
		lang = Language(r.Context())
	})))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Accept-Language", "it")
	handler.ServeHTTP(httptest.NewRecorder(), r)

	if req == nil {
		t.Error("request lost when composed with LanguageMiddleware")
	}
	if lang != "it" {
		t.Errorf("language = %q, want it", lang)
	}
}
