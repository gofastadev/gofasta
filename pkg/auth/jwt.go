package auth

import (
	"fmt"
	"time"

	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/golang-jwt/jwt/v5"
)

// JWTService handles token generation and validation.
type JWTService struct {
	secret        []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
	issuer        string
}

// NewJWTService builds a JWTService from an AuthConfig.
func NewJWTService(cfg *config.AuthConfig) *JWTService {
	return &JWTService{
		secret:        []byte(cfg.JWTSecret),
		accessExpiry:  cfg.AccessTokenExpiry,
		refreshExpiry: cfg.RefreshTokenExpiry,
		issuer:        cfg.Issuer,
	}
}

// Issuer returns the issuer this service stamps and requires, or "" when it is
// not configured to check one.
func (s *JWTService) Issuer() string { return s.issuer }

// GenerateToken creates a signed access token for a user.
func (s *JWTService) GenerateToken(userID, role string) (string, error) {
	return s.GenerateTokenWithRoles(userID, role, nil)
}

// GenerateTokenWithRoles creates a signed access token carrying several roles.
//
// role and roles are both stamped and both honored on the way back in, so a
// service can grant a primary role and additional ones without the caller
// having to know which field a given check reads.
func (s *JWTService) GenerateTokenWithRoles(userID, role string, roles []string) (string, error) {
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    s.issuer,
			Subject:   userID,
		},
		UserID: userID,
		Role:   role,
		Roles:  roles,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// GenerateRefreshToken creates a long-lived refresh token.
func (s *JWTService) GenerateRefreshToken(userID string) (string, error) {
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    s.issuer,
			Subject:   userID,
		},
		UserID: userID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// ValidateToken parses and validates a token string, returning the claims.
//
// Signature, expiry and — when [config.AuthConfig].Issuer is set — the issuer
// are all checked here. Expiry is enforced by the library against the embedded
// RegisteredClaims, which is why [Claims] must never shadow `exp`.
func (s *JWTService) ValidateToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	opts := []jwt.ParserOption{jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()})}
	if s.issuer != "" {
		opts = append(opts, jwt.WithIssuer(s.issuer))
	}

	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	return claims, nil
}

// RefreshToken validates a refresh token and issues a new access token.
func (s *JWTService) RefreshToken(refreshTokenStr string) (string, error) {
	claims, err := s.ValidateToken(refreshTokenStr)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}
	return s.GenerateToken(claims.UserID, claims.Role)
}
