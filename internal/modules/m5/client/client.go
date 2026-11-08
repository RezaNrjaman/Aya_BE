package client

import (
	"context"

	"araya-business-hub/be/internal/modules/m5/domain"
	"araya-business-hub/be/internal/modules/m5/dto"
)

type M2Client interface {
	GetActiveBusinessUnits(ctx context.Context) ([]dto.BusinessUnitSummary, error)
}

type M4Client interface {
	GetActiveProducts(ctx context.Context, unitID *int, category, search string) ([]dto.ProductSummary, error)
	GetProductByID(ctx context.Context, id int) (*dto.ProductSummary, error)
	CreateOrderFromCart(ctx context.Context, req dto.CheckoutRequest, items []domain.CartItem) (*dto.CheckoutResponse, error)
}
