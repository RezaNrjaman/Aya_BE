package rbac

import (
	"context"
	"strings"
)

// PolicyService evaluates permissions and scopes for requests.
type PolicyService struct {
	registry *Registry
	repo     Repository
}

// NewPolicyService initializes an authorization policy evaluator.
func NewPolicyService(registry *Registry, repo Repository) *PolicyService {
	return &PolicyService{
		registry: registry,
		repo:     repo,
	}
}

// CanRolePerform checks if a given role possesses an active permission key.
// Resolves OD-01: changes made to role permissions take effect immediately upon evaluation.
func (p *PolicyService) CanRolePerform(ctx context.Context, roleName, permissionKey string) bool {
	if roleName == "" || permissionKey == "" {
		return false
	}

	// Super-admin escape hatch (optional standard practice for administrative bypass)
	if strings.ToLower(roleName) == "superadmin" {
		return true
	}

	// 1. Verify the permission exists and is globally active in Registry
	perm, err := p.registry.Get(permissionKey)
	if err != nil || !perm.IsActive {
		return false
	}

	// 2. Fetch live permissions mapped to this role from repository
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

// IsScopePermitted checks if an action targeting a specific unit_usaha_id is permitted for the user.
// If userUnitUsahaID is nil, the user has global scope (e.g. administrator).
// If userUnitUsahaID is not nil, it must match the target targetUnitUsahaID.
func (p *PolicyService) IsScopePermitted(userRole string, userUnitUsahaID *int, targetUnitUsahaID int) bool {
	// Superadmin / Admin has global cross-unit scope
	lowerRole := strings.ToLower(userRole)
	if lowerRole == "admin" || lowerRole == "superadmin" {
		return true
	}

	if userUnitUsahaID == nil {
		return true
	}

	return *userUnitUsahaID == targetUnitUsahaID
}
