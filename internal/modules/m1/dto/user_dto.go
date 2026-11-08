package dto

type CreateUserRequest struct {
	Email       string `json:"email" binding:"required"`
	Password    string `json:"password" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Phone       string `json:"phone,omitempty"`
	DivisionID  *int   `json:"division_id,omitempty"`
	Role        string `json:"role,omitempty"`
	UnitUsahaID *int   `json:"unit_usaha_id,omitempty"`
}

type UpdateUserRequest struct {
	Name        *string `json:"name,omitempty"`
	Phone       *string `json:"phone,omitempty"`
	DivisionID  *int    `json:"division_id,omitempty"`
	Role        *string `json:"role,omitempty"`
	UnitUsahaID *int    `json:"unit_usaha_id,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

type AssignRoleRequest struct {
	RoleName    string `json:"role_name" binding:"required"`
	UnitUsahaID *int   `json:"unit_usaha_id,omitempty"`
}

type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
	Description string `json:"description,omitempty"`
}

type UpdateRoleRequest struct {
	DisplayName *string `json:"display_name,omitempty"`
	Description *string `json:"description,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

type AssignPermissionRequest struct {
	PermissionKey string `json:"permission_key" binding:"required"`
}
