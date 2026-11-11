package m1

import (
	"time"

	"araya-business-hub/be/internal/modules/m1/handler"
	"araya-business-hub/be/internal/platform/middleware"
	"araya-business-hub/be/internal/platform/rbac"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
	Auth  *handler.AuthHandler
	User  *handler.UserHandler
	Role  *handler.RoleHandler
	Audit *handler.AuditHandler
}

func registerUserRoutes(rg *gin.RouterGroup, policy *rbac.PolicyService, h *handler.UserHandler) {
	rg.GET("", middleware.RequirePermission(policy, "m1.users.read"), h.ListUsers)
	rg.POST("", middleware.RequirePermission(policy, "m1.users.create"), h.CreateUser)
	rg.GET("/:id", middleware.RequirePermission(policy, "m1.users.read"), h.GetUser)
	rg.PATCH("/:id", middleware.RequirePermission(policy, "m1.users.update"), h.UpdateUser)
	rg.DELETE("/:id", middleware.RequirePermission(policy, "m1.users.delete"), h.DeactivateUser)
	rg.POST("/:id/roles", middleware.RequirePermission(policy, "m1.roles.manage"), h.AssignRole)
}

func registerRoleRoutes(rg *gin.RouterGroup, policy *rbac.PolicyService, h *handler.RoleHandler) {
	rg.GET("", middleware.RequirePermission(policy, "m1.roles.read"), h.ListRoles)
	rg.GET("/permissions", middleware.RequirePermission(policy, "m1.roles.read"), h.ListPermissions)
	rg.POST("", middleware.RequirePermission(policy, "m1.roles.manage"), h.CreateRole)
	rg.PATCH("/:name", middleware.RequirePermission(policy, "m1.roles.manage"), h.UpdateRole)
	rg.POST("/:name/permissions", middleware.RequirePermission(policy, "m1.roles.manage"), h.AssignPermission)
	rg.DELETE("/:name/permissions/:key", middleware.RequirePermission(policy, "m1.roles.manage"), h.RevokePermission)
	rg.PUT("/:name/permissions", middleware.RequirePermission(policy, "m1.roles.manage"), h.SetPermissions)
}

// RegisterRoutes hooks up all M1 endpoints to the router.
func RegisterRoutes(r *gin.Engine, jwtSecret string, policy *rbac.PolicyService, h Handlers) {
	limiter := middleware.NewRateLimiter()
	m1 := r.Group("/api/m1/v1")
	// ARY-002: public auth endpoints are rate limited against brute force and
	// credential/ID enumeration.
	m1.POST("/login", middleware.RateLimit(limiter, 30, time.Minute), h.Auth.Login)
	m1.POST("/auth/login", middleware.RateLimit(limiter, 30, time.Minute), h.Auth.Login)
	m1.POST("/auth/activate", middleware.RateLimit(limiter, 10, time.Minute), h.Auth.ActivateAccount)

	protected := m1.Group("")
	protected.Use(middleware.Authenticate(jwtSecret))
	protected.GET("/me", h.Auth.GetProfile)
	protected.GET("/auth/me", h.Auth.GetProfile)
	protected.PATCH("/password", h.Auth.ChangePassword)

	tenant := protected.Group("")
	tenant.Use(middleware.RequirePartnerTenant())
	registerUserRoutes(tenant.Group("/users"), policy, h.User)
	registerRoleRoutes(tenant.Group("/roles"), policy, h.Role)
	tenant.GET("/audit-logs", middleware.RequirePermission(policy, "m1.audit.read"), h.Audit.ListAuditLogs)
}
