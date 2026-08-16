package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gofastadev/gofasta/pkg/httputil"
	"github.com/gofastadev/gofasta/pkg/middleware"
)

// CredentialResolver turns the credential a client presented into a token this
// service can validate.
//
// It exists for reference tokens (RFC 6749 §1.4, RFC 7662): the client holds an
// opaque handle, and the JWT it stands for is looked up server-side — in Redis,
// in a session table, at an introspection endpoint. The handle is worthless if
// leaked and can be revoked without waiting for an expiry, which a
// self-contained JWT cannot.
//
// Return an error when the handle is unknown, expired or revoked; the request
// is then rejected exactly as a bad token would be. Without a resolver the
// presented credential is validated directly, which is the self-contained-JWT
// case and the default.
type CredentialResolver func(ctx context.Context, presented string) (string, error)

type middlewareOptions struct {
	resolve CredentialResolver
	cookies []string
}

// MiddlewareOption configures [JWTAuth], [OptionalJWTAuth] and [GraphQLAuth].
type MiddlewareOption func(*middlewareOptions)

// WithCredentialResolver installs a [CredentialResolver], making the presented
// credential an opaque reference rather than the token itself.
func WithCredentialResolver(resolve CredentialResolver) MiddlewareOption {
	return func(o *middlewareOptions) { o.resolve = resolve }
}

// WithCookieNames also accepts the credential from these cookies, in order,
// when the Authorization header is absent.
//
// A browser app that keeps its token in an httpOnly cookie cannot read it back
// out to build an Authorization header — the whole point of httpOnly is that
// script cannot touch it. Without this the same token works from a server-side
// caller and fails from the browser.
func WithCookieNames(names ...string) MiddlewareOption {
	return func(o *middlewareOptions) { o.cookies = append(o.cookies, names...) }
}

func newOptions(opts []MiddlewareOption) *middlewareOptions {
	o := &middlewareOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}
	return o
}

// JWTAuth creates a middleware that validates the presented credential.
// On success, stores *Claims and the validated token in the request context.
// On failure, returns 401 Unauthorized.
func JWTAuth(jwtService *JWTService, opts ...MiddlewareOption) middleware.Middleware {
	o := newOptions(opts)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			presented, err := credentialFrom(r, o)
			if err != nil {
				_ = httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
				return
			}

			ctx, ok := authenticate(r.Context(), jwtService, o, presented)
			if !ok {
				_ = httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
				return
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalJWTAuth authenticates when a usable credential is presented and lets
// the request through untouched when it is not.
//
// For endpoints that serve everyone but serve signed-in callers better — a
// public course page that also shows your enrolment. Handlers behind it must
// treat missing claims as anonymous rather than as an error.
func OptionalJWTAuth(jwtService *JWTService, opts ...MiddlewareOption) middleware.Middleware {
	o := newOptions(opts)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if presented, err := credentialFrom(r, o); err == nil {
				if ctx, ok := authenticate(r.Context(), jwtService, o, presented); ok {
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// authenticate resolves, validates, and returns the enriched context. The bool
// is false when the request must not be treated as authenticated.
func authenticate(ctx context.Context, jwtService *JWTService, o *middlewareOptions, presented string) (context.Context, bool) {
	token := presented
	if o.resolve != nil {
		resolved, err := o.resolve(ctx, presented)
		if err != nil || resolved == "" {
			return ctx, false
		}
		token = resolved
	}

	claims, err := jwtService.ValidateToken(token)
	if err != nil {
		return ctx, false
	}

	ctx = context.WithValue(ctx, ClaimsKey, claims)
	return WithToken(ctx, token), true
}

// credentialFrom pulls the presented credential off the request, preferring the
// Authorization header and falling back to the configured cookies.
func credentialFrom(r *http.Request, o *middlewareOptions) (string, error) {
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			return "", errInvalidAuthFormat
		}
		return token, nil
	}

	for _, name := range o.cookies {
		if cookie, err := r.Cookie(name); err == nil && cookie.Value != "" {
			return cookie.Value, nil
		}
	}
	return "", errAuthRequired
}

type authError string

func (e authError) Error() string { return string(e) }

const (
	errAuthRequired      authError = "authentication required"
	errInvalidAuthFormat authError = "invalid authorization format, expected: Bearer <token>"
)

// GraphQLAuth requires authentication for every operation except those named in
// publicOps, which get optional authentication instead.
//
// GraphQL puts every operation behind one URL, so path-based rules cannot
// express "login is public but everything else is not" — the operation name in
// the request body is the only thing that distinguishes them.
//
// publicOps is an allowlist, so an operation nobody remembered to classify is
// protected rather than exposed. A request whose body does not parse is
// likewise treated as non-public: an unreadable body must not be a way to skip
// the check.
func GraphQLAuth(jwtService *JWTService, publicOps map[string]bool, opts ...MiddlewareOption) middleware.Middleware {
	required := JWTAuth(jwtService, opts...)
	optional := OptionalJWTAuth(jwtService, opts...)

	return func(next http.Handler) http.Handler {
		protected, public := required(next), optional(next)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// The body has to be read to find the operation name, and read
			// again by the GraphQL handler, so it is restored either way.
			body, err := io.ReadAll(r.Body)
			if err != nil {
				_ = httputil.JSON(w, http.StatusBadRequest,
					map[string]string{"error": "failed to read request body"})
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			if publicOps[operationNameFrom(body)] {
				public.ServeHTTP(w, r)
				return
			}
			protected.ServeHTTP(w, r)
		})
	}
}

// operationNameFrom returns the operation this request invokes, or "" when the
// body is not a GraphQL request this code can read.
func operationNameFrom(body []byte) string {
	var req struct {
		Query         string `json:"query"`
		OperationName string `json:"operationName"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}
	if req.OperationName != "" {
		return req.OperationName
	}
	return ExtractOperationName(req.Query)
}

// ExtractOperationName returns the operation name declared in a GraphQL query
// string, for clients that do not send operationName alongside it.
//
// It reads the name in `query Name(...)` / `mutation Name {...}` and returns ""
// for anything else — an anonymous operation, a subscription, a query it cannot
// parse. Since the caller uses the result to look up an allowlist, "" fails
// closed.
func ExtractOperationName(query string) string {
	query = strings.TrimSpace(query)

	for _, keyword := range []string{"mutation ", "query "} {
		idx := strings.Index(query, keyword)
		if idx == -1 {
			continue
		}
		remaining := query[idx+len(keyword):]
		end := strings.IndexAny(remaining, "({")
		if end == -1 {
			return ""
		}
		return strings.TrimSpace(remaining[:end])
	}
	return ""
}
