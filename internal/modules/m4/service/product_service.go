package service

import (
	"context"
	"fmt"

	"araya-business-hub/be/internal/modules/m4/domain"
	"araya-business-hub/be/internal/modules/m4/dto"
	"araya-business-hub/be/internal/modules/m4/repository"
	"araya-business-hub/be/internal/platform/audit"
)

type ProductService struct {
	repo     repository.Repository
	auditSvc *audit.Service
}

func NewProductService(repo repository.Repository, auditSvc *audit.Service) *ProductService {
	return &ProductService{repo: repo, auditSvc: auditSvc}
}

func (s *ProductService) CreateProduct(ctx context.Context, req dto.CreateProductRequest, actor string) (*domain.Product, error) {
	p := &domain.Product{
		UnitUsahaID: req.UnitUsahaID, PartnerID: req.PartnerID, Name: req.Name,
		Category: req.Category, Description: req.Description, Price: req.Price,
		Stock: req.Stock, IsActive: true, ImageURL: req.ImageURL,
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.CreateProduct(ctx, p); err != nil {
		return nil, err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "product.created", EntityType: "product", EntityID: fmt.Sprintf("%d", p.ID),
	})
	return p, nil
}

func (s *ProductService) GetProduct(ctx context.Context, id int) (*domain.Product, error) {
	return s.repo.GetProductByID(ctx, id)
}

func applyProductUpdates(p *domain.Product, req dto.UpdateProductRequest) {
	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Category != nil {
		p.Category = *req.Category
	}
	if req.Description != nil {
		p.Description = *req.Description
	}
	if req.Price != nil {
		p.Price = *req.Price
	}
	if req.Stock != nil {
		p.Stock = *req.Stock
	}
	if req.IsActive != nil {
		p.IsActive = *req.IsActive
	}
	if req.ImageURL != nil {
		p.ImageURL = *req.ImageURL
	}
}

func (s *ProductService) UpdateProduct(ctx context.Context, id int, req dto.UpdateProductRequest, actor string) (*domain.Product, error) {
	p, err := s.repo.GetProductByID(ctx, id)
	if err != nil {
		return nil, err
	}
	applyProductUpdates(p, req)
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateProduct(ctx, p); err != nil {
		return nil, err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "product.updated", EntityType: "product", EntityID: fmt.Sprintf("%d", id),
	})
	return p, nil
}

func (s *ProductService) ListProducts(ctx context.Context, unitID *int, cat, search string, activeOnly bool, limit, offset int) ([]domain.Product, int, error) {
	return s.repo.ListProducts(ctx, unitID, cat, search, activeOnly, limit, offset)
}

// ListActiveProducts serves Kontrak K-05: GET /api/m4/v1/products
func (s *ProductService) ListActiveProducts(ctx context.Context, unitID *int, cat, search string, limit, offset int) ([]domain.Product, int, error) {
	return s.repo.ListProducts(ctx, unitID, cat, search, true, limit, offset)
}

