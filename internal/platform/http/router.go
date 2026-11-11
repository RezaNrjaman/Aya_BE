package http

import (
	"context"
	"net/http"
	"time"

	"araya-business-hub/be/internal/platform/audit"
	"araya-business-hub/be/internal/platform/config"
	"araya-business-hub/be/internal/platform/database"
	"araya-business-hub/be/internal/platform/middleware"
	"araya-business-hub/be/internal/platform/rbac"
	"araya-business-hub/be/internal/platform/response"

	"github.com/gin-gonic/gin"
)

type RouterDependencies struct {
	Config        *config.Config
	DB            *database.DB
	RBACRegistry  *rbac.Registry
	RBACRepo      rbac.Repository
	PolicyService *rbac.PolicyService
	AuditService  *audit.Service
}

// NewRouter builds the HTTP router and registers shared platform routes.
func NewRouter(deps RouterDependencies) *gin.Engine {
	setGinMode(deps.Config)
	router := gin.New()
	applyMiddlewares(router, deps.Config)
	registerErrorHandlers(router)
	registerHealthRoutes(router, deps)
	registerPermissionMetadata(router, deps.RBACRegistry)
	return router
}

func setGinMode(cfg *config.Config) {
	if cfg.IsProductionLike() {
		gin.SetMode(gin.ReleaseMode)
	}
}

func applyMiddlewares(router *gin.Engine, cfg *config.Config) {
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS(cfg.AllowedOrigins))
}

func registerErrorHandlers(router *gin.Engine) {
	router.NoRoute(func(c *gin.Context) {
		response.HandleAppError(c, response.NotFound("Endpoint tidak ditemukan"))
	})
	router.NoMethod(func(c *gin.Context) {
		response.Error(c, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
	})
}

func registerHealthRoutes(router *gin.Engine, deps RouterDependencies) {
	registerRootHealth(router)
	registerAPIHealth(router, deps)
}

func registerRootHealth(router *gin.Engine) {
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"service":    "araya-business-hub-api",
			"status":     "running",
			"health":     "/health",
			"api_health": "/api/v1/health",
		})
	})
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})
}

func registerAPIHealth(router *gin.Engine, deps RouterDependencies) {
	router.Group("/api/v1").GET("/health", apiHealthHandler(deps))
}

func apiHealthHandler(deps RouterDependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		response.OK(c, gin.H{
			"status": "running", "environment": deps.Config.AppEnv,
			"database": databaseStatus(c, deps.DB),
			"time":     time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func databaseStatus(c *gin.Context, db *database.DB) string {
	if db == nil {
		return "disconnected"
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if err := db.Ping(ctx); err != nil {
		return "unreachable"
	}
	return "connected"
}

func registerPermissionMetadata(router *gin.Engine, registry *rbac.Registry) {
	if registry == nil {
		return
	}
	router.Group("/api/v1").GET("/permissions", func(c *gin.Context) {
		response.OK(c, registry.List())
	})
}
