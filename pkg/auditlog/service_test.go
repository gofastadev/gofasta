package auditlog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofastadev/gofasta/pkg/auth"
	"github.com/gofastadev/gofasta/pkg/httpcontext"
	"github.com/golang-jwt/jwt/v5"
)

// ctxWithRequest builds the context httpcontext.Middleware would produce.
func ctxWithRequest(remoteAddr, userAgent, forwardedFor string) context.Context {
	r := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	r.RemoteAddr = remoteAddr
	if userAgent != "" {
		r.Header.Set("User-Agent", userAgent)
	}
	if forwardedFor != "" {
		r.Header.Set("X-Forwarded-For", forwardedFor)
	}
	return httpcontext.WithRequest(context.Background(), r)
}

func TestFromContext_CarriesClientAddressAndAgent(t *testing.T) {
	s := NewAuditService(nil, "test")

	_, ip, ua := s.FromContext(ctxWithRequest("203.0.113.7:5555", "probe/1.0", ""))

	if ip != "203.0.113.7:5555" {
		t.Errorf("ipAddress = %q, want the remote address", ip)
	}
	if ua != "probe/1.0" {
		t.Errorf("userAgent = %q", ua)
	}
}

func TestFromContext_ForwardedForWinsOverRemoteAddr(t *testing.T) {
	// Behind a proxy, RemoteAddr is the proxy. Recording it for every user
	// makes the whole column useless.
	s := NewAuditService(nil, "test")

	_, ip, _ := s.FromContext(ctxWithRequest("10.0.0.1:443", "", "198.51.100.9"))

	if ip != "198.51.100.9" {
		t.Errorf("ipAddress = %q, want the forwarded client address", ip)
	}
}

// This is the failure that shipped: a service installing its own
// request-in-context middleware with its own key wrote audit rows whose IP
// address and user agent were empty, and nothing reported it.
func TestFromContext_WithoutTheMiddlewareYieldsNothing(t *testing.T) {
	s := NewAuditService(nil, "test")

	userID, ip, ua := s.FromContext(context.Background())

	if userID != nil {
		t.Errorf("userID = %v, want nil", userID)
	}
	if ip != "" || ua != "" {
		t.Errorf("ip/ua = %q/%q, want empty — the caller must be able to detect this", ip, ua)
	}
}

func TestFromContext_DefaultSubjectReadsFrameworkClaims(t *testing.T) {
	s := NewAuditService(nil, "test")

	ctx := context.WithValue(context.Background(), auth.ClaimsKey, &auth.Claims{UserID: "user-42"})
	userID, _, _ := s.FromContext(ctx)

	if userID == nil || *userID != "user-42" {
		t.Errorf("userID = %v, want user-42", userID)
	}
}

func TestWithSubjectFunc_ReplacesIdentityExtraction(t *testing.T) {
	// The reason the hook exists: a project whose tokens carry the identity in
	// a different claim must not have to fork the package to say so.
	type projectClaims struct{ Sub string }
	type claimsKey struct{}

	s := NewAuditService(nil, "test", WithSubjectFunc(func(ctx context.Context) *string {
		c, ok := ctx.Value(claimsKey{}).(*projectClaims)
		if !ok || c.Sub == "" {
			return nil
		}
		return &c.Sub
	}))

	ctx := context.WithValue(context.Background(), claimsKey{}, &projectClaims{Sub: "solago-subject"})
	userID, _, _ := s.FromContext(ctx)

	if userID == nil || *userID != "solago-subject" {
		t.Errorf("userID = %v, want solago-subject", userID)
	}
}

func TestWithSubjectFunc_NilIsIgnored(t *testing.T) {
	// A nil hook would disable identity extraction entirely and silently.
	s := NewAuditService(nil, "test", WithSubjectFunc(nil))

	ctx := context.WithValue(context.Background(), auth.ClaimsKey, &auth.Claims{UserID: "user-42"})
	if userID, _, _ := s.FromContext(ctx); userID == nil {
		t.Error("a nil subject func displaced the default")
	}
}

func TestFromContext_EndToEndThroughTheMiddleware(t *testing.T) {
	// The wiring a service actually installs, asserted as one path.
	s := NewAuditService(nil, "test")

	var ip, ua string
	handler := httpcontext.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, ip, ua = s.FromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	req.RemoteAddr = "192.0.2.44:1234"
	req.Header.Set("User-Agent", "jedi/2.0")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if ip != "192.0.2.44:1234" || ua != "jedi/2.0" {
		t.Errorf("ip/ua = %q/%q through the real middleware", ip, ua)
	}
}

func TestFromContext_DefaultSubjectReadsAnOIDCSubClaim(t *testing.T) {
	// The Descholar case, and the OIDC case generally: identity arrives in the
	// registered `sub` claim, not `user_id`. Reading UserID directly gives every
	// such caller a subject-less audit row.
	s := NewAuditService(nil, "test")

	ctx := context.WithValue(context.Background(), auth.ClaimsKey, &auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "11111111-2222-3333-4444-555555555555"},
	})
	userID, _, _ := s.FromContext(ctx)

	if userID == nil || *userID != "11111111-2222-3333-4444-555555555555" {
		t.Errorf("userID = %v, want the sub claim", userID)
	}
}

func TestFromContext_ClaimsWithNoIdentityYieldNoSubject(t *testing.T) {
	s := NewAuditService(nil, "test")

	ctx := context.WithValue(context.Background(), auth.ClaimsKey, &auth.Claims{})
	if userID, _, _ := s.FromContext(ctx); userID != nil {
		t.Errorf("userID = %v, want nil so the empty case stays detectable", userID)
	}
}
