package dto

type ProductResponse struct {
	ID          int              `json:"id"`
	Reference   string           `json:"reference"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	PriceHT     int              `json:"price_ht"`
	PriceTTC    int              `json:"price_ttc"`
	Category    CategoryResponse `json:"category"`
}
