package service

import (
	"context"

	"araya-business-hub/be/internal/modules/m5/client"
	"araya-business-hub/be/internal/modules/m5/dto"
)

type PartnerStatusResolver interface {
	IsPartnerPubliclyVisible(ctx context.Context, partnerID int) (bool, error)
}

type CatalogService struct {
	m2Client      client.M2Client
	m4Client      client.M4Client
	partnerStatus PartnerStatusResolver
}

func NewCatalogService(m2 client.M2Client, m4 client.M4Client, status ...PartnerStatusResolver) *CatalogService {
	var resolver PartnerStatusResolver
	if len(status) > 0 {
		resolver = status[0]
	}
	return &CatalogService{m2Client: m2, m4Client: m4, partnerStatus: resolver}
}

func (s *CatalogService) GetBusinessUnits(ctx context.Context) ([]dto.BusinessUnitSummary, error) {
	units, err := s.m2Client.GetActiveBusinessUnits(ctx)
	if err != nil {
		return nil, err
	}
	if s.partnerStatus == nil {
		return units, nil
	}
	visible := make([]dto.BusinessUnitSummary, 0, len(units))
	for _, unit := range units {
		if unit.PartnerID == nil {
			visible = append(visible, unit)
			continue
		}
		ok, err := s.partnerStatus.IsPartnerPubliclyVisible(ctx, *unit.PartnerID)
		if err == nil && ok {
			visible = append(visible, unit)
		}
	}
	return visible, nil
}

func (s *CatalogService) GetProducts(ctx context.Context, f dto.CatalogFilter) ([]dto.ProductSummary, error) {
	return s.m4Client.GetActiveProducts(ctx, f.UnitUsahaID, f.Category, f.Search)
}

func (s *CatalogService) GetCatalog(ctx context.Context, f dto.CatalogFilter) (map[string]any, error) {
	units, err := s.GetBusinessUnits(ctx)
	if err != nil {
		return nil, err
	}
	products, err := s.m4Client.GetActiveProducts(ctx, f.UnitUsahaID, f.Category, f.Search)
	if err != nil {
		return nil, err
	}
	return map[string]any{"business_units": units, "products": products}, nil
}
