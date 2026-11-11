package dto

type CheckoutRequest struct {
	// CustomerID is accepted only for backwards compatibility with internal
	// callers; the public guest checkout flow never trusts or sends it.
	CustomerID      int    `json:"customer_id,omitempty"`
	CustomerName    string `json:"customer_name" binding:"required"`
	CustomerEmail   string `json:"customer_email" binding:"required,email"`
	CustomerPhone   string `json:"customer_phone,omitempty"`
	CustomerAddress string `json:"customer_address,omitempty"`
	IdempotencyKey  string `json:"idempotency_key,omitempty"`
	Notes           string `json:"notes,omitempty"`
}

type CheckoutResponse struct {
	OrderID        int    `json:"order_id"`
	OrderNo        string `json:"order_no"`
	TotalAmount    int64  `json:"total_amount"`
	Status         string `json:"status"`
	CartCleared    bool   `json:"cart_cleared"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}
