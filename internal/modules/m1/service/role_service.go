package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"araya-business-hub/be/internal/modules/m1/dto"
	"araya-business-hub/be/internal/platform/audit"
	"araya-business-hub/be/internal/platform/rbac"
)

// Permission keys that keep the access-management screens reachable.
const (
	PermRolesRead   = "m1.roles.read"
	PermRolesManage = "m1.roles.manage"
)

// ProtectedRoleName is the bootstrap administrator role. It can be edited, but
// it can never be deactivated or lose the permissions needed to manage access,
// so the platform always keeps at least one way to recover configuration.
const ProtectedRoleName = "admin"

var (
	// ErrUnknownPermission is returned for keys not registered by any module.
	ErrUnknownPermission = errors.New("permission tidak terdaftar atau tidak aktif")
	// ErrAccessLockout prevents changes that would remove access management.
	ErrAccessLockout = errors.New("perubahan ditolak: tindakan ini akan menghapus kemampuan mengelola hak akses")
)

// RoleActor identifies who is changing role configuration.
type RoleActor struct {
	Name string
	Role string
}

type RoleService struct {
	rbacRepo rbac.Repository
	policy   *rbac.PolicyService
	auditSvc *audit.Service
}

func NewRoleService(repo rbac.Repository, policy *rbac.PolicyService, auditSvc *audit.Service) *RoleService {
	return &RoleService{rbacRepo: repo, policy: policy, auditSvc: auditSvc}
}

func (s *RoleService) ListPermissions() []rbac.Permission {
	if s.policy == nil {
		return []rbac.Permission{}
	}
	perms := s.policy.ListPermissions()
	sort.Slice(perms, func(a, b int) bool { return perms[a].Key < perms[b].Key })
	return perms
}

func (s *RoleService) permissionIndex() map[string]rbac.Permission {
	registry := make(map[string]rbac.Permission)
	for _, p := range s.ListPermissions() {
		registry[p.Key] = p
	}
	return registry
}

// resolvePermissions turns stored keys into Permission objects with metadata.
func resolvePermissions(keys []string, registry map[string]rbac.Permission) []rbac.Permission {
	perms := make([]rbac.Permission, 0, len(keys))
	for _, key := range keys {
		if p, ok := registry[key]; ok {
			perms = append(perms, p)
			continue
		}
		module := key
		if idx := strings.Index(key, "."); idx > 0 {
			module = key[:idx]
		}
		perms = append(perms, rbac.Permission{Key: key, Module: module, Name: key, IsActive: false})
	}
	sort.Slice(perms, func(a, b int) bool { return perms[a].Key < perms[b].Key })
	return perms
}

func (s *RoleService) ListRoles(ctx context.Context) ([]rbac.Role, error) {
	roles, err := s.rbacRepo.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	registry := s.permissionIndex()
	for i := range roles {
		keys, err := s.storedPermissionKeys(ctx, roles[i])
		if err != nil {
			return nil, err
		}
		roles[i].Permissions = resolvePermissions(keys, registry)
	}
	sort.Slice(roles, func(a, b int) bool { return roles[a].Name < roles[b].Name })
	return roles, nil
}

// storedPermissionKeys returns the configured keys even for inactive roles so
// the administrator can still see and edit them before reactivating the role.
func (s *RoleService) storedPermissionKeys(ctx context.Context, role rbac.Role) ([]string, error) {
	return s.rbacRepo.ListRolePermissionKeys(ctx, role.Name)
}

func (s *RoleService) CreateRole(ctx context.Context, req dto.CreateRoleRequest, actor RoleActor) error {
	name, err := rbac.NormalizeRoleName(req.Name)
	if err != nil {
		return err
	}
	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		return errors.New("nama tampilan role wajib diisi")
	}
	if err := s.validatePermissionKeys(req.PermissionKeys); err != nil {
		return err
	}
	role := rbac.Role{Name: name, DisplayName: displayName, Description: strings.TrimSpace(req.Description), IsActive: true}
	if err := s.rbacRepo.CreateRole(ctx, role); err != nil {
		return err
	}
	if len(req.PermissionKeys) > 0 {
		if err := s.rbacRepo.SetRolePermissions(ctx, name, uniqueKeys(req.PermissionKeys)); err != nil {
			return err
		}
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor.Name, Action: "role.created", EntityType: "role", EntityID: name,
		Details: map[string]any{"permission_keys": uniqueKeys(req.PermissionKeys)},
	})
	return nil
}

