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
	// order.CartID à nil (commande créée par un admin) : aucun panier n'est
	// touché.
	Create(ctx context.Context, order domain.Order) (domain.Order, error)
	// ListAll renvoie toutes les commandes, tous clients confondus. Réservé
	// à l'admin.
	ListAll(ctx context.Context) ([]domain.Order, error)
	// Delete supprime définitivement une commande (et ses lignes). Réservé
	// à l'admin.
	Delete(ctx context.Context, id int) error
}

// OrderLine est une ligne (produit, quantité) choisie par l'admin pour
// créer une commande au nom d'un client : CreateForUser la résout en
// OrderItem (nom et prix HT recopiés depuis le produit).
type OrderLine struct {
	ProductID int
	Quantity  int
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

// ListAll renvoie toutes les commandes, tous clients confondus. Réservé à
// l'admin.
func (s *OrderService) ListAll(ctx context.Context) ([]domain.Order, error) {
	return s.orders.ListAll(ctx)
}

// CreateForUser crée une commande "pending" pour un client donné, à partir
// de lignes (produit, quantité) choisies par l'admin. Chaque produit doit
// être actif ; son nom et son prix HT sont recopiés et figés, comme pour
// une commande issue d'un panier. Réservé à l'admin.
func (s *OrderService) CreateForUser(ctx context.Context, userID int, lines []OrderLine) (domain.Order, error) {
	items := make([]domain.OrderItem, 0, len(lines))
	for _, line := range lines {
		if line.Quantity < 1 {
			return domain.Order{}, domain.ErrInvalidQuantity
		}

		product, err := s.products.FindActiveByID(ctx, line.ProductID)
		if err != nil {
			return domain.Order{}, err
		}

		items = append(items, domain.OrderItem{
			ProductID:   product.ID,
			ProductName: product.Name,
			Quantity:    line.Quantity,
			PriceHT:     product.PriceHT,
		})
	}

	order, err := domain.NewOrderForUser(userID, items, ids.NewOrderRef())
	if err != nil {
		return domain.Order{}, err
	}

	return s.orders.Create(ctx, order)
}

// Ship fait passer une commande payée en "en cours de livraison". Réservé
// à l'admin.
func (s *OrderService) Ship(ctx context.Context, orderID int) (domain.Order, error) {
	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return domain.Order{}, err
	}

	from := order.Status
	if err := order.MarkAsShipping(); err != nil {
		return domain.Order{}, err
	}

	if err := s.orders.UpdateStatus(ctx, order.ID, from, order.Status, nil); err != nil {
		return domain.Order{}, err
	}

	return order, nil
}

// Deliver fait passer une commande en livraison en "livrée". Réservé à
// l'admin.
func (s *OrderService) Deliver(ctx context.Context, orderID int) (domain.Order, error) {
	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return domain.Order{}, err
	}

	from := order.Status
	if err := order.MarkAsDelivered(); err != nil {
		return domain.Order{}, err
	}

	if err := s.orders.UpdateStatus(ctx, order.ID, from, order.Status, nil); err != nil {
		return domain.Order{}, err
	}

	return order, nil
}

// CancelAsAdmin annule n'importe quelle commande, avec une raison
// obligatoire. Contrairement à Cancel, aucune vérification de
// propriétaire : l'admin peut annuler la commande de n'importe quel client.
func (s *OrderService) CancelAsAdmin(ctx context.Context, orderID int, reason string) (domain.Order, error) {
	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return domain.Order{}, err
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

// Delete supprime définitivement une commande, quel que soit son client ou
// son statut. Réservé à l'admin.
func (s *OrderService) Delete(ctx context.Context, orderID int) error {
	return s.orders.Delete(ctx, orderID)
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
