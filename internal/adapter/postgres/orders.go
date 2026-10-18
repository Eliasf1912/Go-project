package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Eliasf1912/go-project/internal/domain"
	"github.com/Eliasf1912/go-project/internal/usecase"
	"github.com/jackc/pgx/v5/pgconn"
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
	return r.listOrders(ctx, "WHERE user_id = $1", userID)
}

// ListAll renvoie toutes les commandes, tous clients confondus, les plus
// récentes en premier. Réservé à l'admin.
func (r *OrderRepository) ListAll(ctx context.Context) ([]domain.Order, error) {
	return r.listOrders(ctx, "")
}

// listOrders factorise la lecture commune à ListByUserID et ListAll : même
// SELECT et même scan, seule la clause WHERE change (vide pour tout
// renvoyer).
func (r *OrderRepository) listOrders(ctx context.Context, where string, args ...any) ([]domain.Order, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, reference, user_id, cart_id, status, cancel_reason, created_at, updated_at
		FROM orders
		`+where+`
		ORDER BY created_at DESC`, args...)
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

// Delete supprime une commande ; ses lignes partent par cascade
// (order_items.order_id est ON DELETE CASCADE). Réservé à l'admin.
func (r *OrderRepository) Delete(ctx context.Context, id int) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM orders WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("suppression de la commande %d : %w", id, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("suppression de la commande %d : %w", id, err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// FindByID renvoie une commande (avec ses lignes) par son id.
func (r *OrderRepository) FindByID(ctx context.Context, id int) (domain.Order, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, reference, user_id, cart_id, status, cancel_reason, created_at, updated_at
		FROM orders
		WHERE id = $1`, id)

	var (
		order        domain.Order
		cartID       sql.NullInt64
		cancelReason sql.NullString
	)
	err := row.Scan(
		&order.ID, &order.Reference, &order.UserID, &cartID,
		&order.Status, &cancelReason, &order.CreatedAt, &order.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Order{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Order{}, fmt.Errorf("recherche de la commande %d : %w", id, err)
	}
	if cartID.Valid {
		cartIDValue := int(cartID.Int64)
		order.CartID = &cartIDValue
	}
	order.CancelReason = nullStringToPtr(cancelReason)

	items, err := r.listItems(ctx, order.ID)
	if err != nil {
		return domain.Order{}, err
	}
	order.Items = items

	return order, nil
}

// UpdateStatus fait passer une commande du statut from au statut to et, si
// fournie, enregistre sa raison d'annulation. La condition "status = from"
// est vérifiée par la base dans la même requête : si deux requêtes arrivent
// en même temps, une seule peut réussir.
func (r *OrderRepository) UpdateStatus(ctx context.Context, id int, from, to domain.OrderStatus, cancelReason *string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE orders SET status = $1, cancel_reason = $2, updated_at = now()
		WHERE id = $3 AND status = $4`, to, stringPtrToNull(cancelReason), id, from)
	if err != nil {
		return fmt.Errorf("mise à jour de la commande %d : %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mise à jour de la commande %d : %w", id, err)
	}
	if rows == 0 {
		return domain.ErrOrderStatusChanged
	}
	return nil
}

// Create vide le panier d'origine (order.CartID), puis enregistre une
// commande "pending" et ses lignes — dans une seule transaction : si une
// étape échoue, rien n'est écrit. Les prix TTC (colonne générée) ne sont
// connus qu'après l'INSERT : on les relit avec RETURNING.
//
// Le panier est vidé en premier pour empêcher deux commandes à partir du
// même panier : si deux validations arrivent en même temps, PostgreSQL
// fait attendre le second DELETE jusqu'à la fin de la première
// transaction ; il ne trouve alors plus aucune ligne et on renvoie
// ErrCartEmpty sans rien créer.
func (r *OrderRepository) Create(ctx context.Context, order domain.Order) (domain.Order, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Order{}, fmt.Errorf("création de la commande : %w", err)
	}
	defer tx.Rollback() // sans effet si Commit a réussi

	if order.CartID != nil {
		result, err := tx.ExecContext(ctx, `
			DELETE FROM cart_items WHERE cart_id = $1`, *order.CartID)
		if err != nil {
			return domain.Order{}, fmt.Errorf("vidage du panier %d : %w", *order.CartID, err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return domain.Order{}, fmt.Errorf("vidage du panier %d : %w", *order.CartID, err)
		}
		if rows == 0 {
			return domain.Order{}, domain.ErrCartEmpty
		}
	}

	row := tx.QueryRowContext(ctx, `
		INSERT INTO orders (reference, user_id, cart_id, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`,
		order.Reference, order.UserID, intPtrToNull(order.CartID), order.Status)
	if err := row.Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt); err != nil {
		if isForeignKeyViolation(err) {
			return domain.Order{}, domain.ErrNotFound // client inexistant
		}
		return domain.Order{}, fmt.Errorf("création de la commande : %w", err)
	}

	for i, item := range order.Items {
		row := tx.QueryRowContext(ctx, `
			INSERT INTO order_items (order_id, product_id, product_name, quantity, price_ht)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, price_ttc`,
			order.ID, item.ProductID, item.ProductName, item.Quantity, item.PriceHT)
		if err := row.Scan(&item.ID, &item.PriceTTC); err != nil {
			if isForeignKeyViolation(err) {
				return domain.Order{}, domain.ErrNotFound // produit supprimé entre-temps
			}
			return domain.Order{}, fmt.Errorf("création d'une ligne de la commande : %w", err)
		}
		order.Items[i] = item
	}

	if err := tx.Commit(); err != nil {
		return domain.Order{}, fmt.Errorf("création de la commande : %w", err)
	}

	return order, nil
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

// isForeignKeyViolation indique si PostgreSQL a refusé l'écriture parce
// qu'elle référence une ligne qui n'existe pas (code 23503), par exemple
// une commande pour un client inexistant.
func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