func (s *RoleService) UpdateRole(ctx context.Context, name string, req dto.UpdateRoleRequest, actor RoleActor) error {
	role, err := s.rbacRepo.GetRole(ctx, name)
	if err != nil {
		return err
	}
	if req.IsActive != nil && !*req.IsActive && s.isGuardedRole(name, actor) {
		return ErrAccessLockout
	}
	applyRoleUpdates(role, req)
	if strings.TrimSpace(role.DisplayName) == "" {
		return errors.New("nama tampilan role wajib diisi")
	}
	if err := s.rbacRepo.UpdateRole(ctx, *role); err != nil {
		return err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor.Name, Action: "role.updated", EntityType: "role", EntityID: name,
		Details: map[string]any{"display_name": role.DisplayName, "is_active": role.IsActive},
	})
	return nil
}

func applyRoleUpdates(role *rbac.Role, req dto.UpdateRoleRequest) {
	if req.DisplayName != nil {
		role.DisplayName = strings.TrimSpace(*req.DisplayName)
	}
	if req.Description != nil {
		role.Description = strings.TrimSpace(*req.Description)
	}
	if req.IsActive != nil {
		role.IsActive = *req.IsActive
	}
}

// isGuardedRole reports whether a role must keep access-management rights:
// the bootstrap admin role and the role the actor is currently using.
func (s *RoleService) isGuardedRole(roleName string, actor RoleActor) bool {
	return roleName == ProtectedRoleName || (actor.Role != "" && strings.EqualFold(roleName, actor.Role))
}

func (s *RoleService) AssignPermission(ctx context.Context, roleName, permKey string, actor RoleActor) error {
	if err := s.validatePermissionKeys([]string{permKey}); err != nil {
		return err
	}
	if err := s.rbacRepo.AssignPermission(ctx, roleName, permKey); err != nil {
		return err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor.Name, Action: "role.permission_assigned", EntityType: "role", EntityID: roleName,
		Details: map[string]any{"permission_key": permKey},
	})
	return nil
}

func (s *RoleService) RevokePermission(ctx context.Context, roleName, permKey string, actor RoleActor) error {
	if _, err := s.rbacRepo.GetRole(ctx, roleName); err != nil {
		return err
	}
	if s.isGuardedRole(roleName, actor) && (permKey == PermRolesManage || permKey == PermRolesRead) {
		return ErrAccessLockout
	}
	if err := s.rbacRepo.RevokePermission(ctx, roleName, permKey); err != nil {
		return err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor.Name, Action: "role.permission_revoked", EntityType: "role", EntityID: roleName,
		Details: map[string]any{"permission_key": permKey},
	})
	return nil
}

// SetPermissions replaces a role's permissions with exactly permKeys. It backs
// the permission-matrix screen so a whole role can be configured in one save.
func (s *RoleService) SetPermissions(ctx context.Context, roleName string, permKeys []string, actor RoleActor) (*rbac.Role, error) {
	role, err := s.rbacRepo.GetRole(ctx, roleName)
	if err != nil {
		return nil, err
	}
	keys := uniqueKeys(permKeys)
	if err := s.validatePermissionKeys(keys); err != nil {
		return nil, err
	}
	if s.isGuardedRole(roleName, actor) && !(contains(keys, PermRolesManage) && contains(keys, PermRolesRead)) {
		return nil, ErrAccessLockout
	}
	before, err := s.storedPermissionKeys(ctx, *role)
	if err != nil {
		return nil, err
	}
	if err := s.rbacRepo.SetRolePermissions(ctx, roleName, keys); err != nil {
		return nil, err
	}
	added, removed := diffKeys(before, keys)
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor.Name, Action: "role.permissions_updated", EntityType: "role", EntityID: roleName,
		Details: map[string]any{"added": added, "removed": removed},
	})
	role.Permissions = resolvePermissions(keys, s.permissionIndex())
	return role, nil
}

func (s *RoleService) validatePermissionKeys(keys []string) error {
	for _, k := range keys {
		if s.policy == nil || !s.policy.IsRegistered(k) {
			return fmt.Errorf("%w: %s", ErrUnknownPermission, k)
		}
	}
	return nil
}

func uniqueKeys(keys []string) []string {
	seen := make(map[string]bool, len(keys))
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(strings.ToLower(k))
		if k != "" && !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func contains(keys []string, key string) bool {
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
}

func diffKeys(before, after []string) (added, removed []string) {
	added, removed = []string{}, []string{}
	for _, k := range after {
		if !contains(before, k) {
			added = append(added, k)
		}
	}
	for _, k := range before {
		if !contains(after, k) {
			removed = append(removed, k)
		}
	}
	return added, removed
}
