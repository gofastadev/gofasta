package auth

import (
	"testing"
	"time"

	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testJWTService() *JWTService {
	return NewJWTService(&config.AuthConfig{
		JWTSecret:          "test-secret-key-for-testing",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 24 * time.Hour,
	})
}

func expiredJWTService() *JWTService {
	return NewJWTService(&config.AuthConfig{
		JWTSecret:          "test-secret-key-for-testing",
		AccessTokenExpiry:  -1 * time.Second, // already expired
		RefreshTokenExpiry: -1 * time.Second,
	})
}

// ---------- NewJWTService ----------

func TestNewJWTService(t *testing.T) {
	svc := testJWTService()
	assert.NotNil(t, svc)
	assert.Equal(t, []byte("test-secret-key-for-testing"), svc.secret)
	assert.Equal(t, 15*time.Minute, svc.accessExpiry)
	assert.Equal(t, 24*time.Hour, svc.refreshExpiry)
}

// ---------- GenerateToken ----------

func TestGenerateToken_Success(t *testing.T) {
	svc := testJWTService()

	token, err := svc.GenerateToken("user-1", "admin")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestGenerateToken_DifferentUsersProduceDifferentTokens(t *testing.T) {
	svc := testJWTService()

	t1, _ := svc.GenerateToken("user-1", "admin")
	t2, _ := svc.GenerateToken("user-2", "admin")
	assert.NotEqual(t, t1, t2)
}

// ---------- ValidateToken ----------

func TestValidateToken_Valid(t *testing.T) {
	svc := testJWTService()

	token, err := svc.GenerateToken("user-1", "editor")
	require.NoError(t, err)

	claims, err := svc.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, "editor", claims.Role)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
}

func TestValidateToken_Expired(t *testing.T) {
	svc := expiredJWTService()

	token, err := svc.GenerateToken("user-1", "admin")
	require.NoError(t, err)

	_, err = svc.ValidateToken(token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestValidateToken_InvalidString(t *testing.T) {
	svc := testJWTService()

	_, err := svc.ValidateToken("not-a-valid-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestValidateToken_WrongSecret(t *testing.T) {
	svc1 := testJWTService()
	svc2 := NewJWTService(&config.AuthConfig{
		JWTSecret:          "different-secret-key",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 24 * time.Hour,
	})

	token, err := svc1.GenerateToken("user-1", "admin")
	require.NoError(t, err)

	_, err = svc2.ValidateToken(token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestValidateToken_EmptyString(t *testing.T) {
	svc := testJWTService()

	_, err := svc.ValidateToken("")
	require.Error(t, err)
}

// ---------- GenerateRefreshToken ----------

func TestGenerateRefreshToken_Success(t *testing.T) {
	svc := testJWTService()

	token, err := svc.GenerateRefreshToken("user-1")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := svc.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Empty(t, claims.Role) // refresh tokens don't include role
}

// ---------- RefreshToken ----------

func TestRefreshToken_Success(t *testing.T) {
	svc := testJWTService()

	refreshToken, err := svc.GenerateRefreshToken("user-1")
	require.NoError(t, err)

	newAccessToken, err := svc.RefreshToken(refreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newAccessToken)

	// Validate the new access token
	claims, err := svc.ValidateToken(newAccessToken)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
}

func TestRefreshToken_InvalidRefreshToken(t *testing.T) {
	svc := testJWTService()

	_, err := svc.RefreshToken("invalid-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid refresh token")
}

func TestRefreshToken_ExpiredRefreshToken(t *testing.T) {
	svc := expiredJWTService()

	token, err := svc.GenerateRefreshToken("user-1")
	require.NoError(t, err)

	_, err = svc.RefreshToken(token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid refresh token")
}

// ---------- Token round-trip ----------

func TestValidateToken_UnexpectedSigningMethod(t *testing.T) {
	svc := testJWTService()
	// Create a token signed with the "none" method (non-HMAC)
	token := jwt.NewWithClaims(jwt.SigningMethodNone, &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		UserID: "user-1",
		Role:   "admin",
	})
	tokenStr, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = svc.ValidateToken(tokenStr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestTokenRoundTrip(t *testing.T) {
	svc := testJWTService()

	// Generate access token
	accessToken, err := svc.GenerateToken("user-42", "moderator")
	require.NoError(t, err)

	// Validate it
	claims, err := svc.ValidateToken(accessToken)
	require.NoError(t, err)
	assert.Equal(t, "user-42", claims.UserID)
	assert.Equal(t, "moderator", claims.Role)

	// Generate refresh token
	refreshToken, err := svc.GenerateRefreshToken("user-42")
	require.NoError(t, err)

	// Use refresh to get new access token
	newAccess, err := svc.RefreshToken(refreshToken)
	require.NoError(t, err)

	// Validate the new access token
	newClaims, err := svc.ValidateToken(newAccess)
	require.NoError(t, err)
	assert.Equal(t, "user-42", newClaims.UserID)
}
