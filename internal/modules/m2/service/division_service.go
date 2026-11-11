package service

import (
	"context"
	"errors"
	"fmt"

	"araya-business-hub/be/internal/modules/m2/domain"
	"araya-business-hub/be/internal/modules/m2/dto"
	"araya-business-hub/be/internal/modules/m2/repository"
	"araya-business-hub/be/internal/platform/audit"
)

// DivisionService manages partner-owned divisions (ARY-006).
type DivisionService struct {
	repo     repository.DivisionRepository
	auditSvc *audit.Service
}

func NewDivisionService(repo repository.DivisionRepository, auditSvc *audit.Service) *DivisionService {
	return &DivisionService{repo: repo, auditSvc: auditSvc}
}

func (s *DivisionService) CreateDivision(ctx context.Context, req dto.CreateDivisionRequest, actor string, scope *int) (*domain.Division, error) {
	d := &domain.Division{Name: req.Name, Description: req.Description, UnitUsahaID: req.UnitUsahaID, CategoryID: req.CategoryID}
	if scope == nil {
		return nil, errors.New("divisi hanya dapat dibuat oleh Partner")
	}
	// Tenant ownership is derived from the authenticated actor.
	d.PartnerID = scope
	if err := d.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.ValidateDivisionPlacement(ctx, *scope, d.UnitUsahaID, d.CategoryID); err != nil {
		return nil, err
	}
	if err := s.repo.CreateDivision(ctx, d); err != nil {
		return nil, err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "division.created", EntityType: "division", EntityID: fmt.Sprintf("%d", d.ID),
	})
	return d, nil
}

func (s *DivisionService) GetDivision(ctx context.Context, id int, scope *int) (*domain.Division, error) {
	d, err := s.repo.GetDivisionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !canAccessTenantUnit(d.PartnerID, scope) {
		return nil, domain.ErrDivisionNotFound
	}
	return d, nil
}

func (s *DivisionService) ListDivisions(ctx context.Context, search string, scope *int, limit, offset int) ([]domain.Division, int, error) {
	return s.repo.ListDivisions(ctx, search, scope, limit, offset)
}

func applyDivisionUpdates(d *domain.Division, req dto.UpdateDivisionRequest) {
	if req.Name != nil {
		d.Name = *req.Name
	}
	if req.Description != nil {
		d.Description = *req.Description
	}
	if req.UnitUsahaID != nil {
		d.UnitUsahaID = *req.UnitUsahaID
	}
	if req.CategoryID != nil {
		d.CategoryID = *req.CategoryID
	}
}

func (s *DivisionService) UpdateDivision(ctx context.Context, id int, req dto.UpdateDivisionRequest, actor string, scope *int) (*domain.Division, error) {
	d, err := s.repo.GetDivisionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !canAccessTenantUnit(d.PartnerID, scope) {
		return nil, domain.ErrDivisionNotFound
	}
	applyDivisionUpdates(d, req)
	if scope == nil {
		return nil, errors.New("divisi hanya dapat dikelola oleh Partner")
	}
	if err := s.repo.ValidateDivisionPlacement(ctx, *scope, d.UnitUsahaID, d.CategoryID); err != nil {
		return nil, err
	}
	if err := d.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateDivision(ctx, d); err != nil {
		return nil, err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "division.updated", EntityType: "division", EntityID: fmt.Sprintf("%d", id),
	})
	return d, nil
}

// DeactivateDivision soft-disables a division. History (users, units,
// transactions) is untouched.
func (s *DivisionService) DeactivateDivision(ctx context.Context, id int, actor string, scope *int) error {
	d, err := s.repo.GetDivisionByID(ctx, id)
	if err != nil {
		return err
	}
	if !canAccessTenantUnit(d.PartnerID, scope) {
		return domain.ErrDivisionNotFound
	}
	if err := s.repo.SetDivisionStatus(ctx, id, false); err != nil {
		return err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "division.deactivated", EntityType: "division", EntityID: fmt.Sprintf("%d", id),
	})
	return nil
}

func (s *DivisionService) ActivateDivision(ctx context.Context, id int, actor string, scope *int) error {
	d, err := s.repo.GetDivisionByID(ctx, id)
	if err != nil {
		return err
	}
	if !canAccessTenantUnit(d.PartnerID, scope) {
		return domain.ErrDivisionNotFound
	}
	if err := s.repo.SetDivisionStatus(ctx, id, true); err != nil {
		return err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "division.activated", EntityType: "division", EntityID: fmt.Sprintf("%d", id),
	})
	return nil
}

// DeleteDivision removes a division outright — only when nothing references it
// anymore (ARY-006: no deletion without moving members first).
func (s *DivisionService) DeleteDivision(ctx context.Context, id int, actor string, scope *int) error {
	d, err := s.repo.GetDivisionByID(ctx, id)
	if err != nil {
		return err
	}
	if !canAccessTenantUnit(d.PartnerID, scope) {
		return domain.ErrDivisionNotFound
	}
	users, units, err := s.repo.CountDivisionReferences(ctx, id)
	if err != nil {
		return err
	}
	if users > 0 || units > 0 {
		return domain.ErrDivisionInUse
	}
	if err := s.repo.DeleteDivision(ctx, id); err != nil {
		return err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "division.deleted", EntityType: "division", EntityID: fmt.Sprintf("%d", id),
	})
	return nil
}
