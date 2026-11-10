package rbac

import (
	"context"
	"errors"
	"strings"
)

// ErrAccessRevoked is returned when the account behind a token no longer exists
// or has been deactivated.
var ErrAccessRevoked = errors.New("akses akun telah dicabut atau dinonaktifkan")

// UserAccessResolver returns the current role and status of a user from the
// owning module (M1). It lets authorization follow role changes and account
// deactivation immediately instead of waiting for the JWT to expire.
type UserAccessResolver interface {
	ResolveUserAccess(ctx context.Context, userID int) (role string, active bool, err error)
}

// PolicyService evaluates permissions and scopes for requests.
type PolicyService struct {
	registry *Registry
	repo     Repository
	users    UserAccessResolver
}

// NewPolicyService initializes an authorization policy evaluator.
func NewPolicyService(registry *Registry, repo Repository) *PolicyService {
	return &PolicyService{
		registry: registry,
		repo:     repo,
	}
}

// SetUserResolver enables live lookup of a user's current role and status.
func (p *PolicyService) SetUserResolver(r UserAccessResolver) {
	p.users = r
}

// CurrentRole returns the role that must be used to authorize the request. When
// a resolver is configured the stored role wins over the role in the token, and
// deactivated or deleted accounts are rejected with ErrAccessRevoked.
func (p *PolicyService) CurrentRole(ctx context.Context, userID int, tokenRole string) (string, error) {
	if p.users == nil {
		return tokenRole, nil
	}
	role, active, err := p.users.ResolveUserAccess(ctx, userID)
	if err != nil {
		return "", err
	}
	if !active {
		return "", ErrAccessRevoked
	}
	return role, nil
}

// CanRolePerform checks if a given role possesses an active permission key.
// Role-permission changes take effect on the next request because the mapping
// is read from the repository every time. There is no name-based bypass: every
// role, including administrators, only has the permissions mapped to it.
func (p *PolicyService) CanRolePerform(ctx context.Context, roleName, permissionKey string) bool {
	if roleName == "" || permissionKey == "" {
		return false
	}

	// 1. Verify the permission exists and is globally active in Registry
	if !p.IsRegistered(permissionKey) {
		return false
	}

	// 2. Fetch live permissions mapped to this role (inactive roles grant nothing)
	permissions, err := p.repo.GetRolePermissions(ctx, roleName)
	if err != nil {
		return false
	}

	for _, k := range permissions {
		if strings.EqualFold(k, permissionKey) {
			return true
		}
	}

	return false
}

// IsRegistered reports whether a permission key is registered and active.
func (p *PolicyService) IsRegistered(permissionKey string) bool {
	if p.registry == nil {
		return false
	}
	perm, err := p.registry.Get(permissionKey)
	return err == nil && perm.IsActive
}

// IsScopePermitted checks if an action targeting a specific unit_usaha_id is permitted for the user.
// If userUnitUsahaID is nil, the user has no unit assignment: only administrators
// reach beyond a single unit; every other role is denied (ARY-003).
// If userUnitUsahaID is not nil, it must match the target targetUnitUsahaID.
func (p *PolicyService) IsScopePermitted(userRole string, userUnitUsahaID *int, targetUnitUsahaID int) bool {
	if strings.EqualFold(userRole, "admin") {
		return true
	}

	if userUnitUsahaID == nil {
		// ARY-003: a missing unit assignment must never be treated as global
		// scope for non-admin roles.
		return false
	}

	return *userUnitUsahaID == targetUnitUsahaID
}

// ListPermissions returns all registered permissions in the platform.
func (p *PolicyService) ListPermissions() []Permission {
	if p.registry == nil {
		return []Permission{}
	}
	return p.registry.List()
}
