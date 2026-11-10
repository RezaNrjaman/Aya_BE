package repository

import (
	"context"
	"strings"
	"sync"
	"time"

	"araya-business-hub/be/internal/modules/m3/domain"
)

type InMemoryPartnerRepository struct {
	mu        sync.RWMutex
	partners  map[int]domain.Partner
	histories []domain.StatusHistory
	nextID    int
	nextHist  int
}

func NewInMemoryPartnerRepository() *InMemoryPartnerRepository {
	return &InMemoryPartnerRepository{
		partners: make(map[int]domain.Partner),
		nextID:   1,
		nextHist: 1,
	}
}

func (r *InMemoryPartnerRepository) Create(ctx context.Context, p *domain.Partner) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p.ID = r.nextID
	r.nextID++
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	r.partners[p.ID] = *p
	return nil
}

func (r *InMemoryPartnerRepository) GetByID(ctx context.Context, id int) (*domain.Partner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, exists := r.partners[id]
	if !exists {
		return nil, domain.ErrPartnerNotFound
	}
	return &p, nil
}

func (r *InMemoryPartnerRepository) GetByEmail(ctx context.Context, email string) (*domain.Partner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	low := strings.ToLower(strings.TrimSpace(email))
	for _, p := range r.partners {
		if strings.ToLower(p.Email) == low {
			return &p, nil
		}
	}
	return nil, domain.ErrPartnerNotFound
}

func (r *InMemoryPartnerRepository) GetByUserID(ctx context.Context, userID int) (*domain.Partner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.partners {
		if p.UserID != nil && *p.UserID == userID {
			return &p, nil
		}
	}
	return nil, domain.ErrPartnerNotFound
}

func (r *InMemoryPartnerRepository) GetByRegistrationCode(ctx context.Context, code string) (*domain.Partner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	code = strings.ToLower(strings.TrimSpace(code))
	if code == "" {
		return nil, domain.ErrPartnerNotFound
	}
	for _, p := range r.partners {
		if strings.EqualFold(p.RegistrationCode, code) {
			return &p, nil
		}
	}
	return nil, domain.ErrPartnerNotFound
}

func (r *InMemoryPartnerRepository) Update(ctx context.Context, p *domain.Partner) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	curr, exists := r.partners[p.ID]
	if !exists {
		return domain.ErrPartnerNotFound
	}
	p.CreatedAt = curr.CreatedAt
	p.UpdatedAt = time.Now().UTC()
	r.partners[p.ID] = *p
	return nil
}

func matchesPartner(p domain.Partner, status, search string) bool {
	if status != "" && !strings.EqualFold(p.Status, status) {
		return false
	}
	if search == "" {
		return true
	}
	s := strings.ToLower(search)
	return strings.Contains(strings.ToLower(p.CompanyName), s) ||
		strings.Contains(strings.ToLower(p.ContactPerson), s) ||
		strings.Contains(strings.ToLower(p.Email), s)
}

func (r *InMemoryPartnerRepository) List(ctx context.Context, status, search string, limit, offset int) ([]domain.Partner, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	filtered := make([]domain.Partner, 0)
	for _, p := range r.partners {
		if matchesPartner(p, status, search) {
			filtered = append(filtered, p)
		}
	}
	total := len(filtered)
	if offset >= total {
		return []domain.Partner{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return filtered[offset:end], total, nil
}

func (r *InMemoryPartnerRepository) ListVerified(ctx context.Context) ([]domain.Partner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	verified := make([]domain.Partner, 0)
	for _, p := range r.partners {
		if p.Status == domain.StatusVerified {
			verified = append(verified, p)
		}
	}
	return verified, nil
}

func (r *InMemoryPartnerRepository) AddHistory(ctx context.Context, h *domain.StatusHistory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	h.ID = r.nextHist
	r.nextHist++
	h.CreatedAt = time.Now().UTC()
	r.histories = append(r.histories, *h)
	return nil
}

func (r *InMemoryPartnerRepository) GetHistories(ctx context.Context, partnerID int) ([]domain.StatusHistory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]domain.StatusHistory, 0)
	for _, h := range r.histories {
		if h.PartnerID == partnerID {
			res = append(res, h)
		}
	}
	return res, nil
}
