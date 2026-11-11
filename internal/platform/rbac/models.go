package rbac

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ErrInvalidRoleName is returned when a role slug does not follow the naming rule.
var ErrInvalidRoleName = errors.New("kode role hanya boleh huruf kecil, angka, dan garis bawah (3-50 karakter, diawali huruf)")

var roleNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{2,49}$`)

// NormalizeRoleName trims and lowercases a role slug, then validates it.
func NormalizeRoleName(name string) (string, error) {
	n := strings.ToLower(strings.TrimSpace(name))
	if !roleNamePattern.MatchString(n) {
		return "", ErrInvalidRoleName
	}
	return n, nil
}

// PartnerRoleSlug builds the globally-unique slug of a partner-owned role
// (ARY-007): "p<partnerID>_<name>". Authorization is keyed on role names, so
// two partners may each own "Finance" ("p7_finance" / "p9_finance") without
// ambiguity. The suffix alone must still follow the role naming rule.
func PartnerRoleSlug(partnerID int, name string) (string, error) {
	prefix := fmt.Sprintf("p%d_", partnerID)
	suffix := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(name)), prefix)
	limit := 50 - len(prefix)
	if len(suffix) < 3 || len(suffix) > limit || !roleNamePattern.MatchString(suffix) {
		return "", ErrInvalidRoleName
	}
	return prefix + suffix, nil
}

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
	ID          int    `json:"id"`
	Name        string `json:"name"`         // Unique slug, e.g. "admin", "manajer", "p7_finance"
	DisplayName string `json:"display_name"` // e.g. "Administrator Sistem", "Finance"
	Description string `json:"description"`
	// PartnerID identifies the owning tenant (ARY-007). NULL marks a system
	// role shared by the whole platform.
	PartnerID    *int   `json:"partner_id,omitempty"`
	DivisionID   *int   `json:"division_id,omitempty"`
	DivisionName string `json:"division_name,omitempty"`
	UnitName     string `json:"unit_name,omitempty"`
	CategoryName string `json:"category_name,omitempty"`
	// IsSystem marks seeded platform roles. They can never be edited,
	// deactivated, or deleted through the role-management flows.
	IsSystem    bool         `json:"is_system"`
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
