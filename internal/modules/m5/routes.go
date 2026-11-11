package m5

import (
	"time"

	"araya-business-hub/be/internal/modules/m5/handler"
	"araya-business-hub/be/internal/platform/middleware"
	"araya-business-hub/be/internal/platform/rbac"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
	Catalog        *handler.CatalogHandler
	ServiceRequest *handler.ServiceRequestHandler
	Cart           *handler.CartHandler
	Checkout       *handler.CheckoutHandler
}

func registerPublicCatalogRoutes(rg *gin.RouterGroup, h Handlers) {
	rg.GET("/business-units", h.Catalog.GetBusinessUnits)
	rg.GET("/products", h.Catalog.GetProducts)
	rg.GET("/catalog", h.Catalog.GetCatalog)
}

func registerPublicCartRoutes(rg *gin.RouterGroup, h Handlers) {
	rg.GET("/cart", h.Cart.GetCart)
	rg.POST("/cart/items", h.Cart.AddItem)
	rg.PATCH("/cart/items/:product_id", h.Cart.UpdateItem)
	rg.DELETE("/cart/items/:product_id", h.Cart.RemoveItem)
	rg.DELETE("/cart", h.Cart.ClearCart)
}

func registerProtectedAdminRoutes(rg *gin.RouterGroup, policy *rbac.PolicyService, h Handlers) {
	perm := middleware.RequirePermission(policy, "m5.services.manage")
	rg.GET("/admin/service-requests", perm, h.ServiceRequest.ListRequests)
	rg.GET("/admin/service-requests/:id", perm, h.ServiceRequest.GetRequest)
	rg.PATCH("/admin/service-requests/:id/status", perm, h.ServiceRequest.UpdateRequestStatus)
}

func RegisterRoutes(r *gin.Engine, jwtSecret string, policy *rbac.PolicyService, h Handlers) {
	m5 := r.Group("/api/m5/v1")
	registerPublicCatalogRoutes(m5, h)
	limiter := middleware.NewRateLimiter()
	m5.POST("/service-requests", middleware.RateLimit(limiter, 10, time.Minute), h.ServiceRequest.CreateRequest)
	m5.POST("/cart/checkout", middleware.RateLimit(limiter, 10, time.Minute), h.Checkout.Checkout)
	registerPublicCartRoutes(m5, h)

	protected := m5.Group("")
	protected.Use(middleware.Authenticate(jwtSecret), middleware.RequirePartnerTenant())
	registerProtectedAdminRoutes(protected, policy, h)
}
