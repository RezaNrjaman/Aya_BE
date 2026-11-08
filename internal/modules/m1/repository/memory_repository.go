package repository

import (
	"context"
	"strings"
	"sync"
	"time"

	"araya-business-hub/be/internal/modules/m1/domain"
)

type InMemoryUserRepository struct {
	mu     sync.RWMutex
	users  map[int]domain.User
	emails map[string]int
	nextID int
}

func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{
		users:  make(map[int]domain.User),
		emails: make(map[string]int),
		nextID: 1,
	}
}

func (r *InMemoryUserRepository) Create(ctx context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	normEmail := strings.ToLower(u.Email)
	if _, exists := r.emails[normEmail]; exists {
		return domain.ErrDuplicateEmail
	}

	u.ID = r.nextID
	r.nextID++
	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now

	r.users[u.ID] = *u
	r.emails[normEmail] = u.ID
	return nil
}

func (r *InMemoryUserRepository) GetByID(ctx context.Context, id int) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, exists := r.users[id]
	if !exists {
		return nil, domain.ErrUserNotFound
	}
	return &u, nil
}

func (r *InMemoryUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, exists := r.emails[strings.ToLower(email)]
	if !exists {
		return nil, domain.ErrUserNotFound
	}
	u := r.users[id]
	return &u, nil
}

func (r *InMemoryUserRepository) Update(ctx context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	curr, exists := r.users[u.ID]
	if !exists {
		return domain.ErrUserNotFound
	}

	u.CreatedAt = curr.CreatedAt
	u.UpdatedAt = time.Now().UTC()
	r.users[u.ID] = *u
	return nil
}

func (r *InMemoryUserRepository) UpdatePassword(ctx context.Context, id int, hash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	u, exists := r.users[id]
	if !exists {
		return domain.ErrUserNotFound
	}

	u.PasswordHash = hash
	u.UpdatedAt = time.Now().UTC()
	r.users[id] = u
	return nil
}

func matchesUser(u domain.User, search, div string) bool {
	if div != "" && !strings.EqualFold(u.DivisionName, div) {
		return false
	}
	if search == "" {
		return true
	}
	s := strings.ToLower(search)
	return strings.Contains(strings.ToLower(u.Name), s) || strings.Contains(strings.ToLower(u.Email), s)
}

func (r *InMemoryUserRepository) List(ctx context.Context, search, div string, limit, offset int) ([]domain.User, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var filtered []domain.User
	for _, u := range r.users {
		if matchesUser(u, search, div) {
			filtered = append(filtered, u)
		}
	}

	total := len(filtered)
	if offset >= total {
		return []domain.User{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return filtered[offset:end], total, nil
}

func (r *InMemoryUserRepository) Deactivate(ctx context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	u, exists := r.users[id]
	if !exists {
		return domain.ErrUserNotFound
	}

	u.IsActive = false
	u.UpdatedAt = time.Now().UTC()
	r.users[id] = u
	return nil
}
