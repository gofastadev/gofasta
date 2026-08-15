package auth

import (
	"net/http"
	"time"

	"github.com/gofastadev/gofasta/pkg/config"
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
