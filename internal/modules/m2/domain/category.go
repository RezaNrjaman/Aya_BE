package domain

import (
	"errors"
	"strings"
	"time"
)

type BusinessCategory struct {
	ID          int       `json:"id"`
	PartnerID   *int      `json:"partner_id,omitempty"`
	UnitUsahaID int       `json:"unit_usaha_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (c *BusinessCategory) Validate() error {
	c.Name = strings.TrimSpace(c.Name)
	if c.UnitUsahaID <= 0 {
		return errors.New("unit usaha wajib dipilih")
	}
	if c.Name == "" {
		return errors.New("nama kategori wajib diisi")
	}
	return nil
}
