package domain

import (
	"errors"
	"time"
)

const (
	InvoiceUnpaid    = "belum_lunas"
	InvoicePaid      = "lunas"
	InvoiceCancelled = "dibatalkan"
)

var (
	ErrInvoiceNotFound = errors.New("invoice tidak ditemukan")
	ErrInvoicePaid     = errors.New("invoice sudah berstatus lunas")
)

type InvoiceItem struct {
	ID        int    `json:"id"`
	InvoiceID int    `json:"invoice_id"`
	ProductID *int   `json:"product_id,omitempty"`
	ItemName  string `json:"item_name"`
	Quantity  int    `json:"quantity"`
	UnitPrice int64  `json:"unit_price"`
	Subtotal  int64  `json:"subtotal"`
}

type Invoice struct {
	ID          int           `json:"id"`
	InvoiceNo   string        `json:"invoice_no"`
	QuotationID *int          `json:"quotation_id,omitempty"`
	// PartnerID is the owning tenant (ARY-001); derived from the approved
	// quotation or the authenticated actor, never from the request body.
	PartnerID   *int          `json:"partner_id,omitempty"`
	CustomerID  int           `json:"customer_id"`
	TotalAmount int64         `json:"total_amount"`
	Status      string        `json:"status"` // belum_lunas, lunas, dibatalkan
	DueDate     *time.Time    `json:"due_date,omitempty"`
	PaidAt      *time.Time    `json:"paid_at,omitempty"`
	Items       []InvoiceItem `json:"items,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

func (inv *Invoice) MarkAsPaid(paidAt time.Time) {
	inv.Status = InvoicePaid
	inv.PaidAt = &paidAt
	inv.UpdatedAt = paidAt
}
