package apiclient

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Eliasf1912/go-project/internal/dto"
)

func (c *Client) ListCategories(ctx context.Context) ([]dto.CategoryResponse, error) {
	var resp []dto.CategoryResponse
	err := c.do(ctx, http.MethodGet, "/admin/categories", nil, &resp)
	return resp, err
}

func (c *Client) CreateCategory(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodPost, "/admin/categories", dto.CategoryRequest{Name: name}, nil)
}

func (c *Client) RenameCategory(ctx context.Context, id int, name string) error {
	path := fmt.Sprintf("/admin/categories/%d", id)
	return c.do(ctx, http.MethodPatch, path, dto.CategoryRequest{Name: name}, nil)
}

func (c *Client) DeleteCategory(ctx context.Context, id int) error {
	path := fmt.Sprintf("/admin/categories/%d", id)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}
