package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Eliasf1912/go-project/internal/domain"
	"github.com/Eliasf1912/go-project/internal/usecase"
)

var _ usecase.CategoryRepository = (*CategoryRepository)(nil)

type CategoryRepository struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) List(ctx context.Context) ([]domain.Category, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, created_at FROM categories ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("liste des catégories : %w", err)
	}
	defer rows.Close()

	categories := []domain.Category{}
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("lecture d'une catégorie : %w", err)
		}
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("liste des catégories : %w", err)
	}
	return categories, nil
}

func (r *CategoryRepository) Create(ctx context.Context, name string) (domain.Category, error) {
	c := domain.Category{Name: name}
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO categories (name) VALUES ($1) RETURNING id, created_at`, name,
	).Scan(&c.ID, &c.CreatedAt)
	if pgErrorCode(err) == codeUniqueViolation {
		return domain.Category{}, domain.ErrCategoryNameTaken
	}
	if err != nil {
		return domain.Category{}, fmt.Errorf("création de la catégorie : %w", err)
	}
	return c, nil
}

func (r *CategoryRepository) Rename(ctx context.Context, id int, name string) (domain.Category, error) {
	c := domain.Category{ID: id, Name: name}
	err := r.db.QueryRowContext(ctx,
		`UPDATE categories SET name = $1 WHERE id = $2 RETURNING created_at`, name, id,
	).Scan(&c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Category{}, domain.ErrNotFound
	}
	if pgErrorCode(err) == codeUniqueViolation {
		return domain.Category{}, domain.ErrCategoryNameTaken
	}
	if err != nil {
		return domain.Category{}, fmt.Errorf("renommage de la catégorie : %w", err)
	}
	return c, nil
}

func (r *CategoryRepository) Delete(ctx context.Context, id int) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if pgErrorCode(err) == codeForeignKeyViolation {
		return domain.ErrCategoryInUse
	}
	if err != nil {
		return fmt.Errorf("suppression de la catégorie : %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("suppression de la catégorie : %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}
