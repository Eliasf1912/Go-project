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

// MarkAsShipping fait passer la commande de paid à shipping. Réservé à
// l'admin : le client n'a aucune action sur la livraison.
func (o *Order) MarkAsShipping() error {
	if o.Status != OrderStatusPaid {
		return ErrOrderNotPaid
	}
	o.Status = OrderStatusShipping
	return nil
}

// MarkAsDelivered fait passer la commande de shipping à delivered. Réservé
// à l'admin.
func (o *Order) MarkAsDelivered() error {
	if o.Status != OrderStatusShipping {
		return ErrOrderNotShipping
	}
	o.Status = OrderStatusDelivered
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

// NewOrderFromCart construit une commande "pending" à partir du panier d'un
// utilisateur : les lignes du panier sont copiées (nom et prix HT figés,
// comme l'exige le projet : l'historique d'une commande ne doit pas changer
// si un prix est modifié plus tard). Refuse un panier vide.
func NewOrderFromCart(cart Cart, reference string) (Order, error) {
	if len(cart.Items) == 0 {
		return Order{}, ErrCartEmpty
	}

	items := make([]OrderItem, 0, len(cart.Items))
	for _, ci := range cart.Items {
		items = append(items, OrderItem{
			ProductID:   ci.ProductID,
			ProductName: ci.ProductName,
			Quantity:    ci.Quantity,
			PriceHT:     ci.PriceHT,
		})
	}

	cartID := cart.ID
	return Order{
		Reference: reference,
		UserID:    cart.UserID,
		CartID:    &cartID,
		Status:    OrderStatusPending,
		Items:     items,
	}, nil
}

// NewOrderForUser construit une commande "pending" créée par un admin pour
// un client : pas de panier associé (CartID reste nil), les lignes sont
// déjà résolues (produit actif, nom et prix HT recopiés) par l'appelant.
// Refuse une commande sans aucun produit.
func NewOrderForUser(userID int, items []OrderItem, reference string) (Order, error) {
	if len(items) == 0 {
		return Order{}, ErrOrderHasNoItems
	}
	return Order{
		Reference: reference,
		UserID:    userID,
		Status:    OrderStatusPending,
		Items:     items,
	}, nil
}

type OrderItem struct {
	ID          int
	ProductID   int
	ProductName string
	Quantity    int
	PriceHT     int
	PriceTTC    int
}
