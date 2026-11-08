package service

import (
	"context"

	"araya-business-hub/be/internal/modules/m5/client"
	"araya-business-hub/be/internal/modules/m5/dto"
)

type CatalogService struct {
	m2Client client.M2Client
	m4Client client.M4Client
}

func NewCatalogService(m2 client.M2Client, m4 client.M4Client) *CatalogService {
	return &CatalogService{m2Client: m2, m4Client: m4}
}

func (s *CatalogService) GetBusinessUnits(ctx context.Context) ([]dto.BusinessUnitSummary, error) {
	return s.m2Client.GetActiveBusinessUnits(ctx)
}

func (s *CatalogService) GetProducts(ctx context.Context, f dto.CatalogFilter) ([]dto.ProductSummary, error) {
	return s.m4Client.GetActiveProducts(ctx, f.UnitUsahaID, f.Category, f.Search)
}

func (s *CatalogService) GetCatalog(ctx context.Context, f dto.CatalogFilter) (map[string]any, error) {
	units, err := s.m2Client.GetActiveBusinessUnits(ctx)
	if err != nil {
		return nil, err
	}
	products, err := s.m4Client.GetActiveProducts(ctx, f.UnitUsahaID, f.Category, f.Search)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"business_units": units,
		"products":       products,
	}, nil
}

