package dto

type CreateDivisionRequest struct {
	Name        string `json:"name" binding:"required"`
	UnitUsahaID int    `json:"unit_usaha_id" binding:"required"`
	CategoryID  int    `json:"category_id" binding:"required"`
	Description string `json:"description,omitempty"`
}

type UpdateDivisionRequest struct {
	Name        *string `json:"name,omitempty"`
	UnitUsahaID *int    `json:"unit_usaha_id,omitempty"`
	CategoryID  *int    `json:"category_id,omitempty"`
	Description *string `json:"description,omitempty"`
}
