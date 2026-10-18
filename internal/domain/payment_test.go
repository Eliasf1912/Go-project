package domain

import (
	"errors"
	"testing"
	"time"
)

func TestValidateCard(t *testing.T) {
	now := time.Date(2026, time.October, 18, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		card Card
		want error
	}{
		{"carte valide", Card{"4242424242424242", "12/27", "123"}, nil},
		{"espaces acceptés", Card{"4242 4242 4242 4242", "12/27", "123"}, nil},
		{"expire ce mois-ci", Card{"4242424242424242", "10/26", "123"}, nil},
		{"trop court", Card{"424242424242", "12/27", "123"}, ErrInvalidCardNumber},
		{"lettres", Card{"4242abcd42424242", "12/27", "123"}, ErrInvalidCardNumber},
		{"clé de Luhn fausse", Card{"4242424242424241", "12/27", "123"}, ErrInvalidCardNumber},
		{"format de date", Card{"4242424242424242", "2027-12", "123"}, ErrInvalidCardExpiry},
		{"mois 13", Card{"4242424242424242", "13/27", "123"}, ErrInvalidCardExpiry},
		{"expirée le mois dernier", Card{"4242424242424242", "09/26", "123"}, ErrCardExpired},
		{"expirée l'an dernier", Card{"4242424242424242", "12/25", "123"}, ErrCardExpired},
		{"CVC trop court", Card{"4242424242424242", "12/27", "12"}, ErrInvalidCVC},
		{"CVC avec lettre", Card{"4242424242424242", "12/27", "12a"}, ErrInvalidCVC},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateCard(tt.card, now); !errors.Is(err, tt.want) {
				t.Errorf("ValidateCard() = %v, attendu %v", err, tt.want)
			}
		})
	}
}
