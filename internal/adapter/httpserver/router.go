package httpserver

import "net/http"

func NewRouter(auth *AuthHandler, orders *OrderHandler, products *ProductHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)

	// Authentification
	mux.HandleFunc("POST /auth/register", auth.Register)
	mux.HandleFunc("POST /auth/confirm", auth.Confirm)
	mux.HandleFunc("POST /auth/login", auth.Login)
	mux.HandleFunc("GET /auth/me", auth.RequireAuth(auth.Me))

	// Produits
	mux.HandleFunc("GET /products", auth.RequireAuth(products.List))

	// Commandes
	mux.HandleFunc("GET /orders", auth.RequireAuth(orders.ListMine))
	mux.HandleFunc("POST /orders/{id}/pay", auth.RequireAuth(orders.Pay))
	mux.HandleFunc("POST /orders/{id}/cancel", auth.RequireAuth(orders.Cancel))

	return mux
}
