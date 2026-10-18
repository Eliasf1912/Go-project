package domain

import "time"

// Product représente un produit du catalogue.
type Product struct {
	ID          int
	Reference   string
	Name        string
	Description string
	PriceHT     int // centimes
	PriceTTC    int // centimes, colonne générée par la base (TVA fixe 20 %)
	Category    Category
	Active      bool // false = archivé (déjà commandé, non supprimable)
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
