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
	ID        int    `json:"id"`
	Reference string `json:"reference"`
	// UserID identifie le client : utile pour l'admin (liste de toutes les
	// commandes) ; sans risque pour le client, qui ne voit que les siennes.
	UserID       int                 `json:"user_id"`
	Status       string              `json:"status"`
	CancelReason *string             `json:"cancel_reason,omitempty"`
	TotalTTC     int                 `json:"total_ttc"`
	CreatedAt    time.Time           `json:"created_at"`
	Items        []OrderItemResponse `json:"items"`
}

// AdminCreateOrderRequest est la requête de l'admin pour créer une
// commande au nom d'un client, à partir d'une liste de produits et de
// quantités (pas de panier impliqué).
type AdminCreateOrderRequest struct {
	UserID int                     `json:"user_id"`
	Items  []AdminOrderItemRequest `json:"items"`
}

type AdminOrderItemRequest struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}
