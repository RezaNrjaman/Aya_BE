package service

import (
	"context"
	"errors"

	"araya-business-hub/be/internal/modules/m5/client"
	"araya-business-hub/be/internal/modules/m5/domain"
	"araya-business-hub/be/internal/modules/m5/dto"
	"araya-business-hub/be/internal/modules/m5/repository"
)

type CartService struct {
	repo     repository.Repository
	m4Client client.M4Client
}

func NewCartService(repo repository.Repository, m4 client.M4Client) *CartService {
	return &CartService{repo: repo, m4Client: m4}
}

func toCartResponse(c *domain.Cart) *dto.CartResponse {
	return &dto.CartResponse{
		ID:          c.ID,
		SessionID:   c.SessionID,
		TotalItems:  c.TotalQuantity(),
		TotalAmount: c.TotalAmount(),
		Items:       c.Items,
	}
}

func (s *CartService) GetCart(ctx context.Context, sessionID string) (*dto.CartResponse, error) {
	c, err := s.repo.GetOrCreateCart(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return toCartResponse(c), nil
}

func (s *CartService) AddItem(ctx context.Context, sessionID string, req dto.AddToCartRequest) (*dto.CartResponse, error) {
	if req.Quantity <= 0 {
		return nil, domain.ErrInvalidQuantity
	}
	prod, err := s.m4Client.GetProductByID(ctx, req.ProductID)
	if err != nil || !prod.IsActive {
		return nil, errors.New("produk tidak ditemukan atau tidak aktif")
	}
	c, err := s.repo.GetOrCreateCart(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	c.AddItem(prod.ID, prod.Name, prod.Price, req.Quantity)
	_ = s.repo.SaveCart(ctx, c)
	return toCartResponse(c), nil
}

func (s *CartService) UpdateItem(ctx context.Context, sessionID string, productID, qty int) (*dto.CartResponse, error) {
	c, err := s.repo.GetOrCreateCart(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if err := c.UpdateQuantity(productID, qty); err != nil {
		return nil, err
	}
	_ = s.repo.SaveCart(ctx, c)
	return toCartResponse(c), nil
}

func (s *CartService) RemoveItem(ctx context.Context, sessionID string, productID int) (*dto.CartResponse, error) {
	c, err := s.repo.GetOrCreateCart(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !c.RemoveItem(productID) {
		return nil, domain.ErrCartItemNotFound
	}
	_ = s.repo.SaveCart(ctx, c)
	return toCartResponse(c), nil
}

func (s *CartService) ClearCart(ctx context.Context, sessionID string) error {
	c, err := s.repo.GetOrCreateCart(ctx, sessionID)
	if err != nil {
		return err
	}
	c.Clear()
	return s.repo.SaveCart(ctx, c)
}

