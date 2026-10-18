package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Eliasf1912/go-project/internal/domain"
	"github.com/Eliasf1912/go-project/internal/usecase"
)

// vérifie à la compilation que OrderRepository implémente bien l'interface
var _ usecase.OrderRepository = (*OrderRepository)(nil)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// ListByUserID renvoie les commandes d'un utilisateur, avec leurs lignes,
// les plus récentes en premier
func (r *OrderRepository) ListByUserID(ctx context.Context, userID int) ([]domain.Order, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, reference, user_id, cart_id, status, cancel_reason, created_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("recherche des commandes : %w", err)
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var (
			order        domain.Order
			cartID       sql.NullInt64
			cancelReason sql.NullString
		)
		if err := rows.Scan(
			&order.ID, &order.Reference, &order.UserID, &cartID,
			&order.Status, &cancelReason, &order.CreatedAt, &order.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("lecture d'une commande : %w", err)
		}
		if cartID.Valid {
			id := int(cartID.Int64)
			order.CartID = &id
		}
		order.CancelReason = nullStringToPtr(cancelReason)
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recherche des commandes : %w", err)
	}

	for i := range orders {
		items, err := r.listItems(ctx, orders[i].ID)
		if err != nil {
			return nil, err
		}
		orders[i].Items = items
	}

	return orders, nil
}

// listItems renvoie les lignes d'une commande (table order_items)
func (r *OrderRepository) listItems(ctx context.Context, orderID int) ([]domain.OrderItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, product_id, product_name, quantity, price_ht, price_ttc
		FROM order_items
		WHERE order_id = $1
		ORDER BY id`, orderID)
	if err != nil {
		return nil, fmt.Errorf("recherche des lignes de la commande %d : %w", orderID, err)
	}
	defer rows.Close()

	var items []domain.OrderItem
	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(
			&item.ID, &item.ProductID, &item.ProductName,
			&item.Quantity, &item.PriceHT, &item.PriceTTC,
		); err != nil {
			return nil, fmt.Errorf("lecture d'une ligne de la commande %d : %w", orderID, err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recherche des lignes de la commande %d : %w", orderID, err)
	}

	return items, nil
}
