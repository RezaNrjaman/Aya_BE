package m4

import (
	"araya-business-hub/be/internal/modules/m4/handler"
	"araya-business-hub/be/internal/platform/middleware"
	"araya-business-hub/be/internal/platform/rbac"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
	Product   *handler.ProductHandler
	Customer  *handler.CustomerHandler
	Quotation *handler.QuotationHandler
	Invoice   *handler.InvoiceHandler
	Order     *handler.OrderHandler
}

func registerProductAndCustomerRoutes(rg *gin.RouterGroup, policy *rbac.PolicyService, h Handlers) {
	rg.POST("/products", middleware.RequirePermission(policy, "m4.products.manage"), h.Product.CreateProduct)
	rg.GET("/admin/products", middleware.RequirePermission(policy, "m4.products.read"), h.Product.ListAdminProducts)
	rg.GET("/products/:id", middleware.RequirePermission(policy, "m4.products.read"), h.Product.GetProduct)
	rg.PATCH("/products/:id", middleware.RequirePermission(policy, "m4.products.manage"), h.Product.UpdateProduct)

	rg.POST("/customers", middleware.RequirePermission(policy, "m4.customers.manage"), h.Customer.CreateCustomer)
	rg.GET("/customers", middleware.RequirePermission(policy, "m4.customers.manage"), h.Customer.ListCustomers)
	rg.GET("/customers/:id", middleware.RequirePermission(policy, "m4.customers.manage"), h.Customer.GetCustomer)
}

func registerDocRoutes(rg *gin.RouterGroup, policy *rbac.PolicyService, h Handlers) {
	rg.POST("/quotations", middleware.RequirePermission(policy, "m4.quotations.manage"), h.Quotation.CreateQuotation)
	rg.GET("/quotations", middleware.RequirePermission(policy, "m4.quotations.manage"), h.Quotation.ListQuotations)
	rg.GET("/quotations/:id", middleware.RequirePermission(policy, "m4.quotations.manage"), h.Quotation.GetQuotation)
	rg.POST("/quotations/:id/approve", middleware.RequirePermission(policy, "m4.quotations.manage"), h.Quotation.ApproveQuotation)
	rg.POST("/quotations/:id/reject", middleware.RequirePermission(policy, "m4.quotations.manage"), h.Quotation.RejectQuotation)

	rg.POST("/invoices", middleware.RequirePermission(policy, "m4.invoices.manage"), h.Invoice.CreateInvoice)
	rg.GET("/invoices", middleware.RequirePermission(policy, "m4.invoices.manage"), h.Invoice.ListInvoices)
	rg.GET("/invoices/:id", middleware.RequirePermission(policy, "m4.invoices.manage"), h.Invoice.GetInvoice)
	rg.POST("/invoices/:id/pay", middleware.RequirePermission(policy, "m4.invoices.pay"), h.Invoice.PayInvoice)
}

func registerOrderAdminRoutes(rg *gin.RouterGroup, policy *rbac.PolicyService, h Handlers) {
	rg.GET("/admin/orders", middleware.RequirePermission(policy, "m4.orders.manage"), h.Order.ListOrders)
	rg.GET("/orders/:id", middleware.RequirePermission(policy, "m4.orders.manage"), h.Order.GetOrder)
	rg.POST("/orders/:id/process", middleware.RequirePermission(policy, "m4.orders.manage"), h.Order.ProcessOrder)
	rg.POST("/orders/:id/complete", middleware.RequirePermission(policy, "m4.orders.manage"), h.Order.CompleteOrder)
	rg.POST("/orders/:id/cancel", middleware.RequirePermission(policy, "m4.orders.manage"), h.Order.CancelOrder)
}

// RegisterRoutes registers all M4 routes into Gin.
func RegisterRoutes(r *gin.Engine, jwtSecret string, policy *rbac.PolicyService, h Handlers) {
	m4 := r.Group("/api/m4/v1")
	// Contract K-05: Read active products
	m4.GET("/products", h.Product.ListActiveProducts)
	// Contract K-04: Create order from checkout
	m4.POST("/orders", h.Order.CreateOrder)
	// Payment webhook / callback
	m4.POST("/payments/callback", h.Invoice.HandlePaymentCallback)

	protected := m4.Group("")
	protected.Use(middleware.Authenticate(jwtSecret))
	registerProductAndCustomerRoutes(protected, policy, h)
	registerDocRoutes(protected, policy, h)
	registerOrderAdminRoutes(protected, policy, h)
}
