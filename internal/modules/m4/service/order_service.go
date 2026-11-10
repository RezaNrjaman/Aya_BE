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
	for _, it := range inputs {
		p, err := s.repo.GetProductByID(ctx, it.ProductID)
		if err != nil || !p.IsActive {
			return nil, 0, nil, errors.New("produk tidak ditemukan atau sedang tidak aktif")
		}
		// ARY-013/ARY-001: revenue must never be mixed across tenants — all
		// items of one order must belong to the same partner.
		if p.PartnerID != nil {
			if partnerID == nil {
				partnerID = p.PartnerID
			} else if *partnerID != *p.PartnerID {
				return nil, 0, nil, errors.New("keranjang berisi produk dari penjual berbeda; silakan checkout terpisah")
			}
		}
		sub := int64(it.Quantity) * p.Price
		total += sub
		items = append(items, domain.OrderItem{
			ProductID: it.ProductID, ItemName: p.Name,
			Quantity: it.Quantity, UnitPrice: p.Price, Subtotal: sub,
		})
	}
	return items, total, partnerID, nil
}

// CreateOrder serves Kontrak K-04: POST /api/m4/v1/orders with idempotency support
func (s *OrderService) CreateOrder(ctx context.Context, req dto.CreateOrderRequest, actor string) (*domain.Order, error) {
	if req.IdempotencyKey != "" {
		if existing, err := s.repo.GetOrderByIdempotency(ctx, req.IdempotencyKey); err == nil && existing != nil {
			return existing, nil
		}
	}
	items, total, partnerID, err := s.resolveOrderItems(ctx, req.Items)
	if err != nil {
		return nil, err
	}
	o := &domain.Order{
		OrderNo:    formatOrderNo(int(time.Now().UnixNano() % 10000)),
		PartnerID:  partnerID, // ARY-001: derived from the products, not the request.
		CustomerID: req.CustomerID, TotalAmount: total, Status: domain.OrderNew,
		IdempotencyKey: req.IdempotencyKey, Items: items,
	}
	// ARY-011: never report success when persistence failed.
	if err := s.repo.CreateOrder(ctx, o); err != nil {
		return nil, err
	}
	o.OrderNo = formatOrderNo(o.ID)
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
