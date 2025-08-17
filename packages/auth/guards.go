package auth

import (
	"strings"

	"github.com/healtronlabs/gofasta/packages/core"
)

// AuthGuard implements JWT-based authentication guard
type AuthGuard struct {
	jwtService *JWTService
}

// NewAuthGuard creates a new authentication guard
func NewAuthGuard(jwtService *JWTService) *AuthGuard {
	return &AuthGuard{
		jwtService: jwtService,
	}
}

// CanActivate implements the Guard interface
func (g *AuthGuard) CanActivate(ctx *core.RequestContext) bool {
	// Extract token from Authorization header
	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" {
		return false
	}

	token := g.jwtService.ExtractTokenFromBearer(authHeader)
	if token == "" {
		return false
	}

	// Validate token and extract user info
	userInfo, err := g.jwtService.GetUserFromToken(token)
	if err != nil {
		return false
	}

	// Store user info in context for use by controllers
	ctx.User = userInfo
	return true
}

// RoleGuard implements role-based authorization guard
type RoleGuard struct {
	requiredRoles []string
	requireAll    bool // If true, user must have ALL roles; if false, user must have ANY role
}

// NewRoleGuard creates a new role-based guard
func NewRoleGuard(requiredRoles []string, requireAll bool) *RoleGuard {
	return &RoleGuard{
		requiredRoles: requiredRoles,
		requireAll:    requireAll,
	}
}

// CanActivate implements the Guard interface
func (g *RoleGuard) CanActivate(ctx *core.RequestContext) bool {
	// Check if user is authenticated
	userInfo, ok := ctx.User.(*UserInfo)
	if !ok || userInfo == nil {
		return false
	}

	if g.requireAll {
		return userInfo.HasAllRoles(g.requiredRoles...)
	}
	return userInfo.HasAnyRole(g.requiredRoles...)
}

// JWTAuthGuard combines authentication and optional role checking
type JWTAuthGuard struct {
	jwtService    *JWTService
	requiredRoles []string
	requireAll    bool
}

// NewJWTAuthGuard creates a new JWT authentication guard with optional role checking
func NewJWTAuthGuard(jwtService *JWTService, requiredRoles []string, requireAll bool) *JWTAuthGuard {
	return &JWTAuthGuard{
		jwtService:    jwtService,
		requiredRoles: requiredRoles,
		requireAll:    requireAll,
	}
}

// CanActivate implements the Guard interface
func (g *JWTAuthGuard) CanActivate(ctx *core.RequestContext) bool {
	// First check authentication
	authGuard := NewAuthGuard(g.jwtService)
	if !authGuard.CanActivate(ctx) {
		return false
	}

	// If no roles required, authentication is sufficient
	if len(g.requiredRoles) == 0 {
		return true
	}

	// Check roles
	roleGuard := NewRoleGuard(g.requiredRoles, g.requireAll)
	return roleGuard.CanActivate(ctx)
}

// AdminGuard is a convenience guard for admin-only access
type AdminGuard struct {
	*JWTAuthGuard
}

// NewAdminGuard creates a new admin guard
func NewAdminGuard(jwtService *JWTService) *AdminGuard {
	return &AdminGuard{
		JWTAuthGuard: NewJWTAuthGuard(jwtService, []string{"admin"}, false),
	}
}

// ModeratorGuard is a convenience guard for moderator+ access
type ModeratorGuard struct {
	*JWTAuthGuard
}

// NewModeratorGuard creates a new moderator guard
func NewModeratorGuard(jwtService *JWTService) *ModeratorGuard {
	return &ModeratorGuard{
		JWTAuthGuard: NewJWTAuthGuard(jwtService, []string{"admin", "moderator"}, false),
	}
}

// OwnershipGuard checks if user owns the resource (simplified implementation)
type OwnershipGuard struct {
	jwtService *JWTService
	idParam    string // URL parameter containing the resource owner ID
}

