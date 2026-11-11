package m3

import (
	"time"

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
	limiter := middleware.NewRateLimiter()
	m3 := r.Group("/api/m3/v1")
	// Public registration and verified partners contract (K-06). ARY-002:
	// public endpoints are rate limited; the status page is addressed by an
	// unguessable registration code instead of the sequential partner ID.
	m3.POST("/partners/register", middleware.RateLimit(limiter, 10, time.Minute), h.Register)
	m3.GET("/partners/verified", h.ListVerifiedPartners)
	m3.GET("/partners/status", middleware.RateLimit(limiter, 30, time.Minute), h.PublicPartnerStatus)

	protected := m3.Group("")
	protected.Use(middleware.Authenticate(jwtSecret), middleware.RequirePlatformAdmin(policy))
	registerAdminRoutes(protected, policy, h)
}
