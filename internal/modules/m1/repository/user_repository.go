package repository

import (
	"context"

	"araya-business-hub/be/internal/modules/m1/domain"
)

// UserRepository specifies persistence operations for users.
type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	GetByID(ctx context.Context, id int) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, u *domain.User) error
	UpdatePassword(ctx context.Context, id int, hash string) error
	List(ctx context.Context, search, division string, limit, offset int) ([]domain.User, int, error)
	Deactivate(ctx context.Context, id int) error
}
