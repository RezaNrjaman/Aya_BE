package m5

import (
	"context"

	"araya-business-hub/be/internal/modules/m5/client"
	"araya-business-hub/be/internal/modules/m5/domain"
	"araya-business-hub/be/internal/modules/m5/handler"
	"araya-business-hub/be/internal/modules/m5/repository"
	"araya-business-hub/be/internal/modules/m5/service"
	"araya-business-hub/be/internal/platform/audit"
	"araya-business-hub/be/internal/platform/rbac"
	"github.com/gin-gonic/gin"
)

type Module struct {
	Repo repository.Repository
}

func buildHandlers(repo repository.Repository, m2 client.M2Client, m4 client.M4Client, auditSvc *audit.Service, partnerStatus service.PartnerStatusResolver) Handlers {
	catSvc := service.NewCatalogService(m2, m4, partnerStatus)
	reqSvc := service.NewServiceRequestService(repo, auditSvc, catSvc)
	cartSvc := service.NewCartService(repo, m4)
	chkSvc := service.NewCheckoutService(repo, m4, auditSvc)

	return Handlers{
		Catalog:        handler.NewCatalogHandler(catSvc),
		ServiceRequest: handler.NewServiceRequestHandler(reqSvc),
		Cart:           handler.NewCartHandler(cartSvc),
		Checkout:       handler.NewCheckoutHandler(chkSvc),
	}
}

func resolveM5Repo(opts ...repository.Repository) repository.Repository {
	if len(opts) > 0 && opts[0] != nil {
		return opts[0]
	}
	return repository.NewInMemoryRepository()
}

func InitModule(r *gin.Engine, secret string, policy *rbac.PolicyService, auditSvc *audit.Service, m2 client.M2Client, m4 client.M4Client, opts ...any) *Module {
	var repo repository.Repository
	var partnerStatus service.PartnerStatusResolver
	hasPersistentRepo := false
	for _, opt := range opts {
		if r, ok := opt.(repository.Repository); ok && r != nil {
			repo = r
			hasPersistentRepo = true
		}
		if status, ok := opt.(service.PartnerStatusResolver); ok {
			partnerStatus = status
		}
	}
	repo = resolveM5Repo(repo)
	h := buildHandlers(repo, m2, m4, auditSvc, partnerStatus)

	RegisterRoutes(r, secret, policy, h)
	if !hasPersistentRepo {
		_ = SeedInitialData(context.Background(), repo)
	}

	return &Module{Repo: repo}
}

func SeedInitialData(ctx context.Context, repo repository.Repository) error {
	sample := domain.ServiceRequest{
		UnitUsahaID:   1,
		CustomerName:  "Budi Santoso",
		CustomerEmail: "budi.santoso@example.com",
		CustomerPhone: "081298765432",
		ServiceType:   "Percetakan",
		Description:   "Mohon penawaran harga cetak buku tahunan sekolah 200 eksemplar hard cover.",
		Status:        domain.RequestPending,
	}
	return repo.CreateServiceRequest(ctx, &sample)
}
