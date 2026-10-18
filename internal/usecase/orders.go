package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Eliasf1912/go-project/internal/domain"
	"github.com/Eliasf1912/go-project/internal/ids"
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
	// Create enregistre une commande "pending" issue d'un panier (elle et
	// ses lignes), puis vide ce panier (order.CartID) — le tout dans une
	// seule transaction côté base : soit tout est écrit, soit rien ne l'est.
	// Renvoie la commande complète, avec les prix TTC calculés par la base.
	Create(ctx context.Context, order domain.Order) (domain.Order, error)
}

type OrderService struct {
	orders   OrderRepository
	carts    CartRepository
	products ProductLookup
}

func NewOrderService(orders OrderRepository, carts CartRepository, products ProductLookup) *OrderService {
	return &OrderService{orders: orders, carts: carts, products: products}
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

// Checkout valide le panier de l'utilisateur : ses lignes deviennent une
// commande "pending", puis le panier est vidé. Refuse un panier vide ou
// absent (ErrCartEmpty), et refuse si l'un des produits a été archivé
// depuis son ajout au panier (ErrCartHasInactiveProduct) : l'utilisateur
// doit alors le retirer lui-même avant de réessayer.
func (s *OrderService) Checkout(ctx context.Context, userID int) (domain.Order, error) {
	cart, err := s.carts.FindByUserID(ctx, userID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.Order{}, domain.ErrCartEmpty
	}
	if err != nil {
		return domain.Order{}, err
	}

	for _, item := range cart.Items {
		if _, err := s.products.FindActiveByID(ctx, item.ProductID); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.Order{}, fmt.Errorf("%w : %s", domain.ErrCartHasInactiveProduct, item.ProductName)
			}
			return domain.Order{}, err
		}
	}

	order, err := domain.NewOrderFromCart(cart, ids.NewOrderRef())
	if err != nil {
		return domain.Order{}, err
	}

	return s.orders.Create(ctx, order)
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
