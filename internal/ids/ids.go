package ids

import (
	"crypto/rand"
)

// alphabet des caractères utilisables pour la partie aléatoire
const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Au-delà de cette valeur, l'octet est rejeté pour éviter un tirage biaisé.
const maxValidByte = 251

// generate tire "length" caractères aléatoires dans alphabet.
func generate(length int) string {
	code := make([]byte, length)
	single := make([]byte, 1)

	for i := 0; i < length; i++ {
		for {
			rand.Read(single) // tire un octet aléatoire
			if single[0] <= maxValidByte {
				code[i] = alphabet[int(single[0])%len(alphabet)] // convertit l'octet en caractère
				break
			}
			// octet rejeté (252-255), on retire
		}
	}

	return string(code)
}

// NewProductID génère un identifiant métier de produit, ex: PDT-7D2K8N
func NewProductID() string {
	return "PDT-" + generate(6)
}

// NewCartID génère un identifiant métier de panier, ex: BSK-1KH8E7
func NewCartID() string {
	return "BSK-" + generate(6)
}

// NewOrderID génère un identifiant métier de commande, ex: CMD-1F2S8B
func NewOrderID() string {
	return "CMD-" + generate(6)
}
