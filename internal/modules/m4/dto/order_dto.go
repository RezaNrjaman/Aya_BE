package dto

type OrderItemInput struct {
	ProductID int `json:"product_id" binding:"required"`
	Quantity  int `json:"quantity" binding:"required"`
}

type CreateOrderRequest struct {
	CustomerID      int              `json:"customer_id,omitempty"`
	CustomerName    string           `json:"customer_name,omitempty"`
	CustomerEmail   string           `json:"customer_email,omitempty"`
	CustomerPhone   string           `json:"customer_phone,omitempty"`
	CustomerAddress string           `json:"customer_address,omitempty"`
	IdempotencyKey  string           `json:"idempotency_key,omitempty"`
	Items           []OrderItemInput `json:"items" binding:"required"`
}
