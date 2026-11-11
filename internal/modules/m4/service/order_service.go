package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"araya-business-hub/be/internal/modules/m4/domain"
	"araya-business-hub/be/internal/modules/m4/dto"
	"araya-business-hub/be/internal/modules/m4/repository"
	"araya-business-hub/be/internal/platform/audit"
)

type OrderService struct {
	repo     repository.Repository
	auditSvc *audit.Service
}

func NewOrderService(repo repository.Repository, auditSvc *audit.Service) *OrderService {
	return &OrderService{repo: repo, auditSvc: auditSvc}
}

func formatOrderNo(id int) string {
	// Full-ID numbering avoids collisions (ARY-011).
	return fmt.Sprintf("ORD-%s-%06d", time.Now().Format("20060102"), id)
}

func (s *OrderService) resolveOrderItems(ctx context.Context, inputs []dto.OrderItemInput) ([]domain.OrderItem, int64, *int, error) {
	if len(inputs) == 0 {
		return nil, 0, nil, domain.ErrOrderItemsRequired
	}
	items := make([]domain.OrderItem, 0, len(inputs))
	var total int64
	var partnerID *int
	sellerKey := ""
	for _, it := range inputs {
		if it.Quantity <= 0 {
			return nil, 0, nil, errors.New("kuantitas produk harus lebih dari nol")
		}
		p, err := s.repo.GetProductByID(ctx, it.ProductID)
		if err != nil || !p.IsActive {
			return nil, 0, nil, errors.New("produk tidak ditemukan atau sedang tidak aktif")
		}
		if p.ProductType != "jasa" && it.Quantity > p.Stock {
			return nil, 0, nil, fmt.Errorf("stok produk %s tidak mencukupi", p.Name)
		}
		currentSeller := "platform"
		if p.PartnerID != nil {
			currentSeller = fmt.Sprintf("partner:%d", *p.PartnerID)
		}
		if sellerKey == "" {
			sellerKey = currentSeller
			if p.PartnerID != nil {
				id := *p.PartnerID
				partnerID = &id
			}
		} else if currentSeller != sellerKey {
			return nil, 0, nil, errors.New("keranjang berisi produk dari penjual berbeda; silakan checkout terpisah")
		}
		sub := int64(it.Quantity) * p.Price
		total += sub
		items = append(items, domain.OrderItem{
			ProductID: p.ID, ItemName: p.Name,
			Quantity: it.Quantity, UnitPrice: p.Price, Subtotal: sub,
		})
	}
	return items, total, partnerID, nil
}

func (s *OrderService) resolveCustomer(ctx context.Context, req dto.CreateOrderRequest, partnerID *int) (int, error) {
	if req.CustomerID > 0 {
		customer, err := s.repo.GetCustomerByID(ctx, req.CustomerID)
		if err != nil {
			return 0, err
		}
		if (partnerID == nil && customer.PartnerID != nil) ||
			(partnerID != nil && (customer.PartnerID == nil || *customer.PartnerID != *partnerID)) {
			return 0, domain.ErrCustomerNotFound
		}
		return customer.ID, nil
	}

	customer := &domain.Customer{
		PartnerID: partnerID,
		Name:      strings.TrimSpace(req.CustomerName),
		Email:     strings.ToLower(strings.TrimSpace(req.CustomerEmail)),
		Phone:     strings.TrimSpace(req.CustomerPhone),
		Address:   strings.TrimSpace(req.CustomerAddress),
	}
	if err := customer.Validate(); err != nil {
		return 0, err
	}
	if !strings.Contains(customer.Email, "@") || !strings.Contains(customer.Email, ".") {
		return 0, errors.New("format email pelanggan tidak valid")
	}
	existing, err := s.repo.GetCustomerByEmail(ctx, customer.Email, partnerID)
	if err == nil {
		return existing.ID, nil
	}
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		return 0, err
	}
	if err := s.repo.CreateCustomer(ctx, customer); err != nil {
		return 0, err
	}
	return customer.ID, nil
}

// CreateOrder serves Kontrak K-04: POST /api/m4/v1/orders with idempotency support
func (s *OrderService) CreateOrder(ctx context.Context, req dto.CreateOrderRequest, actor string, scope *int) (*domain.Order, error) {
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		req.IdempotencyKey = fmt.Sprintf("order-%d", time.Now().UnixNano())
	}
	if req.IdempotencyKey != "" {
		existing, err := s.repo.GetOrderByIdempotency(ctx, req.IdempotencyKey)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			if existing.InvoiceID == nil {
				if err := s.ensureOrderInvoice(ctx, existing); err != nil {
					return nil, err
				}
			}
			return existing, nil
		}
	}
	items, total, partnerID, err := s.resolveOrderItems(ctx, req.Items)
	if err != nil {
		return nil, err
	}
	if scope != nil && (partnerID == nil || *partnerID != *scope) {
		return nil, errors.New("order hanya dapat dibuat dari produk milik perusahaan Anda")
	}
	customerID, err := s.resolveCustomer(ctx, req, partnerID)
	if err != nil {
		return nil, err
	}
	o := &domain.Order{
		OrderNo:    fmt.Sprintf("TMP-ORD-%d", time.Now().UnixNano()),
		PartnerID:  partnerID,
		CustomerID: customerID, TotalAmount: total, Status: domain.OrderNew,
		IdempotencyKey: req.IdempotencyKey, Items: items,
	}
	invoiceItems := make([]domain.InvoiceItem, 0, len(items))
	for _, item := range items {
		pid := item.ProductID
		invoiceItems = append(invoiceItems, domain.InvoiceItem{ProductID: &pid, ItemName: item.ItemName,
			Quantity: item.Quantity, UnitPrice: item.UnitPrice, Subtotal: item.Subtotal})
	}
	inv := &domain.Invoice{InvoiceNo: formatInvoiceNo(int(time.Now().UnixNano() % 10000)),
		PartnerID: partnerID, CustomerID: customerID, TotalAmount: total,
		Status: domain.InvoiceUnpaid, Items: invoiceItems}
	if err := s.repo.CreateOrderWithInvoice(ctx, o, inv); err != nil {
		return nil, err
	}
	o.OrderNo = formatOrderNo(o.ID)
	inv.InvoiceNo = formatInvoiceNo(inv.ID)
	if err := s.repo.UpdateInvoice(ctx, inv); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateOrder(ctx, o); err != nil {
		return nil, err
	}
	if err := s.repo.AddOrderHistory(ctx, &domain.OrderStatusHistory{
		OrderID: o.ID, ToStatus: domain.OrderNew, Reason: "Order baru dibuat dari checkout", ChangedBy: actor,
	}); err != nil {
		return nil, err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "order.created", EntityType: "order", EntityID: fmt.Sprintf("%d", o.ID),
	})
	return o, nil
}

