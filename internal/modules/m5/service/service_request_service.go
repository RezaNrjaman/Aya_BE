package service

import (
	"context"
	"fmt"
	"strings"

	"araya-business-hub/be/internal/modules/m5/domain"
	"araya-business-hub/be/internal/modules/m5/dto"
	"araya-business-hub/be/internal/modules/m5/repository"
	"araya-business-hub/be/internal/platform/audit"
)

type ServiceRequestService struct {
	repo     repository.Repository
	auditSvc *audit.Service
}

func NewServiceRequestService(repo repository.Repository, auditSvc *audit.Service) *ServiceRequestService {
	return &ServiceRequestService{repo: repo, auditSvc: auditSvc}
}

func (s *ServiceRequestService) CreateRequest(ctx context.Context, req dto.CreateServiceRequest) (*domain.ServiceRequest, error) {
	sr := &domain.ServiceRequest{
		UnitUsahaID: req.UnitUsahaID, CustomerName: req.CustomerName,
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
		UserName: "customer", Action: "service_request.created", EntityType: "service_request",
		EntityID: fmt.Sprintf("%d", sr.ID), Details: map[string]any{"unit_usaha_id": sr.UnitUsahaID},
	})
	return sr, nil
}

func (s *ServiceRequestService) GetRequest(ctx context.Context, id int) (*domain.ServiceRequest, error) {
	return s.repo.GetServiceRequestByID(ctx, id)
}

func (s *ServiceRequestService) ListRequests(ctx context.Context, unitID *int, status string, limit, offset int) ([]domain.ServiceRequest, int, error) {
	return s.repo.ListServiceRequests(ctx, unitID, status, limit, offset)
}

func applyStatusChange(sr *domain.ServiceRequest, target string) error {
	switch strings.ToLower(target) {
	case domain.RequestProcessing:
		return sr.Process()
	case domain.RequestCompleted:
		return sr.Complete()
	default:
		return domain.ErrInvalidRequestStatus
	}
}

func (s *ServiceRequestService) UpdateRequestStatus(ctx context.Context, id int, status, actor string) (*domain.ServiceRequest, error) {
	sr, err := s.repo.GetServiceRequestByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := applyStatusChange(sr, status); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateServiceRequest(ctx, sr); err != nil {
		return nil, err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "service_request.status_updated", EntityType: "service_request",
		EntityID: fmt.Sprintf("%d", id), Details: map[string]any{"status": status},
	})
	return sr, nil
}
