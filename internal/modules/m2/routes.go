package m2

import (
	"araya-business-hub/be/internal/modules/m2/handler"
	"araya-business-hub/be/internal/platform/middleware"
	"araya-business-hub/be/internal/platform/rbac"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
	Unit      *handler.UnitHandler
	Dashboard *handler.DashboardHandler
}

func registerAdminUnitRoutes(rg *gin.RouterGroup, policy *rbac.PolicyService, h *handler.UnitHandler) {
	rg.GET("/admin/business-units", middleware.RequirePermission(policy, "m2.business_units.read"), h.ListAdminUnits)
	rg.POST("/business-units", middleware.RequirePermission(policy, "m2.business_units.create"), h.CreateUnit)
	rg.GET("/business-units/:id", middleware.RequirePermission(policy, "m2.business_units.read"), h.GetUnit)
	rg.PATCH("/business-units/:id", middleware.RequirePermission(policy, "m2.business_units.update"), h.UpdateUnit)
	rg.POST("/business-units/:id/deactivate", middleware.RequirePermission(policy, "m2.business_units.update"), h.DeactivateUnit)
	rg.POST("/business-units/:id/activate", middleware.RequirePermission(policy, "m2.business_units.update"), h.ActivateUnit)
}

// RegisterRoutes registers all M2 routes to Gin.
func RegisterRoutes(r *gin.Engine, jwtSecret string, policy *rbac.PolicyService, h Handlers) {
	m2 := r.Group("/api/m2/v1")
	// Contract K-02: Public/module read of active business units
	m2.GET("/business-units", h.Unit.ListActiveUnits)

	protected := m2.Group("")
	protected.Use(middleware.Authenticate(jwtSecret))
	registerAdminUnitRoutes(protected, policy, h.Unit)
	// Contract K-03: Dashboard summary
	protected.GET("/dashboard-summary", middleware.RequirePermission(policy, "m2.dashboard.read"), h.Dashboard.GetSummary)
}
