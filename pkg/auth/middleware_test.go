package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
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
	assert.Equal(t, "user-1", gotClaims.SubjectID())
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

// ---------- CredentialResolver ----------

// referenceStore stands in for the Redis lookup a reference-token deployment
// does: an opaque handle in, the real JWT out.
func referenceStore(t *testing.T, handle, token string) CredentialResolver {
	t.Helper()
	return func(_ context.Context, presented string) (string, error) {
		if presented != handle {
			return "", errors.New("unknown credential")
		}
		return token, nil
	}
}

func TestJWTAuth_ResolvesAnOpaqueReferenceToItsToken(t *testing.T) {
	svc := testJWTService()
	token, err := svc.GenerateToken("user-1", "admin")
	require.NoError(t, err)

	var gotClaims *Claims
	var gotToken string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotClaims, _ = ClaimsFromContext(r.Context())
		gotToken = TokenFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := JWTAuth(svc, WithCredentialResolver(referenceStore(t, "opaque-handle", token)))(inner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer opaque-handle")
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, gotClaims)
	assert.Equal(t, "user-1", gotClaims.SubjectID())
	// The resolved token, not the handle — this is what gets forwarded onward.
	assert.Equal(t, token, gotToken)
}

func TestJWTAuth_UnknownReferenceIsRejected(t *testing.T) {
	// A handle that has been revoked or has expired server-side no longer
	// resolves, and the request must fail exactly as a bad token does. This is
	// the property a self-contained JWT cannot offer.
	svc := testJWTService()
	token, _ := svc.GenerateToken("user-1", "admin")

	called := false
	inner := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	handler := JWTAuth(svc, WithCredentialResolver(referenceStore(t, "live-handle", token)))(inner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer revoked-handle")
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, called)
}

func TestJWTAuth_ResolverReturningEmptyIsRejected(t *testing.T) {
	// A store that misses without erroring would otherwise hand "" to the
	// validator and rely on it to object.
	svc := testJWTService()
	handler := JWTAuth(svc, WithCredentialResolver(
		func(context.Context, string) (string, error) { return "", nil },
	))(noopHandler)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer anything")
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuth_WithoutResolverValidatesThePresentedToken(t *testing.T) {
	// The default path is unchanged: no resolver means the credential IS the
	// token.
	svc := testJWTService()
	token, _ := svc.GenerateToken("user-1", "admin")

	handler := JWTAuth(svc)(noopHandler)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// ---------- cookies ----------

func TestJWTAuth_FallsBackToCookie(t *testing.T) {
	// A browser app keeping its token in an httpOnly cookie cannot read it back
	// out to build an Authorization header; without this the same token works
	// server-side and fails from the browser.
	svc := testJWTService()
	token, _ := svc.GenerateToken("user-1", "admin")

	handler := JWTAuth(svc, WithCookieNames("app_access_token"))(noopHandler)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "app_access_token", Value: token})
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestJWTAuth_HeaderWinsOverCookie(t *testing.T) {
	// A stale cookie must not silently override the credential the caller
	// deliberately sent.
	svc := testJWTService()
	header, _ := svc.GenerateToken("from-header", "admin")

	var got *Claims
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = ClaimsFromContext(r.Context())
	})
	handler := JWTAuth(svc, WithCookieNames("app_access_token"))(inner)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+header)
	req.AddCookie(&http.Cookie{Name: "app_access_token", Value: "stale"})
	handler.ServeHTTP(httptest.NewRecorder(), req)

	require.NotNil(t, got)
	assert.Equal(t, "from-header", got.SubjectID())
}

func TestJWTAuth_NoCookieNamesConfiguredIgnoresCookies(t *testing.T) {
	svc := testJWTService()
	token, _ := svc.GenerateToken("user-1", "admin")

	handler := JWTAuth(svc)(noopHandler)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "app_access_token", Value: token})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ---------- OptionalJWTAuth ----------

