package rbac

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrDuplicatePermission = errors.New("permission key sudah terdaftar")
	ErrPermissionNotFound  = errors.New("permission tidak ditemukan")
)

// Registry manages all registered permissions across modules.
type Registry struct {
	mu          sync.RWMutex
	permissions map[string]Permission
}

// NewRegistry creates a new in-memory permission registry.
func NewRegistry() *Registry {
	return &Registry{
		permissions: make(map[string]Permission),
	}
}

// Register registers a new permission.
func (r *Registry) Register(p Permission) error {
	if err := p.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.permissions[p.Key]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicatePermission, p.Key)
	}

	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	r.permissions[p.Key] = p
	return nil
}

// RegisterBatch registers a slice of permissions.
func (r *Registry) RegisterBatch(perms []Permission) error {
	for _, p := range perms {
		if err := r.Register(p); err != nil {
			return err
		}
	}
	return nil
}

// Get returns a permission by key.
func (r *Registry) Get(key string) (*Permission, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, exists := r.permissions[key]
	if !exists {
		return nil, ErrPermissionNotFound
	}
	return &p, nil
}

// List returns all registered permissions.
func (r *Registry) List() []Permission {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Permission, 0, len(r.permissions))
	for _, p := range r.permissions {
		result = append(result, p)
	}
	return result
}

// Deactivate marks a permission key as inactive.
func (r *Registry) Deactivate(key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, exists := r.permissions[key]
	if !exists {
		return ErrPermissionNotFound
	}

	p.IsActive = false
	r.permissions[key] = p
	return nil
}
