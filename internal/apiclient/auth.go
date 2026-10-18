package apiclient

import (
	"context"
	"net/http"

	"github.com/Eliasf1912/go-project/internal/dto"
)

func (c *Client) Register(ctx context.Context, email, password string) error {
	req := dto.RegisterRequest{Email: email, Password: password}
	return c.do(ctx, http.MethodPost, "/auth/register", req, nil)
}

func (c *Client) Confirm(ctx context.Context, email, code string) error {
	req := dto.ConfirmRequest{Email: email, Code: code}
	return c.do(ctx, http.MethodPost, "/auth/confirm", req, nil)
}

func (c *Client) Login(ctx context.Context, email, password string) (string, error) {
	req := dto.LoginRequest{Email: email, Password: password}
	var resp dto.LoginResponse
	if err := c.do(ctx, http.MethodPost, "/auth/login", req, &resp); err != nil {
		return "", err
	}
	return resp.Token, nil
}

func (c *Client) Me(ctx context.Context) (dto.MeResponse, error) {
	var resp dto.MeResponse
	err := c.do(ctx, http.MethodGet, "/auth/me", nil, &resp)
	return resp, err
}
