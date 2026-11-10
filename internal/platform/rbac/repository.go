package rbac

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrRoleNotFound      = errors.New("role tidak ditemukan")
	ErrRoleInactive      = errors.New("role tidak aktif")
	ErrDuplicateRole     = errors.New("role dengan nama tersebut sudah ada")
	ErrAssignmentInvalid = errors.New("assignment user-role tidak valid")
	ErrSystemRole        = errors.New("role sistem tidak dapat diubah melalui manajemen role")
)

// Repository defines persistence operations for roles and permissions mapping.
type Repository interface {
	CreateRole(ctx context.Context, role Role) error
	GetRole(ctx context.Context, roleName string) (*Role, error)
	// ListRoles returns roles visible to the given tenant scope (ARY-007):
	// nil returns every role; a partner id returns system roles plus that
	// partner's own custom roles.
	ListRoles(ctx context.Context, partnerID *int) ([]Role, error)
	UpdateRole(ctx context.Context, role Role) error
	DeactivateRole(ctx context.Context, roleName string) error
	AssignPermission(ctx context.Context, roleName, permKey string) error
	RevokePermission(ctx context.Context, roleName, permKey string) error
	// SetRolePermissions atomically replaces the full permission set of a role.
	SetRolePermissions(ctx context.Context, roleName string, permKeys []string) error
	GetRolePermissions(ctx context.Context, roleName string) ([]string, error)
	// ListRolePermissionKeys returns the configured keys of a role regardless of
	// whether the role is active (used by the administration screens).
	ListRolePermissionKeys(ctx context.Context, roleName string) ([]string, error)
	AssignUserRole(ctx context.Context, assignment UserRoleAssignment) error
	GetUserRoleAssignments(ctx context.Context, userID int) ([]UserRoleAssignment, error)
}

// InMemoryRepository provides a thread-safe, fast in-memory store for RBAC.
type InMemoryRepository struct {
	mu              sync.RWMutex
	roles           map[string]Role            // roleName -> Role
	rolePermissions map[string]map[string]bool // roleName -> set of permission keys
	userAssignments map[int][]UserRoleAssignment
	nextRoleID      int
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		roles:           make(map[string]Role),
		rolePermissions: make(map[string]map[string]bool),
		userAssignments: make(map[int][]UserRoleAssignment),
		nextRoleID:      1,
	}
}

func (m *InMemoryRepository) CreateRole(ctx context.Context, role Role) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// ARY-007: uniqueness is per tenant. System roles (NULL partner) count as
	// tenant 0; two partners may both own a role with the same slug.
	for _, other := range m.roles {
		if other.Name != role.Name {
			continue
		}
		sameTenant := (role.PartnerID == nil && other.PartnerID == nil) ||
			(role.PartnerID != nil && other.PartnerID != nil && *role.PartnerID == *other.PartnerID)
		if sameTenant {
			return fmt.Errorf("%w: %s", ErrDuplicateRole, role.Name)
		}
	}

	now := time.Now().UTC()
	role.ID = m.nextRoleID
	m.nextRoleID++
	role.CreatedAt = now
	role.UpdatedAt = now
	m.roles[role.Name] = role
	if m.rolePermissions[role.Name] == nil {
		m.rolePermissions[role.Name] = make(map[string]bool)
	}
	return nil
}

func (m *InMemoryRepository) GetRole(ctx context.Context, roleName string) (*Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	role, exists := m.roles[roleName]
	if !exists {
		return nil, ErrRoleNotFound
	}
	return &role, nil
}

func (m *InMemoryRepository) ListRoles(ctx context.Context, partnerID *int) ([]Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]Role, 0, len(m.roles))
	for _, r := range m.roles {
		if partnerID != nil {
			// Tenant-scoped listing: system roles plus the tenant's own roles.
			if r.PartnerID != nil && *r.PartnerID != *partnerID {
				continue
			}
		}
		list = append(list, r)
	}
	return list, nil
}

func (m *InMemoryRepository) UpdateRole(ctx context.Context, role Role) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.roles[role.Name]
	if !exists {
		return ErrRoleNotFound
	}

	role.CreatedAt = existing.CreatedAt
	role.UpdatedAt = time.Now().UTC()
	m.roles[role.Name] = role
	return nil
}

func (m *InMemoryRepository) DeactivateRole(ctx context.Context, roleName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	role, exists := m.roles[roleName]
	if !exists {
		return ErrRoleNotFound
	}

	role.IsActive = false
	role.UpdatedAt = time.Now().UTC()
	m.roles[roleName] = role
	return nil
}

func (m *InMemoryRepository) AssignPermission(ctx context.Context, roleName, permKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	role, exists := m.roles[roleName]
	if !exists {
		return ErrRoleNotFound
	}
	if !role.IsActive {
		return ErrRoleInactive
	}

	if m.rolePermissions[roleName] == nil {
		m.rolePermissions[roleName] = make(map[string]bool)
	}
	m.rolePermissions[roleName][permKey] = true
	return nil
}

func (m *InMemoryRepository) RevokePermission(ctx context.Context, roleName, permKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.roles[roleName]; !exists {
		return ErrRoleNotFound
	}

	if perms, exists := m.rolePermissions[roleName]; exists {
		delete(perms, permKey)
	}
	return nil
}

func (m *InMemoryRepository) SetRolePermissions(ctx context.Context, roleName string, permKeys []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.roles[roleName]; !exists {
		return ErrRoleNotFound
	}
	set := make(map[string]bool, len(permKeys))
	for _, k := range permKeys {
		set[k] = true
	}
	m.rolePermissions[roleName] = set
	return nil
}

func (m *InMemoryRepository) GetRolePermissions(ctx context.Context, roleName string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	role, exists := m.roles[roleName]
	if !exists {
		return nil, ErrRoleNotFound
	}
	if !role.IsActive {
		return nil, ErrRoleInactive
	}

	permsSet := m.rolePermissions[roleName]
	list := make([]string, 0, len(permsSet))
	for k := range permsSet {
		list = append(list, k)
	}
	return list, nil
}

func (m *InMemoryRepository) ListRolePermissionKeys(ctx context.Context, roleName string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, exists := m.roles[roleName]; !exists {
		return nil, ErrRoleNotFound
	}
	list := make([]string, 0, len(m.rolePermissions[roleName]))
	for k := range m.rolePermissions[roleName] {
		list = append(list, k)
	}
	return list, nil
}

func (m *InMemoryRepository) AssignUserRole(ctx context.Context, assignment UserRoleAssignment) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	role, exists := m.roles[assignment.RoleName]
	if !exists {
		return ErrRoleNotFound
	}
	if !role.IsActive {
		return ErrRoleInactive
	}

	assignment.RoleID = role.ID
	assignment.CreatedAt = time.Now().UTC()
	m.userAssignments[assignment.UserID] = append(m.userAssignments[assignment.UserID], assignment)
	return nil
}

func (m *InMemoryRepository) GetUserRoleAssignments(ctx context.Context, userID int) ([]UserRoleAssignment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.userAssignments[userID], nil
}
