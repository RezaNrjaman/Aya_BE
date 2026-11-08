package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrProductNotFound     = errors.New("produk/jasa tidak ditemukan")
	ErrProductNameRequired = errors.New("nama produk/jasa wajib diisi")
	ErrUnitIDRequired      = errors.New("unit usaha wajib dipilih")
	ErrInvalidPrice        = errors.New("harga produk tidak boleh negatif")
	ErrInvalidStock        = errors.New("stok produk tidak boleh negatif")
)

type Product struct {
	ID          int       `json:"id"`
	UnitUsahaID int       `json:"unit_usaha_id"`
	PartnerID   *int      `json:"partner_id,omitempty"`
	Name        string    `json:"name"`
	Category    string    `json:"category,omitempty"`
	Description string    `json:"description,omitempty"`
	Price       int64     `json:"price"` // Integer Rupiah
	Stock       int       `json:"stock"`
	IsActive    bool      `json:"is_active"`
	ImageURL    string    `json:"image_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (p *Product) Validate() error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return ErrProductNameRequired
	}
	if p.UnitUsahaID <= 0 {
		return ErrUnitIDRequired
	}
	if p.Price < 0 {
		return ErrInvalidPrice
	}
	if p.Stock < 0 {
		return ErrInvalidStock
	}
	return nil
}

func (p *Product) SetActive(active bool) {
	p.IsActive = active
	p.UpdatedAt = time.Now().UTC()
}

