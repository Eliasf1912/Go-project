package usecase

import (
	"context"
	"time"

	"github.com/Eliasf1912/go-project/internal/domain"
)

// OrderRepository est le port implémenté par internal/adapter/postgres
type OrderRepository interface {
	ListByUserID(ctx context.Context, userID int) ([]domain.Order, error)
	FindByID(ctx context.Context, id int) (domain.Order, error)
	// UpdateStatus fait passer une commande du statut from au statut to (et
	// enregistre sa raison d'annulation, nil si non applicable). Renvoie
	// ErrOrderStatusChanged si la commande n'est plus au statut from : une
	// autre requête l'a modifiée entre-temps. Réutilisé par Pay et, plus
	// tard, par l'annulation.
	UpdateStatus(ctx context.Context, id int, from, to domain.OrderStatus, cancelReason *string) error
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

// Pay fait passer une commande en "paid". userID doit être le propriétaire
// de la commande : sinon on renvoie ErrNotFound, comme pour une commande
// inexistante, pour ne pas révéler les commandes des autres.
// La carte est seulement validée : aucune donnée bancaire n'est conservée.
func (s *OrderService) Pay(ctx context.Context, userID, orderID int, card domain.Card) (domain.Order, error) {
	if err := domain.ValidateCard(card, time.Now()); err != nil {
		return domain.Order{}, err
	}

	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return domain.Order{}, err
	}

	if order.UserID != userID {
		return domain.Order{}, domain.ErrNotFound
	}

	from := order.Status
	if err := order.MarkAsPaid(); err != nil {
		return domain.Order{}, err
	}

	if err := s.orders.UpdateStatus(ctx, order.ID, from, order.Status, nil); err != nil {
		return domain.Order{}, err
	}

	return order, nil
}

// Cancel annule une commande (statut "cancelled"), avec une raison
// obligatoire. userID doit être le propriétaire de la commande, comme
// pour Pay.
func (s *OrderService) Cancel(ctx context.Context, userID, orderID int, reason string) (domain.Order, error) {
	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return domain.Order{}, err
	}

	if order.UserID != userID {
		return domain.Order{}, domain.ErrNotFound
	}

	from := order.Status
	if err := order.Cancel(reason); err != nil {
		return domain.Order{}, err
	}

	if err := s.orders.UpdateStatus(ctx, order.ID, from, order.Status, order.CancelReason); err != nil {
		return domain.Order{}, err
	}

	return order, nil
}
