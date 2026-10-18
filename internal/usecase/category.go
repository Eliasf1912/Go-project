package usecase

import (
	"context"

	"github.com/Eliasf1912/go-project/internal/domain"
)

type CategoryRepository interface {
	List(ctx context.Context) ([]domain.Category, error)
	Create(ctx context.Context, name string) (domain.Category, error)
	Rename(ctx context.Context, id int, name string) (domain.Category, error)
	Delete(ctx context.Context, id int) error
}

type CategoryService struct {
	categories CategoryRepository
}

func NewCategoryService(categories CategoryRepository) *CategoryService {
	return &CategoryService{categories: categories}
}

func (s *CategoryService) List(ctx context.Context) ([]domain.Category, error) {
	return s.categories.List(ctx)
}

func (s *CategoryService) Create(ctx context.Context, name string) (domain.Category, error) {
	name, err := domain.NormalizeCategoryName(name)
	if err != nil {
		return domain.Category{}, err
	}
	return s.categories.Create(ctx, name)
}

func (s *CategoryService) Rename(ctx context.Context, id int, name string) (domain.Category, error) {
	name, err := domain.NormalizeCategoryName(name)
	if err != nil {
		return domain.Category{}, err
	}
	return s.categories.Rename(ctx, id, name)
}

func (s *CategoryService) Delete(ctx context.Context, id int) error {
	return s.categories.Delete(ctx, id)
}
