package m2

import (
	"context"

	"araya-business-hub/be/internal/modules/m2/domain"
	"araya-business-hub/be/internal/modules/m2/handler"
	"araya-business-hub/be/internal/modules/m2/repository"
	"araya-business-hub/be/internal/modules/m2/service"
	"araya-business-hub/be/internal/platform/audit"
	"araya-business-hub/be/internal/platform/rbac"
	"github.com/gin-gonic/gin"
)

type Module struct {
	Repo         repository.UnitRepository
	UnitSvc      *service.UnitService
	DashboardSvc *service.DashboardService
}

func InitModule(r *gin.Engine, secret string, policy *rbac.PolicyService, auditSvc *audit.Service) *Module {
	repo := repository.NewInMemoryUnitRepository()
	unitSvc := service.NewUnitService(repo, auditSvc)
	dashSvc := service.NewDashboardService(repo)

	h := Handlers{
		Unit:      handler.NewUnitHandler(unitSvc),
		Dashboard: handler.NewDashboardHandler(dashSvc),
	}
	RegisterRoutes(r, secret, policy, h)
	_ = SeedInitialUnits(context.Background(), repo)

	return &Module{Repo: repo, UnitSvc: unitSvc, DashboardSvc: dashSvc}
}

// SeedInitialUnits seeds default BPU business units on startup.
func SeedInitialUnits(ctx context.Context, repo repository.UnitRepository) error {
	units := []domain.BusinessUnit{
		{Name: "Unit Percetakan & Konveksi", CategoryName: "Produksi", Description: "Jasa cetak dan konveksi resmi", IsActive: true},
		{Name: "Unit Penginapan & Wisma", CategoryName: "Akomodasi", Description: "Penginapan nyaman dan wisma tamu", IsActive: true},
		{Name: "Unit Kantin & Katering", CategoryName: "Kuliner", Description: "Layanan makanan dan katering", IsActive: true},
	}
	for _, u := range units {
		_ = repo.Create(ctx, &u)
	}
	return nil
}
