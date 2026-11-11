package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"araya-business-hub/be/internal/modules/m2/domain"
	"araya-business-hub/be/internal/modules/m2/dto"
	"araya-business-hub/be/internal/modules/m2/repository"
	"araya-business-hub/be/internal/platform/audit"
)

type CategoryService struct {
	repo  repository.UnitRepository
	audit *audit.Service
}

func NewCategoryService(repo repository.UnitRepository, auditSvc *audit.Service) *CategoryService {
	return &CategoryService{repo: repo, audit: auditSvc}
}

func (s *CategoryService) Create(ctx context.Context, req dto.CreateCategoryRequest, actor string, partnerID *int) (*domain.BusinessCategory, error) {
	if partnerID == nil {
		return nil, errors.New("Partner tidak teridentifikasi")
	}
	if req.UnitUsahaID <= 0 {
		return nil, errors.New("unit usaha wajib dipilih")
	}
	c := &domain.BusinessCategory{PartnerID: partnerID, UnitUsahaID: req.UnitUsahaID, Name: strings.TrimSpace(req.Name), Description: strings.TrimSpace(req.Description)}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.CreateCategory(ctx, c); err != nil {
		return nil, err
	}
	_ = s.audit.Log(ctx, audit.Record{UserName: actor, Action: "category.created", EntityType: "business_category", EntityID: fmt.Sprint(c.ID)})
	return c, nil
}

func (s *CategoryService) List(ctx context.Context, unitID, partnerID *int, search string, limit, offset int) ([]domain.BusinessCategory, int, error) {
	return s.repo.ListCategories(ctx, unitID, partnerID, search, limit, offset)
}

func (s *CategoryService) Update(ctx context.Context, id int, req dto.UpdateCategoryRequest, actor string, partnerID *int) (*domain.BusinessCategory, error) {
	if partnerID == nil {
		return nil, errors.New("Partner tidak teridentifikasi")
	}
	c, err := s.repo.GetCategoryByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.PartnerID == nil || *c.PartnerID != *partnerID {
		return nil, errors.New("kategori tidak ditemukan")
	}
	if req.Name != nil {
		c.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		c.Description = strings.TrimSpace(*req.Description)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateCategory(ctx, c); err != nil {
		return nil, err
	}
	_ = s.audit.Log(ctx, audit.Record{UserName: actor, Action: "category.updated", EntityType: "business_category", EntityID: fmt.Sprint(id)})
	return c, nil
}

func (s *CategoryService) SetStatus(ctx context.Context, id int, active bool, actor string, partnerID *int) error {
	if partnerID == nil {
		return errors.New("Partner tidak teridentifikasi")
	}
	c, err := s.repo.GetCategoryByID(ctx, id)
	if err != nil {
		return err
	}
	if c.PartnerID == nil || *c.PartnerID != *partnerID {
		return errors.New("kategori tidak ditemukan")
	}
	if err := s.repo.SetCategoryStatus(ctx, id, active); err != nil {
		return err
	}
	action := "category.deactivated"
	if active {
		action = "category.activated"
	}
	_ = s.audit.Log(ctx, audit.Record{UserName: actor, Action: action, EntityType: "business_category", EntityID: fmt.Sprint(id)})
	return nil
}
