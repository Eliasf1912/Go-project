package domain

import "testing"

func TestCart_TotalTTC(t *testing.T) {
	tests := []struct {
		name  string
		items []CartItem
		want  int
	}{
		{"panier vide", nil, 0},
		{"un article", []CartItem{{PriceTTC: 1999, Quantity: 2}}, 3998},
		{
			"plusieurs articles",
			[]CartItem{
				{PriceTTC: 1999, Quantity: 2}, // 3998
				{PriceTTC: 833, Quantity: 1},  // 833
			},
			4831,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cart := Cart{Items: tt.items}
			if got := cart.TotalTTC(); got != tt.want {
				t.Errorf("TotalTTC() = %d, attendu %d", got, tt.want)
			}
		})
	}
}
