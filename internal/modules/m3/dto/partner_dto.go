package dto

import "araya-business-hub/be/internal/modules/m3/domain"

type RegisterPartnerRequest struct {
	CompanyName   string `json:"company_name" binding:"required"`
	ContactPerson string `json:"contact_person" binding:"required"`
	Email         string `json:"email" binding:"required"`
	Phone         string `json:"phone" binding:"required"`
	Address       string `json:"address,omitempty"`
}

type VerifyPartnerRequest struct {
	Action string `json:"action" binding:"required"` // "approve" atau "reject"
	Reason string `json:"reason,omitempty"`
}

type SuspendPartnerRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type PartnerDetailResponse struct {
	Partner   domain.Partner         `json:"partner"`
	Histories []domain.StatusHistory `json:"histories"`
}
