package domain

import "time"

type StatusHistory struct {
	ID         int       `json:"id"`
	PartnerID  int       `json:"partner_id"`
	FromStatus string    `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Reason     string    `json:"reason,omitempty"`
	ChangedBy  string    `json:"changed_by,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}
