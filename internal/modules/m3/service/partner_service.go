package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"araya-business-hub/be/internal/modules/m3/domain"
	"araya-business-hub/be/internal/modules/m3/dto"
	"araya-business-hub/be/internal/modules/m3/repository"
	"araya-business-hub/be/internal/platform/audit"
)

type PartnerService struct {
	repo     repository.PartnerRepository
	auditSvc *audit.Service
}

func NewPartnerService(repo repository.PartnerRepository, auditSvc *audit.Service) *PartnerService {
	return &PartnerService{repo: repo, auditSvc: auditSvc}
}

func (s *PartnerService) Register(ctx context.Context, req dto.RegisterPartnerRequest, actor string) (*domain.Partner, error) {
	if _, err := s.repo.GetByEmail(ctx, req.Email); err == nil {
		return nil, domain.ErrEmailAlreadyRegistered
	}
	p := &domain.Partner{
		CompanyName: req.CompanyName, ContactPerson: req.ContactPerson,
		Email: req.Email, Phone: req.Phone, Address: req.Address, Status: domain.StatusPending,
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	_ = s.repo.AddHistory(ctx, &domain.StatusHistory{
		PartnerID: p.ID, ToStatus: domain.StatusPending, Reason: "Pendaftaran partner baru", ChangedBy: actor,
	})
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "partner.registered", EntityType: "partner", EntityID: fmt.Sprintf("%d", p.ID),
	})
	return p, nil
}

func (s *PartnerService) GetPartner(ctx context.Context, id int) (*domain.Partner, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *PartnerService) GetPartnerDetail(ctx context.Context, id int) (*dto.PartnerDetailResponse, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	histories, err := s.repo.GetHistories(ctx, id)
	if err != nil {
		return nil, err
	}
	return &dto.PartnerDetailResponse{Partner: *p, Histories: histories}, nil
}

func (s *PartnerService) ListPartners(ctx context.Context, status, search string, limit, offset int) ([]domain.Partner, int, error) {
	return s.repo.List(ctx, status, search, limit, offset)
}

func (s *PartnerService) ListVerifiedPartners(ctx context.Context) ([]domain.Partner, error) {
	return s.repo.ListVerified(ctx)
}

func applyVerification(p *domain.Partner, req dto.VerifyPartnerRequest, actor string) (string, error) {
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "approve" {
		return "Disetujui oleh admin", p.Verify(actor)
	}
	if action == "reject" {
		return req.Reason, p.Reject(req.Reason, actor)
	}
	return "", errors.New("aksi verifikasi harus 'approve' atau 'reject'")
}

func (s *PartnerService) VerifyPartner(ctx context.Context, id int, req dto.VerifyPartnerRequest, actor string) (*domain.Partner, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	fromStatus := p.Status
	reason, err := applyVerification(p, req, actor)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	_ = s.repo.AddHistory(ctx, &domain.StatusHistory{
		PartnerID: id, FromStatus: fromStatus, ToStatus: p.Status, Reason: reason, ChangedBy: actor,
	})
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "partner.verified", EntityType: "partner", EntityID: fmt.Sprintf("%d", id),
	})
	return p, nil
}

func (s *PartnerService) SuspendPartner(ctx context.Context, id int, reason, actor string) (*domain.Partner, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	fromStatus := p.Status
	if err := p.Suspend(reason); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	_ = s.repo.AddHistory(ctx, &domain.StatusHistory{
		PartnerID: id, FromStatus: fromStatus, ToStatus: p.Status, Reason: reason, ChangedBy: actor,
	})
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "partner.suspended", EntityType: "partner", EntityID: fmt.Sprintf("%d", id),
	})
	return p, nil
}

func (s *PartnerService) ReactivatePartner(ctx context.Context, id int, actor string) (*domain.Partner, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	fromStatus := p.Status
	if err := p.Reactivate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	_ = s.repo.AddHistory(ctx, &domain.StatusHistory{
		PartnerID: id, FromStatus: fromStatus, ToStatus: p.Status, Reason: "Diaktifkan kembali oleh admin", ChangedBy: actor,
	})
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "partner.reactivated", EntityType: "partner", EntityID: fmt.Sprintf("%d", id),
	})
	return p, nil
}
