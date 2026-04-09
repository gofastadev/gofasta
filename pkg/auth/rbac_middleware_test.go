package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- RequireRole ----------

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name           string
		allowedRoles   []string
		claims         *Claims
		noClaims       bool
		wantStatusCode int
		wantError      string
	}{
		{
			name:           "allowed role",
			allowedRoles:   []string{"admin", "editor"},
			claims:         &Claims{UserID: "user-1", Role: "admin"},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "second allowed role",
			allowedRoles:   []string{"admin", "editor"},
			claims:         &Claims{UserID: "user-2", Role: "editor"},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "denied role",
			allowedRoles:   []string{"admin"},
			claims:         &Claims{UserID: "user-3", Role: "viewer"},
			wantStatusCode: http.StatusForbidden,
			wantError:      "insufficient permissions",
		},
		{
			name:           "no claims in context",
			allowedRoles:   []string{"admin"},
			noClaims:       true,
			wantStatusCode: http.StatusUnauthorized,
			wantError:      "authentication required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mw := RequireRole(tc.allowedRoles...)
			handler := mw(noopHandler)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/resource", nil)

			if !tc.noClaims {
				ctx := context.WithValue(req.Context(), ClaimsKey, tc.claims)
				req = req.WithContext(ctx)
			}

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatusCode, rec.Code)

			if tc.wantError != "" {
				var body map[string]string
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Equal(t, tc.wantError, body["error"])
			}
		})
	}
}

// ---------- RequirePermission ----------

func TestRequirePermission(t *testing.T) {
	cfg := writeRBACFiles(t)
	svc, err := NewRBACService(cfg)
	require.NoError(t, err)

	tests := []struct {
		name           string
		resource       string
		action         string
		claims         *Claims
		noClaims       bool
		wantStatusCode int
		wantError      string
	}{
		{
			name:           "allowed permission",
			resource:       "/users",
			action:         "read",
			claims:         &Claims{UserID: "user-1", Role: "admin"},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "denied permission - wrong role",
			resource:       "/users",
			action:         "write",
			claims:         &Claims{UserID: "user-2", Role: "editor"},
			wantStatusCode: http.StatusForbidden,
			wantError:      "permission denied",
		},
		{
			name:           "denied permission - unknown role",
			resource:       "/users",
			action:         "read",
			claims:         &Claims{UserID: "user-3", Role: "viewer"},
			wantStatusCode: http.StatusForbidden,
			wantError:      "permission denied",
		},
		{
			name:           "no claims in context",
			resource:       "/users",
			action:         "read",
			noClaims:       true,
			wantStatusCode: http.StatusUnauthorized,
			wantError:      "authentication required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mw := RequirePermission(svc, tc.resource, tc.action)
			handler := mw(noopHandler)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/resource", nil)

			if !tc.noClaims {
				ctx := context.WithValue(req.Context(), ClaimsKey, tc.claims)
				req = req.WithContext(ctx)
			}

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatusCode, rec.Code)

			if tc.wantError != "" {
				var body map[string]string
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Equal(t, tc.wantError, body["error"])
			}
		})
	}
}

func TestRequirePermission_NextHandlerCalled(t *testing.T) {
	cfg := writeRBACFiles(t)
	svc, err := NewRBACService(cfg)
	require.NoError(t, err)

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	mw := RequirePermission(svc, "/users", "read")
	handler := mw(inner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	ctx := context.WithValue(req.Context(), ClaimsKey, &Claims{UserID: "user-1", Role: "admin"})
	req = req.WithContext(ctx)

	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequirePermission_NextHandlerNotCalledOnDenied(t *testing.T) {
	cfg := writeRBACFiles(t)
	svc, err := NewRBACService(cfg)
	require.NoError(t, err)

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	mw := RequirePermission(svc, "/users", "read")
	handler := mw(inner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	ctx := context.WithValue(req.Context(), ClaimsKey, &Claims{UserID: "user-2", Role: "viewer"})
	req = req.WithContext(ctx)

	handler.ServeHTTP(rec, req)

	assert.False(t, called)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
