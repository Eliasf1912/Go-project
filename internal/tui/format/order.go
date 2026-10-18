// Package format rassemble les petites fonctions d'affichage partagées
// entre les TUI client et admin (statuts en français, prix en euros) :
// une seule fonction de traduction, comme demandé par le README, plutôt
// qu'une copie dans chaque écran.
package format

import "fmt"

// OrderStatusLabel traduit le statut d'une commande (valeur JSON de
// l'API, en anglais) en libellé français pour l'affichage.
func OrderStatusLabel(status string) string {
	switch status {
	case "pending":
		return "En attente"
	case "paid":
		return "Payée"
	case "shipping":
		return "En cours de livraison"
	case "delivered":
		return "Livrée"
	case "cancelled":
		return "Annulée"
	default:
		return status
	}
}

// Euros affiche un prix en centimes avec deux chiffres après la virgule,
// ex: 1999 -> "19,99 €".
func Euros(cents int) string {
	return fmt.Sprintf("%d,%02d €", cents/100, cents%100)
}
