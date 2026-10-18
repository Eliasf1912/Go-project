package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Eliasf1912/go-project/internal/domain"
	"github.com/Eliasf1912/go-project/internal/usecase"
)

// vérifie à la compilation que ProductRepository implémente bien l'interface
var _ usecase.ProductRepository = (*ProductRepository)(nil)

type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// ListActive renvoie les produits non archivés, avec leur catégorie, triés par nom.
func (r *ProductRepository) ListActive(ctx context.Context) ([]domain.Product, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.id, p.reference, p.name, p.description, p.price_ht, p.price_ttc,
		       p.active, p.created_at, p.updated_at,
		       c.id, c.name, c.created_at
		FROM products p
		JOIN categories c ON c.id = p.category_id
		WHERE p.active
		ORDER BY p.name`)
	if err != nil {
		return nil, fmt.Errorf("recherche des produits : %w", err)
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var (
			product     domain.Product
			description sql.NullString
		)
		if err := rows.Scan(
			&product.ID, &product.Reference, &product.Name, &description,
			&product.PriceHT, &product.PriceTTC, &product.Active,
			&product.CreatedAt, &product.UpdatedAt,
			&product.Category.ID, &product.Category.Name, &product.Category.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("lecture d'un produit : %w", err)
		}
		product.Description = description.String // "" si NULL
		products = append(products, product)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recherche des produits : %w", err)
	}

	return products, nil
}
