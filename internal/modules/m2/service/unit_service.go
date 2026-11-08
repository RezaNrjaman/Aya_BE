package service

import (
	"context"
	"fmt"

	"araya-business-hub/be/internal/modules/m2/domain"
	"araya-business-hub/be/internal/modules/m2/dto"
	"araya-business-hub/be/internal/modules/m2/repository"
	"araya-business-hub/be/internal/platform/audit"
)

type UnitService struct {
	repo     repository.UnitRepository
	auditSvc *audit.Service
}

func NewUnitService(repo repository.UnitRepository, auditSvc *audit.Service) *UnitService {
	return &UnitService{repo: repo, auditSvc: auditSvc}
}

func (s *UnitService) CreateUnit(ctx context.Context, req dto.CreateUnitRequest, actor string) (*domain.BusinessUnit, error) {
	u := &domain.BusinessUnit{
		Name: req.Name, CategoryID: req.CategoryID, CategoryName: req.CategoryName,
		DivisionID: req.DivisionID, DivisionName: req.DivisionName, Description: req.Description,
		ContactEmail: req.ContactEmail, ContactPhone: req.ContactPhone, Address: req.Address,
		LogoURL: req.LogoURL, IsActive: true,
	}
	if err := u.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "business_unit.created", EntityType: "business_unit", EntityID: fmt.Sprintf("%d", u.ID),
	})
	return u, nil
}

func (s *UnitService) GetUnit(ctx context.Context, id int) (*domain.BusinessUnit, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UnitService) ListUnits(ctx context.Context, search, cat string, activeOnly bool, limit, offset int) ([]domain.BusinessUnit, int, error) {
	return s.repo.List(ctx, search, cat, activeOnly, limit, offset)
}

func (s *UnitService) ListActiveUnits(ctx context.Context) ([]domain.BusinessUnit, error) {
	return s.repo.ListActive(ctx)
}

func applyBasicFields(u *domain.BusinessUnit, req dto.UpdateUnitRequest) {
	if req.Name != nil {
		u.Name = *req.Name
	}
	if req.CategoryID != nil {
		u.CategoryID = req.CategoryID
	}
	if req.CategoryName != nil {
		u.CategoryName = *req.CategoryName
	}
	if req.DivisionID != nil {
		u.DivisionID = req.DivisionID
	}
	if req.DivisionName != nil {
		u.DivisionName = *req.DivisionName
	}
	if req.Description != nil {
		u.Description = *req.Description
	}
}

func applyContactFields(u *domain.BusinessUnit, req dto.UpdateUnitRequest) {
	if req.ContactEmail != nil {
		u.ContactEmail = *req.ContactEmail
	}
	if req.ContactPhone != nil {
		u.ContactPhone = *req.ContactPhone
	}
	if req.Address != nil {
		u.Address = *req.Address
	}
	if req.LogoURL != nil {
		u.LogoURL = *req.LogoURL
	}
}

func applyUnitFields(u *domain.BusinessUnit, req dto.UpdateUnitRequest) {
	applyBasicFields(u, req)
	applyContactFields(u, req)
}

func (s *UnitService) UpdateUnit(ctx context.Context, id int, req dto.UpdateUnitRequest, actor string) (*domain.BusinessUnit, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	applyUnitFields(u, req)
	if err := u.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, u); err != nil {
		return nil, err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "business_unit.updated", EntityType: "business_unit", EntityID: fmt.Sprintf("%d", id),
	})
	return u, nil
}

func (s *UnitService) DeactivateUnit(ctx context.Context, id int, actor string) error {
	if err := s.repo.SetStatus(ctx, id, false); err != nil {
		return err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "business_unit.deactivated", EntityType: "business_unit", EntityID: fmt.Sprintf("%d", id),
	})
	return nil
}

func (s *UnitService) ActivateUnit(ctx context.Context, id int, actor string) error {
	if err := s.repo.SetStatus(ctx, id, true); err != nil {
		return err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "business_unit.activated", EntityType: "business_unit", EntityID: fmt.Sprintf("%d", id),
	})
	return nil
}
