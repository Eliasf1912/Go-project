package dto

type CartItemResponse struct {
	ProductID   int    `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	PriceHT     int    `json:"price_ht"`
	PriceTTC    int    `json:"price_ttc"`
}

type CartResponse struct {
	Reference string             `json:"reference,omitempty"`
	Items     []CartItemResponse `json:"items"`
	TotalTTC  int                `json:"total_ttc"`
}

type AddCartItemRequest struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

type UpdateCartItemRequest struct {
	Quantity int `json:"quantity"`
}
