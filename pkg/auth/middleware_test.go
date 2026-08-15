package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var noopHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

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
