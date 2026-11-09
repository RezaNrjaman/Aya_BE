package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"araya-business-hub/be/internal/modules/m4/domain"
	"araya-business-hub/be/internal/modules/m4/dto"
	"araya-business-hub/be/internal/modules/m4/payment"
	"araya-business-hub/be/internal/modules/m4/repository"
	"araya-business-hub/be/internal/platform/audit"
)

type InvoiceService struct {
	repo     repository.Repository
	adapter  *payment.SandboxAdapter
	auditSvc *audit.Service
}

func NewInvoiceService(repo repository.Repository, adapter *payment.SandboxAdapter, auditSvc *audit.Service) *InvoiceService {
	return &InvoiceService{repo: repo, adapter: adapter, auditSvc: auditSvc}
}

func formatInvoiceNo(id int) string {
	return fmt.Sprintf("INV-%s-%04d", time.Now().Format("20060102"), id%10000)
}

func calculateInvoiceItems(inputs []dto.InvoiceItemInput) ([]domain.InvoiceItem, int64) {
	items := make([]domain.InvoiceItem, 0, len(inputs))
	var total int64
	for _, it := range inputs {
		sub := int64(it.Quantity) * it.UnitPrice
		total += sub
		items = append(items, domain.InvoiceItem{
			ProductID: it.ProductID, ItemName: it.ItemName,
			Quantity: it.Quantity, UnitPrice: it.UnitPrice, Subtotal: sub,
		})
	}
	return items, total
}

func buildItemsFromQuotation(q *domain.Quotation) ([]domain.InvoiceItem, int64) {
	items := make([]domain.InvoiceItem, 0, len(q.Items))
	for _, it := range q.Items {
		items = append(items, domain.InvoiceItem{
			ProductID: it.ProductID, ItemName: it.ItemName,
			Quantity: it.Quantity, UnitPrice: it.UnitPrice, Subtotal: it.Subtotal,
		})
	}
	return items, q.TotalAmount
}

func resolveInvoiceItems(ctx context.Context, repo repository.Repository, req dto.CreateInvoiceRequest) ([]domain.InvoiceItem, int64, error) {
	if req.QuotationID != nil {
		q, err := repo.GetQuotationByID(ctx, *req.QuotationID)
		if err != nil {
			return nil, 0, err
		}
		if q.Status != domain.QuotationApproved {
			return nil, 0, errors.New("invoice hanya dapat dibuat dari quotation yang telah disetujui")
		}
		items, total := buildItemsFromQuotation(q)
		return items, total, nil
	}
	items, total := calculateInvoiceItems(req.Items)
	return items, total, nil
}

func (s *InvoiceService) CreateInvoice(ctx context.Context, req dto.CreateInvoiceRequest, actor string) (*domain.Invoice, error) {
	items, total, err := resolveInvoiceItems(ctx, s.repo, req)
	if err != nil {
		return nil, err
	}
	inv := &domain.Invoice{
		InvoiceNo:   formatInvoiceNo(int(time.Now().UnixNano() % 10000)),
		QuotationID: req.QuotationID, CustomerID: req.CustomerID,
		TotalAmount: total, Status: domain.InvoiceUnpaid, Items: items,
	}
	_ = s.repo.CreateInvoice(ctx, inv)
	inv.InvoiceNo = formatInvoiceNo(inv.ID)
	_ = s.repo.UpdateInvoice(ctx, inv)
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "invoice.created", EntityType: "invoice", EntityID: fmt.Sprintf("%d", inv.ID),
	})
	return inv, nil
}

func (s *InvoiceService) GetInvoice(ctx context.Context, id int) (*domain.Invoice, error) {
	return s.repo.GetInvoiceByID(ctx, id)
}

func (s *InvoiceService) ListInvoices(ctx context.Context, status string, limit, offset int) ([]domain.Invoice, int, error) {
	return s.repo.ListInvoices(ctx, status, limit, offset)
}

func checkExistingPayment(ctx context.Context, repo repository.Repository, invoiceID int, status string) (*domain.Payment, bool) {
	if status == domain.InvoicePaid {
		payments, _ := repo.ListPayments(ctx, invoiceID)
		if len(payments) > 0 {
			return &payments[0], true
		}
	}
	return nil, false
}

func (s *InvoiceService) applyPaymentOutcome(ctx context.Context, inv *domain.Invoice, pmt *domain.Payment, actor string) {
	if pmt.Status == domain.PaymentSuccess {
		inv.MarkAsPaid(time.Now().UTC())
		_ = s.repo.UpdateInvoice(ctx, inv)
		_ = s.auditSvc.Log(ctx, audit.Record{
			UserName: actor, Action: "invoice.paid", EntityType: "invoice", EntityID: fmt.Sprintf("%d", inv.ID),
		})
		return
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "invoice.payment_failed", EntityType: "invoice", EntityID: fmt.Sprintf("%d", inv.ID),
	})
}

func (s *InvoiceService) PayInvoice(ctx context.Context, id int, req dto.PayInvoiceRequest, actor string) (*domain.Payment, error) {
	inv, err := s.repo.GetInvoiceByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if pmt, ok := checkExistingPayment(ctx, s.repo, id, inv.Status); ok {
		return pmt, nil
	}
	if req.Amount != inv.TotalAmount {
		return nil, errors.New("nominal pembayaran tidak sesuai dengan total tagihan invoice")
	}
	pmt, err := s.adapter.ProcessPayment(id, req.Amount, req.Method, req.SimulateFailure)
	if err != nil {
		return nil, err
	}
	_ = s.repo.CreatePayment(ctx, pmt)
	s.applyPaymentOutcome(ctx, inv, pmt, actor)
	return pmt, nil
}

func (s *InvoiceService) VerifySignature(payload []byte, sig string) bool {
	return s.adapter.VerifySignature(payload, sig)
}

func (s *InvoiceService) GenerateSignature(payload []byte) string {
	return s.adapter.GenerateSignature(payload)
}

func mapCallbackStatus(raw string) string {
	if strings.EqualFold(raw, "SUCCESS") || strings.EqualFold(raw, "PAID") {
		return domain.PaymentSuccess
	}
	return domain.PaymentFailed
}

func (s *InvoiceService) HandlePaymentCallback(ctx context.Context, req dto.PaymentCallbackRequest) (*domain.Payment, error) {
	inv, err := s.repo.GetInvoiceByID(ctx, req.InvoiceID)
	if err != nil {
		return nil, err
	}
	if pmt, err := s.repo.GetPaymentByReference(ctx, req.PaymentReference); err == nil {
		return pmt, nil
	}
	if req.Amount != inv.TotalAmount {
		return nil, errors.New("nominal pembayaran callback tidak sesuai dengan tagihan invoice")
	}
	pmt := &domain.Payment{
		PaymentReference: req.PaymentReference, InvoiceID: req.InvoiceID,
		Amount: req.Amount, Method: "sandbox_callback", Status: mapCallbackStatus(req.Status),
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	_ = s.repo.CreatePayment(ctx, pmt)
	s.applyPaymentOutcome(ctx, inv, pmt, "system.payment_gateway")
	return pmt, nil
}