func TestOptionalJWTAuth_PassesAnonymousAndAuthenticatedAlike(t *testing.T) {
	svc := testJWTService()
	token, _ := svc.GenerateToken("user-1", "admin")

	var claims *Claims
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, _ = ClaimsFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	handler := OptionalJWTAuth(svc)(inner)

	// Anonymous: through, no claims.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/public", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Nil(t, claims)

	// A bad token: still through, still anonymous — never a 401.
	rec = httptest.NewRecorder()
	bad := httptest.NewRequest(http.MethodGet, "/public", nil)
	bad.Header.Set("Authorization", "Bearer garbage")
	handler.ServeHTTP(rec, bad)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Nil(t, claims)

	// A good token: through, with claims.
	rec = httptest.NewRecorder()
	good := httptest.NewRequest(http.MethodGet, "/public", nil)
	good.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(rec, good)
	assert.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, claims)
	assert.Equal(t, "user-1", claims.SubjectID())
}

// ---------- GraphQLAuth ----------

func graphQLRequest(t *testing.T, body map[string]any) *http.Request {
	t.Helper()
	encoded, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(encoded))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestGraphQLAuth_PublicOperationNeedsNoCredential(t *testing.T) {
	svc := testJWTService()
	handler := GraphQLAuth(svc, map[string]bool{"Login": true})(noopHandler)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, graphQLRequest(t, map[string]any{"operationName": "Login"}))

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGraphQLAuth_UnlistedOperationRequiresCredential(t *testing.T) {
	// The allowlist means an operation nobody classified is protected rather
	// than exposed — the safe direction for the mistake that will happen.
	svc := testJWTService()
	handler := GraphQLAuth(svc, map[string]bool{"Login": true})(noopHandler)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, graphQLRequest(t, map[string]any{"operationName": "DeleteEverything"}))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGraphQLAuth_FallsBackToTheOperationNameInTheQuery(t *testing.T) {
	// Clients that do not send operationName alongside the query still have to
	// be classified correctly, or every one of them is treated as unlisted.
	svc := testJWTService()
	handler := GraphQLAuth(svc, map[string]bool{"Login": true})(noopHandler)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, graphQLRequest(t, map[string]any{
		"query": "mutation Login($e: String!) { login(email: $e) { token } }",
	}))

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGraphQLAuth_UnparseableBodyIsTreatedAsProtected(t *testing.T) {
	// An unreadable body must not become a way to skip the check.
	svc := testJWTService()
	handler := GraphQLAuth(svc, map[string]bool{"Login": true})(noopHandler)

	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader([]byte("{not json")))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGraphQLAuth_BodyStaysReadableForTheHandler(t *testing.T) {
	// The middleware consumes the body to find the operation name. If it does
	// not restore it, every GraphQL request arrives at the resolver empty.
	svc := testJWTService()

	var seen string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		seen = string(body)
		w.WriteHeader(http.StatusOK)
	})
	handler := GraphQLAuth(svc, map[string]bool{"Login": true})(inner)

	handler.ServeHTTP(httptest.NewRecorder(), graphQLRequest(t, map[string]any{
		"operationName": "Login",
		"variables":     map[string]any{"email": "a@b.c"},
	}))

	assert.Contains(t, seen, "a@b.c", "handler received an emptied body")
}

func TestGraphQLAuth_PublicOperationStillAuthenticatesWhenItCan(t *testing.T) {
	// "Public" means "no credential required", not "credential ignored" — a
	// public listing that highlights your own entries needs the claims.
	svc := testJWTService()
	token, _ := svc.GenerateToken("user-1", "learner")

	var claims *Claims
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, _ = ClaimsFromContext(r.Context())
	})
	handler := GraphQLAuth(svc, map[string]bool{"Courses": true})(inner)

	req := graphQLRequest(t, map[string]any{"operationName": "Courses"})
	req.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	require.NotNil(t, claims)
	assert.Equal(t, "user-1", claims.SubjectID())
}

func TestGraphQLAuth_ResolverAppliesToBothPaths(t *testing.T) {
	svc := testJWTService()
	token, _ := svc.GenerateToken("user-1", "admin")
	opt := WithCredentialResolver(referenceStore(t, "handle", token))

	handler := GraphQLAuth(svc, map[string]bool{"Login": true}, opt)(noopHandler)

	protected := graphQLRequest(t, map[string]any{"operationName": "Me"})
	protected.Header.Set("Authorization", "Bearer handle")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, protected)
	assert.Equal(t, http.StatusOK, rec.Code, "resolver not applied on the protected path")
}

