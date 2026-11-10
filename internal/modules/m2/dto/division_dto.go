package dto

type CreateDivisionRequest struct {
	Name string `json:"name" binding:"required"`
	// PartnerID is honored only for platform-wide actors (ARY-006); a
	// partner-scoped actor always owns the division it creates.
	PartnerID   *int   `json:"partner_id,omitempty"`
	Description string `json:"description,omitempty"`
}

type UpdateDivisionRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}
