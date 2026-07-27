package middleware

import (
	"net/http"
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
	origins := make([]string, 0, len(allowedOrigins))
	wildcard := false
	for _, o := range allowedOrigins {
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
				setCORSCommonHeaders(w)
			case wildcard:
				// No credentials with "*" — the pair is invalid per spec.
				w.Header().Set("Access-Control-Allow-Origin", "*")
				setCORSCommonHeaders(w)
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
func setCORSCommonHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Methods", corsAllowedMethods)
	w.Header().Set("Access-Control-Allow-Headers", corsAllowedHeaders)
	w.Header().Set("Access-Control-Expose-Headers", "Set-Cookie, X-Request-ID")
	w.Header().Set("Access-Control-Max-Age", "86400")
}
