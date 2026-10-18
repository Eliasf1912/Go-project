package usecase

import (
	"context"

	"github.com/Eliasf1912/go-project/internal/domain"
)

// ProductRepository est le port implémenté par internal/adapter/postgres
type ProductRepository interface {
	ListActive(ctx context.Context) ([]domain.Product, error)
}

type ProductService struct {
	products ProductRepository
}

func NewProductService(products ProductRepository) *ProductService {
	return &ProductService{products: products}
}

// List renvoie les produits actifs (non archivés), triés par nom.
func (s *ProductService) List(ctx context.Context) ([]domain.Product, error) {
	return s.products.ListActive(ctx)
}
