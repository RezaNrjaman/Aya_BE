package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrDivisionNotFound  = errors.New("divisi tidak ditemukan")
	ErrDivisionInUse     = errors.New("divisi masih digunakan oleh anggota atau unit usaha; pindahkan anggotanya terlebih dahulu")
	ErrDuplicateDivision = errors.New("nama divisi sudah digunakan di tenant ini")
)

// Division is an organisational subdivision owned by a partner tenant
// (ARY-006). A NULL PartnerID marks a platform-level division.
type Division struct {
	ID           int       `json:"id"`
	PartnerID    *int      `json:"partner_id,omitempty"`
	UnitUsahaID  int       `json:"unit_usaha_id"`
	CategoryID   int       `json:"category_id"`
	UnitName     string    `json:"unit_name,omitempty"`
	CategoryName string    `json:"category_name,omitempty"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (d *Division) Validate() error {
	d.Name = strings.TrimSpace(d.Name)
	if d.UnitUsahaID <= 0 || d.CategoryID <= 0 {
		return errors.New("unit usaha dan kategori wajib dipilih")
	}
	if d.Name == "" {
		return ErrNameRequired
	}
	return nil
}
