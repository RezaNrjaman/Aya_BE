package domain

import (
	"errors"
	"time"
)

const (
	OrderNew        = "baru"
	OrderProcessing = "diproses"
	OrderCompleted  = "selesai"
	OrderCancelled  = "dibatalkan"
)

var (
	ErrOrderNotFound      = errors.New("pesanan tidak ditemukan")
	ErrOrderAlreadyFinal  = errors.New("order sudah selesai dan tidak dapat diubah lagi")
	ErrInvalidOrderState  = errors.New("transisi status order tidak valid")
	ErrOrderItemsRequired = errors.New("item order wajib diisi minimal 1 item")
)

type OrderItem struct {
	ID        int    `json:"id"`
	OrderID   int    `json:"order_id"`
	ProductID int    `json:"product_id"`
	ItemName  string `json:"item_name"`
	Quantity  int    `json:"quantity"`
	UnitPrice int64  `json:"unit_price"`
	Subtotal  int64  `json:"subtotal"`
}

type OrderStatusHistory struct {
	ID         int       `json:"id"`
	OrderID    int       `json:"order_id"`
	FromStatus string    `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Reason     string    `json:"reason,omitempty"`
	ChangedBy  string    `json:"changed_by,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Order struct {
	ID      int    `json:"id"`
	OrderNo string `json:"order_no"`
	// PartnerID is the owning tenant (ARY-001): derived from the products in
	// the order. Mixed-partner carts are rejected.
	PartnerID      *int                 `json:"partner_id,omitempty"`
	CustomerID     int                  `json:"customer_id"`
	InvoiceID      *int                 `json:"invoice_id,omitempty"`
	TotalAmount    int64                `json:"total_amount"`
	Status         string               `json:"status"` // baru, diproses, selesai, dibatalkan
	IdempotencyKey string               `json:"idempotency_key,omitempty"`
	Items          []OrderItem          `json:"items,omitempty"`
	Histories      []OrderStatusHistory `json:"histories,omitempty"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

func (o *Order) Process() error {
	if o.Status == OrderCompleted {
		return ErrOrderAlreadyFinal
	}
	if o.Status != OrderNew {
		return ErrInvalidOrderState
	}
	o.Status = OrderProcessing
	o.UpdatedAt = time.Now().UTC()
	return nil
}

func (o *Order) Complete() error {
	if o.Status == OrderCompleted {
		return ErrOrderAlreadyFinal
	}
	if o.Status != OrderProcessing {
		return ErrInvalidOrderState
	}
	o.Status = OrderCompleted
	o.UpdatedAt = time.Now().UTC()
	return nil
}

func (o *Order) Cancel() error {
	if o.Status == OrderCompleted {
		return ErrOrderAlreadyFinal
	}
	o.Status = OrderCancelled
	o.UpdatedAt = time.Now().UTC()
	return nil
}
