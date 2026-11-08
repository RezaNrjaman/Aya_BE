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

func (s *CustomerService) CreateCustomer(ctx context.Context, req dto.CreateCustomerRequest) (*domain.Customer, error) {
	c := &domain.Customer{
		Name: req.Name, Email: req.Email, Phone: req.Phone, Address: req.Address,
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.CreateCustomer(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CustomerService) GetCustomer(ctx context.Context, id int) (*domain.Customer, error) {
	return s.repo.GetCustomerByID(ctx, id)
}

func (s *CustomerService) ListCustomers(ctx context.Context, search string, limit, offset int) ([]domain.Customer, int, error) {
	return s.repo.ListCustomers(ctx, search, limit, offset)
}
