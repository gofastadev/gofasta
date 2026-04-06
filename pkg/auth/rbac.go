package auth

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/gofastadev/gofasta/pkg/config"
)

// RBACService wraps Casbin for role-based access control.
type RBACService struct {
	enforcer *casbin.Enforcer
}

func NewRBACService(cfg *config.AuthConfig) (*RBACService, error) {
	enforcer, err := casbin.NewEnforcer(cfg.RBACModelPath, cfg.RBACPolicyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize RBAC: %w", err)
	}
	return &RBACService{enforcer: enforcer}, nil
}

// Enforce checks if a role is allowed to perform an action on a resource.
func (s *RBACService) Enforce(role, resource, action string) (bool, error) {
	return s.enforcer.Enforce(role, resource, action)
}

// AddPolicy adds a new policy rule at runtime.
func (s *RBACService) AddPolicy(role, resource, action string) (bool, error) {
	return s.enforcer.AddPolicy(role, resource, action)
}

// RemovePolicy removes a policy rule at runtime.
func (s *RBACService) RemovePolicy(role, resource, action string) (bool, error) {
	return s.enforcer.RemovePolicy(role, resource, action)
}
