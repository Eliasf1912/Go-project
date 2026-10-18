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

// Checkout valide le panier de l'utilisateur connecté
func (h *OrderHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeDomainError(w, domain.ErrUnauthorized)
		return
	}

	order, err := h.orders.Checkout(r.Context(), user.ID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toOrderResponse(order))
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

// AdminList renvoie toutes les commandes, tous clients confondus. Réservé
// à l'admin.
func (h *OrderHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	orders, err := h.orders.ListAll(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toOrderResponses(orders))
}

// AdminCreate crée une commande pour un client donné (user_id + liste de
// produits/quantités). Réservé à l'admin.
func (h *OrderHandler) AdminCreate(w http.ResponseWriter, r *http.Request) {
	var req dto.AdminCreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalide")
		return
	}

	lines := make([]usecase.OrderLine, 0, len(req.Items))
	for _, item := range req.Items {
		lines = append(lines, usecase.OrderLine{ProductID: item.ProductID, Quantity: item.Quantity})
	}

	order, err := h.orders.CreateForUser(r.Context(), req.UserID, lines)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toOrderResponse(order))
}

// AdminShip fait passer une commande payée en "en cours de livraison".
// Réservé à l'admin.
func (h *OrderHandler) AdminShip(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant de commande invalide")
		return
	}

	order, err := h.orders.Ship(r.Context(), orderID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toOrderResponse(order))
}

// AdminDeliver fait passer une commande en livraison en "livrée". Réservé
// à l'admin.
func (h *OrderHandler) AdminDeliver(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant de commande invalide")
		return
	}

	order, err := h.orders.Deliver(r.Context(), orderID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toOrderResponse(order))
}

// AdminCancel annule n'importe quelle commande, avec une raison. Réservé à
// l'admin : contrairement à Cancel, aucune vérification de propriétaire.
func (h *OrderHandler) AdminCancel(w http.ResponseWriter, r *http.Request) {
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

	order, err := h.orders.CancelAsAdmin(r.Context(), orderID, req.Reason)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toOrderResponse(order))
}

// AdminDelete supprime définitivement une commande. Réservé à l'admin.
func (h *OrderHandler) AdminDelete(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant de commande invalide")
		return
	}

	if err := h.orders.Delete(r.Context(), orderID); err != nil {
		writeDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
		UserID:       order.UserID,
		Status:       string(order.Status),
		CancelReason: order.CancelReason,
		TotalTTC:     order.TotalTTC(),
		CreatedAt:    order.CreatedAt,
		Items:        items,
	}
}
