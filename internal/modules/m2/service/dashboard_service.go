package service

import (
	"context"
	"time"

	"araya-business-hub/be/internal/modules/m2/domain"
	"araya-business-hub/be/internal/modules/m2/dto"
	"araya-business-hub/be/internal/modules/m2/repository"
)

type DashboardDataProvider interface {
	GetDashboardData(ctx context.Context, partnerID, unitID *int, from, to *time.Time) (dto.DashboardMetrics, []dto.OrderSummaryItem, error)
}

type DashboardService struct {
	unitRepo repository.UnitRepository
	provider DashboardDataProvider
}

func NewDashboardService(repo repository.UnitRepository) *DashboardService {
	return &DashboardService{unitRepo: repo}
}

func (s *DashboardService) SetDataProvider(provider DashboardDataProvider) {
	s.provider = provider
}

func formatPeriod(from, to *time.Time) (string, string) {
	fromStr, toStr := "", ""
	if from != nil {
		fromStr = from.Format("2006-01-02")
	}
	if to != nil {
		toStr = to.Format("2006-01-02")
	}
	return fromStr, toStr
}

func buildBaselineMetrics() dto.DashboardMetrics {
	// Keep the baseline honest: modules that own these entities have not been
	// wired into the dashboard yet, so report zero instead of invented numbers.
	return dto.DashboardMetrics{
		TotalRevenue:         0,
		TotalOrders:          0,
		ActivePartners:       0,
		ActiveProducts:       0,
		PendingVerifications: 0,
	}
}

func (s *DashboardService) GetSummary(ctx context.Context, unitID, partnerID *int, from, to *time.Time) (*dto.DashboardSummaryResponse, error) {
	if unitID != nil && partnerID != nil {
		unit, err := s.unitRepo.GetByID(ctx, *unitID)
		if err != nil || unit.PartnerID == nil || *unit.PartnerID != *partnerID {
			return nil, domain.ErrUnitNotFound
		}
	}
	fromStr, toStr := formatPeriod(from, to)
	metrics := buildBaselineMetrics()
	recentOrders := []dto.OrderSummaryItem{}
	if s.provider != nil {
		var err error
		metrics, recentOrders, err = s.provider.GetDashboardData(ctx, partnerID, unitID, from, to)
		if err != nil {
			return nil, err
		}
	}

	return &dto.DashboardSummaryResponse{
		UnitUsahaID:  unitID,
		PeriodFrom:   fromStr,
		PeriodTo:     toStr,
		Metrics:      metrics,
		RecentOrders: recentOrders,
	}, nil
}
