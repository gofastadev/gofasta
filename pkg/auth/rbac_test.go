package auth

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/casbin/casbin/v2/model"
	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- helpers ----------

const testModel = `[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.sub == p.sub && r.obj == p.obj && r.act == p.act
`

const testPolicy = `p, admin, /users, read
p, admin, /users, write
p, editor, /posts, read
p, editor, /posts, write
`

// writeRBACFiles creates temporary model and policy files and returns the AuthConfig.
func writeRBACFiles(t *testing.T) *config.AuthConfig {
	t.Helper()
	dir := t.TempDir()

	modelPath := filepath.Join(dir, "rbac_model.conf")
	policyPath := filepath.Join(dir, "rbac_policy.csv")

	require.NoError(t, os.WriteFile(modelPath, []byte(testModel), 0644))
	require.NoError(t, os.WriteFile(policyPath, []byte(testPolicy), 0644))

	return &config.AuthConfig{
		RBACModelPath:  modelPath,
		RBACPolicyPath: policyPath,
	}
}

// ---------- NewRBACService ----------

func TestNewRBACService_Success(t *testing.T) {
	cfg := writeRBACFiles(t)

	svc, err := NewRBACService(cfg)
	require.NoError(t, err)
	assert.NotNil(t, svc)
}

func TestNewRBACService_InvalidModelPath(t *testing.T) {
	dir := t.TempDir()
	policyPath := filepath.Join(dir, "rbac_policy.csv")
	require.NoError(t, os.WriteFile(policyPath, []byte(testPolicy), 0644))

	cfg := &config.AuthConfig{
		RBACModelPath:  filepath.Join(dir, "nonexistent_model.conf"),
		RBACPolicyPath: policyPath,
	}

	svc, err := NewRBACService(cfg)
	require.Error(t, err)
	assert.Nil(t, svc)
	assert.Contains(t, err.Error(), "failed to initialize RBAC")
}

func TestNewRBACService_InvalidPolicyPath(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "rbac_model.conf")
	require.NoError(t, os.WriteFile(modelPath, []byte(testModel), 0644))

	cfg := &config.AuthConfig{
		RBACModelPath:  modelPath,
		RBACPolicyPath: filepath.Join(dir, "nonexistent_policy.csv"),
	}

	// Casbin may or may not error on missing policy file depending on version;
	// if it succeeds, the service simply has no policies loaded.
	svc, err := NewRBACService(cfg)
	if err != nil {
		assert.Contains(t, err.Error(), "failed to initialize RBAC")
		assert.Nil(t, svc)
	} else {
		assert.NotNil(t, svc)
	}
}

// ---------- Enforce ----------

func TestEnforce(t *testing.T) {
	cfg := writeRBACFiles(t)
	svc, err := NewRBACService(cfg)
	require.NoError(t, err)

	tests := []struct {
		name     string
		role     string
		resource string
		action   string
		want     bool
	}{
		{
			name:     "admin can read users",
			role:     "admin",
			resource: "/users",
			action:   "read",
			want:     true,
		},
		{
			name:     "admin can write users",
			role:     "admin",
			resource: "/users",
			action:   "write",
			want:     true,
		},
		{
			name:     "editor can read posts",
			role:     "editor",
			resource: "/posts",
			action:   "read",
			want:     true,
		},
		{
			name:     "editor can write posts",
			role:     "editor",
			resource: "/posts",
			action:   "write",
			want:     true,
		},
		{
			name:     "admin denied posts read",
			role:     "admin",
			resource: "/posts",
			action:   "read",
			want:     false,
		},
		{
			name:     "editor denied users write",
			role:     "editor",
			resource: "/users",
			action:   "write",
			want:     false,
		},
		{
			name:     "unknown role denied",
			role:     "viewer",
			resource: "/users",
			action:   "read",
			want:     false,
		},
		{
			name:     "admin denied unknown resource",
			role:     "admin",
			resource: "/settings",
			action:   "read",
			want:     false,
		},
		{
			name:     "admin denied unknown action",
			role:     "admin",
			resource: "/users",
			action:   "delete",
			want:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			allowed, err := svc.Enforce(tc.role, tc.resource, tc.action)
			require.NoError(t, err)
			assert.Equal(t, tc.want, allowed)
		})
	}
}

// ---------- AddPolicy ----------

