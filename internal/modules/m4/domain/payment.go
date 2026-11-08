package domain

import "time"

const (
	PaymentPending = "pending"
	PaymentSuccess = "success"
	PaymentFailed  = "failed"
)

type Payment struct {
	ID               int       `json:"id"`
	PaymentReference string    `json:"payment_reference"`
	InvoiceID        int       `json:"invoice_id"`
	Amount           int64     `json:"amount"`
	Method           string    `json:"method"`
	Status           string    `json:"status"` // pending, success, failed
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

