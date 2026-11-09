package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"araya-business-hub/be/internal/modules/m1/domain"
	"araya-business-hub/be/internal/modules/m1/dto"
	"araya-business-hub/be/internal/modules/m1/repository"
	"araya-business-hub/be/internal/platform/audit"
	"araya-business-hub/be/internal/platform/rbac"
)

var (
	// ErrRoleChangeForbidden is returned when the actor tries to set a role or
	// scope without holding m1.roles.manage.
	ErrRoleChangeForbidden = errors.New("mengubah role atau scope pengguna memerlukan izin m1.roles.manage")
	// ErrSelfAccessChange prevents administrators from locking themselves out.
	ErrSelfAccessChange = errors.New("tidak dapat mengubah role, scope, atau status akun sendiri")
	// ErrRoleUnavailable is returned for unknown or inactive roles.
	ErrRoleUnavailable = errors.New("role tidak ditemukan atau tidak aktif")
)

// Actor describes the authenticated user performing a user-management action.
type Actor struct {
	UserID         int
	Name           string
	Role           string
	PartnerID      *int
	CanManageRoles bool
}

// PartnerScoped reports whether the actor may only see and manage members of a
// single partner. Partner owners and their members must never see other tenants.
func (a Actor) PartnerScoped() bool {
	return a.PartnerID != nil || strings.EqualFold(a.Role, "partner")
}

func (a Actor) partnerFilter() *int {
	if a.PartnerScoped() {
		return a.PartnerID
	}
	return nil
}

type UserService struct {
	userRepo repository.UserRepository
	rbacRepo rbac.Repository
	auditSvc *audit.Service
}

func NewUserService(uRepo repository.UserRepository, rRepo rbac.Repository, aSvc *audit.Service) *UserService {
	return &UserService{userRepo: uRepo, rbacRepo: rRepo, auditSvc: aSvc}
}

func (s *UserService) CreateUser(ctx context.Context, req dto.CreateUserRequest, actor Actor) (*domain.User, error) {
	if actor.PartnerScoped() && actor.PartnerID == nil {
		return nil, fmt.Errorf("akun mitra belum terhubung ke data partner")
	}
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
	if actor.PartnerScoped() {
		// Partner-created accounts always belong to the actor's partner and
		// cannot escalate to an internal role.
		u.PartnerID = actor.PartnerID
		u.Role = "partner"
		u.DivisionID = nil
		u.UnitUsahaID = nil
	} else if err := s.checkRoleChange(ctx, actor, u.Role, u.UnitUsahaID); err != nil {
		return nil, err
	}
	if err := u.Validate(); err != nil {
		return nil, err
	}
	if err := s.userRepo.Create(ctx, u); err != nil {
		return nil, err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor.Name, Action: "user.created", EntityType: "user", EntityID: fmt.Sprintf("%d", u.ID),
	})
	return u, nil
}

// ensureRoleUsable verifies that a role exists and is active.
func (s *UserService) ensureRoleUsable(ctx context.Context, roleName string) error {
	role, err := s.rbacRepo.GetRole(ctx, roleName)
	if err != nil {
		if errors.Is(err, rbac.ErrRoleNotFound) {
			return ErrRoleUnavailable
		}
		return err
	}
	if !role.IsActive {
		return ErrRoleUnavailable
	}
	return nil
}

// checkRoleChange enforces that only actors with m1.roles.manage can set a
// role or business-unit scope, and that the target role is usable.
func (s *UserService) checkRoleChange(ctx context.Context, actor Actor, roleName string, unitID *int) error {
	if roleName == "" && unitID == nil {
		return nil
	}
	if !actor.CanManageRoles {
		return ErrRoleChangeForbidden
	}
	if roleName == "" {
		return nil
	}
	return s.ensureRoleUsable(ctx, roleName)
}

func (s *UserService) getScopedUser(ctx context.Context, id int, actor Actor) (*domain.User, error) {
	u, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if actor.PartnerScoped() {
		if actor.PartnerID == nil || u.PartnerID == nil || *u.PartnerID != *actor.PartnerID {
			return nil, domain.ErrUserNotFound
		}
	}
	return u, nil
}

func (s *UserService) GetUser(ctx context.Context, id int, actor Actor) (*domain.User, error) {
	return s.getScopedUser(ctx, id, actor)
}

