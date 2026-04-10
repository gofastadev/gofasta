package auth

import (
	"context"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

// ClaimsKey is the context key under which JWT Claims are stored.
const ClaimsKey contextKey = "claims"

// Claims holds JWT token payload data.
type Claims struct {
	jwt.RegisteredClaims
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// ClaimsFromContext extracts Claims from the request context.
// Returns error if no claims are present (unauthenticated request).
func ClaimsFromContext(ctx context.Context) (*Claims, error) {
	claims, ok := ctx.Value(ClaimsKey).(*Claims)
	if !ok || claims == nil {
		return nil, fmt.Errorf("no claims in context")
	}
	return claims, nil
}
