package audit

import (
	"context"
	"sync"
	"time"

	"araya-business-hub/be/internal/platform/logging"
)

// Record represents a single audit trail entry.
type Record struct {
	ID         int64          `json:"id"`
	UserID     *int           `json:"user_id,omitempty"`
	UserName   string         `json:"user_name,omitempty"`
	Action     string         `json:"action"`      // e.g. "user.created", "permission.revoked"
	EntityType string         `json:"entity_type"` // e.g. "user", "role", "invoice"
	EntityID   string         `json:"entity_id,omitempty"`
	Details    map[string]any `json:"details,omitempty"`
	IPAddress  string         `json:"ip_address,omitempty"`
	UserAgent  string         `json:"user_agent,omitempty"`
	RequestID  string         `json:"request_id,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// Repository defines audit log storage operations.
type Repository interface {
	Save(ctx context.Context, record *Record) error
	List(ctx context.Context, limit, offset int) ([]Record, int, error)
}

// InMemoryRepository stores audit records in memory (useful for testing and fallback).
type InMemoryRepository struct {
	mu      sync.RWMutex
	records []Record
	nextID  int64
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		records: make([]Record, 0),
		nextID:  1,
	}
}

func (m *InMemoryRepository) Save(ctx context.Context, r *Record) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	r.ID = m.nextID
	m.nextID++
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}

	m.records = append(m.records, *r)
	return nil
}

func (m *InMemoryRepository) List(ctx context.Context, limit, offset int) ([]Record, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := len(m.records)
	if offset >= total {
		return []Record{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	// Return copy in reverse chronological order
	res := make([]Record, 0, end-offset)
	for i := end - 1; i >= offset; i-- {
		res = append(res, m.records[i])
	}

	return res, total, nil
}

// Service coordinates audit logging with sensitive data masking.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Log records an audit event after sanitizing sensitive details.
func (s *Service) Log(ctx context.Context, r Record) error {
	if r.Details != nil {
		r.Details = logging.RedactMap(r.Details)
	}
	return s.repo.Save(ctx, &r)
}

// List returns audit trail history.
func (s *Service) List(ctx context.Context, limit, offset int) ([]Record, int, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.List(ctx, limit, offset)
}
