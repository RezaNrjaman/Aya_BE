package m4

import (
	"context"

	"araya-business-hub/be/internal/modules/m4/domain"
	"araya-business-hub/be/internal/modules/m4/handler"
	"araya-business-hub/be/internal/modules/m4/payment"
	"araya-business-hub/be/internal/modules/m4/repository"
	"araya-business-hub/be/internal/modules/m4/service"
	"araya-business-hub/be/internal/platform/audit"
	"araya-business-hub/be/internal/platform/rbac"
	"github.com/gin-gonic/gin"
)

type Module struct {
	Repo       repository.Repository
	ProductSvc *service.ProductService
	OrderSvc   *service.OrderService
}

func resolveM4Repo(opts ...repository.Repository) repository.Repository {
	if len(opts) > 0 && opts[0] != nil {
		return opts[0]
	}
	return repository.NewInMemoryRepository()
}

// InitModule wires module m4. partnerStatus (ARY-004) may be nil in tests
// that do not exercise the storefront visibility rules.
func InitModule(r *gin.Engine, secret string, policy *rbac.PolicyService, auditSvc *audit.Service, partnerStatus service.PartnerStatusResolver, repoOpt ...repository.Repository) *Module {
	repo := resolveM4Repo(repoOpt...)
	adapter := payment.NewSandboxAdapter("")
	prodSvc := service.NewProductService(repo, auditSvc, partnerStatus)
	custSvc := service.NewCustomerService(repo)
	quotSvc := service.NewQuotationService(repo, auditSvc)
	invSvc := service.NewInvoiceService(repo, adapter, auditSvc)
	ordSvc := service.NewOrderService(repo, auditSvc)

	h := Handlers{
		Product:   handler.NewProductHandler(prodSvc),
		Customer:  handler.NewCustomerHandler(custSvc),
		Quotation: handler.NewQuotationHandler(quotSvc),
		Invoice:   handler.NewInvoiceHandler(invSvc),
		Order:     handler.NewOrderHandler(ordSvc),
	}
	RegisterRoutes(r, secret, policy, h)
	_ = SeedInitialData(context.Background(), repo)

	return &Module{Repo: repo, ProductSvc: prodSvc, OrderSvc: ordSvc}
}

func seedProducts(ctx context.Context, repo repository.Repository) {
	products := []domain.Product{
		{UnitUsahaID: 1, Name: "Cetak Banner Outdoor 3x1m", Category: "Percetakan", Price: 75000, Stock: 100, IsActive: true},
		{UnitUsahaID: 1, Name: "Brosur Full Color A4 (1 Rim)", Category: "Percetakan", Price: 350000, Stock: 50, IsActive: true},
		{UnitUsahaID: 2, Name: "Kamar Standar Wisma Tamu (1 Malam)", Category: "Penginapan", Price: 250000, Stock: 10, IsActive: true},
		{UnitUsahaID: 3, Name: "Paket Nasi Kotak Premium", Category: "Katering", Price: 30000, Stock: 500, IsActive: true},
	}
	for _, p := range products {
		_ = repo.CreateProduct(ctx, &p)
	}
}

func seedCustomers(ctx context.Context, repo repository.Repository) {
	customers := []domain.Customer{
		{Name: "Universitas Brawijaya", Email: "ub@ub.ac.id", Phone: "0341551611", Address: "Jl. Veteran Malang"},
		{Name: "CV Rekan Usaha", Email: "rekan@cvrekan.co.id", Phone: "081233445566", Address: "Jl. Ijen No. 12 Malang"},
	}
	for _, c := range customers {
		_ = repo.CreateCustomer(ctx, &c)
	}
}

func SeedInitialData(ctx context.Context, repo repository.Repository) error {
	seedProducts(ctx, repo)
	seedCustomers(ctx, repo)
	return nil
}
