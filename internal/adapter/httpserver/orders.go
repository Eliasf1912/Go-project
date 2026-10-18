package httpserver

import (
	"net/http"

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
// Protégée par auth.RequireAuth dans router.go
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

func toOrderResponses(orders []domain.Order) []dto.OrderResponse {
	responses := make([]dto.OrderResponse, 0, len(orders))
	for _, order := range orders {
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
		responses = append(responses, dto.OrderResponse{
			ID:           order.ID,
			Reference:    order.Reference,
			Status:       string(order.Status),
			CancelReason: order.CancelReason,
			CreatedAt:    order.CreatedAt,
			Items:        items,
		})
	}
	return responses
}
