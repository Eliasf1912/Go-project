package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Eliasf1912/go-project/internal/domain"
	"github.com/Eliasf1912/go-project/internal/ids"
	"github.com/Eliasf1912/go-project/internal/usecase"
)

// vérifie à la compilation que CartRepository implémente bien l'interface
var _ usecase.CartRepository = (*CartRepository)(nil)

type CartRepository struct {
	db *sql.DB
}

func NewCartRepository(db *sql.DB) *CartRepository {
	return &CartRepository{db: db}
}

// FindByUserID renvoie le panier de l'utilisateur, avec ses lignes (prix
// courants des produits). ErrNotFound si l'utilisateur n'a pas encore de
// panier (carts.user_id est UNIQUE : au plus une ligne par utilisateur).
func (r *CartRepository) FindByUserID(ctx context.Context, userID int) (domain.Cart, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, reference, user_id, created_at, updated_at
		FROM carts
		WHERE user_id = $1`, userID)

	var cart domain.Cart
	err := row.Scan(&cart.ID, &cart.Reference, &cart.UserID, &cart.CreatedAt, &cart.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Cart{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Cart{}, fmt.Errorf("recherche du panier de l'utilisateur %d : %w", userID, err)
	}

	items, err := r.listItems(ctx, cart.ID)
	if err != nil {
		return domain.Cart{}, err
	}
	cart.Items = items

	return cart, nil
}

// Create crée le panier de l'utilisateur, avec une référence générée
// (BSK-XXXXXX). carts.user_id est UNIQUE en base : si deux requêtes tentent
// de créer le panier en même temps, ON CONFLICT DO NOTHING fait perdre la
// course à l'une des deux sans erreur ; on relit alors le panier créé par
// l'autre requête.
func (r *CartRepository) Create(ctx context.Context, userID int) (domain.Cart, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO carts (reference, user_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO NOTHING
		RETURNING id, reference, user_id, created_at, updated_at`,
		ids.NewCartRef(), userID)

	var cart domain.Cart
	err := row.Scan(&cart.ID, &cart.Reference, &cart.UserID, &cart.CreatedAt, &cart.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		// une autre requête a créé le panier entre-temps : on relit celui-ci.
		return r.FindByUserID(ctx, userID)
	}
	if err != nil {
		return domain.Cart{}, fmt.Errorf("création du panier de l'utilisateur %d : %w", userID, err)
	}
	return cart, nil
}

// AddItem ajoute quantity au panier pour ce produit : augmente la
// quantité si la ligne existe déjà (ON CONFLICT), la crée sinon.
func (r *CartRepository) AddItem(ctx context.Context, cartID, productID, quantity int) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO cart_items (cart_id, product_id, quantity)
		VALUES ($1, $2, $3)
		ON CONFLICT (cart_id, product_id)
		DO UPDATE SET quantity = cart_items.quantity + EXCLUDED.quantity`,
		cartID, productID, quantity)
	if err != nil {
		return fmt.Errorf("ajout au panier %d : %w", cartID, err)
	}
	return nil
}

// SetItemQuantity fixe la quantité d'une ligne déjà présente dans le
// panier (quantity > 0 : 0 est traité par le usecase via RemoveItem).
// ErrNotFound si le produit n'est pas dans le panier.
func (r *CartRepository) SetItemQuantity(ctx context.Context, cartID, productID, quantity int) error {
	if quantity == 0 {
		return r.RemoveItem(ctx, cartID, productID)
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE cart_items SET quantity = $1
		WHERE cart_id = $2 AND product_id = $3`, quantity, cartID, productID)
	if err != nil {
		return fmt.Errorf("mise à jour du panier %d : %w", cartID, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mise à jour du panier %d : %w", cartID, err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// RemoveItem supprime une ligne du panier. ErrNotFound si elle n'y est pas.
func (r *CartRepository) RemoveItem(ctx context.Context, cartID, productID int) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM cart_items WHERE cart_id = $1 AND product_id = $2`, cartID, productID)
	if err != nil {
		return fmt.Errorf("suppression d'une ligne du panier %d : %w", cartID, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("suppression d'une ligne du panier %d : %w", cartID, err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// listItems renvoie les lignes du panier, avec le nom et le prix courants
// du produit (table de liaison cart_items, sans prix figé).
func (r *CartRepository) listItems(ctx context.Context, cartID int) ([]domain.CartItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT ci.product_id, p.name, ci.quantity, p.price_ht, p.price_ttc
		FROM cart_items ci
		JOIN products p ON p.id = ci.product_id
		WHERE ci.cart_id = $1
		ORDER BY p.name`, cartID)
	if err != nil {
		return nil, fmt.Errorf("recherche des lignes du panier %d : %w", cartID, err)
	}
	defer rows.Close()

	var items []domain.CartItem
	for rows.Next() {
		var item domain.CartItem
		if err := rows.Scan(
			&item.ProductID, &item.ProductName, &item.Quantity, &item.PriceHT, &item.PriceTTC,
		); err != nil {
			return nil, fmt.Errorf("lecture d'une ligne du panier %d : %w", cartID, err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recherche des lignes du panier %d : %w", cartID, err)
	}

	return items, nil
}
