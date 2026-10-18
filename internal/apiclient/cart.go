package apiclient

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Eliasf1912/go-project/internal/dto"
)

// GetCart renvoie le panier de l'utilisateur connecté (vide si jamais créé).
func (c *Client) GetCart(ctx context.Context) (dto.CartResponse, error) {
	var resp dto.CartResponse
	err := c.do(ctx, http.MethodGet, "/cart", nil, &resp)
	return resp, err
}

// AddCartItem ajoute quantity exemplaires d'un produit au panier (la
// quantité s'additionne si le produit y est déjà).
func (c *Client) AddCartItem(ctx context.Context, productID, quantity int) (dto.CartResponse, error) {
	req := dto.AddCartItemRequest{ProductID: productID, Quantity: quantity}
	var resp dto.CartResponse
	err := c.do(ctx, http.MethodPost, "/cart/items", req, &resp)
	return resp, err
}

// UpdateCartItem fixe la quantité d'un produit déjà dans le panier ;
// quantity = 0 supprime la ligne.
func (c *Client) UpdateCartItem(ctx context.Context, productID, quantity int) (dto.CartResponse, error) {
	req := dto.UpdateCartItemRequest{Quantity: quantity}
	var resp dto.CartResponse
	err := c.do(ctx, http.MethodPut, fmt.Sprintf("/cart/items/%d", productID), req, &resp)
	return resp, err
}

// RemoveCartItem supprime un produit du panier.
func (c *Client) RemoveCartItem(ctx context.Context, productID int) (dto.CartResponse, error) {
	var resp dto.CartResponse
	err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/cart/items/%d", productID), nil, &resp)
	return resp, err
}
