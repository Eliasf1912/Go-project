package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Eliasf1912/go-project/internal/domain"
	"github.com/Eliasf1912/go-project/internal/dto"
	"github.com/Eliasf1912/go-project/internal/usecase"
)

type OrderHandler struct {
	orders *usecase.OrderService
}

func NewOrderHandler(orders *usecase.OrderService) *OrderHandler {
	return &OrderHandler{orders: orders}
}

// ListMine renvoie les commandes de l'utilisateur authentifié
func (h *OrderHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeDomainError(w, domain.ErrUnauthorized)
		return
	}

	orders, err := h.orders.ListMine(r.Context(), user.ID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toOrderResponses(orders))
}

// Pay fait passer une commande en "paid"
func (h *OrderHandler) Pay(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeDomainError(w, domain.ErrUnauthorized)
		return
	}

	orderID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant de commande invalide")
		return
	}

	var req dto.PayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalide")
		return
	}

	card := domain.Card{Number: req.CardNumber, Expiry: req.Expiry, CVC: req.CVC}
	order, err := h.orders.Pay(r.Context(), user.ID, orderID, card)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toOrderResponse(order))
}

// Cancel fait passer une commande en "cancelled", avec une raison
func (h *OrderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeDomainError(w, domain.ErrUnauthorized)
		return
	}

	orderID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant de commande invalide")
		return
	}

	var req dto.CancelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalide")
		return
	}

	order, err := h.orders.Cancel(r.Context(), user.ID, orderID, req.Reason)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toOrderResponse(order))
}

func toOrderResponses(orders []domain.Order) []dto.OrderResponse {
	responses := make([]dto.OrderResponse, 0, len(orders))
	for _, order := range orders {
		responses = append(responses, toOrderResponse(order))
	}
	return responses
}

func toOrderResponse(order domain.Order) dto.OrderResponse {
	items := make([]dto.OrderItemResponse, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, dto.OrderItemResponse{
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			PriceHT:     item.PriceHT,
			PriceTTC:    item.PriceTTC,
		})
	}
	return dto.OrderResponse{
		ID:           order.ID,
		Reference:    order.Reference,
		Status:       string(order.Status),
		CancelReason: order.CancelReason,
		TotalTTC:     order.TotalTTC(),
		CreatedAt:    order.CreatedAt,
		Items:        items,
	}
}
