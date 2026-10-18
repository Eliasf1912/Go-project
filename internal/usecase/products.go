package usecase

import (
	"context"

	"github.com/Eliasf1912/go-project/internal/domain"
)

// ProductFilter décrit les critères de recherche de produits, tous
// optionnels (nil ou "" = critère ignoré) et combinés avec un ET logique.
// Name et Description sont des recherches partielles, insensibles à la
// casse. CategoryID filtre sur une catégorie précise. Les bornes de prix
// sont inclusives et exprimées en centimes.
// Un produit archivé (Active = false) n'est jamais renvoyé : c'est la
// responsabilité du repository, pas de ce filtre.
// Un ProductFilter{} (zéro valeur) renvoie tous les produits actifs,
// comme le faisait l'ancien ListActive.
type ProductFilter struct {
	Name        string
	Description string
	CategoryID  *int
	MinPriceHT  *int
	MaxPriceHT  *int
	MinPriceTTC *int
	MaxPriceTTC *int
}

// ProductRepository est le port implémenté par internal/adapter/postgres
type ProductRepository interface {
	// Search renvoie les produits actifs correspondant au filtre, avec leur
	// catégorie, triés par nom.
	Search(ctx context.Context, filter ProductFilter) ([]domain.Product, error)
}

type ProductService struct {
	products ProductRepository
}

func NewProductService(products ProductRepository) *ProductService {
	return &ProductService{products: products}
}

// Search valide les bornes du filtre puis délègue au repository.
func (s *ProductService) Search(ctx context.Context, filter ProductFilter) ([]domain.Product, error) {
	if err := validatePriceRange(filter.MinPriceHT, filter.MaxPriceHT); err != nil {
		return nil, err
	}
	if err := validatePriceRange(filter.MinPriceTTC, filter.MaxPriceTTC); err != nil {
		return nil, err
	}

	return s.products.Search(ctx, filter)
}

// validatePriceRange refuse un intervalle où le minimum dépasse le maximum ;
// un seul des deux bornes (ou aucune) est toujours valide.
func validatePriceRange(minPrice, maxPrice *int) error {
	if minPrice != nil && maxPrice != nil && *minPrice > *maxPrice {
		return domain.ErrInvalidPriceRange
	}
	return nil
}
