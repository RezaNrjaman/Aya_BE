package dto

type InvoiceItemInput struct {
	ProductID *int   `json:"product_id,omitempty"`
	ItemName  string `json:"item_name" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required"`
	UnitPrice int64  `json:"unit_price" binding:"required"`
}

type CreateInvoiceRequest struct {
	QuotationID *int               `json:"quotation_id,omitempty"`
	CustomerID  int                `json:"customer_id" binding:"required"`
	DueDateDays int                `json:"due_date_days,omitempty"`
	Items       []InvoiceItemInput `json:"items,omitempty"`
}

type PayInvoiceRequest struct {
	Method          string `json:"method" binding:"required"` // sandbox_qris, sandbox_va, sandbox_bank_transfer
	Amount          int64  `json:"amount" binding:"required"`
	SimulateFailure bool   `json:"simulate_failure,omitempty"`
}

type PaymentCallbackRequest struct {
	PaymentReference string `json:"payment_reference" binding:"required"`
	InvoiceID        int    `json:"invoice_id" binding:"required"`
	Amount           int64  `json:"amount" binding:"required"`
	Status           string `json:"status" binding:"required"` // SUCCESS, FAILED
}
