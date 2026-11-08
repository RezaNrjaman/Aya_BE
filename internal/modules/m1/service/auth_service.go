package service

import (
	"context"
	"errors"

	"araya-business-hub/be/internal/modules/m1/domain"
	"araya-business-hub/be/internal/modules/m1/dto"
	"araya-business-hub/be/internal/modules/m1/repository"
	"araya-business-hub/be/internal/platform/audit"
	"araya-business-hub/be/internal/platform/rbac"
	"araya-business-hub/be/internal/platform/security"
)

var ErrInvalidCredentials = errors.New("email atau kata sandi tidak sesuai")

type AuthService struct {
	userRepo  repository.UserRepository
	rbacRepo  rbac.Repository
	auditSvc  *audit.Service
	jwtSecret string
	expHours  int
}

func NewAuthService(uRepo repository.UserRepository, rRepo rbac.Repository, aSvc *audit.Service, secret string, exp int) *AuthService {
	return &AuthService{
		userRepo:  uRepo,
		rbacRepo:  rRepo,
		auditSvc:  aSvc,
		jwtSecret: secret,
		expHours:  exp,
	}
}

func (s *AuthService) verifyUserPassword(user *domain.User, password string) error {
	if !domain.CheckPassword(password, user.PasswordHash) {
		return ErrInvalidCredentials
	}
	if !user.CanLogin() {
		return domain.ErrUserInactive
	}
	return nil
}

func (s *AuthService) generateUserJWT(user *domain.User) (string, error) {
	claims := security.PlatformClaims{
		UserID:      user.ID,
		Name:        user.Name,
		Role:        user.Role,
		Division:    user.DivisionName,
		UnitUsahaID: user.UnitUsahaID,
	}
	return security.GenerateToken(claims, s.jwtSecret, s.expHours)
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if err := s.verifyUserPassword(user, req.Password); err != nil {
		return nil, err
	}
	token, err := s.generateUserJWT(user)
	if err != nil {
		return nil, err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserID: &user.ID, UserName: user.Name, Action: "user.login", EntityType: "user", EntityID: user.Email,
	})
	return &dto.LoginResponse{
		Token: token,
		User: dto.UserSummary{
			ID: user.ID, Email: user.Email, Name: user.Name, Role: user.Role, Division: user.DivisionName, UnitUsahaID: user.UnitUsahaID,
		},
	}, nil
}

func (s *AuthService) GetProfile(ctx context.Context, userID int) (*dto.ProfileResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	perms, _ := s.rbacRepo.GetRolePermissions(ctx, user.Role)
	if perms == nil {
		perms = []string{}
	}
	return &dto.ProfileResponse{
		ID: user.ID, Email: user.Email, Name: user.Name, Role: user.Role, Division: user.DivisionName, UnitUsahaID: user.UnitUsahaID, Permissions: perms,
	}, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID int, req dto.ChangePasswordRequest) error {
	if err := domain.ValidatePasswordStrength(req.NewPassword); err != nil {
		return err
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if !domain.CheckPassword(req.OldPassword, user.PasswordHash) {
		return domain.ErrPasswordMismatch
	}
	newHash, err := domain.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	return s.userRepo.UpdatePassword(ctx, userID, newHash)
}
