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
	categories     map[string]int
	categoryRows   map[int]domain.BusinessCategory
	nextCategoryID int
	nextID         int
	nextDivisionID int
}

func NewInMemoryUnitRepository() *InMemoryUnitRepository {
	return &InMemoryUnitRepository{
		units:          make(map[int]domain.BusinessUnit),
		divisions:      make(map[int]domain.Division),
		categories:     make(map[string]int),
		categoryRows:   make(map[int]domain.BusinessCategory),
		nextCategoryID: 1,
		nextID:         1,
		nextDivisionID: 1,
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

func (r *InMemoryUnitRepository) GetOrCreateCategoryID(ctx context.Context, name string) (*int, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.categories[strings.ToLower(name)]
	if !ok {
		id = r.nextCategoryID
		r.nextCategoryID++
		r.categories[strings.ToLower(name)] = id
	}
	return &id, nil
}

func (r *InMemoryUnitRepository) ValidatePartnerDivision(ctx context.Context, partnerID int, divisionID *int) error {
	return nil
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
		if sameTenant && other.CategoryID == d.CategoryID && strings.EqualFold(other.Name, d.Name) {
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

func (r *InMemoryUnitRepository) CreateCategory(ctx context.Context, c *domain.BusinessCategory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	unit, ok := r.units[c.UnitUsahaID]
	if !ok || unit.PartnerID == nil || c.PartnerID == nil || *unit.PartnerID != *c.PartnerID {
		return domain.ErrUnitNotFound
	}
	for _, existing := range r.categoryRows {
		if existing.UnitUsahaID == c.UnitUsahaID && strings.EqualFold(existing.Name, c.Name) {
			return domain.ErrDuplicateDivision
		}
	}
	c.ID = r.nextCategoryID
	r.nextCategoryID++
	c.IsActive = true
	c.CreatedAt = time.Now().UTC()
	c.UpdatedAt = c.CreatedAt
	r.categoryRows[c.ID] = *c
	return nil
}

func (r *InMemoryUnitRepository) ListCategories(ctx context.Context, unitID *int, partnerID *int, search string, limit, offset int) ([]domain.BusinessCategory, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := []domain.BusinessCategory{}
	for _, c := range r.categoryRows {
		if unitID != nil && c.UnitUsahaID != *unitID {
			continue
		}
		if partnerID != nil && (c.PartnerID == nil || *c.PartnerID != *partnerID) {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(c.Name), strings.ToLower(search)) {
			continue
		}
		items = append(items, c)
	}
	total := len(items)
	if offset >= total {
		return []domain.BusinessCategory{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return items[offset:end], total, nil
}

func (r *InMemoryUnitRepository) GetCategoryByID(ctx context.Context, id int) (*domain.BusinessCategory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.categoryRows[id]
	if !ok {
		return nil, domain.ErrUnitNotFound
	}
	return &c, nil
}

func (r *InMemoryUnitRepository) UpdateCategory(ctx context.Context, c *domain.BusinessCategory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	old, ok := r.categoryRows[c.ID]
	if !ok {
		return domain.ErrUnitNotFound
	}
	c.CreatedAt = old.CreatedAt
	c.UpdatedAt = time.Now().UTC()
	c.UnitUsahaID = old.UnitUsahaID
	c.PartnerID = old.PartnerID
	r.categoryRows[c.ID] = *c
	return nil
}

func (r *InMemoryUnitRepository) SetCategoryStatus(ctx context.Context, id int, active bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.categoryRows[id]
	if !ok {
		return domain.ErrUnitNotFound
	}
	c.IsActive = active
	c.UpdatedAt = time.Now().UTC()
	r.categoryRows[id] = c
	return nil
}

func (r *InMemoryUnitRepository) ValidateCategoryPlacement(ctx context.Context, partnerID int, unitID, categoryID int) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, uok := r.units[unitID]
	c, cok := r.categoryRows[categoryID]
	if !uok || !cok || u.PartnerID == nil || *u.PartnerID != partnerID ||
		c.PartnerID == nil || *c.PartnerID != partnerID || c.UnitUsahaID != unitID || !u.IsActive || !c.IsActive {
		return domain.ErrUnitNotFound
	}
	return nil
}

func (r *InMemoryUnitRepository) ValidateDivisionPlacement(ctx context.Context, partnerID, unitID, categoryID int) error {
	return r.ValidateCategoryPlacement(ctx, partnerID, unitID, categoryID)
}
