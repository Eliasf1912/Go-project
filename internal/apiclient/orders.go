package apiclient

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Eliasf1912/go-project/internal/dto"
)

// ListMyOrders renvoie les commandes de l'utilisateur connecté.
func (c *Client) ListMyOrders(ctx context.Context) ([]dto.OrderResponse, error) {
	var resp []dto.OrderResponse
	err := c.do(ctx, http.MethodGet, "/orders", nil, &resp)
	return resp, err
}

// Checkout valide le panier de l'utilisateur connecté : crée une commande
// "pending" à partir de ses lignes, puis vide le panier.
func (c *Client) Checkout(ctx context.Context) (dto.OrderResponse, error) {
	var resp dto.OrderResponse
	err := c.do(ctx, http.MethodPost, "/orders", nil, &resp)
	return resp, err
}

// Pay paie une commande en attente.
func (c *Client) Pay(ctx context.Context, orderID int, cardNumber, expiry, cvc string) (dto.OrderResponse, error) {
	req := dto.PayRequest{CardNumber: cardNumber, Expiry: expiry, CVC: cvc}
	var resp dto.OrderResponse
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/orders/%d/pay", orderID), req, &resp)
	return resp, err
}

// CancelOrder annule une commande (en attente ou payée), avec une raison.
func (c *Client) CancelOrder(ctx context.Context, orderID int, reason string) (dto.OrderResponse, error) {
	req := dto.CancelRequest{Reason: reason}
	var resp dto.OrderResponse
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/orders/%d/cancel", orderID), req, &resp)
	return resp, err
}

// AdminOrderLine est une ligne (produit, quantité) choisie par l'admin
// pour créer une commande au nom d'un client.
type AdminOrderLine struct {
	ProductID int
	Quantity  int
}

// AdminListOrders renvoie toutes les commandes, tous clients confondus.
// Réservé à l'admin.
func (c *Client) AdminListOrders(ctx context.Context) ([]dto.OrderResponse, error) {
	var resp []dto.OrderResponse
	err := c.do(ctx, http.MethodGet, "/admin/orders", nil, &resp)
	return resp, err
}

// AdminCreateOrder crée une commande pour un client donné, à partir de
// lignes (produit, quantité). Réservé à l'admin.
func (c *Client) AdminCreateOrder(ctx context.Context, userID int, lines []AdminOrderLine) (dto.OrderResponse, error) {
	req := dto.AdminCreateOrderRequest{UserID: userID, Items: make([]dto.AdminOrderItemRequest, 0, len(lines))}
	for _, line := range lines {
		req.Items = append(req.Items, dto.AdminOrderItemRequest{ProductID: line.ProductID, Quantity: line.Quantity})
	}

	var resp dto.OrderResponse
	err := c.do(ctx, http.MethodPost, "/admin/orders", req, &resp)
	return resp, err
}

// AdminShipOrder fait passer une commande payée en "en cours de livraison".
// Réservé à l'admin.
func (c *Client) AdminShipOrder(ctx context.Context, orderID int) (dto.OrderResponse, error) {
	var resp dto.OrderResponse
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/admin/orders/%d/ship", orderID), nil, &resp)
	return resp, err
}

// AdminDeliverOrder fait passer une commande en livraison en "livrée".
// Réservé à l'admin.
func (c *Client) AdminDeliverOrder(ctx context.Context, orderID int) (dto.OrderResponse, error) {
	var resp dto.OrderResponse
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/admin/orders/%d/deliver", orderID), nil, &resp)
	return resp, err
}

// AdminCancelOrder annule n'importe quelle commande, avec une raison.
// Réservé à l'admin.
func (c *Client) AdminCancelOrder(ctx context.Context, orderID int, reason string) (dto.OrderResponse, error) {
	req := dto.CancelRequest{Reason: reason}
	var resp dto.OrderResponse
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/admin/orders/%d/cancel", orderID), req, &resp)
	return resp, err
}

// AdminDeleteOrder supprime définitivement une commande. Réservé à l'admin.
func (c *Client) AdminDeleteOrder(ctx context.Context, orderID int) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/admin/orders/%d", orderID), nil, nil)
}
