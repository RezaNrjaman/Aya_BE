package dto

import "araya-business-hub/be/internal/modules/m5/domain"

type AddToCartRequest struct {
	ProductID int `json:"product_id" binding:"required"`
	Quantity  int `json:"quantity" binding:"required"`
}

type UpdateCartItemRequest struct {
	Quantity int `json:"quantity" binding:"required"`
}

type CartResponse struct {
	ID          int               `json:"id"`
	SessionID   string            `json:"session_id"`
	TotalItems  int               `json:"total_items"`
	TotalAmount int64             `json:"total_amount"`
	Items       []domain.CartItem `json:"items"`
}
