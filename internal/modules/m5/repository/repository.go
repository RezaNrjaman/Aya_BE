package repository

import (
	"context"

	"araya-business-hub/be/internal/modules/m5/domain"
)

type Repository interface {
	// Service Requests
	CreateServiceRequest(ctx context.Context, req *domain.ServiceRequest) error
	GetServiceRequestByID(ctx context.Context, id int, partnerID *int, restricted bool) (*domain.ServiceRequest, error)
	UpdateServiceRequest(ctx context.Context, req *domain.ServiceRequest, partnerID *int, restricted bool) error
	ListServiceRequests(ctx context.Context, unitID, partnerID *int, restricted bool, status string, limit, offset int) ([]domain.ServiceRequest, int, error)

	// Shopping Carts
	GetOrCreateCart(ctx context.Context, sessionID string) (*domain.Cart, error)
	GetCartBySession(ctx context.Context, sessionID string) (*domain.Cart, error)
	SaveCart(ctx context.Context, cart *domain.Cart) error
}
