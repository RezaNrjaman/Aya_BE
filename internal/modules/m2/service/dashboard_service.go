package service

import (
	"context"
	"time"

	"araya-business-hub/be/internal/modules/m2/dto"
	"araya-business-hub/be/internal/modules/m2/repository"
)

type DashboardService struct {
	unitRepo repository.UnitRepository
}

func NewDashboardService(repo repository.UnitRepository) *DashboardService {
	return &DashboardService{unitRepo: repo}
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

func (s *DashboardService) GetSummary(ctx context.Context, unitID *int, from, to *time.Time) (*dto.DashboardSummaryResponse, error) {
	fromStr, toStr := formatPeriod(from, to)
	metrics := buildBaselineMetrics()

	return &dto.DashboardSummaryResponse{
		UnitUsahaID:  unitID,
		PeriodFrom:   fromStr,
		PeriodTo:     toStr,
		Metrics:      metrics,
		RecentOrders: []dto.OrderSummaryItem{},
	}, nil
}
