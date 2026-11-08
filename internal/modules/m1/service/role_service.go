package service

import (
	"context"

	"araya-business-hub/be/internal/modules/m1/dto"
	"araya-business-hub/be/internal/platform/audit"
	"araya-business-hub/be/internal/platform/rbac"
)

type RoleService struct {
	rbacRepo rbac.Repository
	auditSvc *audit.Service
}

func NewRoleService(repo rbac.Repository, auditSvc *audit.Service) *RoleService {
	return &RoleService{rbacRepo: repo, auditSvc: auditSvc}
}

func (s *RoleService) ListRoles(ctx context.Context) ([]rbac.Role, error) {
	return s.rbacRepo.ListRoles(ctx)
}

func (s *RoleService) CreateRole(ctx context.Context, req dto.CreateRoleRequest, actor string) error {
	role := rbac.Role{
		Name: req.Name, DisplayName: req.DisplayName, Description: req.Description, IsActive: true,
	}
	if err := s.rbacRepo.CreateRole(ctx, role); err != nil {
		return err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "role.created", EntityType: "role", EntityID: req.Name,
	})
	return nil
}

func (s *RoleService) UpdateRole(ctx context.Context, name string, req dto.UpdateRoleRequest, actor string) error {
	role, err := s.rbacRepo.GetRole(ctx, name)
	if err != nil {
		return err
	}
	if req.DisplayName != nil {
		role.DisplayName = *req.DisplayName
	}
	if req.Description != nil {
		role.Description = *req.Description
	}
	if req.IsActive != nil {
		role.IsActive = *req.IsActive
	}
	if err := s.rbacRepo.UpdateRole(ctx, *role); err != nil {
		return err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "role.updated", EntityType: "role", EntityID: name,
	})
	return nil
}

func (s *RoleService) AssignPermission(ctx context.Context, roleName, permKey, actor string) error {
	if err := s.rbacRepo.AssignPermission(ctx, roleName, permKey); err != nil {
		return err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "role.permission_assigned", EntityType: "role", EntityID: roleName,
		Details: map[string]any{"permission_key": permKey},
	})
	return nil
}

func (s *RoleService) RevokePermission(ctx context.Context, roleName, permKey, actor string) error {
	if err := s.rbacRepo.RevokePermission(ctx, roleName, permKey); err != nil {
		return err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "role.permission_revoked", EntityType: "role", EntityID: roleName,
		Details: map[string]any{"permission_key": permKey},
	})
	return nil
}
