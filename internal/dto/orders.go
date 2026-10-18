package dto

import "time"

type OrderItemResponse struct {
	ProductID   int    `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	PriceHT     int    `json:"price_ht"`
	PriceTTC    int    `json:"price_ttc"`
}

type OrderResponse struct {
	ID           int                 `json:"id"`
	Reference    string              `json:"reference"`
	Status       string              `json:"status"`
	CancelReason *string             `json:"cancel_reason,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
	Items        []OrderItemResponse `json:"items"`
}
