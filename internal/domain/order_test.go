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
