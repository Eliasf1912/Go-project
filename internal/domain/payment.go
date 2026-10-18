package domain

import (
	"strconv"
	"strings"
	"time"
)

type Card struct {
	Number string
	Expiry string // format MM/AA
	CVC    string
}

func ValidateCard(card Card, now time.Time) error {
	number := strings.ReplaceAll(card.Number, " ", "")
	if len(number) != 16 || !isDigits(number) || !luhnValid(number) {
		return ErrInvalidCardNumber
	}

	month, year, ok := parseExpiry(card.Expiry)
	if !ok {
		return ErrInvalidCardExpiry
	}
	// La carte reste valable jusqu'au dernier jour de son mois d'expiration
	if year < now.Year() || (year == now.Year() && month < int(now.Month())) {
		return ErrCardExpired
	}

	if len(card.CVC) != 3 || !isDigits(card.CVC) {
		return ErrInvalidCVC
	}

	return nil
}

// parseExpiry lit une date "MM/AA" et renvoie le mois et l'année sur 4 chiffres.
func parseExpiry(expiry string) (month, year int, ok bool) {
	mm, yy, found := strings.Cut(expiry, "/")
	if !found || len(mm) != 2 || len(yy) != 2 || !isDigits(mm) || !isDigits(yy) {
		return 0, 0, false
	}
	month, _ = strconv.Atoi(mm)
	year, _ = strconv.Atoi(yy)
	if month < 1 || month > 12 {
		return 0, 0, false
	}
	return month, 2000 + year, true
}

// luhnValid applique l'algorithme de Luhn : en partant de la droite, on
// double un chiffre sur deux (en retirant 9 si le résultat dépasse 9) ;
// la somme de tous les chiffres doit être un multiple de 10.
func luhnValid(number string) bool {
	sum := 0
	for i := len(number) - 1; i >= 0; i-- {
		digit := int(number[i] - '0')
		if (len(number)-i)%2 == 0 {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}
	return sum%10 == 0
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return s != ""
}
