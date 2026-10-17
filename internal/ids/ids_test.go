package ids

import (
	"regexp"
	"testing"
)

func TestFormat(t *testing.T) {
	tests := []struct {
		name    string
		gen     func() string
		pattern string
	}{
		{"produit", NewProductRef, `^PDT-[A-Z0-9]{6}$`},
		{"panier", NewCartRef, `^BSK-[A-Z0-9]{6}$`},
		{"commande", NewOrderRef, `^CMD-[A-Z0-9]{6}$`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re := regexp.MustCompile(tt.pattern)
			for range 1000 {
				if ref := tt.gen(); !re.MatchString(ref) {
					t.Fatalf("format invalide : %s", ref)
				}
			}
		})
	}
}

// Test probabiliste : sur 1 000 tirages, un vrai doublon a environ 0,02 % de chances d'arriver.
func TestUnique(t *testing.T) {
	seen := make(map[string]bool)
	for range 1000 {
		ref := NewOrderRef()
		if seen[ref] {
			t.Fatalf("doublon : %s", ref)
		}
		seen[ref] = true
	}
}
