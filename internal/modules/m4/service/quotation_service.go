package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"araya-business-hub/be/internal/modules/m4/domain"
	"araya-business-hub/be/internal/modules/m4/dto"
	"araya-business-hub/be/internal/modules/m4/repository"
	"araya-business-hub/be/internal/platform/audit"
)

type QuotationService struct {
	repo     repository.Repository
	auditSvc *audit.Service
}

func NewQuotationService(repo repository.Repository, auditSvc *audit.Service) *QuotationService {
	return &QuotationService{repo: repo, auditSvc: auditSvc}
}

func formatQuotationNo(id int) string {
	// Full-ID numbering avoids collisions when two documents are created in
	// the same period (ARY-011).
	return fmt.Sprintf("Q-%s-%06d", time.Now().Format("20060102"), id)
}

func calculateQuotationItems(inputs []dto.QuotationItemInput) ([]domain.QuotationItem, int64, error) {
	items := make([]domain.QuotationItem, 0, len(inputs))
	var total int64
	for _, it := range inputs {
		if it.Quantity <= 0 || it.UnitPrice < 0 || it.ItemName == "" {
			return nil, 0, errors.New("nama item, kuantitas positif, dan harga valid wajib diisi")
		}
		sub := int64(it.Quantity) * it.UnitPrice
		total += sub
		items = append(items, domain.QuotationItem{
			ProductID: it.ProductID, ItemName: it.ItemName,
			Quantity: it.Quantity, UnitPrice: it.UnitPrice, Subtotal: sub,
		})
	}
	return items, total, nil
}

func (s *QuotationService) CreateQuotation(ctx context.Context, req dto.CreateQuotationRequest, actor string, scope *int) (*domain.Quotation, error) {
	if len(req.Items) == 0 {
		return nil, domain.ErrQuotationItemsRequired
	}
	// ARY-001: partner_id is derived from the authenticated actor. Only a
	// platform-wide actor (super admin) may stamp a specific tenant.
	partnerID := scope
	if partnerID == nil {
		partnerID = req.PartnerID
	}
	if err := s.repo.ValidateUnitOwnership(ctx, req.UnitUsahaID, partnerID); err != nil {
		return nil, err
	}
	if err := ensureCustomerTenant(ctx, s.repo, req.CustomerID, partnerID); err != nil {
		return nil, err
	}
	items, total, err := calculateQuotationItems(req.Items)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if err := ensureProductTenant(ctx, s.repo, item.ProductID, partnerID); err != nil {
			return nil, err
		}
	}
	q := &domain.Quotation{
		QuotationNo: formatQuotationNo(int(time.Now().UnixNano() % 10000)),
		UnitUsahaID: req.UnitUsahaID, PartnerID: partnerID, CustomerID: req.CustomerID,
		TotalAmount: total, Status: domain.QuotationDraft, Items: items,
	}
	if err := s.repo.CreateQuotation(ctx, q); err != nil {
		return nil, err
	}
	q.QuotationNo = formatQuotationNo(q.ID)
	if err := s.repo.UpdateQuotation(ctx, q); err != nil {
		return nil, err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "quotation.created", EntityType: "quotation", EntityID: fmt.Sprintf("%d", q.ID),
	})
	return q, nil
}

func (s *QuotationService) GetQuotation(ctx context.Context, id int, scope *int) (*domain.Quotation, error) {
	q, err := s.repo.GetQuotationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !canAccessTenant(q.PartnerID, scope) {
		return nil, domain.ErrQuotationNotFound
	}
	return q, nil
}

func (s *QuotationService) ApproveQuotation(ctx context.Context, id int, actor string, scope *int) (*domain.Quotation, error) {
	q, err := s.GetQuotation(ctx, id, scope)
	if err != nil {
		return nil, err
	}
	if err := q.Approve(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateQuotation(ctx, q); err != nil {
		return nil, err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "quotation.approved", EntityType: "quotation", EntityID: fmt.Sprintf("%d", id),
	})
	return q, nil
}

func (s *QuotationService) RejectQuotation(ctx context.Context, id int, actor string, scope *int) (*domain.Quotation, error) {
	q, err := s.GetQuotation(ctx, id, scope)
	if err != nil {
		return nil, err
	}
	if err := q.Reject(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateQuotation(ctx, q); err != nil {
		return nil, err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "quotation.rejected", EntityType: "quotation", EntityID: fmt.Sprintf("%d", id),
	})
	return q, nil
}

func (s *QuotationService) ListQuotations(ctx context.Context, status string, scope *int, limit, offset int) ([]domain.Quotation, int, error) {
	return s.repo.ListQuotations(ctx, status, scope, limit, offset)
}
