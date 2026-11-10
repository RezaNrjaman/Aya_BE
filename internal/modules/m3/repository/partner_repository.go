package repository

import (
	"context"

	"araya-business-hub/be/internal/modules/m3/domain"
)

type PartnerRepository interface {
	Create(ctx context.Context, p *domain.Partner) error
	GetByID(ctx context.Context, id int) (*domain.Partner, error)
	GetByEmail(ctx context.Context, email string) (*domain.Partner, error)
	GetByUserID(ctx context.Context, userID int) (*domain.Partner, error)
	// GetByRegistrationCode finds a partner by its unguessable tracking code
	// (public status page, ARY-002).
	GetByRegistrationCode(ctx context.Context, code string) (*domain.Partner, error)
	Update(ctx context.Context, p *domain.Partner) error
	List(ctx context.Context, status, search string, limit, offset int) ([]domain.Partner, int, error)
	ListVerified(ctx context.Context) ([]domain.Partner, error)
	AddHistory(ctx context.Context, h *domain.StatusHistory) error
	GetHistories(ctx context.Context, partnerID int) ([]domain.StatusHistory, error)
}
