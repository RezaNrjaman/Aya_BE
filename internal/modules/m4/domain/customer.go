package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrCustomerNotFound = errors.New("pelanggan tidak ditemukan")
	ErrCustomerNameReq  = errors.New("nama pelanggan wajib diisi")
	ErrCustomerEmailReq = errors.New("email pelanggan wajib diisi")
)

type Customer struct {
	ID        int       `json:"id"`
	// PartnerID is the owning tenant (ARY-001). NULL means a platform-level
	// contact usable for public guest checkout before attribution.
	PartnerID *int      `json:"partner_id,omitempty"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone,omitempty"`
	Address   string    `json:"address,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (c *Customer) Validate() error {
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" {
		return ErrCustomerNameReq
	}
	c.Email = strings.TrimSpace(strings.ToLower(c.Email))
	if c.Email == "" {
		return ErrCustomerEmailReq
	}
	return nil
}
