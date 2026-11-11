package domain

import (
	"errors"
	"strings"
	"time"
)

const (
	RequestPending    = "pending"
	RequestProcessing = "diproses"
	RequestCompleted  = "selesai"
)

var (
	ErrRequestNotFound      = errors.New("permintaan layanan tidak ditemukan")
	ErrInvalidRequestStatus = errors.New("transisi status permintaan tidak valid")
	ErrInvalidEmail         = errors.New("format email pelanggan tidak valid")
	ErrRequiredFieldMissing = errors.New("unit usaha, nama, email, dan deskripsi wajib diisi")
)

type ServiceRequest struct {
	ID            int       `json:"id"`
	PartnerID     *int      `json:"partner_id,omitempty"`
	UnitUsahaID   int       `json:"unit_usaha_id"`
	CustomerName  string    `json:"customer_name"`
	CustomerEmail string    `json:"customer_email"`
	CustomerPhone string    `json:"customer_phone,omitempty"`
	ServiceType   string    `json:"service_type,omitempty"`
	Description   string    `json:"description"`
	Status        string    `json:"status"` // pending, diproses, selesai
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (r *ServiceRequest) Validate() error {
	if r.UnitUsahaID <= 0 || strings.TrimSpace(r.CustomerName) == "" ||
		strings.TrimSpace(r.CustomerEmail) == "" || strings.TrimSpace(r.Description) == "" {
		return ErrRequiredFieldMissing
	}
	if !strings.Contains(r.CustomerEmail, "@") || !strings.Contains(r.CustomerEmail, ".") {
		return ErrInvalidEmail
	}
	return nil
}

func (r *ServiceRequest) Process() error {
	if r.Status != RequestPending {
		return ErrInvalidRequestStatus
	}
	r.Status = RequestProcessing
	r.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *ServiceRequest) Complete() error {
	if r.Status != RequestProcessing {
		return ErrInvalidRequestStatus
	}
	r.Status = RequestCompleted
	r.UpdatedAt = time.Now().UTC()
	return nil
}
