package domain

import (
	"strings"
	"time"
)

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

// MarkAsPaid fait passer la commande de pending à paid. Impossible depuis
// un autre statut : une commande déjà payée, livrée ou annulée ne peut
// pas repasser à paid.
func (o *Order) MarkAsPaid() error {
	if o.Status != OrderStatusPending {
		return ErrOrderNotPending
	}
	o.Status = OrderStatusPaid
	return nil
}

// Cancel fait passer la commande en "cancelled", avec une raison
// obligatoire. Possible seulement depuis pending ou paid : une commande déjà en livraison, livrée ou annulée ne peut
// pas être annulée.
func (o *Order) Cancel(reason string) error {
	if o.Status != OrderStatusPending && o.Status != OrderStatusPaid {
		return ErrOrderNotCancellable
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrCancelReasonRequired
	}
	o.Status = OrderStatusCancelled
	o.CancelReason = &reason
	return nil
}

// TotalTTC renvoie le total TTC de la commande : la somme, pour chaque
// ligne, du prix TTC unitaire multiplié par la quantité. Toujours
// calculé ici, jamais stocké en base.
func (o Order) TotalTTC() int {
	total := 0
	for _, item := range o.Items {
		total += item.PriceTTC * item.Quantity
	}
	return total
}

type OrderItem struct {
	ID          int
	ProductID   int
	ProductName string
	Quantity    int
	PriceHT     int
	PriceTTC    int
}
