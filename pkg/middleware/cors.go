package middleware

import (
	"net/http"
	"os"
	"strings"
)

// corsAllowedHeaders is the default Access-Control-Allow-Headers value.
//
// Apollo-Require-Preflight is included because Apollo Client sends it on every
// non-simple request to force a preflight; omitting it from the allow-list
// makes the preflight fail and takes down every GraphQL call from an Apollo
// frontend. Accept and X-Requested-With are common enough that leaving them
// out surprises people for no benefit.
const corsAllowedHeaders = "Content-Type, Authorization, Accept, X-Requested-With, X-Request-ID, Apollo-Require-Preflight"

// corsAllowedMethods is the default Access-Control-Allow-Methods value.
const corsAllowedMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"

// corsExposedHeaders lists the response headers JavaScript may read.
// Set-Cookie is included because cookie-based refresh flows are otherwise
// invisible to the client, and X-Request-ID because RequestID() sets it and
// correlating a failed request to its ID is the main reason to read it.
const corsExposedHeaders = "Set-Cookie, X-Request-ID"

// corsMaxAge is the default preflight cache lifetime, in seconds.
const corsMaxAge = "86400"

// CORS adds Cross-Origin Resource Sharing headers and handles preflight
// requests.
//
// Access-Control-Allow-Origin is a single-value header: the spec allows one
// origin or the literal "*", never a list. The request's Origin is therefore
// matched against allowedOrigins and echoed back when it matches, which is the
// only way to support more than one origin. (A previous implementation joined
// the configured origins with ", " and sent that as the header value; browsers
// reject it, so any project with two or more origins had CORS failing on every
// request.)
//
// The wildcard "*" is honored when it appears in allowedOrigins, but only for
// requests without credentials — the Fetch spec forbids pairing "*" with
// Access-Control-Allow-Credentials: true, and browsers reject the combination.
// When a concrete origin matches, credentials are allowed.
//
// A request whose Origin is not allowed is still served; it simply receives no
// CORS headers, which is what makes the browser block the response. Rejecting
// it outright would break non-browser clients, which send no Origin at all.
func CORS(allowedOrigins []string) Middleware {
	return CORSWith(CORSOptions{AllowedOrigins: allowedOrigins})
}

// CORSOptions configures CORS beyond the origin list.
//
// Every field is optional; an empty one keeps the default. The defaults are
// what most projects want, but not all: a service whose clients send a custom
// header (a tenant id, an API key, a client identifier) must add it to
// AllowedHeaders or every preflight for those requests fails, and there is no
// way to express that through a bare origin list.
type CORSOptions struct {
	// AllowedOrigins is matched against the request Origin and echoed back.
	// "*" is honored for credential-less requests.
	AllowedOrigins []string

	// AllowedMethods defaults to GET, POST, PUT, PATCH, DELETE, OPTIONS.
	AllowedMethods []string

	// AllowedHeaders defaults to the common set plus Apollo-Require-Preflight.
	// Supplying this REPLACES the defaults rather than adding to them, so
	// include the ones you still need.
	AllowedHeaders []string

	// ExposedHeaders defaults to Set-Cookie and X-Request-ID — the two a
	// browser client typically needs to read.
	ExposedHeaders []string

	// MaxAge is the preflight cache lifetime in seconds. Defaults to 86400.
	MaxAge string
}

// CORSWith is CORS with the header policy under the caller's control.
func CORSWith(opts CORSOptions) Middleware {
	methods := corsAllowedMethods
	if len(opts.AllowedMethods) > 0 {
		methods = strings.Join(opts.AllowedMethods, ", ")
	}
	headers := corsAllowedHeaders
	if len(opts.AllowedHeaders) > 0 {
		headers = strings.Join(opts.AllowedHeaders, ", ")
	}
	exposed := corsExposedHeaders
	if len(opts.ExposedHeaders) > 0 {
		exposed = strings.Join(opts.ExposedHeaders, ", ")
	}
	maxAge := corsMaxAge
	if opts.MaxAge != "" {
		maxAge = opts.MaxAge
	}

	origins := make([]string, 0, len(opts.AllowedOrigins))
	wildcard := false
	for _, o := range opts.AllowedOrigins {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if o == "*" {
			wildcard = true
			continue
		}
		origins = append(origins, o)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Vary tells caches the response differs per Origin. Without it a
			// shared cache can serve one origin's Allow-Origin header to another.
			w.Header().Add("Vary", "Origin")

			switch {
			case origin != "" && originAllowed(origins, origin):
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				setCORSCommonHeaders(w, methods, headers, exposed, maxAge)
			case wildcard:
				// No credentials with "*" — the pair is invalid per spec.
				w.Header().Set("Access-Control-Allow-Origin", "*")
				setCORSCommonHeaders(w, methods, headers, exposed, maxAge)
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// originAllowed reports whether origin exactly matches a configured origin.
// Matching is exact rather than prefix- or suffix-based: a suffix match on
// "example.com" would also accept "evil-example.com".
func originAllowed(origins []string, origin string) bool {
	for _, allowed := range origins {
		if allowed == origin {
			return true
		}
	}
	return false
}

// setCORSCommonHeaders writes the headers that are identical whether the
// response echoes a concrete origin or the wildcard.
//
// Expose-Headers lists response headers JavaScript may read. Set-Cookie is
// included because cookie-based refresh flows are otherwise invisible to the
// client, and X-Request-ID because RequestID() sets it and correlating a
// failed request to its ID is the main reason to read it.
func setCORSCommonHeaders(w http.ResponseWriter, methods, headers, exposed, maxAge string) {
	w.Header().Set("Access-Control-Allow-Methods", methods)
	w.Header().Set("Access-Control-Allow-Headers", headers)
	w.Header().Set("Access-Control-Expose-Headers", exposed)
	w.Header().Set("Access-Control-Max-Age", maxAge)
}

// CORSOriginsFromEnv reads the allowed origins from CORS_ALLOWED_ORIGINS, a
// comma-separated list.
//
// Provided because every service that configures CORS from the environment
// writes the same five lines otherwise, and because the fallback matters: with
// no value set this returns the two localhost ports a web frontend uses in
// development, so a fresh checkout works without configuration. A deployment
// that sets nothing therefore allows only localhost — which fails closed
// rather than open.
//
// Projects whose origins come from config.yaml should use
// cfg.Server.AllowedOrigins instead; this is for the ones reading the
// environment directly.
func CORSOriginsFromEnv() []string {
	raw := os.Getenv("CORS_ALLOWED_ORIGINS")
	if strings.TrimSpace(raw) == "" {
		return []string{"http://localhost:3000", "http://localhost:3001"}
	}

	origins := make([]string, 0, strings.Count(raw, ",")+1)
	for _, o := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}
