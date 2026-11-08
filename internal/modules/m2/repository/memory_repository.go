package repository

import (
	"context"
	"strings"
	"sync"
	"time"

	"araya-business-hub/be/internal/modules/m2/domain"
)

type InMemoryUnitRepository struct {
	mu     sync.RWMutex
	units  map[int]domain.BusinessUnit
	nextID int
}

func NewInMemoryUnitRepository() *InMemoryUnitRepository {
	return &InMemoryUnitRepository{units: make(map[int]domain.BusinessUnit), nextID: 1}
}

func (r *InMemoryUnitRepository) Create(ctx context.Context, u *domain.BusinessUnit) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	u.ID = r.nextID
	r.nextID++
	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now
	r.units[u.ID] = *u
	return nil
}

func (r *InMemoryUnitRepository) GetByID(ctx context.Context, id int) (*domain.BusinessUnit, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, exists := r.units[id]
	if !exists {
		return nil, domain.ErrUnitNotFound
	}
	return &u, nil
}

func (r *InMemoryUnitRepository) Update(ctx context.Context, u *domain.BusinessUnit) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	curr, exists := r.units[u.ID]
	if !exists {
		return domain.ErrUnitNotFound
	}
	u.CreatedAt = curr.CreatedAt
	u.UpdatedAt = time.Now().UTC()
	r.units[u.ID] = *u
	return nil
}

func matchesUnit(u domain.BusinessUnit, search, cat string, activeOnly bool) bool {
	if activeOnly && !u.IsActive {
		return false
	}
	if cat != "" && !strings.EqualFold(u.CategoryName, cat) {
		return false
	}
	if search == "" {
		return true
	}
	s := strings.ToLower(search)
	return strings.Contains(strings.ToLower(u.Name), s) || strings.Contains(strings.ToLower(u.Description), s)
}

func (r *InMemoryUnitRepository) List(ctx context.Context, search, cat string, activeOnly bool, limit, offset int) ([]domain.BusinessUnit, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	filtered := make([]domain.BusinessUnit, 0)
	for _, u := range r.units {
		if matchesUnit(u, search, cat, activeOnly) {
			filtered = append(filtered, u)
		}
	}
	total := len(filtered)
	if offset >= total {
		return []domain.BusinessUnit{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return filtered[offset:end], total, nil
}

func (r *InMemoryUnitRepository) ListActive(ctx context.Context) ([]domain.BusinessUnit, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	active := make([]domain.BusinessUnit, 0)
	for _, u := range r.units {
		if u.IsActive {
			active = append(active, u)
		}
	}
	return active, nil
}

func (r *InMemoryUnitRepository) SetStatus(ctx context.Context, id int, active bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	u, exists := r.units[id]
	if !exists {
		return domain.ErrUnitNotFound
	}
	u.IsActive = active
	u.UpdatedAt = time.Now().UTC()
	r.units[id] = u
	return nil
}
