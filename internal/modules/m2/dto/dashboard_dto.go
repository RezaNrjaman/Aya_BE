package dto

type DashboardMetrics struct {
	TotalRevenue         int64 `json:"total_revenue"` // In Rupiah integer
	TotalOrders          int   `json:"total_orders"`
	ActivePartners       int   `json:"active_partners"`
	ActiveProducts       int   `json:"active_products"`
	PendingVerifications int   `json:"pending_verifications"`
}

type OrderSummaryItem struct {
	OrderNo     string `json:"order_no"`
	UnitName    string `json:"unit_name"`
	TotalAmount int64  `json:"total_amount"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

type DashboardSummaryResponse struct {
	UnitUsahaID  *int               `json:"unit_usaha_id,omitempty"`
	PeriodFrom   string             `json:"period_from,omitempty"`
	PeriodTo     string             `json:"period_to,omitempty"`
	Metrics      DashboardMetrics   `json:"metrics"`
	RecentOrders []OrderSummaryItem `json:"recent_orders"`
}