// ---------- ExtractOperationName ----------

func TestExtractOperationName(t *testing.T) {
	tests := []struct {
		name, query, want string
	}{
		{"mutation with variables", "mutation Login($e: String!) { login }", "Login"},
		{"query with a brace", "query Me { me { id } }", "Me"},
		{"leading whitespace", "\n\t query Courses { courses }", "Courses"},
		{"anonymous query", "{ me { id } }", ""},
		{"no delimiter after the name", "query Me", ""},
		{"empty", "", ""},
		// Fails closed: the caller looks the result up in an allowlist, so ""
		// is the safe answer for anything unreadable.
		{"subscription is not classified", "subscription OnPing { ping }", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ExtractOperationName(tt.query))
		})
	}
}

// ---------- the migration gate ----------

// TestJWTAuth_TokenTableOutcomes pins the full accept/reject contract of the
// reference-token deployment in one place, so a migration onto this middleware
// can be checked against it rather than against a reading of the code.
//
// Each row is a way a credential can be wrong, and every one of them must end
// as 401 with the handler untouched.
func TestJWTAuth_TokenTableOutcomes(t *testing.T) {
	const issuer = "descholar"
	svc := issuerJWTService(issuer)

	valid, err := svc.GenerateToken("user-1", "admin")
	require.NoError(t, err)
	expired, err := NewJWTService(&config.AuthConfig{
		JWTSecret:         "test-secret-key-for-testing",
		AccessTokenExpiry: -time.Second,
		Issuer:            issuer,
	}).GenerateToken("user-1", "admin")
	require.NoError(t, err)
	wrongIssuer, err := issuerJWTService("someone-else").GenerateToken("user-1", "admin")
	require.NoError(t, err)
	wrongSecret, err := NewJWTService(&config.AuthConfig{
		JWTSecret:         "a-completely-different-secret",
		AccessTokenExpiry: 15 * time.Minute,
		Issuer:            issuer,
	}).GenerateToken("user-1", "admin")
	require.NoError(t, err)

	// The store resolves exactly one handle, standing for "this reference is
	// live"; every other handle is unknown, revoked, or expired server-side.
	const liveHandle = "live-reference"
	store := map[string]string{
		liveHandle:         valid,
		"expired-token":    expired,
		"wrong-issuer":     wrongIssuer,
		"wrong-secret":     wrongSecret,
		"resolves-to-junk": "not-a-jwt",
	}
	resolver := WithCredentialResolver(func(_ context.Context, presented string) (string, error) {
		token, ok := store[presented]
		if !ok {
			return "", errors.New("no such reference")
		}
		return token, nil
	})

	tests := []struct {
		name       string
		credential string
		wantStatus int
	}{
		{"live reference", liveHandle, http.StatusOK},
		{"expired token behind a live reference", "expired-token", http.StatusUnauthorized},
		{"token minted for another service", "wrong-issuer", http.StatusUnauthorized},
		{"token signed with another secret", "wrong-secret", http.StatusUnauthorized},
		{"reference resolving to a non-token", "resolves-to-junk", http.StatusUnauthorized},
		{"unknown reference", "never-issued", http.StatusUnauthorized},
		{"empty bearer value", "", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reached := false
			inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				reached = true
				w.WriteHeader(http.StatusOK)
			})
			handler := JWTAuth(svc, resolver)(inner)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+tt.credential)
			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantStatus == http.StatusOK, reached,
				"handler reachability must match the status")
		})
	}
}

// TestJWTAuth_UnsetIssuerAcceptsAnyIssuer records the one behavior that is NOT
// preserved by default when moving onto this middleware.
//
// A deployment that checked the issuer before must set config.AuthConfig.Issuer
// here, or the check silently disappears — tokens minted for a sibling service
// start being accepted, and nothing fails to announce it.
func TestJWTAuth_UnsetIssuerAcceptsAnyIssuer(t *testing.T) {
	foreign, err := issuerJWTService("someone-else").GenerateToken("user-1", "admin")
	require.NoError(t, err)

	handler := JWTAuth(testJWTService())(noopHandler) // Issuer unset
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+foreign)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code,
		"an unset issuer means no issuer check — configure Issuer to restore it")
}
