package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"araya-business-hub/be/internal/modules/m1"
	m1Repo "araya-business-hub/be/internal/modules/m1/repository"
	"araya-business-hub/be/internal/modules/m2"
	m2Repo "araya-business-hub/be/internal/modules/m2/repository"
	m2Service "araya-business-hub/be/internal/modules/m2/service"
	"araya-business-hub/be/internal/modules/m3"
	m3Repo "araya-business-hub/be/internal/modules/m3/repository"
	"araya-business-hub/be/internal/modules/m4"
	m4Repo "araya-business-hub/be/internal/modules/m4/repository"
	"araya-business-hub/be/internal/modules/m5"
	m5Client "araya-business-hub/be/internal/modules/m5/client"
	m5Repo "araya-business-hub/be/internal/modules/m5/repository"
	"araya-business-hub/be/internal/platform/audit"
	"araya-business-hub/be/internal/platform/config"
	"araya-business-hub/be/internal/platform/database"
	platformHttp "araya-business-hub/be/internal/platform/http"
	"araya-business-hub/be/internal/platform/logging"
	"araya-business-hub/be/internal/platform/rbac"

	"github.com/gin-gonic/gin"
)

func initDB(ctx context.Context, cfg *config.Config) *database.DB {
	if cfg.DatabaseURL == "" {
		logging.Info("DATABASE_URL not set; running in local memory mode", nil)
		return nil
	}
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logging.Error("Failed connecting database", err, nil)
		if cfg.IsProductionLike() {
			os.Exit(1)
		}
	}
	return db
}

func initRBAC(ctx context.Context, db *database.DB) (*rbac.Registry, rbac.Repository, *rbac.PolicyService) {
	reg := rbac.NewRegistry()
	_ = rbac.SeedPlatformPermissions(reg)
	var repo rbac.Repository = rbac.NewInMemoryRepository()
	if db != nil {
		repo = rbac.NewPostgresRepository(db.Pool)
	}
	_ = rbac.SeedInitialRoles(ctx, repo)
	return reg, repo, rbac.NewPolicyService(reg, repo)
}

func initAudit(db *database.DB) *audit.Service {
	if db != nil {
		return audit.NewService(audit.NewPostgresRepository(db.Pool))
	}
	return audit.NewService(audit.NewInMemoryRepository())
}

func resolveDBRepos(db *database.DB) (m1Repo.UserRepository, m2Repo.UnitRepository, m3Repo.PartnerRepository, m4Repo.Repository, m5Repo.Repository) {
	if db == nil {
		return nil, nil, nil, nil, nil
	}
	return m1Repo.NewPostgresUserRepository(db.Pool),
		m2Repo.NewPostgresUnitRepository(db.Pool),
		m3Repo.NewPostgresPartnerRepository(db.Pool),
		m4Repo.NewPostgresRepository(db.Pool),
		m5Repo.NewPostgresRepository(db.Pool)
}

func initModules(r *gin.Engine, cfg *config.Config, rRepo rbac.Repository, pSvc *rbac.PolicyService, aSvc *audit.Service, db *database.DB) {
	uR, buR, pR, m4R, m5R := resolveDBRepos(db)
	m1Mod := m1.InitModule(r, cfg.JWTSecret, cfg.JWTExpirationHours, rRepo, pSvc, aSvc, uR)
	m2Mod := m2.InitModule(r, cfg.JWTSecret, pSvc, aSvc, buR)
	m3Mod := m3.InitModule(r, cfg.JWTSecret, pSvc, aSvc, m1Mod.Provisioner, pR)
	// ARY-004: m3's partner status gates the public storefront in m4.
	m4Mod := m4.InitModule(r, cfg.JWTSecret, pSvc, aSvc, m3Mod.Svc, m4R)
	if provider, ok := m4Mod.Repo.(m2Service.DashboardDataProvider); ok {
		m2Mod.DashboardSvc.SetDataProvider(provider)
	}

	m2Cli := m5Client.NewInternalM2Adapter(m2Mod.UnitSvc)
	m4Cli := m5Client.NewInternalM4Adapter(m4Mod.ProductSvc, m4Mod.OrderSvc)
	m5.InitModule(r, cfg.JWTSecret, pSvc, aSvc, m2Cli, m4Cli, m5R, m3Mod.Svc)
}

func startServer(srv *http.Server) {
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logging.Error("Server terminated unexpectedly", err, nil)
			os.Exit(1)
		}
	}()
}

func waitForShutdown(srv *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logging.Error("Server forced shutdown", err, nil)
	}
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		logging.Error("Configuration failure", err, nil)
		os.Exit(1)
	}
	run(cfg)
}

func run(cfg *config.Config) {
	ctx := context.Background()
	db := initDB(ctx, cfg)
	defer db.Close()
	reg, rbacRepo, policySvc := initRBAC(ctx, db)
	auditSvc := initAudit(db)
	router := platformHttp.NewRouter(platformHttp.RouterDependencies{
		Config: cfg, DB: db, RBACRegistry: reg, RBACRepo: rbacRepo, PolicyService: policySvc, AuditService: auditSvc,
	})
	initModules(router, cfg, rbacRepo, policySvc, auditSvc, db)
	srv := &http.Server{Addr: fmt.Sprintf(":%s", cfg.Port), Handler: router}
	startServer(srv)
	waitForShutdown(srv)
}
