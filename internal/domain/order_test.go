package domain

import (
	"errors"
	"testing"
)

func TestMarkAsPaid(t *testing.T) {
	tests := []struct {
		from       OrderStatus
		wantErr    error
		wantStatus OrderStatus
	}{
		{OrderStatusPending, nil, OrderStatusPaid},
		{OrderStatusPaid, ErrOrderNotPending, OrderStatusPaid},
		{OrderStatusShipping, ErrOrderNotPending, OrderStatusShipping},
		{OrderStatusDelivered, ErrOrderNotPending, OrderStatusDelivered},
		{OrderStatusCancelled, ErrOrderNotPending, OrderStatusCancelled},
	}

	for _, tt := range tests {
		t.Run(string(tt.from), func(t *testing.T) {
			order := Order{Status: tt.from}

			err := order.MarkAsPaid()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("MarkAsPaid() = %v, attendu %v", err, tt.wantErr)
			}
			// En cas d'erreur, le statut ne doit pas avoir bougé.
			if order.Status != tt.wantStatus {
				t.Errorf("statut = %s, attendu %s", order.Status, tt.wantStatus)
			}
		})
	}
}

func TestMarkAsShipping(t *testing.T) {
	tests := []struct {
		from       OrderStatus
		wantErr    error
		wantStatus OrderStatus
	}{
		{OrderStatusPaid, nil, OrderStatusShipping},
		{OrderStatusPending, ErrOrderNotPaid, OrderStatusPending},
		{OrderStatusShipping, ErrOrderNotPaid, OrderStatusShipping},
		{OrderStatusDelivered, ErrOrderNotPaid, OrderStatusDelivered},
		{OrderStatusCancelled, ErrOrderNotPaid, OrderStatusCancelled},
	}

	for _, tt := range tests {
		t.Run(string(tt.from), func(t *testing.T) {
			order := Order{Status: tt.from}

			err := order.MarkAsShipping()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("MarkAsShipping() = %v, attendu %v", err, tt.wantErr)
			}
			if order.Status != tt.wantStatus {
				t.Errorf("statut = %s, attendu %s", order.Status, tt.wantStatus)
			}
		})
	}
}

func TestMarkAsDelivered(t *testing.T) {
	tests := []struct {
		from       OrderStatus
		wantErr    error
		wantStatus OrderStatus
	}{
		{OrderStatusShipping, nil, OrderStatusDelivered},
		{OrderStatusPending, ErrOrderNotShipping, OrderStatusPending},
		{OrderStatusPaid, ErrOrderNotShipping, OrderStatusPaid},
		{OrderStatusDelivered, ErrOrderNotShipping, OrderStatusDelivered},
		{OrderStatusCancelled, ErrOrderNotShipping, OrderStatusCancelled},
	}

	for _, tt := range tests {
		t.Run(string(tt.from), func(t *testing.T) {
			order := Order{Status: tt.from}

			err := order.MarkAsDelivered()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("MarkAsDelivered() = %v, attendu %v", err, tt.wantErr)
			}
			if order.Status != tt.wantStatus {
				t.Errorf("statut = %s, attendu %s", order.Status, tt.wantStatus)
			}
		})
	}
}

func TestNewOrderForUser(t *testing.T) {
	t.Run("sans article", func(t *testing.T) {
		_, err := NewOrderForUser(10, nil, "CMD-TEST00")
		if !errors.Is(err, ErrOrderHasNoItems) {
			t.Errorf("NewOrderForUser() = %v, attendu %v", err, ErrOrderHasNoItems)
		}
	})

	t.Run("avec des articles", func(t *testing.T) {
		items := []OrderItem{
			{ProductID: 1, ProductName: "Clavier", Quantity: 2, PriceHT: 4999},
		}

		order, err := NewOrderForUser(10, items, "CMD-TEST00")
		if err != nil {
			t.Fatalf("NewOrderForUser() erreur inattendue : %v", err)
		}
		if order.UserID != 10 || order.Reference != "CMD-TEST00" || order.Status != OrderStatusPending {
			t.Errorf("commande = %+v, champs de base inattendus", order)
		}
		// Pas de panier d'origine pour une commande créée par l'admin.
		if order.CartID != nil {
			t.Errorf("CartID = %v, attendu nil", order.CartID)
		}
		if len(order.Items) != 1 || order.Items[0].ProductID != 1 {
			t.Errorf("articles = %+v, inattendus", order.Items)
		}
	})
}

