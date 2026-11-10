package repository

import (
	"context"

	"araya-business-hub/be/internal/modules/m2/domain"
)

type UnitRepository interface {
	Create(ctx context.Context, u *domain.BusinessUnit) error
	GetByID(ctx context.Context, id int) (*domain.BusinessUnit, error)
	Update(ctx context.Context, u *domain.BusinessUnit) error
	// List filters by partnerID (ARY-005): nil means all tenants.
	List(ctx context.Context, search, category string, activeOnly bool, partnerID *int, limit, offset int) ([]domain.BusinessUnit, int, error)
	ListActive(ctx context.Context) ([]domain.BusinessUnit, error)
	SetStatus(ctx context.Context, id int, active bool) error
}
