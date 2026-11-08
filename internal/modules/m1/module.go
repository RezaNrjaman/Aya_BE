package m1

import (
	"context"

	"araya-business-hub/be/internal/modules/m1/domain"
	"araya-business-hub/be/internal/modules/m1/handler"
	"araya-business-hub/be/internal/modules/m1/repository"
	"araya-business-hub/be/internal/modules/m1/service"
	"araya-business-hub/be/internal/platform/audit"
	"araya-business-hub/be/internal/platform/rbac"
	"github.com/gin-gonic/gin"
)

type Module struct {
	UserRepo repository.UserRepository
	AuthSvc  *service.AuthService
	UserSvc  *service.UserService
	RoleSvc  *service.RoleService
}

func resolveUserRepo(opts ...repository.UserRepository) repository.UserRepository {
	if len(opts) > 0 && opts[0] != nil {
		return opts[0]
	}
	return repository.NewInMemoryUserRepository()
}

func InitModule(r *gin.Engine, secret string, exp int, rRepo rbac.Repository, pSvc *rbac.PolicyService, aSvc *audit.Service, uRepoOpt ...repository.UserRepository) *Module {
	uRepo := resolveUserRepo(uRepoOpt...)
	authSvc := service.NewAuthService(uRepo, rRepo, aSvc, secret, exp)
	userSvc := service.NewUserService(uRepo, rRepo, aSvc)
	roleSvc := service.NewRoleService(rRepo, pSvc, aSvc)

	h := Handlers{
		Auth:  handler.NewAuthHandler(authSvc),
		User:  handler.NewUserHandler(userSvc),
		Role:  handler.NewRoleHandler(roleSvc),
		Audit: handler.NewAuditHandler(aSvc),
	}
	RegisterRoutes(r, secret, pSvc, h)

	_ = SeedInitialAdmin(context.Background(), uRepo)

	return &Module{UserRepo: uRepo, AuthSvc: authSvc, UserSvc: userSvc, RoleSvc: roleSvc}
}

// SeedInitialAdmin seeds the primary administrative account on bootstrap.
func SeedInitialAdmin(ctx context.Context, uRepo repository.UserRepository) error {
	hash, _ := domain.HashPassword("Admin1234!")
	admin := &domain.User{
		Email:        "admin@araya.com",
		PasswordHash: hash,
		Name:         "Administrator Utama",
		Role:         "admin",
		DivisionName: "operasional",
		IsActive:     true,
	}
	return uRepo.Create(ctx, admin)
}
