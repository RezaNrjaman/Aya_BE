package service

import (
	"context"
	"errors"
	"fmt"

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
	if err != nil || len(c.Items) == 0 {
		return nil, domain.ErrCartEmpty
	}
	if req.CustomerID <= 0 {
		return nil, errors.New("customer ID wajib diisi untuk checkout")
	}
	if err := validateCartProducts(ctx, s.m4Client, c.Items); err != nil {
		return nil, err
	}
	resp, err := s.m4Client.CreateOrderFromCart(ctx, req, c.Items)
	if err != nil {
		return nil, err
	}
	c.Clear()
	_ = s.repo.SaveCart(ctx, c)
	_ = s.auditSvc.Log(ctx, audit.Record{
		UserName: "customer", Action: "cart.checkout", EntityType: "order",
		EntityID: fmt.Sprintf("%d", resp.OrderID), Details: map[string]any{"session_id": sessionID},
	})
	return resp, nil
}
