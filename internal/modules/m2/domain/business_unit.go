package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrUnitNotFound = errors.New("unit usaha tidak ditemukan")
	ErrNameRequired = errors.New("nama unit usaha wajib diisi")
)

type BusinessUnit struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// PartnerID identifies the owning tenant (ARY-005). NULL means the unit
	// is operated by the platform itself.
	PartnerID    *int      `json:"partner_id,omitempty"`
	CategoryID   *int      `json:"category_id,omitempty"`
	CategoryName string    `json:"category,omitempty"`
	DivisionID   *int      `json:"division_id,omitempty"`
	DivisionName string    `json:"division,omitempty"`
	Description  string    `json:"description,omitempty"`
	ContactEmail string    `json:"contact_email,omitempty"`
	ContactPhone string    `json:"contact_phone,omitempty"`
	Address      string    `json:"address,omitempty"`
	LogoURL      string    `json:"logo_url,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (b *BusinessUnit) Validate() error {
	b.Name = strings.TrimSpace(b.Name)
	if b.Name == "" {
		return ErrNameRequired
	}
	return nil
}

func (b *BusinessUnit) Deactivate() {
	b.IsActive = false
	b.UpdatedAt = time.Now().UTC()
}

func (b *BusinessUnit) Activate() {
	b.IsActive = true
	b.UpdatedAt = time.Now().UTC()
}
