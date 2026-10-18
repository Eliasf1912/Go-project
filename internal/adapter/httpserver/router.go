package httpserver

import "net/http"

func NewRouter(auth *AuthHandler, orders *OrderHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)

	// Authentification
	mux.HandleFunc("POST /auth/register", auth.Register)
	mux.HandleFunc("POST /auth/confirm", auth.Confirm)
	mux.HandleFunc("POST /auth/login", auth.Login)
	mux.HandleFunc("GET /auth/me", auth.RequireAuth(auth.Me))

	// Commandes
	mux.HandleFunc("GET /orders", auth.RequireAuth(orders.ListMine))

	return mux
}
