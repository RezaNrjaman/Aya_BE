package service

import (
	"context"
	"errors"

	"araya-business-hub/be/internal/modules/m4/domain"
	"araya-business-hub/be/internal/modules/m4/repository"
)

func ensureCustomerTenant(ctx context.Context, repo repository.Repository, customerID int, partnerID *int) error {
	customer, err := repo.GetCustomerByID(ctx, customerID)
	if err != nil {
		return err
	}
	if partnerID == nil {
		if customer.PartnerID != nil {
			return domain.ErrCustomerNotFound
		}
		return nil
	}
	if customer.PartnerID == nil || *customer.PartnerID != *partnerID {
		return domain.ErrCustomerNotFound
	}
	return nil
}

func ensureProductTenant(ctx context.Context, repo repository.Repository, productID *int, partnerID *int) error {
	if productID == nil {
		return nil
	}
	product, err := repo.GetProductByID(ctx, *productID)
	if err != nil {
		return err
	}
	if partnerID == nil {
		if product.PartnerID != nil {
			return errors.New("produk bukan milik platform")
		}
		return nil
	}
	if product.PartnerID == nil || *product.PartnerID != *partnerID {
		return errors.New("produk bukan milik perusahaan Anda")
	}
	return nil
}
