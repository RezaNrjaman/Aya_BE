package dto

type QuotationItemInput struct {
	ProductID *int   `json:"product_id,omitempty"`
	ItemName  string `json:"item_name" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required"`
	UnitPrice int64  `json:"unit_price" binding:"required"`
}

type CreateQuotationRequest struct {
	UnitUsahaID int                  `json:"unit_usaha_id" binding:"required"`
	PartnerID   *int                 `json:"partner_id,omitempty"`
	CustomerID  int                  `json:"customer_id" binding:"required"`
	Items       []QuotationItemInput `json:"items" binding:"required"`
}