// NewOwnershipGuard creates a new ownership guard
func NewOwnershipGuard(jwtService *JWTService, idParam string) *OwnershipGuard {
	return &OwnershipGuard{
		jwtService: jwtService,
		idParam:    idParam,
	}
}

// CanActivate implements the Guard interface
func (g *OwnershipGuard) CanActivate(ctx *core.RequestContext) bool {
	// First check authentication
	authGuard := NewAuthGuard(g.jwtService)
	if !authGuard.CanActivate(ctx) {
		return false
	}

	userInfo, ok := ctx.User.(*UserInfo)
	if !ok || userInfo == nil {
		return false
	}

	// Check if user is admin (admins can access everything)
	if userInfo.HasRole("admin") {
		return true
	}

	// Check ownership
	resourceOwnerID := ctx.GetParam(g.idParam)
	return userInfo.ID == resourceOwnerID
}

// PublicGuard allows all requests (used for public endpoints)
type PublicGuard struct{}

// NewPublicGuard creates a new public guard
func NewPublicGuard() *PublicGuard {
	return &PublicGuard{}
}

// CanActivate implements the Guard interface
func (g *PublicGuard) CanActivate(ctx *core.RequestContext) bool {
	return true
}

// ConditionalGuard applies different guards based on conditions
type ConditionalGuard struct {
	conditions []GuardCondition
	defaultGuard core.Guard
}

// GuardCondition represents a conditional guard
type GuardCondition struct {
	Condition func(ctx *core.RequestContext) bool
	Guard     core.Guard
}

// NewConditionalGuard creates a new conditional guard
func NewConditionalGuard(defaultGuard core.Guard) *ConditionalGuard {
	return &ConditionalGuard{
		conditions:   make([]GuardCondition, 0),
		defaultGuard: defaultGuard,
	}
}

// AddCondition adds a condition to the guard
func (g *ConditionalGuard) AddCondition(condition func(ctx *core.RequestContext) bool, guard core.Guard) *ConditionalGuard {
	g.conditions = append(g.conditions, GuardCondition{
		Condition: condition,
		Guard:     guard,
	})
	return g
}

// CanActivate implements the Guard interface
func (g *ConditionalGuard) CanActivate(ctx *core.RequestContext) bool {
	// Check conditions in order
	for _, condition := range g.conditions {
		if condition.Condition(ctx) {
			return condition.Guard.CanActivate(ctx)
		}
	}
	
	// Use default guard if no conditions match
	if g.defaultGuard != nil {
		return g.defaultGuard.CanActivate(ctx)
	}
	
	return false
}

// Common condition functions
func IsHTTPMethod(method string) func(ctx *core.RequestContext) bool {
	return func(ctx *core.RequestContext) bool {
		return strings.ToUpper(ctx.Request.Method) == strings.ToUpper(method)
	}
}

func HasHeader(headerName string) func(ctx *core.RequestContext) bool {
	return func(ctx *core.RequestContext) bool {
		return ctx.GetHeader(headerName) != ""
	}
}

func HasQueryParam(paramName string) func(ctx *core.RequestContext) bool {
	return func(ctx *core.RequestContext) bool {
		return ctx.GetQuery(paramName) != ""
	}
}

func PathMatches(pattern string) func(ctx *core.RequestContext) bool {
	return func(ctx *core.RequestContext) bool {
		return strings.Contains(ctx.Request.URL.Path, pattern)
	}
}

// Example usage:
/*
// Different guards for different HTTP methods
conditionalGuard := NewConditionalGuard(NewAuthGuard(jwtService)).
    AddCondition(IsHTTPMethod("GET"), NewPublicGuard()).
    AddCondition(IsHTTPMethod("POST"), NewAdminGuard(jwtService)).
    AddCondition(IsHTTPMethod("PUT"), NewOwnershipGuard(jwtService, "id"))
*/