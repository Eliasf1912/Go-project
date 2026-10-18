package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Eliasf1912/go-project/internal/domain"
	"github.com/Eliasf1912/go-project/internal/usecase"
)

// vérifie à la compilation que ProductRepository implémente bien les
// interfaces (ProductRepository pour la recherche, ProductLookup pour le
// panier)
var (
	_ usecase.ProductRepository = (*ProductRepository)(nil)
	_ usecase.ProductLookup     = (*ProductRepository)(nil)
)

type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// Search renvoie les produits actifs (non archivés) correspondant au
// filtre, avec leur catégorie, triés par nom. Name et Description sont des
// recherches partielles insensibles à la casse (ILIKE) ; les autres
// critères sont des égalités ou des bornes (inclusives). Tous les critères
// sont combinés avec AND ; un filtre vide renvoie tous les produits actifs
// (comportement de l'ancien ListActive).
func (r *ProductRepository) Search(ctx context.Context, filter usecase.ProductFilter) ([]domain.Product, error) {
	query := strings.Builder{}
	query.WriteString(`
		SELECT p.id, p.reference, p.name, p.description, p.price_ht, p.price_ttc,
		       p.active, p.created_at, p.updated_at,
		       c.id, c.name, c.created_at
		FROM products p
		JOIN categories c ON c.id = p.category_id
		WHERE p.active`)

	var args []any
	addCondition := func(condition string, value any) {
		args = append(args, value)
		query.WriteString(fmt.Sprintf(" AND %s $%d", condition, len(args)))
	}

	if filter.Name != "" {
		addCondition("p.name ILIKE", "%"+escapeLikePattern(filter.Name)+"%")
	}
	if filter.Description != "" {
		addCondition("p.description ILIKE", "%"+escapeLikePattern(filter.Description)+"%")
	}
	if filter.CategoryID != nil {
		addCondition("p.category_id =", *filter.CategoryID)
	}
	if filter.MinPriceHT != nil {
		addCondition("p.price_ht >=", *filter.MinPriceHT)
	}
	if filter.MaxPriceHT != nil {
		addCondition("p.price_ht <=", *filter.MaxPriceHT)
	}
	if filter.MinPriceTTC != nil {
		addCondition("p.price_ttc >=", *filter.MinPriceTTC)
	}
	if filter.MaxPriceTTC != nil {
		addCondition("p.price_ttc <=", *filter.MaxPriceTTC)
	}

	query.WriteString(" ORDER BY p.name")

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
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

// FindActiveByID renvoie un produit actif par son id, avec sa catégorie.
// ErrNotFound s'il n'existe pas ou est archivé : dans les deux cas, il ne
// peut pas être ajouté au panier.
func (r *ProductRepository) FindActiveByID(ctx context.Context, id int) (domain.Product, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT p.id, p.reference, p.name, p.description, p.price_ht, p.price_ttc,
		       p.active, p.created_at, p.updated_at,
		       c.id, c.name, c.created_at
		FROM products p
		JOIN categories c ON c.id = p.category_id
		WHERE p.id = $1 AND p.active`, id)

	var (
		product     domain.Product
		description sql.NullString
	)
	err := row.Scan(
		&product.ID, &product.Reference, &product.Name, &description,
		&product.PriceHT, &product.PriceTTC, &product.Active,
		&product.CreatedAt, &product.UpdatedAt,
		&product.Category.ID, &product.Category.Name, &product.Category.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Product{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Product{}, fmt.Errorf("recherche du produit %d : %w", id, err)
	}
	product.Description = description.String // "" si NULL

	return product, nil
}
