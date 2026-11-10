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
	// Full-ID numbering avoids collisions (ARY-011).
	return fmt.Sprintf("INV-%s-%06d", time.Now().Format("20060102"), id)
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

// calculateInvoiceItemsSafe validates manual invoice items: quantities and
// prices must be positive so a malformed request can't create a broken
// document (ARY-011).
func calculateInvoiceItemsSafe(inputs []dto.InvoiceItemInput) ([]domain.InvoiceItem, int64, error) {
	if len(inputs) == 0 {
		return nil, 0, errors.New("item invoice wajib diisi minimal 1 item")
	}
	items := make([]domain.InvoiceItem, 0, len(inputs))
	var total int64
	for _, it := range inputs {
		if it.Quantity <= 0 || it.UnitPrice < 0 {
			return nil, 0, errors.New("kuantitas dan harga item invoice harus valid")
		}
		sub := int64(it.Quantity) * it.UnitPrice
		total += sub
		items = append(items, domain.InvoiceItem{
			ProductID: it.ProductID, ItemName: it.ItemName,
			Quantity: it.Quantity, UnitPrice: it.UnitPrice, Subtotal: sub,
		})
	}
	return items, total, nil
}

func (s *InvoiceService) CreateInvoice(ctx context.Context, req dto.CreateInvoiceRequest, actor string, scope *int) (*domain.Invoice, error) {
	if req.CustomerID <= 0 {
		return nil, errors.New("pelanggan invoice wajib dipilih")
	}
	var quotation *domain.Quotation
	var items []domain.InvoiceItem
	var total int64
	if req.QuotationID != nil {
		q, err := s.repo.GetQuotationByID(ctx, *req.QuotationID)
		if err != nil {
			return nil, err
		}
		if !canAccessTenant(q.PartnerID, scope) {
			return nil, domain.ErrQuotationNotFound
		}
		if q.Status != domain.QuotationApproved {
			return nil, errors.New("invoice hanya dapat dibuat dari quotation yang telah disetujui")
		}
		quotation = q
		items, total = buildItemsFromQuotation(q)
	} else {
		var err error
		items, total, err = calculateInvoiceItemsSafe(req.Items)
		if err != nil {
			return nil, err
		}
	}
	// ARY-001: tenant is the quotation's tenant, else the actor's scope.
	// A request never dictates the tenant.
	partnerID := scope
	if quotation != nil {
		partnerID = quotation.PartnerID
	}
	inv := &domain.Invoice{
		InvoiceNo:   formatInvoiceNo(int(time.Now().UnixNano() % 10000)),
		QuotationID: req.QuotationID, PartnerID: partnerID, CustomerID: req.CustomerID,
		TotalAmount: total, Status: domain.InvoiceUnpaid, Items: items,
	}
	if req.DueDateDays > 0 {
		due := time.Now().AddDate(0, 0, req.DueDateDays)
		inv.DueDate = &due
	}
	// ARY-011: surface persistence failures instead of fabricating a success.
	if err := s.repo.CreateInvoice(ctx, inv); err != nil {
		return nil, err
	}
	inv.InvoiceNo = formatInvoiceNo(inv.ID)
	if err := s.repo.UpdateInvoice(ctx, inv); err != nil {
		return nil, err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "invoice.created", EntityType: "invoice", EntityID: fmt.Sprintf("%d", inv.ID),
	})
	return inv, nil
}

func (s *InvoiceService) GetInvoice(ctx context.Context, id int, scope *int) (*domain.Invoice, error) {
	inv, err := s.repo.GetInvoiceByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !canAccessTenant(inv.PartnerID, scope) {
		return nil, domain.ErrInvoiceNotFound
	}
	return inv, nil
}

func (s *InvoiceService) ListInvoices(ctx context.Context, status string, scope *int, limit, offset int) ([]domain.Invoice, int, error) {
	return s.repo.ListInvoices(ctx, status, scope, limit, offset)
}

func checkExistingPayment(ctx context.Context, repo repository.Repository, invoiceID int, status string) (*domain.Payment, bool, error) {
	if status == domain.InvoicePaid {
		payments, err := repo.ListPayments(ctx, invoiceID)
		if err != nil {
			return nil, false, err
		}
		if len(payments) > 0 {
			return &payments[0], true, nil
		}
	}
	return nil, false, nil
}

func (s *InvoiceService) applyPaymentOutcome(ctx context.Context, inv *domain.Invoice, pmt *domain.Payment, actor string) error {
	if pmt.Status == domain.PaymentSuccess {
		inv.MarkAsPaid(time.Now().UTC())
		if err := s.repo.UpdateInvoice(ctx, inv); err != nil {
			return err
		}
		_ = s.auditSvc.Log(ctx, audit.Record{
			UserName: actor, Action: "invoice.paid", EntityType: "invoice", EntityID: fmt.Sprintf("%d", inv.ID),
		})
		return nil
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "invoice.payment_failed", EntityType: "invoice", EntityID: fmt.Sprintf("%d", inv.ID),
	})
	return nil
}

func (s *InvoiceService) PayInvoice(ctx context.Context, id int, req dto.PayInvoiceRequest, actor string, scope *int) (*domain.Payment, error) {
	inv, err := s.GetInvoice(ctx, id, scope)
	if err != nil {
		return nil, err
	}
	if pmt, ok, err := checkExistingPayment(ctx, s.repo, id, inv.Status); err != nil {
		return nil, err
	} else if ok {
		return pmt, nil
	}
	if req.Amount != inv.TotalAmount {
		return nil, errors.New("nominal pembayaran tidak sesuai dengan total tagihan invoice")
	}
	pmt, err := s.adapter.ProcessPayment(id, req.Amount, req.Method, req.SimulateFailure)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreatePayment(ctx, pmt); err != nil {
		return nil, err
	}
	if err := s.applyPaymentOutcome(ctx, inv, pmt, actor); err != nil {
		return nil, err
	}
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
		// Duplicate callback: idempotent replay of the recorded outcome.
		return pmt, nil
	}
	if req.Amount != inv.TotalAmount {
		return nil, errors.New("nominal pembayaran callback tidak sesuai dengan tagihan invoice")
	}
	if inv.Status == domain.InvoiceCancelled {
		return nil, errors.New("invoice sudah dibatalkan")
	}
	pmt := &domain.Payment{
		PaymentReference: req.PaymentReference, InvoiceID: req.InvoiceID,
		Amount: req.Amount, Method: "sandbox_callback", Status: mapCallbackStatus(req.Status),
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := s.repo.CreatePayment(ctx, pmt); err != nil {
		return nil, err
	}
	if err := s.applyPaymentOutcome(ctx, inv, pmt, "system.payment_gateway"); err != nil {
		return nil, err
	}
	return pmt, nil
}
