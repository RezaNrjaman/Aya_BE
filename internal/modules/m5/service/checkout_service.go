package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"araya-business-hub/be/internal/modules/m5/client"
	"araya-business-hub/be/internal/modules/m5/domain"
	"araya-business-hub/be/internal/modules/m5/dto"
	"araya-business-hub/be/internal/modules/m5/repository"
	"araya-business-hub/be/internal/platform/audit"
)

type CheckoutService struct {
	repo     repository.Repository
	m4Client client.M4Client
	auditSvc *audit.Service
}

func NewCheckoutService(repo repository.Repository, m4 client.M4Client, auditSvc *audit.Service) *CheckoutService {
	return &CheckoutService{repo: repo, m4Client: m4, auditSvc: auditSvc}
}

func validateCartProducts(ctx context.Context, m4 client.M4Client, items []domain.CartItem) error {
	for _, it := range items {
		prod, err := m4.GetProductByID(ctx, it.ProductID)
		if err != nil || !prod.IsActive {
			return fmt.Errorf("produk ID %d tidak tersedia atau sudah tidak aktif", it.ProductID)
		}
	}
	return nil
}

func (s *CheckoutService) Checkout(ctx context.Context, sessionID string, req dto.CheckoutRequest) (*dto.CheckoutResponse, error) {
	c, err := s.repo.GetCartBySession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrCartEmpty) {
			return nil, domain.ErrCartEmpty
		}
		return nil, err
	}
	if len(c.Items) == 0 {
		return nil, domain.ErrCartEmpty
	}
	if strings.TrimSpace(req.CustomerName) == "" || strings.TrimSpace(req.CustomerEmail) == "" {
		return nil, errors.New("nama dan email customer wajib diisi untuk checkout")
	}
	if !strings.Contains(req.CustomerEmail, "@") || !strings.Contains(req.CustomerEmail, ".") {
		return nil, errors.New("format email customer tidak valid")
	}
	if err := validateCartProducts(ctx, s.m4Client, c.Items); err != nil {
		return nil, err
	}
	resp, err := s.m4Client.CreateOrderFromCart(ctx, req, c.Items)
	if err != nil {
		return nil, err
	}
	c.Clear()
	if err := s.repo.SaveCart(ctx, c); err != nil {
		// Keep the idempotency key on the client; retrying returns the already
		// created order and attempts to clear the cart again without duplicating it.
		return nil, fmt.Errorf("pesanan %s sudah dibuat, tetapi keranjang belum berhasil dikosongkan; silakan ulangi proses checkout", resp.OrderNo)
	}
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: "customer", Action: "cart.checkout", EntityType: "order",
		EntityID: fmt.Sprintf("%d", resp.OrderID), Details: map[string]any{"session_id": sessionID},
	})
	return resp, nil
}
