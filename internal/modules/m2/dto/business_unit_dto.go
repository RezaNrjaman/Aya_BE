package dto

type CreateUnitRequest struct {
	Name string `json:"name" binding:"required"`
	// PartnerID is honored only for platform-wide actors (ARY-005); a
	// partner-scoped actor always owns the unit it creates.
	PartnerID    *int   `json:"partner_id,omitempty"`
	CategoryID   *int   `json:"category_id,omitempty"`
	CategoryName string `json:"category,omitempty"`
	DivisionID   *int   `json:"division_id,omitempty"`
	DivisionName string `json:"division,omitempty"`
	Description  string `json:"description,omitempty"`
	ContactEmail string `json:"contact_email,omitempty"`
	ContactPhone string `json:"contact_phone,omitempty"`
	Address      string `json:"address,omitempty"`
	LogoURL      string `json:"logo_url,omitempty"`
}

type UpdateUnitRequest struct {
	Name         *string `json:"name,omitempty"`
	CategoryID   *int    `json:"category_id,omitempty"`
	CategoryName *string `json:"category,omitempty"`
	DivisionID   *int    `json:"division_id,omitempty"`
	DivisionName *string `json:"division,omitempty"`
	Description  *string `json:"description,omitempty"`
	ContactEmail *string `json:"contact_email,omitempty"`
	ContactPhone *string `json:"contact_phone,omitempty"`
	Address      *string `json:"address,omitempty"`
	LogoURL      *string `json:"logo_url,omitempty"`
}
