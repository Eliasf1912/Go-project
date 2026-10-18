package httpserver

import (
	"fmt"
	"net/http"
	"strconv"

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

// Search renvoie les produits actifs du catalogue correspondant aux
// paramètres de requête, tous optionnels : name, description, category_id,
// min_price_ht, max_price_ht, min_price_ttc, max_price_ttc. Sans aucun
// paramètre, renvoie tous les produits actifs (comportement de l'ancien
// List/ListActive).
// Protégée par auth.RequireAuth dans router.go : accessible à tout utilisateur connecté.
func (h *ProductHandler) Search(w http.ResponseWriter, r *http.Request) {
	filter, err := parseProductFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	products, err := h.products.Search(r.Context(), filter)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toProductResponses(products))
}

func parseProductFilter(r *http.Request) (usecase.ProductFilter, error) {
	query := r.URL.Query()

	filter := usecase.ProductFilter{
		Name:        query.Get("name"),
		Description: query.Get("description"),
	}

	intParams := []struct {
		name string
		dest **int
	}{
		{"category_id", &filter.CategoryID},
		{"min_price_ht", &filter.MinPriceHT},
		{"max_price_ht", &filter.MaxPriceHT},
		{"min_price_ttc", &filter.MinPriceTTC},
		{"max_price_ttc", &filter.MaxPriceTTC},
	}

	for _, p := range intParams {
		raw := query.Get(p.name)
		if raw == "" {
			continue
		}
		value, err := strconv.Atoi(raw)
		if err != nil {
			return usecase.ProductFilter{}, fmt.Errorf("le paramètre %q doit être un nombre entier", p.name)
		}
		*p.dest = &value
	}

	return filter, nil
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
