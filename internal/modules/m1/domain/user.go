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
	// ErrAccountNotActivated is returned when an account exists but its owner
	// has not finished the activation invitation yet (no password set).
	ErrAccountNotActivated = errors.New("akun belum diaktivasi, silakan gunakan tautan aktivasi yang dikirim ke email Anda")
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
	// ActivationTokenHash stores the SHA-256 hash of the single-use activation
	// invitation token; the plaintext token is never persisted (ARY-002).
	ActivationTokenHash string     `json:"-"`
	ActivationExpiresAt *time.Time `json:"-"`
	// IsManuallyDisabled marks accounts disabled by an administrator directly
	// (as opposed to being disabled because the partner was suspended), so a
	// partner reactivation never re-enables them automatically (ARY-004).
	IsManuallyDisabled bool      `json:"-"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// CanLogin checks if the user account is allowed to log in. Accounts that have
// not completed activation (no password set) can never log in.
func (u *User) CanLogin() bool {
	return u.IsActive && u.PasswordHash != ""
}

// NeedsActivation reports whether the account is waiting for its owner to set
// an initial password through an activation invitation.
func (u *User) NeedsActivation() bool {
	return u.PasswordHash == ""
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
