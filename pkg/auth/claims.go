package auth

import (
	"context"
	"fmt"
	"slices"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	// ClaimsKey is the context key under which JWT Claims are stored.
	ClaimsKey contextKey = "claims"

	// TokenKey is the context key under which the validated token string is
	// stored, so a handler can forward the caller's own credential to another
	// service instead of minting one.
	TokenKey contextKey = "token"
)

// Claims holds JWT token payload data.
//
// Two identity conventions are in the wild and both are supported. Tokens this
// package issues carry the user id in `user_id`; tokens issued by an OAuth 2.0
// / OIDC provider carry it in the registered `sub` claim. Read it through
// [Claims.SubjectID] rather than either field, or half the tokens read as
// anonymous.
//
// Roles work the same way: a single `role`, or a `roles` array. [Claims.HasRole]
// and friends look at both.
//
// Do not add fields that shadow a registered claim — an `Exp int64` with tag
// `json:"exp"` wins the unmarshal over the embedded RegisteredClaims.ExpiresAt,
// leaves it nil, and the library then has no expiry to check. Expired tokens
// validate clean and nothing reports it.
type Claims struct {
	jwt.RegisteredClaims
	UserID string   `json:"user_id"`
	Role   string   `json:"role"`
	Roles  []string `json:"roles,omitempty"`
}

// SubjectID returns the authenticated user's identifier: UserID when the token
// carries one, otherwise the registered `sub` claim.
//
// Anything recording *who* did something — an audit row, an ownership check —
// must go through this. Reading UserID directly yields "" for every OIDC token
// and the record silently loses its subject.
func (c *Claims) SubjectID() string {
	if c == nil {
		return ""
	}
	if c.UserID != "" {
		return c.UserID
	}
	return c.Subject
}

// AllRoles returns every role the token grants, from both the singular Role and
// the plural Roles, de-duplicated. Order follows Role first, then Roles as
// given; it carries no precedence meaning — see [RoleLadder] for that.
func (c *Claims) AllRoles() []string {
	if c == nil {
		return nil
	}
	all := make([]string, 0, len(c.Roles)+1)
	seen := make(map[string]struct{}, len(c.Roles)+1)
	for _, r := range append([]string{c.Role}, c.Roles...) {
		if r == "" {
			continue
		}
		if _, dup := seen[r]; dup {
			continue
		}
		seen[r] = struct{}{}
		all = append(all, r)
	}
	return all
}

// HasRole reports whether the token grants role exactly.
func (c *Claims) HasRole(role string) bool {
	if c == nil || role == "" {
		return false
	}
	return c.Role == role || slices.Contains(c.Roles, role)
}

// HasAnyRole reports whether the token grants at least one of roles.
//
// Called with no roles it returns false: an empty requirement list is far more
// often a bug at the call site than a deliberate "anyone may pass", and the
// failure should be a denied request rather than an open endpoint.
func (c *Claims) HasAnyRole(roles ...string) bool {
	return slices.ContainsFunc(roles, c.HasRole)
}

// RoleLadder orders roles by authority, highest first, so a check can ask for a
// minimum rather than an exact match.
//
// The ladder is the application's, not the framework's — "admin" outranking
// "editor" is a product decision, and a framework that hardcoded one would be
// wrong for the next application. Declare it once and pass it around:
//
//	var Ladder = auth.RoleLadder{"super_admin", "admin", "facilitator", "learner"}
type RoleLadder []string

// Precedence returns role's index in the ladder, 0 being the highest authority,
// or -1 when the ladder does not list it.
func (l RoleLadder) Precedence(role string) int {
	for i, r := range l {
		if r == role {
			return i
		}
	}
	return -1
}

// Highest returns the most authoritative of the token's roles, or "" when it
// holds none the ladder lists.
func (l RoleLadder) Highest(c *Claims) string {
	if c == nil {
		return ""
	}
	held := make(map[string]struct{})
	for _, r := range c.AllRoles() {
		held[r] = struct{}{}
	}
	for _, r := range l {
		if _, ok := held[r]; ok {
			return r
		}
	}
	return ""
}

// HasHigherOrEqual reports whether the token's best role is at least as
// authoritative as required.
//
// A role the ladder does not list denies, in both positions. An unknown
// required role is a typo at the call site, and resolving a typo to "allow"
// turns a mistake into an open door.
func (l RoleLadder) HasHigherOrEqual(c *Claims, required string) bool {
	highest := l.Highest(c)
	if highest == "" {
		return false
	}
	heldPrec, requiredPrec := l.Precedence(highest), l.Precedence(required)
	if heldPrec == -1 || requiredPrec == -1 {
		return false
	}
	return heldPrec <= requiredPrec
}

// ClaimsFromContext extracts Claims from the request context.
// Returns error if no claims are present (unauthenticated request).
func ClaimsFromContext(ctx context.Context) (*Claims, error) {
	claims, ok := ctx.Value(ClaimsKey).(*Claims)
	if !ok || claims == nil {
		return nil, fmt.Errorf("no claims in context")
	}
	return claims, nil
}

// WithToken returns a context carrying the validated token string.
func WithToken(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	return context.WithValue(ctx, TokenKey, token)
}

// TokenFromContext returns the validated token this request presented, for
// forwarding to a downstream service on the caller's behalf. Empty when the
// request was not authenticated.
func TokenFromContext(ctx context.Context) string {
	token, _ := ctx.Value(TokenKey).(string)
	return token
}
