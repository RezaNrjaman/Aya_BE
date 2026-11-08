package domain

import (
	"errors"
	"time"
)

const (
	QuotationDraft     = "draft"
	QuotationSubmitted = "diajukan"
	QuotationApproved  = "disetujui"
	QuotationRejected  = "ditolak"
)

var (
	ErrQuotationNotFound       = errors.New("quotation tidak ditemukan")
	ErrQuotationAlreadyHandled = errors.New("quotation sudah disetujui atau ditolak")
	ErrQuotationItemsRequired  = errors.New("item penawaran wajib diisi minimal 1 item")
)

type QuotationItem struct {
	ID          int    `json:"id"`
	QuotationID int    `json:"quotation_id"`
	ProductID   *int   `json:"product_id,omitempty"`
	ItemName    string `json:"item_name"`
	Quantity    int    `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
	Subtotal    int64  `json:"subtotal"`
}

type Quotation struct {
	ID          int             `json:"id"`
	QuotationNo string          `json:"quotation_no"`
	UnitUsahaID int             `json:"unit_usaha_id"`
	PartnerID   *int            `json:"partner_id,omitempty"`
	CustomerID  int             `json:"customer_id"`
	TotalAmount int64           `json:"total_amount"`
	Status      string          `json:"status"` // draft, diajukan, disetujui, ditolak
	ValidUntil  *time.Time      `json:"valid_until,omitempty"`
	Items       []QuotationItem `json:"items,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func (q *Quotation) Approve() error {
	if q.Status == QuotationApproved || q.Status == QuotationRejected {
		return ErrQuotationAlreadyHandled
	}
	q.Status = QuotationApproved
	q.UpdatedAt = time.Now().UTC()
	return nil
}

func (q *Quotation) Reject() error {
	if q.Status == QuotationApproved || q.Status == QuotationRejected {
		return ErrQuotationAlreadyHandled
	}
	q.Status = QuotationRejected
	q.UpdatedAt = time.Now().UTC()
	return nil
}

