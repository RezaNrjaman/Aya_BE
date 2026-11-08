package m1

import (
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
	rg.POST("", middleware.RequirePermission(policy, "m1.roles.manage"), h.CreateRole)
	rg.PATCH("/:name", middleware.RequirePermission(policy, "m1.roles.manage"), h.UpdateRole)
	rg.POST("/:name/permissions", middleware.RequirePermission(policy, "m1.roles.manage"), h.AssignPermission)
	rg.DELETE("/:name/permissions/:key", middleware.RequirePermission(policy, "m1.roles.manage"), h.RevokePermission)
}

// RegisterRoutes hooks up all M1 endpoints to the router.
func RegisterRoutes(r *gin.Engine, jwtSecret string, policy *rbac.PolicyService, h Handlers) {
	m1 := r.Group("/api/m1/v1")
	m1.POST("/login", h.Auth.Login)

	protected := m1.Group("")
	protected.Use(middleware.Authenticate(jwtSecret))
	protected.GET("/me", h.Auth.GetProfile)
	protected.PATCH("/password", h.Auth.ChangePassword)

	registerUserRoutes(protected.Group("/users"), policy, h.User)
	registerRoleRoutes(protected.Group("/roles"), policy, h.Role)
	protected.GET("/audit-logs", middleware.RequirePermission(policy, "m1.audit.read"), h.Audit.ListAuditLogs)
}
