package service

import (
	"context"
	"fmt"

	"araya-business-hub/be/internal/modules/m4/domain"
	"araya-business-hub/be/internal/modules/m4/dto"
	"araya-business-hub/be/internal/modules/m4/repository"
	"araya-business-hub/be/internal/platform/audit"
)

// PartnerStatusResolver reports whether a partner may sell through the public
// storefront (ARY-004). It is implemented by module m3, so m4 stays decoupled
// from partner management internals.
type PartnerStatusResolver interface {
	IsPartnerPubliclyVisible(ctx context.Context, partnerID int) (bool, error)
}

type ProductService struct {
	repo          repository.Repository
	auditSvc      *audit.Service
	partnerStatus PartnerStatusResolver
}

func NewProductService(repo repository.Repository, auditSvc *audit.Service, partnerStatus ...PartnerStatusResolver) *ProductService {
	var ps PartnerStatusResolver
	if len(partnerStatus) > 0 {
		ps = partnerStatus[0]
	}
	return &ProductService{repo: repo, auditSvc: auditSvc, partnerStatus: ps}
}

// isPubliclyVisible reports whether a product may appear in the public
// storefront (ARY-004). Platform-owned products (nil partner) are always
// visible; partner-owned products require the partner to be verified. The
// check fails closed when the resolver errors, so a suspension is never
// bypassed by an infrastructure hiccup.
func (s *ProductService) isPubliclyVisible(ctx context.Context, p *domain.Product) bool {
	if p.PartnerID == nil {
		return true
	}
	if s.partnerStatus == nil {
		return true
	}
	visible, err := s.partnerStatus.IsPartnerPubliclyVisible(ctx, *p.PartnerID)
	return err == nil && visible
}

// canAccessTenant reports whether a resource owned by tenant partnerID may be
// accessed by an actor restricted to scope (ARY-001). A nil scope means the
// actor is platform-wide (super admin or an explicitly public flow).
func canAccessTenant(resourcePartnerID *int, scope *int) bool {
	if scope == nil {
		return true
	}
	return resourcePartnerID != nil && *resourcePartnerID == *scope
}

func (s *ProductService) CreateProduct(ctx context.Context, req dto.CreateProductRequest, actor string, scope *int) (*domain.Product, error) {
	p := &domain.Product{
		UnitUsahaID: req.UnitUsahaID, PartnerID: req.PartnerID, Name: req.Name,
		Category: req.Category, Description: req.Description, Price: req.Price,
		Stock: req.Stock, IsActive: true, ImageURL: req.ImageURL,
	}
	if scope != nil {
		// ARY-001: partner-scoped actors always own their products; a
		// request-body partner_id is never trusted.
		p.PartnerID = scope
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

func (s *ProductService) GetProduct(ctx context.Context, id int, scope *int) (*domain.Product, error) {
	p, err := s.repo.GetProductByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !canAccessTenant(p.PartnerID, scope) {
		return nil, domain.ErrProductNotFound
	}
	return p, nil
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

func (s *ProductService) UpdateProduct(ctx context.Context, id int, req dto.UpdateProductRequest, actor string, scope *int) (*domain.Product, error) {
	p, err := s.repo.GetProductByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !canAccessTenant(p.PartnerID, scope) {
		return nil, domain.ErrProductNotFound
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

func (s *ProductService) ListProducts(ctx context.Context, unitID *int, cat, search string, activeOnly bool, scope *int, limit, offset int) ([]domain.Product, int, error) {
	return s.repo.ListProducts(ctx, unitID, cat, search, activeOnly, scope, limit, offset)
}

// ListActiveProducts serves Kontrak K-05: GET /api/m4/v1/products.
// ARY-004: products of suspended or not-yet-verified partners are hidden from
// the storefront; platform products (partner_id NULL) always remain listed.
func (s *ProductService) ListActiveProducts(ctx context.Context, unitID *int, cat, search string, limit, offset int) ([]domain.Product, int, error) {
	prods, _, err := s.repo.ListProducts(ctx, unitID, cat, search, true, nil, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	visible := make([]domain.Product, 0, len(prods))
	for i := range prods {
		if s.isPubliclyVisible(ctx, &prods[i]) {
			visible = append(visible, prods[i])
		}
	}
	return visible, len(visible), nil
}

// GetPublicProduct returns a product for storefront consumption (cart and
// checkout, ARY-004) only when it is active and its partner is publicly
// visible. Platform-owned products always pass the partner check.
func (s *ProductService) GetPublicProduct(ctx context.Context, id int) (*domain.Product, error) {
	p, err := s.repo.GetProductByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !p.IsActive || !s.isPubliclyVisible(ctx, p) {
		return nil, domain.ErrProductNotFound
	}
	return p, nil
}
