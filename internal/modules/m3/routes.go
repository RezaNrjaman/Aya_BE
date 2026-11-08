package m3

import (
	"araya-business-hub/be/internal/modules/m3/handler"
	"araya-business-hub/be/internal/platform/middleware"
	"araya-business-hub/be/internal/platform/rbac"
	"github.com/gin-gonic/gin"
)

func registerAdminRoutes(rg *gin.RouterGroup, policy *rbac.PolicyService, h *handler.PartnerHandler) {
	rg.GET("/partners", middleware.RequirePermission(policy, "m3.partners.read"), h.ListPartners)
	rg.GET("/partners/:id", middleware.RequirePermission(policy, "m3.partners.read"), h.GetPartner)
	rg.POST("/partners/:id/verify", middleware.RequirePermission(policy, "m3.partners.verify"), h.Verify)
	rg.POST("/partners/:id/suspend", middleware.RequirePermission(policy, "m3.partners.suspend"), h.Suspend)
	rg.POST("/partners/:id/reactivate", middleware.RequirePermission(policy, "m3.partners.suspend"), h.Reactivate)
}

// RegisterRoutes registers all M3 routes into Gin engine.
func RegisterRoutes(r *gin.Engine, jwtSecret string, policy *rbac.PolicyService, h *handler.PartnerHandler) {
	m3 := r.Group("/api/m3/v1")
	// Public registration and verified partners contract (K-06)
	m3.POST("/partners/register", h.Register)
	m3.GET("/partners/verified", h.ListVerifiedPartners)
	m3.GET("/partners/:id/status", h.GetPartner)

	protected := m3.Group("")
	protected.Use(middleware.Authenticate(jwtSecret))
	registerAdminRoutes(protected, policy, h)
}