func TestCancel(t *testing.T) {
	tests := []struct {
		name       string
		from       OrderStatus
		reason     string
		wantErr    error
		wantStatus OrderStatus
		wantReason string
	}{
		{"pending", OrderStatusPending, "Je me suis trompé", nil, OrderStatusCancelled, "Je me suis trompé"},
		{"paid", OrderStatusPaid, "Je me suis trompé", nil, OrderStatusCancelled, "Je me suis trompé"},
		{"espaces autour retirés", OrderStatusPending, "  Trop cher  ", nil, OrderStatusCancelled, "Trop cher"},
		{"shipping", OrderStatusShipping, "Trop tard", ErrOrderNotCancellable, OrderStatusShipping, ""},
		{"delivered", OrderStatusDelivered, "Trop tard", ErrOrderNotCancellable, OrderStatusDelivered, ""},
		{"déjà annulée", OrderStatusCancelled, "Encore", ErrOrderNotCancellable, OrderStatusCancelled, ""},
		{"raison vide", OrderStatusPending, "", ErrCancelReasonRequired, OrderStatusPending, ""},
		{"raison faite d'espaces", OrderStatusPending, "   ", ErrCancelReasonRequired, OrderStatusPending, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := Order{Status: tt.from}

			err := order.Cancel(tt.reason)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Cancel() = %v, attendu %v", err, tt.wantErr)
			}
			if order.Status != tt.wantStatus {
				t.Errorf("statut = %s, attendu %s", order.Status, tt.wantStatus)
			}
			if tt.wantReason == "" {
				// En cas d'erreur, aucune raison ne doit être enregistrée.
				if order.CancelReason != nil {
					t.Errorf("raison = %q, attendu aucune", *order.CancelReason)
				}
			} else if order.CancelReason == nil || *order.CancelReason != tt.wantReason {
				t.Errorf("raison = %v, attendu %q", order.CancelReason, tt.wantReason)
			}
		})
	}
}

func TestNewOrderFromCart(t *testing.T) {
	t.Run("panier vide", func(t *testing.T) {
		_, err := NewOrderFromCart(Cart{ID: 1, UserID: 10}, "CMD-TEST00")
		if !errors.Is(err, ErrCartEmpty) {
			t.Errorf("NewOrderFromCart() = %v, attendu %v", err, ErrCartEmpty)
		}
	})

	t.Run("panier avec des lignes", func(t *testing.T) {
		cart := Cart{
			ID:     1,
			UserID: 10,
			Items: []CartItem{
				{ProductID: 1, ProductName: "Clavier", Quantity: 2, PriceHT: 4999, PriceTTC: 5999},
				{ProductID: 2, ProductName: "Souris", Quantity: 1, PriceHT: 1999, PriceTTC: 2399},
			},
		}

		order, err := NewOrderFromCart(cart, "CMD-TEST00")
		if err != nil {
			t.Fatalf("NewOrderFromCart() erreur inattendue : %v", err)
		}
		if order.Reference != "CMD-TEST00" || order.UserID != 10 || order.Status != OrderStatusPending {
			t.Errorf("commande = %+v, champs de base inattendus", order)
		}
		if order.CartID == nil || *order.CartID != 1 {
			t.Errorf("CartID = %v, attendu 1", order.CartID)
		}
		if len(order.Items) != 2 {
			t.Fatalf("attendu 2 lignes, obtenu %d", len(order.Items))
		}
		// Le prix TTC n'est jamais recopié : il est recalculé par la base
		// (colonne générée) à partir du prix HT figé ici.
		first := order.Items[0]
		if first.ProductID != 1 || first.ProductName != "Clavier" || first.Quantity != 2 || first.PriceHT != 4999 {
			t.Errorf("première ligne = %+v, inattendue", first)
		}
	})
}

func TestTotalTTC(t *testing.T) {
	tests := []struct {
		name  string
		items []OrderItem
		want  int
	}{
		{"aucun article", nil, 0},
		{
			"un article, quantité 1",
			[]OrderItem{{PriceTTC: 1000, Quantity: 1}},
			1000,
		},
		{

			"un article, quantité > 1",
			[]OrderItem{{PriceTTC: 1000, Quantity: 3}},
			3000,
		},
		{
			"plusieurs articles",
			[]OrderItem{
				{PriceTTC: 1000, Quantity: 2}, // 2000
				{PriceTTC: 500, Quantity: 1},  // 500
				{PriceTTC: 250, Quantity: 4},  // 1000
			},
			3500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := Order{Items: tt.items}

			if got := order.TotalTTC(); got != tt.want {
				t.Errorf("TotalTTC() = %d, attendu %d", got, tt.want)
			}
		})
	}
}
