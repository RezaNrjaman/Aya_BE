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
	Repo repository.PartnerRepository
	Svc  *service.PartnerService
}

func parseInitOptions(opts ...any) (service.UserProvisioner, repository.PartnerRepository) {
	var prov service.UserProvisioner
	var repo repository.PartnerRepository
	for _, opt := range opts {
		if p, ok := opt.(service.UserProvisioner); ok {
			prov = p
		} else if r, ok := opt.(repository.PartnerRepository); ok {
			repo = r
		}
	}
	if repo == nil {
		repo = repository.NewInMemoryPartnerRepository()
	}
	return prov, repo
}

func InitModule(r *gin.Engine, secret string, policy *rbac.PolicyService, auditSvc *audit.Service, opts ...any) *Module {
	hasPersistentRepo := false
	for _, opt := range opts {
		if repo, ok := opt.(repository.PartnerRepository); ok && repo != nil {
			hasPersistentRepo = true
		}
	}
	prov, repo := parseInitOptions(opts...)
	svc := service.NewPartnerService(repo, auditSvc, prov)
	h := handler.NewPartnerHandler(svc)

	RegisterRoutes(r, secret, policy, h)
	if !hasPersistentRepo {
		_ = SeedInitialPartners(context.Background(), repo, prov)
	}

	return &Module{Repo: repo, Svc: svc}
}

func defaultSeedPartners() []domain.Partner {
	return []domain.Partner{
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
}

// SeedInitialPartners seeds default verified and pending partners for development.
// It is idempotent and also repairs the user <-> partner linkage on existing rows.
func SeedInitialPartners(ctx context.Context, repo repository.PartnerRepository, prov service.UserProvisioner) error {
	for _, seed := range defaultSeedPartners() {
		existing, err := repo.GetByEmail(ctx, seed.Email)
		if err != nil {
			if err := repo.Create(ctx, &seed); err != nil {
				continue
			}
			existing = &seed
		}

		if existing.Status != domain.StatusVerified || prov == nil {
			continue
		}
		uid, _, err := prov.ProvisionPartner(ctx, existing.Email, existing.ContactPerson, existing.Phone)
		if err != nil || uid <= 0 {
			continue
		}
		if existing.UserID == nil || *existing.UserID != uid {
			existing.UserID = &uid
			_ = repo.Update(ctx, existing)
		}
		_ = prov.LinkPartnerUser(ctx, uid, existing.ID)
	}
	return nil
}