// ResolvePartnerID returns the partner linkage for a user account. It is used as
// a fallback for tokens issued before partner_id was added to the JWT claims.
func (s *UserService) ResolvePartnerID(ctx context.Context, userID int) *int {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil
	}
	return u.PartnerID
}

func (s *UserService) ListUsers(ctx context.Context, search, div string, actor Actor, limit, offset int) ([]domain.User, int, error) {
	if actor.PartnerScoped() && actor.PartnerID == nil {
		return []domain.User{}, 0, nil
	}
	return s.userRepo.List(ctx, search, div, actor.partnerFilter(), limit, offset)
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

func (s *UserService) checkUserUpdate(ctx context.Context, id int, req dto.UpdateUserRequest, actor Actor) error {
	changesAccess := req.Role != nil || req.UnitUsahaID != nil || req.IsActive != nil
	if changesAccess && actor.UserID != 0 && actor.UserID == id {
		return ErrSelfAccessChange
	}
	if actor.PartnerScoped() {
		return nil
	}
	role := ""
	if req.Role != nil {
		role = *req.Role
	}
	return s.checkRoleChange(ctx, actor, role, req.UnitUsahaID)
}

func (s *UserService) UpdateUser(ctx context.Context, id int, req dto.UpdateUserRequest, actor Actor) (*domain.User, error) {
	u, err := s.getScopedUser(ctx, id, actor)
	if err != nil {
		return nil, err
	}
	if err := s.checkUserUpdate(ctx, id, req, actor); err != nil {
		return nil, err
	}
	applyUserUpdates(u, req)
	if actor.PartnerScoped() {
		u.PartnerID = actor.PartnerID
		u.Role = "partner"
		u.DivisionID = nil
		u.UnitUsahaID = nil
	}
	if err := u.Validate(); err != nil {
		return nil, err
	}
	if err := s.userRepo.Update(ctx, u); err != nil {
		return nil, err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor.Name, Action: "user.updated", EntityType: "user", EntityID: fmt.Sprintf("%d", id),
	})
	return u, nil
}

func (s *UserService) DeactivateUser(ctx context.Context, id int, actor Actor) error {
	if actor.UserID != 0 && actor.UserID == id {
		return ErrSelfAccessChange
	}
	u, err := s.getScopedUser(ctx, id, actor)
	if err != nil {
		return err
	}
	if err := s.userRepo.Deactivate(ctx, u.ID); err != nil {
		return err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor.Name, Action: "user.deactivated", EntityType: "user", EntityID: fmt.Sprintf("%d", id),
	})
	return nil
}

func (s *UserService) AssignRole(ctx context.Context, userID int, req dto.AssignRoleRequest, actor Actor) error {
	u, err := s.getScopedUser(ctx, userID, actor)
	if err != nil {
		return err
	}
	if actor.UserID != 0 && actor.UserID == userID {
		return ErrSelfAccessChange
	}
	if actor.PartnerScoped() && !strings.EqualFold(req.RoleName, "partner") {
		return fmt.Errorf("mitra hanya dapat menetapkan role partner")
	}
	if err := s.ensureRoleUsable(ctx, req.RoleName); err != nil {
		return err
	}
	previousRole, previousUnit := u.Role, u.UnitUsahaID
	u.Role = req.RoleName
	u.UnitUsahaID = req.UnitUsahaID
	if actor.PartnerScoped() {
		u.PartnerID = actor.PartnerID
		u.UnitUsahaID = nil
	}
	// Update replaces the user's effective assignment atomically (ADR-001: one
	// effective role and scope per session).
	if err := s.userRepo.Update(ctx, u); err != nil {
		return err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor.Name, Action: "user.role_assigned", EntityType: "user", EntityID: fmt.Sprintf("%d", userID),
		Details: map[string]any{
			"previous_role": previousRole, "new_role": u.Role,
			"previous_unit_usaha_id": previousUnit, "new_unit_usaha_id": u.UnitUsahaID,
		},
	})
	return nil
}

// ResolveUserAccess implements rbac.UserAccessResolver so that authorization
// always uses the user's current role and account status.
func (s *UserService) ResolveUserAccess(ctx context.Context, userID int) (string, bool, error) {
	u, err := s.userRepo.GetByID(ctx, userID)
	if errors.Is(err, domain.ErrUserNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return u.Role, u.IsActive, nil
}
