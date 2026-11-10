package service

import (
	"context"

	"araya-business-hub/be/internal/modules/m4/domain"
	"araya-business-hub/be/internal/modules/m4/dto"
	"araya-business-hub/be/internal/modules/m4/repository"
)

type CustomerService struct {
	repo repository.Repository
}

func NewCustomerService(repo repository.Repository) *CustomerService {
	return &CustomerService{repo: repo}
}

func (s *CustomerService) CreateCustomer(ctx context.Context, req dto.CreateCustomerRequest, scope *int) (*domain.Customer, error) {
	c := &domain.Customer{
		// ARY-001: customers are contacts owned by the creating tenant.
		PartnerID: scope,
		Name:      req.Name, Email: req.Email, Phone: req.Phone, Address: req.Address,
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.CreateCustomer(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CustomerService) GetCustomer(ctx context.Context, id int, scope *int) (*domain.Customer, error) {
	c, err := s.repo.GetCustomerByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !canAccessTenant(c.PartnerID, scope) {
		return nil, domain.ErrCustomerNotFound
	}
	return c, nil
}

func (s *CustomerService) ListCustomers(ctx context.Context, search string, scope *int, limit, offset int) ([]domain.Customer, int, error) {
	return s.repo.ListCustomers(ctx, search, scope, limit, offset)
}
