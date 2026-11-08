package service

import (
	"context"
	"fmt"

	"araya-business-hub/be/internal/modules/m1/domain"
	"araya-business-hub/be/internal/modules/m1/dto"
	"araya-business-hub/be/internal/modules/m1/repository"
	"araya-business-hub/be/internal/platform/audit"
	"araya-business-hub/be/internal/platform/rbac"
)

type UserService struct {
	userRepo repository.UserRepository
	rbacRepo rbac.Repository
	auditSvc *audit.Service
}

func NewUserService(uRepo repository.UserRepository, rRepo rbac.Repository, aSvc *audit.Service) *UserService {
	return &UserService{userRepo: uRepo, rbacRepo: rRepo, auditSvc: aSvc}
}

func (s *UserService) CreateUser(ctx context.Context, req dto.CreateUserRequest, actor string) (*domain.User, error) {
	if err := domain.ValidatePasswordStrength(req.Password); err != nil {
		return nil, err
	}
	hash, err := domain.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	u := &domain.User{
		Email: req.Email, PasswordHash: hash, Name: req.Name, Phone: req.Phone,
		DivisionID: req.DivisionID, Role: req.Role, UnitUsahaID: req.UnitUsahaID, IsActive: true,
	}
	if err := u.Validate(); err != nil {
		return nil, err
	}
	if err := s.userRepo.Create(ctx, u); err != nil {
		return nil, err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "user.created", EntityType: "user", EntityID: fmt.Sprintf("%d", u.ID),
	})
	return u, nil
}

func (s *UserService) GetUser(ctx context.Context, id int) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *UserService) ListUsers(ctx context.Context, search, div string, limit, offset int) ([]domain.User, int, error) {
	return s.userRepo.List(ctx, search, div, limit, offset)
}

func applyUserUpdates(u *domain.User, req dto.UpdateUserRequest) {
	if req.Name != nil {
		u.Name = *req.Name
	}
	if req.Phone != nil {
		u.Phone = *req.Phone
	}
	if req.DivisionID != nil {
		u.DivisionID = req.DivisionID
	}
	if req.Role != nil {
		u.Role = *req.Role
	}
	if req.UnitUsahaID != nil {
		u.UnitUsahaID = req.UnitUsahaID
	}
	if req.IsActive != nil {
		u.IsActive = *req.IsActive
	}
}

func (s *UserService) UpdateUser(ctx context.Context, id int, req dto.UpdateUserRequest, actor string) (*domain.User, error) {
	u, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	applyUserUpdates(u, req)
	if err := u.Validate(); err != nil {
		return nil, err
	}
	if err := s.userRepo.Update(ctx, u); err != nil {
		return nil, err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "user.updated", EntityType: "user", EntityID: fmt.Sprintf("%d", id),
	})
	return u, nil
}

func (s *UserService) DeactivateUser(ctx context.Context, id int, actor string) error {
	if err := s.userRepo.Deactivate(ctx, id); err != nil {
		return err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "user.deactivated", EntityType: "user", EntityID: fmt.Sprintf("%d", id),
	})
	return nil
}

func (s *UserService) AssignRole(ctx context.Context, userID int, req dto.AssignRoleRequest, actor string) error {
	assignment := rbac.UserRoleAssignment{
		UserID:      userID,
		RoleName:    req.RoleName,
		UnitUsahaID: req.UnitUsahaID,
	}
	if err := s.rbacRepo.AssignUserRole(ctx, assignment); err != nil {
		return err
	}
	u, _ := s.userRepo.GetByID(ctx, userID)
	if u != nil {
		u.Role = req.RoleName
		u.UnitUsahaID = req.UnitUsahaID
		_ = s.userRepo.Update(ctx, u)
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "user.role_assigned", EntityType: "user", EntityID: fmt.Sprintf("%d", userID),
	})
	return nil
}
