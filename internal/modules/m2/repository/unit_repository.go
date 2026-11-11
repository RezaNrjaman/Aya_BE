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
	ValidatePartnerDivision(ctx context.Context, partnerID int, divisionID *int) error
	GetOrCreateCategoryID(ctx context.Context, name string) (*int, error)
	CreateCategory(ctx context.Context, category *domain.BusinessCategory) error
	ListCategories(ctx context.Context, unitID *int, partnerID *int, search string, limit, offset int) ([]domain.BusinessCategory, int, error)
	GetCategoryByID(ctx context.Context, id int) (*domain.BusinessCategory, error)
	UpdateCategory(ctx context.Context, category *domain.BusinessCategory) error
	SetCategoryStatus(ctx context.Context, id int, active bool) error
	ValidateCategoryPlacement(ctx context.Context, partnerID int, unitID, categoryID int) error
}
