package rbac

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Permission represents a system permission registered by a module.
type Permission struct {
	ID          int       `json:"id"`
	Key         string    `json:"key"`         // e.g. "m1.users.read"
	Module      string    `json:"module"`      // "m1", "m2", "m3", "m4", "m5"
	Name        string    `json:"name"`        // e.g. "Lihat Daftar Pengguna"
	Description string    `json:"description"` // e.g. "Dapat membaca data seluruh pengguna"
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// Validate ensures key follows the lowercase namespaced format: "<module>.<resource>.<action>"
func (p *Permission) Validate() error {
	p.Key = strings.TrimSpace(strings.ToLower(p.Key))
	p.Module = strings.TrimSpace(strings.ToLower(p.Module))

	if p.Key == "" {
		return errors.New("permission key cannot be empty")
	}
	parts := strings.Split(p.Key, ".")
	if len(parts) < 3 {
		return fmt.Errorf("permission key '%s' must follow format '<module>.<resource>.<action>' (e.g. m1.users.read)", p.Key)
	}
	if parts[0] != p.Module {
		return fmt.Errorf("permission key '%s' module prefix must match module '%s'", p.Key, p.Module)
	}
	if p.Name == "" {
		return errors.New("permission name is required")
	}
	return nil
}

// Role represents a configurable dynamic role in the system.
type Role struct {
	ID          int          `json:"id"`
	Name        string       `json:"name"`         // Unique slug, e.g. "admin", "manajer", "partner", "kasir"
	DisplayName string       `json:"display_name"` // e.g. "Administrator Sistem"
	Description string       `json:"description"`
	IsActive    bool         `json:"is_active"`
	Permissions []Permission `json:"permissions,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// UserRoleAssignment maps a user to a role with optional business unit scope.
type UserRoleAssignment struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	RoleID      int       `json:"role_id"`
	RoleName    string    `json:"role_name"`
	UnitUsahaID *int      `json:"unit_usaha_id,omitempty"` // Nullable for global scope
	CreatedAt   time.Time `json:"created_at"`
}
