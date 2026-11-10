package repository

import (
	"context"
	"strings"
	"sync"
	"time"

	"araya-business-hub/be/internal/modules/m2/domain"
)

type InMemoryUnitRepository struct {
	mu             sync.RWMutex
	units          map[int]domain.BusinessUnit
	divisions      map[int]domain.Division
	nextID         int
	nextDivisionID int
}

func NewInMemoryUnitRepository() *InMemoryUnitRepository {
	return &InMemoryUnitRepository{
		units:     make(map[int]domain.BusinessUnit),
		divisions: make(map[int]domain.Division),
		nextID:    1, nextDivisionID: 1,
	}
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

func matchesUnit(u domain.BusinessUnit, search, cat string, activeOnly bool, partnerID *int) bool {
	if activeOnly && !u.IsActive {
		return false
	}
	if partnerID != nil {
		// ARY-005: partner-scoped listings only see their own units. Platform
		// units (NULL partner) belong to no tenant and are excluded as well.
		if u.PartnerID == nil || *u.PartnerID != *partnerID {
			return false
		}
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

func (r *InMemoryUnitRepository) List(ctx context.Context, search, cat string, activeOnly bool, partnerID *int, limit, offset int) ([]domain.BusinessUnit, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	filtered := make([]domain.BusinessUnit, 0)
	for _, u := range r.units {
		if matchesUnit(u, search, cat, activeOnly, partnerID) {
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

// --- DivisionRepository (ARY-006) ---

func (r *InMemoryUnitRepository) CreateDivision(ctx context.Context, d *domain.Division) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Per-tenant name uniqueness: the same name may exist under different
	// partners, but never twice within one tenant.
	for _, other := range r.divisions {
		sameTenant := (d.PartnerID == nil && other.PartnerID == nil) ||
			(d.PartnerID != nil && other.PartnerID != nil && *d.PartnerID == *other.PartnerID)
		if sameTenant && strings.EqualFold(other.Name, d.Name) {
			return domain.ErrDuplicateDivision
		}
	}
	d.ID = r.nextDivisionID
	r.nextDivisionID++
	now := time.Now().UTC()
	d.CreatedAt = now
	d.UpdatedAt = now
	d.IsActive = true
	r.divisions[d.ID] = *d
	return nil
}

func (r *InMemoryUnitRepository) GetDivisionByID(ctx context.Context, id int) (*domain.Division, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	d, exists := r.divisions[id]
	if !exists {
		return nil, domain.ErrDivisionNotFound
	}
	return &d, nil
}

func (r *InMemoryUnitRepository) UpdateDivision(ctx context.Context, d *domain.Division) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	curr, exists := r.divisions[d.ID]
	if !exists {
		return domain.ErrDivisionNotFound
	}
	d.CreatedAt = curr.CreatedAt
	d.UpdatedAt = time.Now().UTC()
	d.IsActive = curr.IsActive
	r.divisions[d.ID] = *d
	return nil
}

func (r *InMemoryUnitRepository) ListDivisions(ctx context.Context, search string, partnerID *int, limit, offset int) ([]domain.Division, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	filtered := make([]domain.Division, 0)
	for _, d := range r.divisions {
		if partnerID != nil && (d.PartnerID == nil || *d.PartnerID != *partnerID) {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(d.Name), strings.ToLower(search)) &&
			!strings.Contains(strings.ToLower(d.Description), strings.ToLower(search)) {
			continue
		}
		filtered = append(filtered, d)
	}
	total := len(filtered)
	if offset >= total {
		return []domain.Division{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return filtered[offset:end], total, nil
}

func (r *InMemoryUnitRepository) SetDivisionStatus(ctx context.Context, id int, active bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	d, exists := r.divisions[id]
	if !exists {
		return domain.ErrDivisionNotFound
	}
	d.IsActive = active
	d.UpdatedAt = time.Now().UTC()
	r.divisions[id] = d
	return nil
}

func (r *InMemoryUnitRepository) DeleteDivision(ctx context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.divisions[id]; !exists {
		return domain.ErrDivisionNotFound
	}
	delete(r.divisions, id)
	return nil
}

func (r *InMemoryUnitRepository) CountDivisionReferences(ctx context.Context, id int) (int, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	units := 0
	for _, u := range r.units {
		if u.DivisionID != nil && *u.DivisionID == id {
			units++
		}
	}
	return 0, units, nil
}
