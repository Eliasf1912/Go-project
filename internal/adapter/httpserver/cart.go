package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Eliasf1912/go-project/internal/domain"
	"github.com/Eliasf1912/go-project/internal/dto"
	"github.com/Eliasf1912/go-project/internal/usecase"
)

type CartHandler struct {
	carts *usecase.CartService
}

func NewCartHandler(carts *usecase.CartService) *CartHandler {
	return &CartHandler{carts: carts}
}

// GetMine renvoie le panier de l'utilisateur authentifié et son total TTC.
// Protégée par auth.RequireAuth dans router.go.
func (h *CartHandler) GetMine(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeDomainError(w, domain.ErrUnauthorized)
		return
	}

	cart, err := h.carts.GetMine(r.Context(), user.ID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toCartResponse(cart))
}

// AddItem ajoute un produit (et sa quantité) au panier de l'utilisateur
// authentifié. Protégée par auth.RequireAuth dans router.go.
func (h *CartHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeDomainError(w, domain.ErrUnauthorized)
		return
	}

	var req dto.AddCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalide")
		return
	}

	cart, err := h.carts.AddItem(r.Context(), user.ID, req.ProductID, req.Quantity)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toCartResponse(cart))
}

// UpdateItem fixe la quantité d'un produit du panier de l'utilisateur
// authentifié ; une quantité de 0 supprime la ligne. Protégée par
// auth.RequireAuth dans router.go.
func (h *CartHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeDomainError(w, domain.ErrUnauthorized)
		return
	}

	productID, err := strconv.Atoi(r.PathValue("productID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant de produit invalide")
		return
	}

	var req dto.UpdateCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalide")
		return
	}

	cart, err := h.carts.UpdateItem(r.Context(), user.ID, productID, req.Quantity)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toCartResponse(cart))
}

// RemoveItem supprime un produit du panier de l'utilisateur authentifié.
// Protégée par auth.RequireAuth dans router.go.
func (h *CartHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeDomainError(w, domain.ErrUnauthorized)
		return
	}

	productID, err := strconv.Atoi(r.PathValue("productID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant de produit invalide")
		return
	}

	cart, err := h.carts.RemoveItem(r.Context(), user.ID, productID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toCartResponse(cart))
}

func toCartResponse(cart domain.Cart) dto.CartResponse {
	items := make([]dto.CartItemResponse, 0, len(cart.Items))
	for _, item := range cart.Items {
		items = append(items, dto.CartItemResponse{
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			PriceHT:     item.PriceHT,
			PriceTTC:    item.PriceTTC,
		})
	}
	return dto.CartResponse{
		Reference: cart.Reference,
		Items:     items,
		TotalTTC:  cart.TotalTTC(),
	}
}
