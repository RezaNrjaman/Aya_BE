package repository

import (
	"context"
	"strings"
	"sync"
	"time"

	"araya-business-hub/be/internal/modules/m5/domain"
)

type InMemoryRepository struct {
	mu         sync.RWMutex
	requests   map[int]domain.ServiceRequest
	carts      map[string]domain.Cart
	nextReqID  int
	nextCartID int
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		requests:   make(map[int]domain.ServiceRequest),
		carts:      make(map[string]domain.Cart),
		nextReqID:  1,
		nextCartID: 1,
	}
}

func (r *InMemoryRepository) CreateServiceRequest(ctx context.Context, req *domain.ServiceRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	req.ID = r.nextReqID
	r.nextReqID++
	now := time.Now().UTC()
	req.CreatedAt, req.UpdatedAt = now, now
	r.requests[req.ID] = *req
	return nil
}

func (r *InMemoryRepository) GetServiceRequestByID(ctx context.Context, id int, partnerID *int, restricted bool) (*domain.ServiceRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	req, ok := r.requests[id]
	if !ok || (restricted && (partnerID == nil || req.PartnerID == nil || *req.PartnerID != *partnerID)) {
		return nil, domain.ErrRequestNotFound
	}
	return &req, nil
}

func (r *InMemoryRepository) UpdateServiceRequest(ctx context.Context, req *domain.ServiceRequest, partnerID *int, restricted bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.requests[req.ID]
	if !ok || (restricted && (partnerID == nil || current.PartnerID == nil || *current.PartnerID != *partnerID)) {
		return domain.ErrRequestNotFound
	}
	req.UpdatedAt = time.Now().UTC()
	r.requests[req.ID] = *req
	return nil
}

func filterRequests(all map[int]domain.ServiceRequest, unitID, partnerID *int, restricted bool, status string) []domain.ServiceRequest {
	res := make([]domain.ServiceRequest, 0)
	for _, req := range all {
		if restricted && (partnerID == nil || req.PartnerID == nil || *req.PartnerID != *partnerID) {
			continue
		}
		if unitID != nil && req.UnitUsahaID != *unitID {
			continue
		}
		if status != "" && !strings.EqualFold(req.Status, status) {
			continue
		}
		res = append(res, req)
	}
	return res
}

func paginateRequests(list []domain.ServiceRequest, limit, offset int) []domain.ServiceRequest {
	total := len(list)
	if offset >= total {
		return []domain.ServiceRequest{}
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return list[offset:end]
}

func (r *InMemoryRepository) ListServiceRequests(ctx context.Context, unitID, partnerID *int, restricted bool, status string, limit, offset int) ([]domain.ServiceRequest, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	filtered := filterRequests(r.requests, unitID, partnerID, restricted, status)
	paged := paginateRequests(filtered, limit, offset)
	return paged, len(filtered), nil
}

func (r *InMemoryRepository) GetOrCreateCart(ctx context.Context, sessionID string) (*domain.Cart, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.carts[sessionID]; ok {
		return &c, nil
	}
	now := time.Now().UTC()
	c := domain.Cart{
		ID: r.nextCartID, SessionID: sessionID,
		Items: []domain.CartItem{}, CreatedAt: now, UpdatedAt: now,
	}
	r.nextCartID++
	r.carts[sessionID] = c
	return &c, nil
}

func (r *InMemoryRepository) GetCartBySession(ctx context.Context, sessionID string) (*domain.Cart, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.carts[sessionID]
	if !ok {
		return nil, domain.ErrCartEmpty
	}
	return &c, nil
}

func (r *InMemoryRepository) SaveCart(ctx context.Context, cart *domain.Cart) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cart.UpdatedAt = time.Now().UTC()
	r.carts[cart.SessionID] = *cart
	return nil
}
