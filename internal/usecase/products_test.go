package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Eliasf1912/go-project/internal/domain"
	"github.com/Eliasf1912/go-project/internal/usecase"
)

var _ usecase.ProductRepository = (*fakeProductRepo)(nil)

// fakeProductRepo ignore le filtre et renvoie simplement les produits
// fournis : seule la validation faite par ProductService est testée ici,
// pas le filtrage SQL (couvert par les tests d'intégration du repository).
type fakeProductRepo struct {
	products       []domain.Product
	receivedFilter usecase.ProductFilter
}

func (r *fakeProductRepo) Search(_ context.Context, filter usecase.ProductFilter) ([]domain.Product, error) {
	r.receivedFilter = filter
	return r.products, nil
}

func intPtr(v int) *int { return &v }

func TestProductSearch_DelegueAuRepository(t *testing.T) {
	repo := &fakeProductRepo{products: []domain.Product{{ID: 1, Name: "Clavier"}}}
	service := usecase.NewProductService(repo)

	filter := usecase.ProductFilter{Name: "clav", CategoryID: intPtr(3)}
	products, err := service.Search(context.Background(), filter)
	if err != nil {
		t.Fatalf("Search() erreur inattendue : %v", err)
	}
	if len(products) != 1 || products[0].Name != "Clavier" {
		t.Errorf("Search() = %v, attendu le produit du repository", products)
	}
	if repo.receivedFilter.Name != "clav" || *repo.receivedFilter.CategoryID != 3 {
		t.Errorf("filtre transmis au repository = %+v, attendu %+v", repo.receivedFilter, filter)
	}
}

func TestProductSearch_IntervallePrixInvalide(t *testing.T) {
	tests := []struct {
		name   string
		filter usecase.ProductFilter
	}{
		{"prix HT min > max", usecase.ProductFilter{MinPriceHT: intPtr(2000), MaxPriceHT: intPtr(1000)}},
		{"prix TTC min > max", usecase.ProductFilter{MinPriceTTC: intPtr(2000), MaxPriceTTC: intPtr(1000)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeProductRepo{}
			service := usecase.NewProductService(repo)

			_, err := service.Search(context.Background(), tt.filter)
			if !errors.Is(err, domain.ErrInvalidPriceRange) {
				t.Errorf("Search() = %v, attendu %v", err, domain.ErrInvalidPriceRange)
			}
		})
	}
}

func TestProductSearch_IntervallePrixValide(t *testing.T) {
	tests := []struct {
		name   string
		filter usecase.ProductFilter
	}{
		{"min = max", usecase.ProductFilter{MinPriceHT: intPtr(1000), MaxPriceHT: intPtr(1000)}},
		{"min seul", usecase.ProductFilter{MinPriceHT: intPtr(1000)}},
		{"max seul", usecase.ProductFilter{MaxPriceHT: intPtr(1000)}},
		{"aucune borne", usecase.ProductFilter{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeProductRepo{}
			service := usecase.NewProductService(repo)

			if _, err := service.Search(context.Background(), tt.filter); err != nil {
				t.Errorf("Search() erreur inattendue : %v", err)
			}
		})
	}
}
