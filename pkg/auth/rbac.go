package auth

import (
	"fmt"
	"slices"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/persist"
	"github.com/gofastadev/gofasta/pkg/config"
)

// RBACService wraps Casbin for role-based access control.
//
// The embedded enforcer is a [casbin.SyncedEnforcer] rather than a plain
// Enforcer. Casbin's plain enforcer is not safe for concurrent use, and this
// type exposes both Enforce and AddPolicy — so any service adding a policy at
// runtime while a request is being authorized would race on the policy model.
// SyncedEnforcer takes the lock for us, and a watcher reloading policy in the
// background makes that mandatory rather than merely prudent.
type RBACService struct {
	enforcer *casbin.SyncedEnforcer
}

type rbacOptions struct {
	watcher persist.Watcher
}

// RBACOption configures an RBACService at construction.
type RBACOption func(*rbacOptions)

// WithWatcher installs a Casbin watcher so this process reloads its policy when
// another process changes it.
//
// Without one, a policy edit reaches only the replica that served the request
// that made it. Every other replica keeps enforcing the old policy until it
// restarts — so a revoked permission stays usable, on some replicas, for as
// long as they stay up. See [NewRedisWatcher].
func WithWatcher(w persist.Watcher) RBACOption {
	return func(o *rbacOptions) { o.watcher = w }
}

// NewRBACService constructs an RBACService from the Casbin model and policy
// paths declared in AuthConfig.
//
// Policy comes from a file and is therefore read-only in practice: AddPolicy
// changes this process's copy and nothing else. For policy that several
// processes share and change at runtime, use [NewRBACServiceWithAdapter].
func NewRBACService(cfg *config.AuthConfig) (*RBACService, error) {
	enforcer, err := casbin.NewSyncedEnforcer(cfg.RBACModelPath, cfg.RBACPolicyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize RBAC: %w", err)
	}
	return &RBACService{enforcer: enforcer}, nil
}

// NewRBACServiceWithAdapter constructs an RBACService whose policy lives in a
// Casbin adapter — a database table, typically, via [NewGormAdapter] — rather
// than a file.
//
// This is what a multi-replica deployment needs: the policy is shared, edits
// are durable, and with [WithWatcher] they propagate. The policy is loaded
// once here; the watcher reloads it thereafter.
func NewRBACServiceWithAdapter(modelPath string, adapter persist.Adapter, opts ...RBACOption) (*RBACService, error) {
	if modelPath == "" {
		return nil, fmt.Errorf("rbac: model path is required")
	}
	if adapter == nil {
		return nil, fmt.Errorf("rbac: adapter is required")
	}

	o := &rbacOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}

	enforcer, err := casbin.NewSyncedEnforcer(modelPath, adapter)
	if err != nil {
		return nil, fmt.Errorf("rbac: enforcer init: %w", err)
	}

	if o.watcher != nil {
		if err := enforcer.SetWatcher(o.watcher); err != nil {
			// Not survivable by design. A watcher that failed to attach leaves
			// this replica enforcing a policy that silently stops tracking
			// reality, and the only symptom is an authorization decision that
			// disagrees with every other replica.
			return nil, fmt.Errorf("rbac: attaching watcher: %w", err)
		}
	}

	if err := enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("rbac: loading policy: %w", err)
	}

	return &RBACService{enforcer: enforcer}, nil
}

// Enforcer exposes the underlying Casbin enforcer for the operations this
// wrapper does not cover. Prefer the methods here where they exist.
func (s *RBACService) Enforcer() *casbin.SyncedEnforcer { return s.enforcer }

// LoadPolicy re-reads the policy from the adapter.
func (s *RBACService) LoadPolicy() error { return s.enforcer.LoadPolicy() }

// Enforce checks if a role is allowed to perform an action on a resource.
//
// This is the three-argument model: subject, object, action. Multi-tenant
// deployments want [RBACService.EnforceInDomain] instead.
func (s *RBACService) Enforce(role, resource, action string) (bool, error) {
	return s.enforcer.Enforce(role, resource, action)
}

// EnforceInDomain checks a request against a domain-scoped policy:
// (subject, domain, object, action).
//
// The domain is the tenant — an organization or institution id — and it is
// what keeps one tenant's grant from applying inside another. It requires a
// model declaring the domain, Casbin's rbac_with_domains: `r = sub, dom, obj,
// act` with `g = _, _, _`. Calling this against a three-argument model returns
// an error rather than a decision, because a wrong number of arguments there
// cannot be resolved into a safe answer.
//
// Pass "*" for a deliberately cross-tenant check, and only where the model's
// matcher treats it as a wildcard.
func (s *RBACService) EnforceInDomain(subject, domain, object, action string) (bool, error) {
	return s.enforcer.Enforce(subject, domain, object, action)
}

// AddPolicy adds a new policy rule at runtime.
func (s *RBACService) AddPolicy(role, resource, action string) (bool, error) {
	return s.enforcer.AddPolicy(role, resource, action)
}

