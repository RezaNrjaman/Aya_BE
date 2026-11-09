package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrUserInactive   = errors.New("akun dinonaktifkan, silakan hubungi administrator")
	ErrInvalidEmail   = errors.New("format email tidak valid")
	ErrNameRequired   = errors.New("nama wajib diisi")
	ErrUserNotFound   = errors.New("pengguna tidak ditemukan")
	ErrDuplicateEmail = errors.New("email sudah terdaftar di sistem")
)

type User struct {
	ID           int       `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	Phone        string    `json:"phone,omitempty"`
	DivisionID   *int      `json:"division_id,omitempty"`
	DivisionName string    `json:"division,omitempty"`
	Role         string    `json:"role,omitempty"`
	UnitUsahaID  *int      `json:"unit_usaha_id,omitempty"`
	PartnerID    *int      `json:"partner_id,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CanLogin checks if the user account is allowed to log in.
func (u *User) CanLogin() bool {
	return u.IsActive
}

// Validate checks essential user fields.
func (u *User) Validate() error {
	u.Email = strings.TrimSpace(strings.ToLower(u.Email))
	u.Name = strings.TrimSpace(u.Name)

	if u.Email == "" || !strings.Contains(u.Email, "@") {
		return ErrInvalidEmail
	}
	if u.Name == "" {
		return ErrNameRequired
	}
	return nil
}