func (s *OrderService) ensureOrderInvoice(ctx context.Context, o *domain.Order) error {
	if o.InvoiceID != nil {
		return nil
	}
	items := make([]domain.InvoiceItem, 0, len(o.Items))
	for _, item := range o.Items {
		pid := item.ProductID
		items = append(items, domain.InvoiceItem{ProductID: &pid, ItemName: item.ItemName,
			Quantity: item.Quantity, UnitPrice: item.UnitPrice, Subtotal: item.Subtotal})
	}
	inv := &domain.Invoice{InvoiceNo: formatInvoiceNo(int(time.Now().UnixNano() % 10000)),
		PartnerID: o.PartnerID, CustomerID: o.CustomerID, TotalAmount: o.TotalAmount,
		Status: domain.InvoiceUnpaid, Items: items}
	if err := s.repo.CreateInvoice(ctx, inv); err != nil {
		return err
	}
	inv.InvoiceNo = formatInvoiceNo(inv.ID)
	if err := s.repo.UpdateInvoice(ctx, inv); err != nil {
		return err
	}
	o.InvoiceID = &inv.ID
	return s.repo.UpdateOrder(ctx, o)
}

func (s *OrderService) GetPublicOrder(ctx context.Context, orderNo, email string) (*domain.Order, error) {
	if strings.TrimSpace(orderNo) == "" || strings.TrimSpace(email) == "" {
		return nil, domain.ErrOrderNotFound
	}
	o, err := s.repo.GetOrderByOrderNoAndEmail(ctx, strings.TrimSpace(orderNo), strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return nil, err
	}
	histories, err := s.repo.GetOrderHistories(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Histories = histories
	return o, nil
}

func (s *OrderService) GetOrder(ctx context.Context, id int, scope *int) (*domain.Order, error) {
	o, err := s.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !canAccessTenant(o.PartnerID, scope) {
		return nil, domain.ErrOrderNotFound
	}
	histories, _ := s.repo.GetOrderHistories(ctx, id)
	o.Histories = histories
	return o, nil
}

func (s *OrderService) ListOrders(ctx context.Context, status string, scope *int, limit, offset int) ([]domain.Order, int, error) {
	return s.repo.ListOrders(ctx, status, scope, limit, offset)
}

func (s *OrderService) ProcessOrder(ctx context.Context, id int, actor string, scope *int) (*domain.Order, error) {
	o, err := s.GetOrder(ctx, id, scope)
	if err != nil {
		return nil, err
	}
	if o.InvoiceID != nil {
		inv, err := s.repo.GetInvoiceByID(ctx, *o.InvoiceID)
		if err != nil {
			return nil, err
		}
		if inv.Status != domain.InvoicePaid {
			return nil, errors.New("pesanan hanya dapat diproses setelah pembayaran berhasil")
		}
	}
	from := o.Status
	if err := o.Process(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateOrder(ctx, o); err != nil {
		return nil, err
	}
	if err := s.repo.AddOrderHistory(ctx, &domain.OrderStatusHistory{
		OrderID: id, FromStatus: from, ToStatus: o.Status, Reason: "Order diproses oleh operasional/partner", ChangedBy: actor,
	}); err != nil {
		return nil, err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "order.processed", EntityType: "order", EntityID: fmt.Sprintf("%d", id),
	})
	return o, nil
}

func (s *OrderService) CompleteOrder(ctx context.Context, id int, actor string, scope *int) (*domain.Order, error) {
	o, err := s.GetOrder(ctx, id, scope)
	if err != nil {
		return nil, err
	}
	from := o.Status
	if err := o.Complete(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateOrder(ctx, o); err != nil {
		return nil, err
	}
	if err := s.repo.AddOrderHistory(ctx, &domain.OrderStatusHistory{
		OrderID: id, FromStatus: from, ToStatus: o.Status, Reason: "Order selesai", ChangedBy: actor,
	}); err != nil {
		return nil, err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "order.completed", EntityType: "order", EntityID: fmt.Sprintf("%d", id),
	})
	return o, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, id int, reason string, actor string, scope *int) (*domain.Order, error) {
	o, err := s.GetOrder(ctx, id, scope)
	if err != nil {
		return nil, err
	}
	from := o.Status
	if err := o.Cancel(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateOrder(ctx, o); err != nil {
		return nil, err
	}
	if err := s.repo.AddOrderHistory(ctx, &domain.OrderStatusHistory{
		OrderID: id, FromStatus: from, ToStatus: o.Status, Reason: reason, ChangedBy: actor,
	}); err != nil {
		return nil, err
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "order.cancelled", EntityType: "order", EntityID: fmt.Sprintf("%d", id),
	})
	return o, nil
}
