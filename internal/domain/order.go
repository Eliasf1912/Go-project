package domain

import "time"

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusShipping  OrderStatus = "shipping"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCancelled OrderStatus = "cancelled"
)

type Order struct {
	ID        int
	Reference string
	UserID    int
	CartID    *int
	Status    OrderStatus
	// CancelReason n'est rempli que si Status == OrderStatusCancelled.
	CancelReason *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Items        []OrderItem
}

type OrderItem struct {
	ID          int
	ProductID   int
	ProductName string
	Quantity    int
	PriceHT     int
	PriceTTC    int
}
