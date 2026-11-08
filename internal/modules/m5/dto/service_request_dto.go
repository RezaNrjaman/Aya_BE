package dto

type CreateServiceRequest struct {
	UnitUsahaID   int    `json:"unit_usaha_id" binding:"required"`
	CustomerName  string `json:"customer_name" binding:"required"`
	CustomerEmail string `json:"customer_email" binding:"required"`
	CustomerPhone string `json:"customer_phone,omitempty"`
	ServiceType   string `json:"service_type,omitempty"`
	Description   string `json:"description" binding:"required"`
}

type UpdateServiceRequestStatus struct {
	Status string `json:"status" binding:"required"` // diproses, selesai
}
