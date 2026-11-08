package dto

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserSummary struct {
	ID          int      `json:"id"`
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	Role        string   `json:"role"`
	Division    string   `json:"division"`
	UnitUsahaID *int     `json:"unit_usaha_id,omitempty"`
	Permissions []string `json:"permissions"`
}

type LoginResponse struct {
	Token string      `json:"token"`
	User  UserSummary `json:"user"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

type ProfileResponse struct {
	ID          int      `json:"id"`
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	Role        string   `json:"role"`
	Division    string   `json:"division"`
	UnitUsahaID *int     `json:"unit_usaha_id,omitempty"`
	Permissions []string `json:"permissions"`
}
