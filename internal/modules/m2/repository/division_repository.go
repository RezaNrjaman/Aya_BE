package repository

import (
	"context"

	"araya-business-hub/be/internal/modules/m2/domain"
)

// DivisionRepository persists partner-owned divisions (ARY-006). It is
// implemented by the same storage types as UnitRepository.
type DivisionRepository interface {
	CreateDivision(ctx context.Context, d *domain.Division) error
	GetDivisionByID(ctx context.Context, id int) (*domain.Division, error)
	UpdateDivision(ctx context.Context, d *domain.Division) error
	// ListDivisions filters by partnerID: nil means all tenants.
	ListDivisions(ctx context.Context, search string, partnerID *int, limit, offset int) ([]domain.Division, int, error)
	SetDivisionStatus(ctx context.Context, id int, active bool) error
	DeleteDivision(ctx context.Context, id int) error
	// CountDivisionReferences returns how many users and business units still
	// point at the division. A division in use must not be deleted outright.
	CountDivisionReferences(ctx context.Context, id int) (users int, units int, err error)
}
