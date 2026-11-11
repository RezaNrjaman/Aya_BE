package repository

import (
	"context"
	"time"

	"araya-business-hub/be/internal/modules/m1/domain"
)

// UserRepository specifies persistence operations for users.
type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	GetByID(ctx context.Context, id int) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, u *domain.User) error
	UpdatePassword(ctx context.Context, id int, hash string) error
	List(ctx context.Context, search, division string, partnerID *int, limit, offset int) ([]domain.User, int, error)
	ValidatePartnerPlacement(ctx context.Context, partnerID int, divisionID, unitID *int) error
	Deactivate(ctx context.Context, id int) error
	SetPartnerUsersSuspended(ctx context.Context, partnerID int, suspended bool) error
	// GetByActivationTokenHash finds the account a (hashed) activation or
	// password-reset invitation token belongs to (ARY-002).
	GetByActivationTokenHash(ctx context.Context, tokenHash string) (*domain.User, error)
	// IssueActivationToken stores a hashed invitation token and its expiry,
	// replacing any previous token for the account.
	IssueActivationToken(ctx context.Context, id int, tokenHash string, expiresAt time.Time) error
	// ActivateAccount sets the initial password, clears the invitation token
	// (single use) and enables the account.
	ActivateAccount(ctx context.Context, id int, passwordHash string) error
}
