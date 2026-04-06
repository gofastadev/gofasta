package auth

import (
	"net/http"

	"github.com/gofastadev/gofasta/pkg/httputil"
	"github.com/gofastadev/gofasta/pkg/middleware"
)

// RequireRole creates a middleware that checks if the authenticated user has one of the allowed roles.
// Must be used after JWTAuth middleware (requires Claims in context).
func RequireRole(roles ...string) middleware.Middleware {
	roleSet := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := ClaimsFromContext(r.Context())
			if err != nil {
				httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
				return
			}
			if !roleSet[claims.Role] {
				httputil.JSON(w, http.StatusForbidden, map[string]string{"error": "insufficient permissions"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission creates a middleware that checks Casbin policies for the user's role.
// Must be used after JWTAuth middleware.
func RequirePermission(rbac *RBACService, resource, action string) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := ClaimsFromContext(r.Context())
			if err != nil {
				httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
				return
			}
			allowed, err := rbac.Enforce(claims.Role, resource, action)
			if err != nil || !allowed {
				httputil.JSON(w, http.StatusForbidden, map[string]string{"error": "permission denied"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
