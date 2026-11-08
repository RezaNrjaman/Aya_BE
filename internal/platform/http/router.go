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

// NewRouter sets up the Gin router with all global middlewares and routes.
func NewRouter(deps RouterDependencies) *gin.Engine {
	if deps.Config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Global middlewares
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS(deps.Config.AllowedOrigins))

	// Handle 404 with standard API envelope
	r.NoRoute(func(c *gin.Context) {
		response.HandleAppError(c, response.NotFound("Endpoint tidak ditemukan"))
	})

	// Handle 405 Method Not Allowed
	r.NoMethod(func(c *gin.Context) {
		response.Error(c, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
	})

	// Root simple health check (e.g. for container / load balancer probes)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Platform API Health Check
	apiV1 := r.Group("/api/v1")
	{
		apiV1.GET("/health", func(c *gin.Context) {
			dbStatus := "disconnected"
			if deps.DB != nil {
				ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
				defer cancel()
				if err := deps.DB.Ping(ctx); err == nil {
					dbStatus = "connected"
				} else {
					dbStatus = "unreachable"
				}
			}

			response.OK(c, gin.H{
				"status":      "running",
				"environment": deps.Config.AppEnv,
				"database":    dbStatus,
				"time":        time.Now().UTC().Format(time.RFC3339),
			})
		})

		// Public or Protected RBAC Metadata endpoints (useful for UI Permission Matrix)
		if deps.RBACRegistry != nil {
			apiV1.GET("/permissions", func(c *gin.Context) {
				list := deps.RBACRegistry.List()
				response.OK(c, list)
			})
		}
	}

	return r
}
