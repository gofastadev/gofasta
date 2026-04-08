package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- helpers ----------

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

var noopHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// ---------- ClaimsFromContext ----------

func TestClaimsFromContext_WithClaims(t *testing.T) {
	claims := &Claims{UserID: "user-123", Role: "admin"}
	ctx := context.WithValue(context.Background(), ClaimsKey, claims)

	got, err := ClaimsFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, "user-123", got.UserID)
	assert.Equal(t, "admin", got.Role)
}

func TestClaimsFromContext_NoClaims(t *testing.T) {
	ctx := context.Background()

	got, err := ClaimsFromContext(ctx)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "no claims in context")
}

func TestClaimsFromContext_NilClaims(t *testing.T) {
	ctx := context.WithValue(context.Background(), ClaimsKey, (*Claims)(nil))

	got, err := ClaimsFromContext(ctx)
	require.Error(t, err)
	assert.Nil(t, got)
}

func TestClaimsFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), ClaimsKey, "not-claims")

	got, err := ClaimsFromContext(ctx)
	require.Error(t, err)
	assert.Nil(t, got)
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

// ---------- JWTAuth middleware ----------

func TestJWTAuth_ValidToken(t *testing.T) {
	svc := testJWTService()
	token, _ := svc.GenerateToken("user-1", "admin")

	var gotClaims *Claims
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotClaims, _ = ClaimsFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := JWTAuth(svc)(inner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, gotClaims)
	assert.Equal(t, "user-1", gotClaims.UserID)
	assert.Equal(t, "admin", gotClaims.Role)
}

func TestJWTAuth_MissingAuthHeader(t *testing.T) {
	svc := testJWTService()
	handler := JWTAuth(svc)(noopHandler)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var body map[string]string
	json.Unmarshal(rec.Body.Bytes(), &body)
	assert.Equal(t, "authentication required", body["error"])
}

func TestJWTAuth_InvalidFormat_NoBearerPrefix(t *testing.T) {
	svc := testJWTService()
	handler := JWTAuth(svc)(noopHandler)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Basic sometoken")
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var body map[string]string
	json.Unmarshal(rec.Body.Bytes(), &body)
	assert.Contains(t, body["error"], "invalid authorization format")
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	svc := testJWTService()
	handler := JWTAuth(svc)(noopHandler)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-string")
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var body map[string]string
	json.Unmarshal(rec.Body.Bytes(), &body)
	assert.Equal(t, "invalid or expired token", body["error"])
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	svc := expiredJWTService()
	token, _ := svc.GenerateToken("user-1", "admin")

	// Use a valid service to verify the expired token
	validSvc := testJWTService()
	handler := JWTAuth(validSvc)(noopHandler)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuth_TokenFromDifferentSecret(t *testing.T) {
	svc1 := testJWTService()
	svc2 := NewJWTService(&config.AuthConfig{
		JWTSecret:          "completely-different-secret",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 24 * time.Hour,
	})

	token, _ := svc2.GenerateToken("user-1", "admin")
	handler := JWTAuth(svc1)(noopHandler)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuth_DoesNotCallNextOnFailure(t *testing.T) {
	svc := testJWTService()
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := JWTAuth(svc)(inner)

	// No auth header
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	handler.ServeHTTP(rec, req)

	assert.False(t, called)
}

func TestJWTAuth_BearerOnlyPrefix(t *testing.T) {
	svc := testJWTService()
	handler := JWTAuth(svc)(noopHandler)

	// "Bearer" without a space and token
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer")
	handler.ServeHTTP(rec, req)

	// "Bearer" without space is treated as no Bearer prefix
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ---------- Claims key ----------

func TestClaimsKey_Value(t *testing.T) {
	assert.Equal(t, contextKey("claims"), ClaimsKey)
}

// ---------- Token round-trip ----------

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
