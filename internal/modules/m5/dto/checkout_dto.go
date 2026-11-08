package dto

type CheckoutRequest struct {
	CustomerID     int    `json:"customer_id" binding:"required"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
	Notes          string `json:"notes,omitempty"`
}

type CheckoutResponse struct {
	OrderID        int    `json:"order_id"`
	OrderNo        string `json:"order_no"`
	TotalAmount    int64  `json:"total_amount"`
	Status         string `json:"status"`
	CartCleared    bool   `json:"cart_cleared"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}
