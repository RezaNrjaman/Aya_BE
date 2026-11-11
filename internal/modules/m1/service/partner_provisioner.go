package service

import (
	"context"
	"errors"
	"time"

	"araya-business-hub/be/internal/modules/m1/domain"
	"araya-business-hub/be/internal/modules/m1/repository"
)

// ErrUserNotFound re-exported for convenience of M3 callers.
var ErrUserNotFound = domain.ErrUserNotFound

// PartnerProvisioner provisions partner owner accounts. ARY-002: accounts are
// created WITHOUT a password. The owner receives a single-use, time-limited
// activation invitation token and must choose a strong password before the
// account can log in.
type PartnerProvisioner struct {
	userRepo repository.UserRepository
}

func NewPartnerProvisioner(userRepo repository.UserRepository) *PartnerProvisioner {
	return &PartnerProvisioner{userRepo: userRepo}
}

// ProvisionPartner creates (or re-links) the owner account for a partner and
// returns its ID together with a fresh activation token (plaintext, to be
// delivered out-of-band). If the account already exists and is activated, the
// returned token is empty.
func (p *PartnerProvisioner) ProvisionPartner(ctx context.Context, email, name, phone string) (int, string, error) {
	existing, err := p.userRepo.GetByEmail(ctx, email)
	if err == nil && existing != nil {
		existing.IsActive = existing.IsActive || existing.NeedsActivation()
		existing.Role = "partner"
		if err := p.userRepo.Update(ctx, existing); err != nil {
			return existing.ID, "", err
		}
		if !existing.NeedsActivation() {
			return existing.ID, "", nil
		}
		token, err := p.issueToken(ctx, existing.ID)
		if err != nil {
			return existing.ID, "", err
		}
		return existing.ID, token, nil
	}
	u := &domain.User{
		Email:    email,
		Name:     name,
		Phone:    phone,
		Role:     "partner",
		IsActive: true, // unusable until a password is set (see CanLogin).
	}
	if err := p.userRepo.Create(ctx, u); err != nil {
		return 0, "", err
	}
	token, err := p.issueToken(ctx, u.ID)
	if err != nil {
		return u.ID, "", err
	}
	return u.ID, token, nil
}

func (p *PartnerProvisioner) issueToken(ctx context.Context, userID int) (string, error) {
	token, hash, err := domain.GenerateActivationToken()
	if err != nil {
		return "", err
	}
	if err := p.userRepo.IssueActivationToken(ctx, userID, hash, time.Now().Add(domain.ActivationTokenTTL)); err != nil {
		return "", err
	}
	return token, nil
}

// IssueActivationTokenForEmail (re)issues an activation invitation for an
// account that has not been activated yet. It returns an empty token when the
// account does not exist or has already been activated.
func (p *PartnerProvisioner) IssueActivationTokenForEmail(ctx context.Context, email string) (string, error) {
	u, err := p.userRepo.GetByEmail(ctx, email)
	if err != nil || u == nil {
		return "", nil // do not reveal whether the email exists.
	}
	if !u.NeedsActivation() {
		return "", nil
	}
	return p.issueToken(ctx, u.ID)
}

// ActivateAccount consumes a single-use invitation token and sets the
// account's first password.
func (p *PartnerProvisioner) ActivateAccount(ctx context.Context, token, newPassword string) error {
	if err := domain.ValidatePasswordStrength(newPassword); err != nil {
		return err
	}
	if token == "" {
		return domain.ErrActivationTokenInvalid
	}
	u, err := p.userRepo.GetByActivationTokenHash(ctx, domain.HashToken(token))
	if err != nil || u == nil {
		return domain.ErrActivationTokenInvalid
	}
	if u.ActivationExpiresAt != nil && time.Now().After(*u.ActivationExpiresAt) {
		return domain.ErrActivationTokenExpired
	}
	hash, err := domain.HashPassword(newPassword)
	if err != nil {
		return err
	}
	// Single use: activation clears the stored token hash.
	return p.userRepo.ActivateAccount(ctx, u.ID, hash)
}

// SetUserStatus toggles the partner owner login. Disabling marks the account
// as suspend-disabled; enabling never reactivates accounts that were disabled
// manually by an administrator (ARY-004).
func (p *PartnerProvisioner) SetUserStatus(ctx context.Context, email string, isActive bool) error {
	existing, err := p.userRepo.GetByEmail(ctx, email)
	if err != nil || existing == nil {
		return nil
	}
	if existing.IsManuallyDisabled {
		// An administrator disabled this account directly; a partner
		// suspend/reactivate cycle must never change that state (ARY-004).
		return nil
	}
	existing.IsActive = isActive
	return p.userRepo.Update(ctx, existing)
}

// LinkPartnerUser associates a user account with its partner organisation. This
// is what lets M1 and M3 scope data to the logged-in partner's own members.
func (p *PartnerProvisioner) LinkPartnerUser(ctx context.Context, userID, partnerID int) error {
	existing, err := p.userRepo.GetByID(ctx, userID)
	if err != nil || existing == nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil
		}
		return err
	}
	existing.PartnerID = &partnerID
	return p.userRepo.Update(ctx, existing)
}

// SetPartnerStatus applies suspension to every member of the tenant.
func (p *PartnerProvisioner) SetPartnerStatus(ctx context.Context, partnerID int, active bool) error {
	return p.userRepo.SetPartnerUsersSuspended(ctx, partnerID, !active)
}
