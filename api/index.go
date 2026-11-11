package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"

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
	"araya-business-hub/be/internal/platform/rbac"

	"github.com/gin-gonic/gin"
)

var (
	once       sync.Once
	appHandler http.Handler
	initErr    error
)

func buildHandler() (http.Handler, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return nil, errors.New("DATABASE_URL wajib diatur pada Vercel Environment Variables")
	}
	db, err := database.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	registry := rbac.NewRegistry()
	_ = rbac.SeedPlatformPermissions(registry)
	rbacRepo := rbac.NewPostgresRepository(db.Pool)
	if err := rbac.SeedInitialRoles(context.Background(), rbacRepo); err != nil {
		db.Close()
		return nil, err
	}
	policy := rbac.NewPolicyService(registry, rbacRepo)
	auditSvc := audit.NewService(audit.NewPostgresRepository(db.Pool))

	router := platformHttp.NewRouter(platformHttp.RouterDependencies{
		Config: cfg, DB: db, RBACRegistry: registry, RBACRepo: rbacRepo,
		PolicyService: policy, AuditService: auditSvc,
	})

	userRepo := m1Repo.NewPostgresUserRepository(db.Pool)
	unitRepo := m2Repo.NewPostgresUnitRepository(db.Pool)
	partnerRepo := m3Repo.NewPostgresPartnerRepository(db.Pool)
	productRepo := m4Repo.NewPostgresRepository(db.Pool)
	transactionRepo := m5Repo.NewPostgresRepository(db.Pool)

	m1Module := m1.InitModule(router, cfg.JWTSecret, cfg.JWTExpirationHours, rbacRepo, policy, auditSvc, userRepo)
	m2Module := m2.InitModule(router, cfg.JWTSecret, policy, auditSvc, unitRepo)
	m3Module := m3.InitModule(router, cfg.JWTSecret, policy, auditSvc, m1Module.Provisioner, partnerRepo)
	m4Module := m4.InitModule(router, cfg.JWTSecret, policy, auditSvc, m3Module.Svc, productRepo)
	if provider, ok := m4Module.Repo.(m2Service.DashboardDataProvider); ok {
		m2Module.DashboardSvc.SetDataProvider(provider)
	}
	m2Client := m5Client.NewInternalM2Adapter(m2Module.UnitSvc)
	m4Client := m5Client.NewInternalM4Adapter(m4Module.ProductSvc, m4Module.OrderSvc)
	m5.InitModule(router, cfg.JWTSecret, policy, auditSvc, m2Client, m4Client, transactionRepo, m3Module.Svc)

	// Vercel reuses warm instances. Keep the database pool alive with this handler.
	return router, nil
}

// Handler is the Vercel Go Function entry point.
// vercel.json rewrites public paths to this function and passes the original path
// in __vercel_path so Gin can continue matching the existing /api/... routes.
func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(func() {
		appHandler, initErr = buildHandler()
		if initErr != nil {
			log.Printf("ARAYA API initialization failed: %v", initErr)
		}
	})
	if initErr != nil || appHandler == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error": map[string]string{
				"code":    "SERVICE_UNAVAILABLE",
				"message": "API gagal diinisialisasi. Periksa environment variables dan koneksi database.",
			},
		})
		return
	}

	if originalPath := r.URL.Query().Get("__vercel_path"); originalPath != "" {
		query := r.URL.Query()
		query.Del("__vercel_path")
		r.URL.Path = originalPath
		r.URL.RawPath = ""
		r.URL.RawQuery = query.Encode()
	}
	if r.URL.Path == "" || r.URL.Path == "/api" {
		r.URL.Path = "/"
	}
	appHandler.ServeHTTP(w, r)
}
