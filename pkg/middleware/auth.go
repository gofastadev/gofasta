package middleware

import (
	"context"
	"net/http"
	"strings"
)

// Auth is a skeleton authentication middleware. It extracts the Authorization
// header, validates the token (placeholder logic), and stores claims in context.
// Implement actual JWT/OAuth validation per your project requirements.
func Auth() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == authHeader {
				http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
				return
			}

			// TODO: Replace with actual token validation (JWT, OAuth, etc.)
			// claims, err := validateToken(token)
			// if err != nil {
			//     http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			//     return
			// }

			ctx := context.WithValue(r.Context(), UserClaimsKey, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
