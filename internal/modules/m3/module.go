package m3

import (
	"context"

	"araya-business-hub/be/internal/modules/m3/domain"
	"araya-business-hub/be/internal/modules/m3/handler"
	"araya-business-hub/be/internal/modules/m3/repository"
	"araya-business-hub/be/internal/modules/m3/service"
	"araya-business-hub/be/internal/platform/audit"
	"araya-business-hub/be/internal/platform/rbac"
	"github.com/gin-gonic/gin"
)

type Module struct {
	Repo *repository.InMemoryPartnerRepository
	Svc  *service.PartnerService
}

func InitModule(r *gin.Engine, secret string, policy *rbac.PolicyService, auditSvc *audit.Service) *Module {
	repo := repository.NewInMemoryPartnerRepository()
	svc := service.NewPartnerService(repo, auditSvc)
	h := handler.NewPartnerHandler(svc)

	RegisterRoutes(r, secret, policy, h)
	_ = SeedInitialPartners(context.Background(), repo)

	return &Module{Repo: repo, Svc: svc}
}

// SeedInitialPartners seeds default verified and pending partners for development.
func SeedInitialPartners(ctx context.Context, repo repository.PartnerRepository) error {
	partners := []domain.Partner{
		{
			CompanyName: "CV Mitra Grafika Mandiri", ContactPerson: "Budi Santoso",
			Email: "budi@mitragrafika.com", Phone: "081298765432", Address: "Jl. Industri No. 12",
			Status: domain.StatusVerified,
		},
		{
			CompanyName: "PT Katering Berkah Nusantara", ContactPerson: "Siti Rahma",
			Email: "siti@berkahkatering.com", Phone: "081387654321", Address: "Jl. Kuliner No. 45",
			Status: domain.StatusVerified,
		},
		{
			CompanyName: "CV Logistik Cepat Aman", ContactPerson: "Rian Hidayat",
			Email: "rian@logistikcepat.com", Phone: "081122334455", Address: "Jl. Cargo No. 8",
			Status: domain.StatusPending,
		},
	}
	for _, p := range partners {
		_ = repo.Create(ctx, &p)
	}
	return nil
}

