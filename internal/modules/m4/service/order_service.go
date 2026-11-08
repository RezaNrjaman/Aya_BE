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
	return fmt.Sprintf("ORD-%s-%04d", time.Now().Format("20060102"), id%10000)
}

func (s *OrderService) resolveOrderItems(ctx context.Context, inputs []dto.OrderItemInput) ([]domain.OrderItem, int64, error) {
	if len(inputs) == 0 {
		return nil, 0, domain.ErrOrderItemsRequired
	}
	items := make([]domain.OrderItem, 0, len(inputs))
	var total int64
	for _, it := range inputs {
		p, err := s.repo.GetProductByID(ctx, it.ProductID)
		if err != nil || !p.IsActive {
			return nil, 0, errors.New("produk tidak ditemukan atau sedang tidak aktif")
		}
		sub := int64(it.Quantity) * p.Price
		total += sub
		items = append(items, domain.OrderItem{
			ProductID: it.ProductID, ItemName: p.Name,
			Quantity: it.Quantity, UnitPrice: p.Price, Subtotal: sub,
		})
	}
	return items, total, nil
}

// CreateOrder serves Kontrak K-04: POST /api/m4/v1/orders with idempotency support
func (s *OrderService) CreateOrder(ctx context.Context, req dto.CreateOrderRequest, actor string) (*domain.Order, error) {
	if req.IdempotencyKey != "" {
		if existing, err := s.repo.GetOrderByIdempotency(ctx, req.IdempotencyKey); err == nil {
			return existing, nil
		}
	}
	items, total, err := s.resolveOrderItems(ctx, req.Items)
	if err != nil {
		return nil, err
	}
	o := &domain.Order{
		CustomerID: req.CustomerID, TotalAmount: total, Status: domain.OrderNew,
		IdempotencyKey: req.IdempotencyKey, Items: items,
	}
	_ = s.repo.CreateOrder(ctx, o)
	o.OrderNo = formatOrderNo(o.ID)
	_ = s.repo.UpdateOrder(ctx, o)
	_ = s.repo.AddOrderHistory(ctx, &domain.OrderStatusHistory{
		OrderID: o.ID, ToStatus: domain.OrderNew, Reason: "Order baru dibuat dari checkout", ChangedBy: actor,
	})
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "order.created", EntityType: "order", EntityID: fmt.Sprintf("%d", o.ID),
	})
	return o, nil
}

func (s *OrderService) GetOrder(ctx context.Context, id int) (*domain.Order, error) {
	o, err := s.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	histories, _ := s.repo.GetOrderHistories(ctx, id)
	o.Histories = histories
	return o, nil
}

func (s *OrderService) ListOrders(ctx context.Context, status string, limit, offset int) ([]domain.Order, int, error) {
	return s.repo.ListOrders(ctx, status, limit, offset)
}

func (s *OrderService) ProcessOrder(ctx context.Context, id int, actor string) (*domain.Order, error) {
	o, err := s.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	from := o.Status
	if err := o.Process(); err != nil {
		return nil, err
	}
	_ = s.repo.UpdateOrder(ctx, o)
	_ = s.repo.AddOrderHistory(ctx, &domain.OrderStatusHistory{
		OrderID: id, FromStatus: from, ToStatus: o.Status, Reason: "Order diproses oleh operasional/partner", ChangedBy: actor,
	})
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "order.processed", EntityType: "order", EntityID: fmt.Sprintf("%d", id),
	})
	return o, nil
}

func (s *OrderService) CompleteOrder(ctx context.Context, id int, actor string) (*domain.Order, error) {
	o, err := s.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	from := o.Status
	if err := o.Complete(); err != nil {
		return nil, err
	}
	_ = s.repo.UpdateOrder(ctx, o)
	_ = s.repo.AddOrderHistory(ctx, &domain.OrderStatusHistory{
		OrderID: id, FromStatus: from, ToStatus: o.Status, Reason: "Order selesai", ChangedBy: actor,
	})
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "order.completed", EntityType: "order", EntityID: fmt.Sprintf("%d", id),
	})
	return o, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, id int, reason string, actor string) (*domain.Order, error) {
	o, err := s.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	from := o.Status
	if err := o.Cancel(); err != nil {
		return nil, err
	}
	_ = s.repo.UpdateOrder(ctx, o)
	_ = s.repo.AddOrderHistory(ctx, &domain.OrderStatusHistory{
		OrderID: id, FromStatus: from, ToStatus: o.Status, Reason: reason, ChangedBy: actor,
	})
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: actor, Action: "order.cancelled", EntityType: "order", EntityID: fmt.Sprintf("%d", id),
	})
	return o, nil
}
