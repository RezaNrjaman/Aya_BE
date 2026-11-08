package dto

type CatalogFilter struct {
	UnitUsahaID *int   `form:"unit_usaha_id"`
	Category    string `form:"category"`
	Search      string `form:"search"`
}

type BusinessUnitSummary struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description,omitempty"`
	LogoURL     string `json:"logo_url,omitempty"`
}

type ProductSummary struct {
	ID          int    `json:"id"`
	UnitUsahaID int    `json:"unit_usaha_id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Price       int64  `json:"price"`
	Stock       int    `json:"stock"`
	IsActive    bool   `json:"is_active"`
}
