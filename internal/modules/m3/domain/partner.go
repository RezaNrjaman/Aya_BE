package domain

import (
	"errors"
	"strings"
	"time"
)

const (
	StatusPending   = "pending"
	StatusVerified  = "terverifikasi"
	StatusRejected  = "ditolak"
	StatusSuspended = "ditangguhkan"
)

var (
	ErrPartnerNotFound          = errors.New("partner tidak ditemukan")
	ErrEmailAlreadyRegistered   = errors.New("email partner sudah terdaftar")
	ErrInvalidStatusTransition  = errors.New("transisi status partner tidak diizinkan")
	ErrRejectionReasonRequired  = errors.New("alasan penolakan wajib diisi")
	ErrSuspensionReasonRequired = errors.New("alasan penangguhan wajib diisi")
	ErrCompanyNameRequired      = errors.New("nama perusahaan partner wajib diisi")
	ErrContactPersonRequired    = errors.New("nama kontak person wajib diisi")
	ErrEmailRequired            = errors.New("email partner wajib diisi")
	ErrPhoneRequired            = errors.New("nomor telepon partner wajib diisi")
)

type Partner struct {
	ID            int    `json:"id"`
	UserID        *int   `json:"user_id,omitempty"`
	CompanyName   string `json:"company_name"`
	ContactPerson string `json:"contact_person"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	Address       string `json:"address,omitempty"`
	Status        string `json:"status"`
	// RegistrationCode is an unguessable tracking code handed to the registrant
	// at sign-up. It replaces sequential-ID lookups on the public status page
	// (ARY-002).
	RegistrationCode string     `json:"registration_code,omitempty"`
	RejectionReason  string     `json:"rejection_reason,omitempty"`
	SuspensionReason string     `json:"suspension_reason,omitempty"`
	VerifiedAt       *time.Time `json:"verified_at,omitempty"`
	VerifiedBy       string     `json:"verified_by,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (p *Partner) Validate() error {
	p.CompanyName = strings.TrimSpace(p.CompanyName)
	if p.CompanyName == "" {
		return ErrCompanyNameRequired
	}
	p.ContactPerson = strings.TrimSpace(p.ContactPerson)
	if p.ContactPerson == "" {
		return ErrContactPersonRequired
	}
	p.Email = strings.TrimSpace(strings.ToLower(p.Email))
	if p.Email == "" {
		return ErrEmailRequired
	}
	p.Phone = strings.TrimSpace(p.Phone)
	if p.Phone == "" {
		return ErrPhoneRequired
	}
	return nil
}

func (p *Partner) Verify(verifier string) error {
	if p.Status != StatusPending {
		return ErrInvalidStatusTransition
	}
	now := time.Now().UTC()
	p.Status = StatusVerified
	p.VerifiedAt = &now
	p.VerifiedBy = verifier
	p.UpdatedAt = now
	return nil
}

func (p *Partner) Reject(reason, verifier string) error {
	if p.Status != StatusPending {
		return ErrInvalidStatusTransition
	}
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" {
		return ErrRejectionReasonRequired
	}
	p.Status = StatusRejected
	p.RejectionReason = trimmed
	p.VerifiedBy = verifier
	p.UpdatedAt = time.Now().UTC()
	return nil
}

func (p *Partner) Suspend(reason string) error {
	if p.Status != StatusVerified {
		return ErrInvalidStatusTransition
	}
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" {
		return ErrSuspensionReasonRequired
	}
	p.Status = StatusSuspended
	p.SuspensionReason = trimmed
	p.UpdatedAt = time.Now().UTC()
	return nil
}

func (p *Partner) Reactivate() error {
	if p.Status != StatusSuspended {
		return ErrInvalidStatusTransition
	}
	p.Status = StatusVerified
	p.SuspensionReason = ""
	p.UpdatedAt = time.Now().UTC()
	return nil
}
