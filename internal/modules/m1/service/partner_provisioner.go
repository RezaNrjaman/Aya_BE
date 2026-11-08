package service

import (
	"context"

	"araya-business-hub/be/internal/modules/m1/domain"
	"araya-business-hub/be/internal/modules/m1/repository"
)

type PartnerProvisioner struct {
	userRepo repository.UserRepository
}

func NewPartnerProvisioner(userRepo repository.UserRepository) *PartnerProvisioner {
	return &PartnerProvisioner{userRepo: userRepo}
}

func (p *PartnerProvisioner) ProvisionPartner(ctx context.Context, email, name, phone, password string) (int, error) {
	existing, err := p.userRepo.GetByEmail(ctx, email)
	if err == nil && existing != nil {
		existing.IsActive = true
		existing.Role = "partner"
		_ = p.userRepo.Update(ctx, existing)
		return existing.ID, nil
	}
	if password == "" {
		password = "Partner123!"
	}
	hash, _ := domain.HashPassword(password)
	u := &domain.User{
		Email:        email,
		PasswordHash: hash,
		Name:         name,
		Phone:        phone,
		Role:         "partner",
		DivisionName: "mitra",
		IsActive:     true,
	}
	if err := p.userRepo.Create(ctx, u); err != nil {
		return 0, err
	}
	return u.ID, nil
}

func (p *PartnerProvisioner) SetUserStatus(ctx context.Context, email string, isActive bool) error {
	existing, err := p.userRepo.GetByEmail(ctx, email)
	if err != nil || existing == nil {
		return nil
	}
	existing.IsActive = isActive
	return p.userRepo.Update(ctx, existing)
}
