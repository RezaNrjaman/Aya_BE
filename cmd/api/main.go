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
	"araya-business-hub/be/internal/modules/m2"
	"araya-business-hub/be/internal/modules/m3"
	"araya-business-hub/be/internal/modules/m4"
	"araya-business-hub/be/internal/modules/m5"
	m5Client "araya-business-hub/be/internal/modules/m5/client"
	"araya-business-hub/be/internal/platform/audit"
	"araya-business-hub/be/internal/platform/config"
	"araya-business-hub/be/internal/platform/database"
	platformHttp "araya-business-hub/be/internal/platform/http"
	"araya-business-hub/be/internal/platform/logging"
	"araya-business-hub/be/internal/platform/rbac"
)

func initDB(ctx context.Context, cfg *config.Config) *database.DB {
	if cfg.DatabaseURL == "" {
		logging.Info("DATABASE_URL not set; running in local memory mode", nil)
		return nil
	}
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logging.Error("Failed connecting database", err, nil)
		if cfg.IsProduction() {
			os.Exit(1)
		}
	}
	return db
}

func initRBAC(ctx context.Context) (*rbac.Registry, rbac.Repository, *rbac.PolicyService) {
	reg := rbac.NewRegistry()
	_ = rbac.SeedPlatformPermissions(reg)
	repo := rbac.NewInMemoryRepository()
	_ = rbac.SeedInitialRoles(ctx, repo)
	return reg, repo, rbac.NewPolicyService(reg, repo)
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
	ctx := context.Background()
	db := initDB(ctx, cfg)
	defer db.Close()

	reg, rbacRepo, policySvc := initRBAC(ctx)
	auditSvc := audit.NewService(audit.NewInMemoryRepository())
	router := platformHttp.NewRouter(platformHttp.RouterDependencies{
		Config: cfg, DB: db, RBACRegistry: reg, RBACRepo: rbacRepo, PolicyService: policySvc, AuditService: auditSvc,
	})
	m1.InitModule(router, cfg.JWTSecret, cfg.JWTExpirationHours, rbacRepo, policySvc, auditSvc)
	m2Mod := m2.InitModule(router, cfg.JWTSecret, policySvc, auditSvc)
	m3.InitModule(router, cfg.JWTSecret, policySvc, auditSvc)
	m4Mod := m4.InitModule(router, cfg.JWTSecret, policySvc, auditSvc)

	m2Cli := m5Client.NewInternalM2Adapter(m2Mod.UnitSvc)
	m4Cli := m5Client.NewInternalM4Adapter(m4Mod.ProductSvc, m4Mod.OrderSvc)
	m5.InitModule(router, cfg.JWTSecret, policySvc, auditSvc, m2Cli, m4Cli)

	srv := &http.Server{Addr: fmt.Sprintf(":%s", cfg.Port), Handler: router}
	startServer(srv)
	waitForShutdown(srv)
}
