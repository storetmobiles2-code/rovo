// Package auth holds the authentication primitives shared by every module: audiences and roles,
// the request Principal, Ed25519 access JWTs, opaque refresh tokens, session cookies, argon2id
// password hashing, AES-GCM field encryption and TOTP. Policy (who may log in, session rules)
// lives in the identity module.
package auth

import (
	"context"
	"slices"

	"github.com/google/uuid"
)

// Audience is the app a request belongs to, derived from Host (R14, AUTH-D03).
type Audience string

// Audiences (doc 12 §4.1). Webhook is the api. host, which accepts no cookies.
const (
	AudienceCustomer   Audience = "customer"
	AudienceRestaurant Audience = "restaurant"
	AudienceRider      Audience = "rider"
	AudienceAdmin      Audience = "admin"
	AudienceWebhook    Audience = "webhook"
)

// Valid reports whether a is a known audience.
func (a Audience) Valid() bool {
	switch a {
	case AudienceCustomer, AudienceRestaurant, AudienceRider, AudienceAdmin, AudienceWebhook:
		return true
	}
	return false
}

// IsApp reports whether a is one of the four browser app audiences.
func (a Audience) IsApp() bool { return a.Valid() && a != AudienceWebhook }

// ClientHeaderValue is the X-Rovo-Client value required on unsafe methods (AUTH-D06).
func (a Audience) ClientHeaderValue() string { return string(a) + "-web" }

// Role is an RBAC role (00 §2) plus the internal SYSTEM principal (R26).
type Role string

// Roles.
const (
	RoleCustomer        Role = "CUSTOMER"
	RoleRestaurantOwner Role = "RESTAURANT_OWNER"
	RoleRestaurantStaff Role = "RESTAURANT_STAFF"
	RoleRider           Role = "RIDER"
	RoleAdminSuper      Role = "ADMIN_SUPER"
	RoleAdminOps        Role = "ADMIN_OPS"
	RoleAdminSupport    Role = "ADMIN_SUPPORT"
	RoleAdminFinance    Role = "ADMIN_FINANCE"
	RoleSystem          Role = "SYSTEM"
)

// AdminRoles lists every admin sub-role.
var AdminRoles = []Role{RoleAdminSuper, RoleAdminOps, RoleAdminSupport, RoleAdminFinance}

// IsAdmin reports whether r is an admin sub-role.
func (r Role) IsAdmin() bool { return slices.Contains(AdminRoles, r) }

// Principal is the authenticated caller of a request. Roles is a snapshot from the access
// token for coarse checks; scope decisions (restaurant, city) re-read user_roles (AUTH-D05).
type Principal struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	Audience  Audience
	Roles     []Role
	// Context is the active context: "restaurant:<uuid>" on the restaurant host, "rider" on the
	// rider host, empty otherwise.
	Context string
	// CityScopes are admin city scopes; empty with an admin role means all cities.
	CityScopes []uuid.UUID
	AMR        []string
}

// HasRole reports whether p holds any of roles.
func (p *Principal) HasRole(roles ...Role) bool {
	if p == nil {
		return false
	}
	for _, r := range roles {
		if slices.Contains(p.Roles, r) {
			return true
		}
	}
	return false
}

// IsAdmin reports whether p holds any admin role.
func (p *Principal) IsAdmin() bool { return p.HasRole(AdminRoles...) }

// InCityScope reports whether an admin may act in cityID (empty scopes = all cities).
func (p *Principal) InCityScope(cityID uuid.UUID) bool {
	if p == nil || !p.IsAdmin() {
		return false
	}
	return len(p.CityScopes) == 0 || slices.Contains(p.CityScopes, cityID)
}

// RestaurantContext returns the restaurant id from Context ("restaurant:<uuid>").
func (p *Principal) RestaurantContext() (uuid.UUID, bool) {
	if p == nil || len(p.Context) <= len("restaurant:") || p.Context[:len("restaurant:")] != "restaurant:" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(p.Context[len("restaurant:"):])
	return id, err == nil
}

type principalKey struct{}

// WithPrincipal stores p in ctx.
func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// PrincipalFrom returns the request principal, or nil for anonymous requests.
func PrincipalFrom(ctx context.Context) *Principal {
	p, _ := ctx.Value(principalKey{}).(*Principal)
	return p
}

// SystemPrincipal is used by jobs for automated transitions (R26).
func SystemPrincipal() *Principal { return &Principal{Roles: []Role{RoleSystem}} }
