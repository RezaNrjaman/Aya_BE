package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"araya-business-hub/be/internal/modules/m5/domain"
	"araya-business-hub/be/internal/modules/m5/dto"
	"araya-business-hub/be/internal/modules/m5/repository"
	"araya-business-hub/be/internal/platform/audit"
)

type PublicUnitCatalog interface {
	GetBusinessUnits(ctx context.Context) ([]dto.BusinessUnitSummary, error)
}

type ServiceRequestService struct {
	repo     repository.Repository
	auditSvc *audit.Service
	catalog  PublicUnitCatalog
}

func NewServiceRequestService(repo repository.Repository, auditSvc *audit.Service, catalog PublicUnitCatalog) *ServiceRequestService {
	return &ServiceRequestService{repo: repo, auditSvc: auditSvc, catalog: catalog}
}

func (s *ServiceRequestService) resolvePublicUnit(ctx context.Context, unitID int) (*int, error) {
	if s.catalog == nil {
		return nil, errors.New("katalog unit usaha belum tersedia")
	}
	units, err := s.catalog.GetBusinessUnits(ctx)
	if err != nil {
		return nil, err
	}
	for _, unit := range units {
		if unit.ID == unitID {
			return unit.PartnerID, nil
		}
	}
	return nil, errors.New("unit usaha tidak aktif atau tidak tersedia untuk publik")
}

func (s *ServiceRequestService) CreateRequest(ctx context.Context, req dto.CreateServiceRequest) (*domain.ServiceRequest, error) {
	partnerID, err := s.resolvePublicUnit(ctx, req.UnitUsahaID)
	if err != nil {
		return nil, err
	}
	sr := &domain.ServiceRequest{
		PartnerID: partnerID, UnitUsahaID: req.UnitUsahaID, CustomerName: req.CustomerName,
		CustomerEmail: req.CustomerEmail, CustomerPhone: req.CustomerPhone,
		ServiceType: req.ServiceType, Description: req.Description,
		Status: domain.RequestPending,
	}
	if err := sr.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.CreateServiceRequest(ctx, sr); err != nil {
		return nil, err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		PartnerID: partnerID, UserName: "customer", Action: "service_request.created", EntityType: "service_request",
		EntityID: fmt.Sprintf("%d", sr.ID), Details: map[string]any{"unit_usaha_id": sr.UnitUsahaID},
	})
	return sr, nil
}

func (s *ServiceRequestService) GetRequest(ctx context.Context, id int, partnerID *int, restricted bool) (*domain.ServiceRequest, error) {
	return s.repo.GetServiceRequestByID(ctx, id, partnerID, restricted)
}

func (s *ServiceRequestService) ListRequests(ctx context.Context, unitID *int, status string, partnerID *int, restricted bool, limit, offset int) ([]domain.ServiceRequest, int, error) {
	if restricted && partnerID == nil {
		return []domain.ServiceRequest{}, 0, nil
	}
	return s.repo.ListServiceRequests(ctx, unitID, partnerID, restricted, status, limit, offset)
}

func applyStatusChange(sr *domain.ServiceRequest, target string) error {
	switch strings.ToLower(strings.TrimSpace(target)) {
	case domain.RequestProcessing:
		return sr.Process()
	case domain.RequestCompleted:
		return sr.Complete()
	default:
		return domain.ErrInvalidRequestStatus
	}
}

func (s *ServiceRequestService) UpdateRequestStatus(ctx context.Context, id int, status, actor string, partnerID *int, restricted bool) (*domain.ServiceRequest, error) {
	sr, err := s.repo.GetServiceRequestByID(ctx, id, partnerID, restricted)
	if err != nil {
		return nil, err
	}
	if err := applyStatusChange(sr, status); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateServiceRequest(ctx, sr, partnerID, restricted); err != nil {
		return nil, err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		PartnerID: partnerID, UserName: actor, Action: "service_request.status_updated", EntityType: "service_request",
		EntityID: fmt.Sprintf("%d", id), Details: map[string]any{"status": status},
	})
	return sr, nil
}
