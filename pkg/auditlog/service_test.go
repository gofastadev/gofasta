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

	subjectID, ip, ua := s.FromContext(context.Background())

	if subjectID != nil {
		t.Errorf("subjectID = %v, want nil", subjectID)
	}
	if ip != "" || ua != "" {
		t.Errorf("ip/ua = %q/%q, want empty — the caller must be able to detect this", ip, ua)
	}
}

func TestFromContext_DefaultSubjectReadsTheSubClaim(t *testing.T) {
	// The subject of an audit row is the token's registered `sub` claim, which
	// is what every OAuth 2.0 / OIDC issuer emits — including Solago, whose
	// tokens carry a UUID there.
	s := NewAuditService(nil, "test")

	ctx := context.WithValue(context.Background(), auth.ClaimsKey, &auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "11111111-2222-3333-4444-555555555555"},
	})
	subjectID, _, _ := s.FromContext(ctx)

	if subjectID == nil || *subjectID != "11111111-2222-3333-4444-555555555555" {
		t.Errorf("subjectID = %v, want the sub claim", subjectID)
	}
}

func TestFromContext_SubjectMayBeAClientNotAUser(t *testing.T) {
	// Under the client-credentials grant `sub` holds a client id. The column is
	// called subject_id rather than user_id precisely so this row is not read as
	// naming a person.
	s := NewAuditService(nil, "test")

	ctx := context.WithValue(context.Background(), auth.ClaimsKey, &auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "jedi_lms"},
	})
	subjectID, _, _ := s.FromContext(ctx)

	if subjectID == nil || *subjectID != "jedi_lms" {
		t.Errorf("subjectID = %v, want the client id", subjectID)
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
	subjectID, _, _ := s.FromContext(ctx)

	if subjectID == nil || *subjectID != "solago-subject" {
		t.Errorf("subjectID = %v, want solago-subject", subjectID)
	}
}

func TestWithSubjectFunc_NilIsIgnored(t *testing.T) {
	// A nil hook would disable identity extraction entirely and silently.
	s := NewAuditService(nil, "test", WithSubjectFunc(nil))

	ctx := context.WithValue(context.Background(), auth.ClaimsKey, &auth.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "user-42"}})
	if subjectID, _, _ := s.FromContext(ctx); subjectID == nil {
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

func TestFromContext_ClaimsWithNoIdentityYieldNoSubject(t *testing.T) {
	s := NewAuditService(nil, "test")

	ctx := context.WithValue(context.Background(), auth.ClaimsKey, &auth.Claims{})
	if subjectID, _, _ := s.FromContext(ctx); subjectID != nil {
		t.Errorf("subjectID = %v, want nil so the empty case stays detectable", subjectID)
	}
}
