package domain

import "time"

// Cart représente le panier d'un utilisateur. carts.user_id est UNIQUE en
// base : un utilisateur n'a jamais qu'un seul panier, créé au premier
// ajout et réutilisé indéfiniment (vidé après chaque validation, jamais
// recréé).
type Cart struct {
	ID        int
	Reference string
	UserID    int
	Items     []CartItem
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CartItem représente une ligne du panier, avec les informations du
// produit au moment de la lecture. Contrairement à OrderItem, rien n'est
// figé : le panier reflète toujours le nom et le prix courants du produit.
type CartItem struct {
	ProductID   int
	ProductName string
	Quantity    int
	PriceHT     int
	PriceTTC    int
}

// TotalTTC renvoie le total TTC du panier : somme des (prix TTC unitaire ×
// quantité), sans arrondi supplémentaire. La livraison est toujours
// gratuite, donc ce total est aussi celui à payer.
func (c Cart) TotalTTC() int {
	total := 0
	for _, item := range c.Items {
		total += item.PriceTTC * item.Quantity
	}
	return total
}
