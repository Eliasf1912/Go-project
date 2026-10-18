package apiclient

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Eliasf1912/go-project/internal/dto"
)

func (c *Client) ListUsers(ctx context.Context) ([]dto.UserResponse, error) {
	var resp []dto.UserResponse
	err := c.do(ctx, http.MethodGet, "/admin/users", nil, &resp)
	return resp, err
}

// DeleteUser renvoie true si le compte a été anonymisé au lieu d'être supprimé.
func (c *Client) DeleteUser(ctx context.Context, id int) (bool, error) {
	var resp dto.DeleteUserResponse
	err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/admin/users/%d", id), nil, &resp)
	return resp.Anonymized, err
}
