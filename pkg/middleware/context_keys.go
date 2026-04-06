package middleware

type contextKey string

const (
	RequestIDKey  contextKey = "requestID"
	UserClaimsKey contextKey = "userClaims"
)
