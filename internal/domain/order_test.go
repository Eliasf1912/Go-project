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
