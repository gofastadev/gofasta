package auth

import (
	"os"
	"path/filepath"
	"testing"

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
