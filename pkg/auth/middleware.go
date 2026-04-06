package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/healtronlabs/gofasta/pkg/httputil"
	"github.com/healtronlabs/gofasta/pkg/middleware"
)

// JWTAuth creates a middleware that validates JWT Bearer tokens.
// On success, stores *Claims in the request context.
// On failure, returns 401 Unauthorized.
func JWTAuth(jwtService *JWTService) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == authHeader {
				httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid authorization format, expected: Bearer <token>"})
				return
			}

			claims, err := jwtService.ValidateToken(token)
			if err != nil {
				httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
