package usecase

import (
	"context"

	"github.com/Eliasf1912/go-project/internal/domain"
)

// OrderRepository est le port implémenté par internal/adapter/postgres
type OrderRepository interface {
	ListByUserID(ctx context.Context, userID int) ([]domain.Order, error)
}

type OrderService struct {
	orders OrderRepository
}

func NewOrderService(orders OrderRepository) *OrderService {
	return &OrderService{orders: orders}
}

// ListMine renvoie les commandes d'un utilisateur, les plus récentes en premier.
func (s *OrderService) ListMine(ctx context.Context, userID int) ([]domain.Order, error) {
	return s.orders.ListByUserID(ctx, userID)
}
