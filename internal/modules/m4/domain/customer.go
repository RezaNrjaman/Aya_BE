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

