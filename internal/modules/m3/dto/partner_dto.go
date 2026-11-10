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

// PublicPartnerView is the safe projection of a partner for the public
// verified-partners list: it never exposes tracking codes, verification
// metadata or linkage IDs (ARY-002).
type PublicPartnerView struct {
	ID            int    `json:"id"`
	CompanyName   string `json:"company_name"`
	ContactPerson string `json:"contact_person"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	Address       string `json:"address,omitempty"`
	Status        string `json:"status"`
}

// PublicPartnerStatusResponse is the minimal status payload returned by the
// public status page. It is addressed by an unguessable registration code —
// never by sequential ID — and omits contact details and internal history
// (ARY-002). ActivationToken is only present while the partner account has
// never been activated, simulating the invitation e-mail delivery.
type PublicPartnerStatusResponse struct {
	PartnerID        int    `json:"partner_id"`
	CompanyName      string `json:"company_name"`
	Status           string `json:"status"`
	RejectionReason  string `json:"rejection_reason,omitempty"`
	SuspensionReason string `json:"suspension_reason,omitempty"`
	ActivationToken  string `json:"activation_token,omitempty"`
}
