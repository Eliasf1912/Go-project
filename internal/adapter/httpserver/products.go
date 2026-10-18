package httpserver

import (
	"net/http"

	"github.com/Eliasf1912/go-project/internal/domain"
	"github.com/Eliasf1912/go-project/internal/dto"
	"github.com/Eliasf1912/go-project/internal/usecase"
)

type ProductHandler struct {
	products *usecase.ProductService
}

func NewProductHandler(products *usecase.ProductService) *ProductHandler {
	return &ProductHandler{products: products}
}

// List renvoie les produits actifs du catalogue.
// Protégée par auth.RequireAuth dans router.go : accessible à tout utilisateur connecté.
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	products, err := h.products.List(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toProductResponses(products))
}

func toProductResponses(products []domain.Product) []dto.ProductResponse {
	responses := make([]dto.ProductResponse, 0, len(products))
	for _, p := range products {
		responses = append(responses, dto.ProductResponse{
			ID:          p.ID,
			Reference:   p.Reference,
			Name:        p.Name,
			Description: p.Description,
			PriceHT:     p.PriceHT,
			PriceTTC:    p.PriceTTC,
			Category: dto.CategoryResponse{
				ID:   p.Category.ID,
				Name: p.Category.Name,
			},
		})
	}
	return responses
}
