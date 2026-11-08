package service

import (
	"context"
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
	return fmt.Sprintf("Q-%s-%04d", time.Now().Format("20060102"), id%10000)
}

func calculateQuotationItems(inputs []dto.QuotationItemInput) ([]domain.QuotationItem, int64) {
	items := make([]domain.QuotationItem, 0, len(inputs))
	var total int64
	for _, it := range inputs {
		sub := int64(it.Quantity) * it.UnitPrice
		total += sub
		items = append(items, domain.QuotationItem{
			ProductID: it.ProductID, ItemName: it.ItemName,
			Quantity: it.Quantity, UnitPrice: it.UnitPrice, Subtotal: sub,
		})
	}
	return items, total
}

func (s *QuotationService) CreateQuotation(ctx context.Context, req dto.CreateQuotationRequest, actor string) (*domain.Quotation, error) {
	if len(req.Items) == 0 {
		return nil, domain.ErrQuotationItemsRequired
	}
	items, total := calculateQuotationItems(req.Items)
	q := &domain.Quotation{
		UnitUsahaID: req.UnitUsahaID, PartnerID: req.PartnerID, CustomerID: req.CustomerID,
		TotalAmount: total, Status: domain.QuotationDraft, Items: items,
	}
	if err := s.repo.CreateQuotation(ctx, q); err != nil {
		return nil, err
	}
	q.QuotationNo = formatQuotationNo(q.ID)
	_ = s.repo.UpdateQuotation(ctx, q)
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "quotation.created", EntityType: "quotation", EntityID: fmt.Sprintf("%d", q.ID),
	})
	return q, nil
}

func (s *QuotationService) GetQuotation(ctx context.Context, id int) (*domain.Quotation, error) {
	return s.repo.GetQuotationByID(ctx, id)
}

func (s *QuotationService) ApproveQuotation(ctx context.Context, id int, actor string) (*domain.Quotation, error) {
	q, err := s.repo.GetQuotationByID(ctx, id)
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

func (s *QuotationService) RejectQuotation(ctx context.Context, id int, actor string) (*domain.Quotation, error) {
	q, err := s.repo.GetQuotationByID(ctx, id)
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

func (s *QuotationService) ListQuotations(ctx context.Context, status string, limit, offset int) ([]domain.Quotation, int, error) {
	return s.repo.ListQuotations(ctx, status, limit, offset)
}
