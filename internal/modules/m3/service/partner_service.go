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

type UserProvisioner interface {
	// ProvisionPartner creates the owner account without any password and
	// returns (userID, activationToken). ARY-002.
	ProvisionPartner(ctx context.Context, email, name, phone string) (int, string, error)
	SetUserStatus(ctx context.Context, email string, isActive bool) error
	LinkPartnerUser(ctx context.Context, userID, partnerID int) error
	IssueActivationTokenForEmail(ctx context.Context, email string) (string, error)
}

type PartnerService struct {
	repo        repository.PartnerRepository
	auditSvc    *audit.Service
	provisioner UserProvisioner
}

func NewPartnerService(repo repository.PartnerRepository, auditSvc *audit.Service, prov ...UserProvisioner) *PartnerService {
	var p UserProvisioner
	if len(prov) > 0 {
		p = prov[0]
	}
	return &PartnerService{repo: repo, auditSvc: auditSvc, provisioner: p}
}

func (s *PartnerService) Register(ctx context.Context, req dto.RegisterPartnerRequest, actor string) (*domain.Partner, error) {
	_, lookupErr := s.repo.GetByEmail(ctx, req.Email)
	if lookupErr == nil {
		return nil, domain.ErrEmailAlreadyRegistered
	}
	if !errors.Is(lookupErr, domain.ErrPartnerNotFound) {
		// Do not mistake a database/schema outage for an unused e-mail address.
		return nil, lookupErr
	}
	code, err := domain.GenerateRegistrationCode()
	if err != nil {
		return nil, err
	}
	p := &domain.Partner{
		CompanyName: req.CompanyName, ContactPerson: req.ContactPerson,
		Email: req.Email, Phone: req.Phone, Address: req.Address, Status: domain.StatusPending,
		RegistrationCode: code,
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

func (s *PartnerService) GetPartnerByUserID(ctx context.Context, userID int) (*domain.Partner, error) {
	return s.repo.GetByUserID(ctx, userID)
}

// IsPartnerPubliclyVisible reports whether a partner may currently sell
// through the public storefront (ARY-004). Only partners in status
// "terverifikasi" are visible; suspended, pending and rejected partners are
// hidden. It satisfies m4's PartnerStatusResolver without creating an import
// from m4 to m3.
func (s *PartnerService) IsPartnerPubliclyVisible(ctx context.Context, partnerID int) (bool, error) {
	p, err := s.repo.GetByID(ctx, partnerID)
	if err != nil {
		if errors.Is(err, domain.ErrPartnerNotFound) {
			return false, nil
		}
		return false, err
	}
	return p.Status == domain.StatusVerified, nil
}

// EnsurePartnerAccess blocks partner-scoped actors from reading or mutating a
// different partner record. Admins (scope == nil, restricted == false) keep
// global access.
func (s *PartnerService) EnsurePartnerAccess(ctx context.Context, id int, scope *int, restricted bool) error {
	if !restricted {
		return nil
	}
	if scope == nil || *scope != id {
		return domain.ErrPartnerNotFound
	}
	return nil
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

// ListPartnersScoped returns only the actor's own partner for partner-scoped
// users. Search and status filters are intentionally ignored for that view.
func (s *PartnerService) ListPartnersScoped(ctx context.Context, scope *int, restricted bool, status, search string, limit, offset int) ([]domain.Partner, int, error) {
	if !restricted {
		return s.repo.List(ctx, status, search, limit, offset)
	}
	if scope == nil {
		return []domain.Partner{}, 0, nil
	}
	p, err := s.repo.GetByID(ctx, *scope)
	if err != nil {
		if errors.Is(err, domain.ErrPartnerNotFound) {
			return []domain.Partner{}, 0, nil
		}
		return nil, 0, err
	}
	return []domain.Partner{*p}, 1, nil
}

func (s *PartnerService) GetPartnerDetailScoped(ctx context.Context, id int, scope *int, restricted bool) (*dto.PartnerDetailResponse, error) {
	if err := s.EnsurePartnerAccess(ctx, id, scope, restricted); err != nil {
		return nil, err
	}
	return s.GetPartnerDetail(ctx, id)
}

// GetPublicStatusByCode returns the minimal registration status addressed by
// the partner's unguessable tracking code (ARY-002). It never exposes contact
// details or internal history. When the partner has just been verified and its
// account has never been activated, a fresh single-use activation invitation
// token is returned — the sandbox equivalent of the activation e-mail.
func (s *PartnerService) GetPublicStatusByCode(ctx context.Context, code string) (*dto.PublicPartnerStatusResponse, error) {
	p, err := s.repo.GetByRegistrationCode(ctx, code)
	if err != nil {
		return nil, domain.ErrPartnerNotFound
	}
	resp := &dto.PublicPartnerStatusResponse{
		PartnerID:        p.ID,
		CompanyName:      p.CompanyName,
		Status:           p.Status,
		RejectionReason:  p.RejectionReason,
		SuspensionReason: p.SuspensionReason,
	}
	if p.Status == domain.StatusVerified && s.provisioner != nil {
		if token, err := s.provisioner.IssueActivationTokenForEmail(ctx, p.Email); err == nil {
			resp.ActivationToken = token
		}
	}
	return resp, nil
}

func (s *PartnerService) ListVerifiedPartners(ctx context.Context) ([]domain.Partner, error) {
	return s.repo.ListVerified(ctx)
}

// PublicVerifiedPartners projects the verified partner list to its safe public
// shape (no tracking codes, verification metadata or linkage IDs).
func (s *PartnerService) PublicVerifiedPartners(ctx context.Context) ([]dto.PublicPartnerView, error) {
	partners, err := s.repo.ListVerified(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]dto.PublicPartnerView, 0, len(partners))
	for _, p := range partners {
		views = append(views, dto.PublicPartnerView{
			ID: p.ID, CompanyName: p.CompanyName, ContactPerson: p.ContactPerson,
			Email: p.Email, Phone: p.Phone, Address: p.Address, Status: p.Status,
		})
	}
	return views, nil
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
	if strings.EqualFold(strings.TrimSpace(req.Action), "approve") && s.provisioner != nil {
		// ARY-002: no universal initial password. The owner receives a
		// single-use activation invitation instead.
		uid, _, err := s.provisioner.ProvisionPartner(ctx, p.Email, p.ContactPerson, p.Phone)
		if err != nil {
			return nil, fmt.Errorf("gagal menyiapkan akun partner: %w", err)
		}
		if uid > 0 {
			p.UserID = &uid
			if err := s.provisioner.LinkPartnerUser(ctx, uid, p.ID); err != nil {
				return nil, fmt.Errorf("gagal menghubungkan akun partner: %w", err)
			}
		}
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
	previous := *p
	fromStatus := p.Status
	if err := p.Suspend(reason); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	if s.provisioner != nil {
		if tenantStatus, ok := s.provisioner.(interface {
			SetPartnerStatus(context.Context, int, bool) error
		}); ok {
			if err := tenantStatus.SetPartnerStatus(ctx, p.ID, false); err != nil {
				_ = s.repo.Update(ctx, &previous)
				return nil, fmt.Errorf("gagal menangguhkan seluruh akun partner: %w", err)
			}
		} else if err := s.provisioner.SetUserStatus(ctx, p.Email, false); err != nil {
			_ = s.repo.Update(ctx, &previous)
			return nil, fmt.Errorf("gagal menonaktifkan akun partner: %w", err)
		}
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
	previous := *p
	fromStatus := p.Status
	if err := p.Reactivate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	if s.provisioner != nil {
		if tenantStatus, ok := s.provisioner.(interface {
			SetPartnerStatus(context.Context, int, bool) error
		}); ok {
			if err := tenantStatus.SetPartnerStatus(ctx, p.ID, true); err != nil {
				_ = s.repo.Update(ctx, &previous)
				return nil, fmt.Errorf("gagal mengaktifkan kembali seluruh akun partner: %w", err)
			}
		} else if err := s.provisioner.SetUserStatus(ctx, p.Email, true); err != nil {
			_ = s.repo.Update(ctx, &previous)
			return nil, fmt.Errorf("gagal mengaktifkan kembali akun partner: %w", err)
		}
	}
	_ = s.repo.AddHistory(ctx, &domain.StatusHistory{
		PartnerID: id, FromStatus: fromStatus, ToStatus: p.Status, Reason: "Diaktifkan kembali oleh admin", ChangedBy: actor,
	})
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "partner.reactivated", EntityType: "partner", EntityID: fmt.Sprintf("%d", id),
	})
	return p, nil
}
