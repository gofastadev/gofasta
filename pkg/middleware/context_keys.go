package middleware

type contextKey string

// Context keys used by the middleware stack.
const (
	RequestIDKey  contextKey = "requestID"
	UserClaimsKey contextKey = "userClaims"
)