func TestAddPolicy(t *testing.T) {
	cfg := writeRBACFiles(t)
	svc, err := NewRBACService(cfg)
	require.NoError(t, err)

	// Verify viewer cannot read users initially.
	allowed, err := svc.Enforce("viewer", "/users", "read")
	require.NoError(t, err)
	assert.False(t, allowed)

	// Add a policy for viewer.
	ok, err := svc.AddPolicy("viewer", "/users", "read")
	require.NoError(t, err)
	assert.True(t, ok)

	// Now viewer should be allowed.
	allowed, err = svc.Enforce("viewer", "/users", "read")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestAddPolicy_Duplicate(t *testing.T) {
	cfg := writeRBACFiles(t)
	svc, err := NewRBACService(cfg)
	require.NoError(t, err)

	// Adding an already-existing policy should return false (no new rule added).
	ok, err := svc.AddPolicy("admin", "/users", "read")
	require.NoError(t, err)
	assert.False(t, ok)
}

// ---------- RemovePolicy ----------

func TestRemovePolicy(t *testing.T) {
	cfg := writeRBACFiles(t)
	svc, err := NewRBACService(cfg)
	require.NoError(t, err)

	// Verify admin can read users initially.
	allowed, err := svc.Enforce("admin", "/users", "read")
	require.NoError(t, err)
	assert.True(t, allowed)

	// Remove the policy.
	ok, err := svc.RemovePolicy("admin", "/users", "read")
	require.NoError(t, err)
	assert.True(t, ok)

	// Now admin should be denied.
	allowed, err = svc.Enforce("admin", "/users", "read")
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestRemovePolicy_NonExistent(t *testing.T) {
	cfg := writeRBACFiles(t)
	svc, err := NewRBACService(cfg)
	require.NoError(t, err)

	// Removing a policy that does not exist should return false.
	ok, err := svc.RemovePolicy("viewer", "/settings", "delete")
	require.NoError(t, err)
	assert.False(t, ok)
}

// ---------- domain-scoped enforcement ----------

// domainModel is Casbin's rbac_with_domains shape, with a deny effect and
// pattern matching on object and action. It is the model a multi-tenant
// deployment needs: without the domain, a grant made inside one tenant applies
// inside every other.
const domainModel = `[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act, eft

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow)) && !some(where (p.eft == deny))

[matchers]
m = (g(r.sub, p.sub, r.dom) || g(r.sub, p.sub, "*")) && (p.dom == "*" || r.dom == p.dom) && keyMatch(r.obj, p.obj) && regexMatch(r.act, p.act)
`

func writeDomainModel(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rbac_model.conf")
	require.NoError(t, os.WriteFile(path, []byte(domainModel), 0o600))
	return path
}

// domainFixture is the policy every decision below is checked against. It is
// deliberately fixed: the point of these tests is that the same fixture yields
// the same answers before and after a change to how the enforcer is built.
func domainFixture(t *testing.T) *RBACService {
	t.Helper()

	svc, err := NewRBACServiceWithAdapter(writeDomainModel(t), newSQLiteAdapter(t))
	require.NoError(t, err)

	// Roles, scoped to a tenant.
	_, err = svc.AddRoleForUserInDomain("alice", "admin", "inst-a")
	require.NoError(t, err)
	_, err = svc.AddRoleForUserInDomain("bob", "facilitator", "inst-b")
	require.NoError(t, err)
	// A platform-wide grant, expressed as the wildcard domain.
	_, err = svc.AddRoleForUserInDomain("root", "super_admin", "*")
	require.NoError(t, err)

	for _, p := range [][]string{
		{"admin", "inst-a", "/courses/*", "(create|update|delete)", "allow"},
		{"admin", "inst-a", "/billing/*", "read", "allow"},
		{"facilitator", "inst-b", "/courses/*", "update", "allow"},
		// A deny that must beat the allow above it.
		{"facilitator", "inst-b", "/courses/archived", "update", "deny"},
		{"super_admin", "*", "/*", ".*", "allow"},
	} {
		_, err := svc.AddPolicyInDomain(p[0], p[1], p[2], p[3], p[4])
		require.NoError(t, err)
	}
	return svc
}

func TestEnforceInDomain_DecisionTable(t *testing.T) {
	svc := domainFixture(t)

	tests := []struct {
		name               string
		sub, dom, obj, act string
		want               bool
	}{
		{"role grants the action in its own tenant", "alice", "inst-a", "/courses/intro", "create", true},
		{"same grant does not reach another tenant", "alice", "inst-b", "/courses/intro", "create", false},
		{"action outside the granted pattern", "alice", "inst-a", "/courses/intro", "publish", false},
		{"object outside the granted pattern", "alice", "inst-a", "/library/books", "create", false},
		{"second policy for the same role", "alice", "inst-a", "/billing/invoices", "read", true},
		{"a different role in a different tenant", "bob", "inst-b", "/courses/intro", "update", true},
		{"that role cannot create", "bob", "inst-b", "/courses/intro", "create", false},
		// The deny effect: an explicit deny outranks a matching allow, which is
		// what makes a carve-out expressible at all.
		{"explicit deny beats the matching allow", "bob", "inst-b", "/courses/archived", "update", false},
		{"wildcard-domain role reaches every tenant", "root", "inst-a", "/anything", "delete", true},
		{"wildcard-domain role, other tenant", "root", "inst-b", "/anything", "delete", true},
		{"a subject with no grant at all", "mallory", "inst-a", "/courses/intro", "create", false},
		{"empty domain matches nothing", "alice", "", "/courses/intro", "create", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.EnforceInDomain(tt.sub, tt.dom, tt.obj, tt.act)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEnforceInDomain_SurvivesAReload(t *testing.T) {
	// A watcher reloads policy from the adapter behind the enforcer's back.
	// The same fixture must decide identically afterwards, or a policy change
	// anywhere becomes an authorization change everywhere.
	svc := domainFixture(t)

	before, err := svc.EnforceInDomain("alice", "inst-a", "/courses/intro", "create")
	require.NoError(t, err)
	require.True(t, before)

	require.NoError(t, svc.LoadPolicy())

	after, err := svc.EnforceInDomain("alice", "inst-a", "/courses/intro", "create")
	require.NoError(t, err)
	assert.Equal(t, before, after)

	denied, err := svc.EnforceInDomain("alice", "inst-b", "/courses/intro", "create")
	require.NoError(t, err)
	assert.False(t, denied, "tenant isolation lost across a reload")
}

// ---------- PermissionsInDomain ----------

func TestPermissionsInDomain(t *testing.T) {
	svc := domainFixture(t)

	alice, err := svc.PermissionsInDomain("alice", "inst-a")
	require.NoError(t, err)
	assert.Equal(t, []string{"/billing/*:read", "/courses/*:(create|update|delete)"}, alice,
		"sorted, so the serialized result is stable")

	// The permissions of a tenant the subject holds no role in must not leak.
	none, err := svc.PermissionsInDomain("alice", "inst-b")
	require.NoError(t, err)
	assert.Empty(t, none)

	// A wildcard-domain role reports its permissions in every tenant, which is
	// how a platform administrator is expressed.
	root, err := svc.PermissionsInDomain("root", "inst-a")
	require.NoError(t, err)
	assert.Equal(t, []string{"/*:.*"}, root)

	unknown, err := svc.PermissionsInDomain("mallory", "inst-a")
	require.NoError(t, err)
	assert.Empty(t, unknown)
}

// ---------- construction ----------

func TestNewRBACServiceWithAdapter_Validation(t *testing.T) {
	_, err := NewRBACServiceWithAdapter("", newSQLiteAdapter(t))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model path")

	_, err = NewRBACServiceWithAdapter(writeDomainModel(t), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "adapter")
}

func TestNewRBACServiceWithAdapter_PolicyOutlivesTheEnforcer(t *testing.T) {
	// The reason to use an adapter rather than a file: the policy is in the
	// database, so a replica starting later sees what an earlier one wrote.
	db := newSQLiteDB(t)
	modelPath := writeDomainModel(t)

	first, err := NewRBACServiceWithAdapter(modelPath, adapterFor(t, db))
	require.NoError(t, err)
	_, err = first.AddPolicyInDomain("admin", "inst-a", "/courses/*", "create", EffectAllow)
	require.NoError(t, err)
	_, err = first.AddRoleForUserInDomain("alice", "admin", "inst-a")
	require.NoError(t, err)

	second, err := NewRBACServiceWithAdapter(modelPath, adapterFor(t, db))
	require.NoError(t, err)

	allowed, err := second.EnforceInDomain("alice", "inst-a", "/courses/intro", "create")
	require.NoError(t, err)
	assert.True(t, allowed, "a second replica did not see the first one's policy")
}

func TestRemoveRoleForUserInDomain_RevokesAccess(t *testing.T) {
	svc := domainFixture(t)

	allowed, err := svc.EnforceInDomain("alice", "inst-a", "/courses/intro", "create")
	require.NoError(t, err)
	require.True(t, allowed)

	_, err = svc.RemoveRoleForUserInDomain("alice", "admin", "inst-a")
	require.NoError(t, err)

	allowed, err = svc.EnforceInDomain("alice", "inst-a", "/courses/intro", "create")
	require.NoError(t, err)
	assert.False(t, allowed, "revoking the role did not revoke the access")
}

// ---------- construction failures ----------

// flakyAdapter loads successfully the first time and fails afterwards, which is
// what an adapter whose database goes away between startup and the first reload
// looks like. Casbin loads once inside NewSyncedEnforcer, so a stub that always
// failed would never get past construction.
type flakyAdapter struct {
	loads int
	err   error
}

func (a *flakyAdapter) LoadPolicy(model.Model) error {
	a.loads++
	if a.loads > 1 {
		return a.err
	}
	return nil
}
func (a *flakyAdapter) SavePolicy(model.Model) error               { return nil }
func (a *flakyAdapter) AddPolicy(_, _ string, _ []string) error    { return nil }
func (a *flakyAdapter) RemovePolicy(_, _ string, _ []string) error { return nil }
func (a *flakyAdapter) RemoveFilteredPolicy(_, _ string, _ int, _ ...string) error {
	return nil
}

// refusingWatcher fails to attach, the way a watcher whose Redis is unreachable
// at the moment the enforcer is built does.
type refusingWatcher struct{ err error }

func (w refusingWatcher) SetUpdateCallback(func(string)) error { return w.err }
func (w refusingWatcher) Update() error                        { return nil }
func (w refusingWatcher) Close()                               {}

func TestNewRBACServiceWithAdapter_RejectsAnUnreadableModel(t *testing.T) {
	svc, err := NewRBACServiceWithAdapter(
		filepath.Join(t.TempDir(), "absent_model.conf"), newSQLiteAdapter(t))

	require.Error(t, err)
	assert.Nil(t, svc)
	assert.Contains(t, err.Error(), "enforcer init")
}

func TestNewRBACServiceWithAdapter_AWatcherThatCannotAttachIsFatal(t *testing.T) {
	// Deliberately not survivable. A watcher that failed to attach leaves this
	// replica enforcing a policy that silently stops tracking reality, and the
	// only symptom is an authorization decision that disagrees with every other
	// replica — the failure mode a watcher exists to prevent.
	svc, err := NewRBACServiceWithAdapter(
		writeDomainModel(t),
		newSQLiteAdapter(t),
		WithWatcher(refusingWatcher{err: errors.New("dial redis: connection refused")}),
	)

	require.Error(t, err)
	assert.Nil(t, svc)
	assert.Contains(t, err.Error(), "attaching watcher")
	assert.Contains(t, err.Error(), "connection refused", "the cause must survive")
}

func TestNewRBACServiceWithAdapter_ReportsAFailedPolicyLoad(t *testing.T) {
	svc, err := NewRBACServiceWithAdapter(
		writeDomainModel(t),
		&flakyAdapter{err: errors.New("policy table vanished")},
	)

	require.Error(t, err)
	assert.Nil(t, svc)
	assert.Contains(t, err.Error(), "loading policy")
}

func TestNewRBACServiceWithAdapter_IgnoresANilOption(t *testing.T) {
	// Options are commonly built into a slice by a caller that appends
	// conditionally; a nil left in it must not take the process down.
	svc, err := NewRBACServiceWithAdapter(writeDomainModel(t), newSQLiteAdapter(t), nil)

	require.NoError(t, err)
	assert.NotNil(t, svc)
}

// ---------- Enforcer ----------

func TestEnforcer_ExposesTheUnderlyingEnforcer(t *testing.T) {
	// The escape hatch for the Casbin operations this wrapper does not cover.
	// It must be the same enforcer, not a copy: a policy added through the
	// wrapper has to be visible through the handle it hands out.
	svc := domainFixture(t)

	enforcer := svc.Enforcer()
	require.NotNil(t, enforcer)

	_, err := svc.AddPolicyInDomain("auditor", "inst-a", "/reports", "read", EffectAllow)
	require.NoError(t, err)

	allowed, err := enforcer.Enforce("auditor", "inst-a", "/reports", "read")
	require.NoError(t, err)
	assert.True(t, allowed, "the wrapper and the exposed enforcer disagree")
}

// ---------- RemovePolicyInDomain ----------

func TestRemovePolicyInDomain_RevokesTheGrant(t *testing.T) {
	svc := domainFixture(t)

	_, err := svc.AddRoleForUserInDomain("mara", "auditor", "inst-a")
	require.NoError(t, err)
	_, err = svc.AddPolicyInDomain("auditor", "inst-a", "/reports", "read", EffectAllow)
	require.NoError(t, err)

	allowed, err := svc.EnforceInDomain("mara", "inst-a", "/reports", "read")
	require.NoError(t, err)
	require.True(t, allowed, "the fixture never granted the permission")

	removed, err := svc.RemovePolicyInDomain("auditor", "inst-a", "/reports", "read", EffectAllow)
	require.NoError(t, err)
	assert.True(t, removed)

	allowed, err = svc.EnforceInDomain("mara", "inst-a", "/reports", "read")
	require.NoError(t, err)
	assert.False(t, allowed, "the permission survived its removal")
}

func TestRemovePolicyInDomain_MustMatchTheStoredRuleExactly(t *testing.T) {
	// Including the effect. A rule stored with one and removed without it is a
	// different rule. Casbin reports that as (false, nil) rather than an error,
	// so a caller that only checks err believes access was revoked when the
	// grant is still there — the returned bool is the only signal.
	svc := domainFixture(t)

	_, err := svc.AddRoleForUserInDomain("mara", "auditor", "inst-a")
	require.NoError(t, err)
	_, err = svc.AddPolicyInDomain("auditor", "inst-a", "/reports", "read", EffectAllow)
	require.NoError(t, err)

	removed, err := svc.RemovePolicyInDomain("auditor", "inst-a", "/reports", "read")

	require.NoError(t, err)
	assert.False(t, removed, "nothing matched, and the caller has only this to go on")

	allowed, err := svc.EnforceInDomain("mara", "inst-a", "/reports", "read")
	require.NoError(t, err)
	assert.True(t, allowed, "the grant is still in force after a mismatched removal")
}

// ---------- permissionsInDomain, at the boundary ----------

// stubReader returns canned policy, including shapes a healthy enforcer would
// not produce.
type stubReader struct {
	groupings    [][]string
	groupingsErr error
	policies     [][]string
	policiesErr  error
}

func (r stubReader) GetFilteredGroupingPolicy(int, ...string) ([][]string, error) {
	return r.groupings, r.groupingsErr
}

func (r stubReader) GetFilteredPolicy(int, ...string) ([][]string, error) {
	return r.policies, r.policiesErr
}

func TestPermissionsInDomain_ReportsAFailureToReadGrants(t *testing.T) {
	// An empty list and a failed read look identical to a caller that ignores
	// the error, and a UI built on the first hides every action the user has.
	_, err := permissionsInDomain(
		stubReader{groupingsErr: errors.New("connection reset")}, "alice", "inst-a")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading role grants")
	assert.Contains(t, err.Error(), "connection reset")
}

func TestPermissionsInDomain_ReportsAFailureToReadPolicies(t *testing.T) {
	_, err := permissionsInDomain(stubReader{
		groupings:   [][]string{{"alice", "admin", "inst-a"}},
		policiesErr: errors.New("connection reset"),
	}, "alice", "inst-a")

	require.Error(t, err)
	assert.Contains(t, err.Error(), `reading policies for role "admin"`)
}

func TestPermissionsInDomain_SkipsGrantsWithNoDomain(t *testing.T) {
	// A two-field grant belongs to a three-argument model and cannot answer a
	// domain-scoped question. Reading grant[2] to find out would panic.
	got, err := permissionsInDomain(stubReader{
		groupings: [][]string{
			{"alice", "admin"},
			{"alice", "auditor", "inst-a"},
		},
		policies: [][]string{{"auditor", "inst-a", "/reports", "read"}},
	}, "alice", "inst-a")

	require.NoError(t, err)
	assert.Equal(t, []string{"/reports:read"}, got)
}

func TestPermissionsInDomain_SkipsPoliciesWithNoAction(t *testing.T) {
	// Same guard on the other side: a three-field policy is a non-domain rule,
	// and p[3] does not exist.
	got, err := permissionsInDomain(stubReader{
		groupings: [][]string{{"alice", "admin", "inst-a"}},
		policies: [][]string{
			{"admin", "/courses", "read"},
			{"admin", "inst-a", "/courses", "create"},
		},
	}, "alice", "inst-a")

	require.NoError(t, err)
	assert.Equal(t, []string{"/courses:create"}, got)
}