// RemovePolicy removes a policy rule at runtime.
func (s *RBACService) RemovePolicy(role, resource, action string) (bool, error) {
	return s.enforcer.RemovePolicy(role, resource, action)
}

// Effect values for models whose policy declares a trailing `eft` field.
const (
	EffectAllow = "allow"
	EffectDeny  = "deny"
)

// AddPolicyInDomain adds a domain-scoped policy rule at runtime.
//
// effect is optional and must be supplied exactly when the model's policy
// declares an `eft` field — `p = sub, dom, obj, act, eft`, the shape a model
// needs to express a deny that carves an exception out of a broader allow.
// Casbin rejects a rule whose field count differs from the model's, so passing
// the wrong number returns an error rather than storing something that would
// never match.
func (s *RBACService) AddPolicyInDomain(role, domain, object, action string, effect ...string) (bool, error) {
	return s.enforcer.AddPolicy(policyRule(role, domain, object, action, effect)...)
}

// RemovePolicyInDomain removes a domain-scoped policy rule at runtime. The
// arguments must match the stored rule exactly, effect included.
func (s *RBACService) RemovePolicyInDomain(role, domain, object, action string, effect ...string) (bool, error) {
	return s.enforcer.RemovePolicy(policyRule(role, domain, object, action, effect)...)
}

// policyRule assembles the rule as []interface{}, which is what Casbin's
// variadic AddPolicy/RemovePolicy take. Passing a []string would work through
// their reflection path, but only by accident of a check that is not part of
// the documented signature.
func policyRule(role, domain, object, action string, effect []string) []interface{} {
	rule := make([]interface{}, 0, 4+len(effect))
	for _, field := range append([]string{role, domain, object, action}, effect...) {
		rule = append(rule, field)
	}
	return rule
}

// AddRoleForUserInDomain grants a subject a role within one domain.
func (s *RBACService) AddRoleForUserInDomain(subject, role, domain string) (bool, error) {
	return s.enforcer.AddGroupingPolicy(subject, role, domain)
}

// RemoveRoleForUserInDomain revokes a subject's role within one domain.
func (s *RBACService) RemoveRoleForUserInDomain(subject, role, domain string) (bool, error) {
	return s.enforcer.RemoveGroupingPolicy(subject, role, domain)
}

// PermissionsInDomain returns the flat set of "object:action" strings a subject
// holds in a domain, through every role granted to them there.
//
// It exists for user interfaces: a client that knows what the caller may do can
// hide what they may not, rather than offering an action and reporting a denial
// afterwards. It is not an authorization decision — the enforcer is. A client
// that trusts this list in place of enforcement has moved the access check to
// where the user can edit it.
//
// A subject holding a role granted in the wildcard domain "*" sees that role's
// permissions in every domain, which is how a platform-wide administrator is
// expressed. Passing an empty domain returns permissions across all of them.
func (s *RBACService) PermissionsInDomain(subject, domain string) ([]string, error) {
	return permissionsInDomain(s.enforcer, subject, domain)
}

// policyReader is the part of the enforcer permissionsInDomain reads through.
//
// It is an interface so the two failure paths below can be exercised: a live
// SyncedEnforcer backed by a file or a table returns an error from these only
// when its adapter is already broken, and code that has never run once is code
// that reports the wrong thing at the moment it finally does.
type policyReader interface {
	GetFilteredGroupingPolicy(fieldIndex int, fieldValues ...string) ([][]string, error)
	GetFilteredPolicy(fieldIndex int, fieldValues ...string) ([][]string, error)
}

func permissionsInDomain(reader policyReader, subject, domain string) ([]string, error) {
	groupings, err := reader.GetFilteredGroupingPolicy(0, subject)
	if err != nil {
		return nil, fmt.Errorf("rbac: reading role grants: %w", err)
	}

	seen := make(map[string]struct{})
	for _, grant := range groupings {
		// subject, role, domain — a grant without a domain belongs to a
		// three-argument model and cannot answer a domain-scoped question.
		if len(grant) < 3 {
			continue
		}
		role, grantDomain := grant[1], grant[2]
		if domain != "" && grantDomain != "*" && grantDomain != domain {
			continue
		}

		policies, err := reader.GetFilteredPolicy(0, role, grantDomain)
		if err != nil {
			return nil, fmt.Errorf("rbac: reading policies for role %q: %w", role, err)
		}
		for _, p := range policies {
			// role, domain, object, action
			if len(p) < 4 {
				continue
			}
			seen[p[2]+":"+p[3]] = struct{}{}
		}
	}

	out := make([]string, 0, len(seen))
	for permission := range seen {
		out = append(out, permission)
	}
	// Sorted so the result is stable: it is often serialized into a response
	// body, and an unstable order defeats both caching and diffing.
	slices.Sort(out)
	return out, nil
}
