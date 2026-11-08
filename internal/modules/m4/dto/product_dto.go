package dto

type CreateProductRequest struct {
	UnitUsahaID int    `json:"unit_usaha_id" binding:"required"`
	PartnerID   *int   `json:"partner_id,omitempty"`
	Name        string `json:"name" binding:"required"`
	Category    string `json:"category,omitempty"`
	Description string `json:"description,omitempty"`
	Price       int64  `json:"price" binding:"required"`
	Stock       int    `json:"stock"`
	ImageURL    string `json:"image_url,omitempty"`
}

type UpdateProductRequest struct {
	Name        *string `json:"name,omitempty"`
	Category    *string `json:"category,omitempty"`
	Description *string `json:"description,omitempty"`
	Price       *int64  `json:"price,omitempty"`
	Stock       *int    `json:"stock,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
	ImageURL    *string `json:"image_url,omitempty"`
}
