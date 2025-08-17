package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/healtronlabs/gofasta/packages/core"
)

// JWTService provides JWT token management
type JWTService struct {
	secretKey       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	issuer          string
}

// JWTConfig represents JWT configuration
type JWTConfig struct {
	SecretKey       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	Issuer          string
}

// NewJWTService creates a new JWT service
func NewJWTService(config *JWTConfig) *JWTService {
	if config.AccessTokenTTL == 0 {
		config.AccessTokenTTL = 15 * time.Minute
	}
	if config.RefreshTokenTTL == 0 {
		config.RefreshTokenTTL = 24 * time.Hour * 7 // 7 days
	}
	if config.Issuer == "" {
		config.Issuer = "gofasta"
	}

	return &JWTService{
		secretKey:       []byte(config.SecretKey),
		accessTokenTTL:  config.AccessTokenTTL,
		refreshTokenTTL: config.RefreshTokenTTL,
		issuer:          config.Issuer,
	}
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// Claims represents JWT claims
type Claims struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

// GenerateTokenPair generates access and refresh tokens for a user
func (s *JWTService) GenerateTokenPair(userID, email string, roles []string) (*TokenPair, error) {
	now := time.Now()
	
	// Generate access token
	accessClaims := &Claims{
		UserID: userID,
		Email:  email,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.issuer,
			Subject:   userID,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.secretKey)
	if err != nil {
		return nil, core.NewInternalServerException("Failed to generate access token", err)
	}

	// Generate refresh token
	refreshClaims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.issuer,
			Subject:   userID,
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.secretKey)
	if err != nil {
		return nil, core.NewInternalServerException("Failed to generate refresh token", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    now.Add(s.accessTokenTTL),
		TokenType:    "Bearer",
	}, nil
}

// ValidateToken validates a JWT token and returns claims
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, core.NewUnauthorizedException("Invalid token signing method")
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, core.NewUnauthorizedException("Invalid token")
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, core.NewUnauthorizedException("Invalid token claims")
}

// RefreshToken generates a new access token using a refresh token
func (s *JWTService) RefreshToken(refreshTokenString string) (*TokenPair, error) {
	claims, err := s.ValidateToken(refreshTokenString)
	if err != nil {
		return nil, core.NewUnauthorizedException("Invalid refresh token")
	}

	// Generate new token pair
	return s.GenerateTokenPair(claims.UserID, claims.Email, claims.Roles)
}

// ExtractTokenFromBearer extracts token from Bearer authorization header
func (s *JWTService) ExtractTokenFromBearer(authHeader string) string {
	const bearerPrefix = "Bearer "
	if len(authHeader) > len(bearerPrefix) && authHeader[:len(bearerPrefix)] == bearerPrefix {
		return authHeader[len(bearerPrefix):]
	}
	return ""
}

// GetUserFromToken extracts user information from a valid token
func (s *JWTService) GetUserFromToken(tokenString string) (*UserInfo, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	return &UserInfo{
		ID:    claims.UserID,
		Email: claims.Email,
		Roles: claims.Roles,
	}, nil
}

// UserInfo represents user information extracted from token
type UserInfo struct {
	ID    string   `json:"id"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

// HasRole checks if user has a specific role
func (u *UserInfo) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole checks if user has any of the specified roles
func (u *UserInfo) HasAnyRole(roles ...string) bool {
	for _, requiredRole := range roles {
		if u.HasRole(requiredRole) {
			return true
		}
	}
	return false
}

// HasAllRoles checks if user has all of the specified roles
func (u *UserInfo) HasAllRoles(roles ...string) bool {
	for _, requiredRole := range roles {
		if !u.HasRole(requiredRole) {
			return false
		}
	}
	return true
}