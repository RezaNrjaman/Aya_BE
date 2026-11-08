package dto

type CreateCustomerRequest struct {
	Name    string `json:"name" binding:"required"`
	Email   string `json:"email" binding:"required"`
	Phone   string `json:"phone,omitempty"`
	Address string `json:"address,omitempty"`
}

type UpdateCustomerRequest struct {
	Name    *string `json:"name,omitempty"`
	Email   *string `json:"email,omitempty"`
	Phone   *string `json:"phone,omitempty"`
	Address *string `json:"address,omitempty"`
}
