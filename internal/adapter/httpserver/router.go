package httpserver

import "net/http"

// NewRouter crée le routeur de l'API et enregistre toutes les routes.
//
// Pour ajouter une fonctionnalité : écrire ses handlers dans un fichier
// dédié (auth.go, products.go...) puis enregistrer ses routes ici, avec
// la méthode dans le motif ("GET /products", "POST /cart/items"...).
func NewRouter(auth *AuthHandler, categories *CategoryHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)

	// Authentification
	mux.HandleFunc("POST /auth/register", auth.Register)
	mux.HandleFunc("POST /auth/confirm", auth.Confirm)
	mux.HandleFunc("POST /auth/login", auth.Login)
	mux.HandleFunc("GET /auth/me", auth.RequireAuth(auth.Me))
	mux.HandleFunc("POST /auth/logout", auth.RequireAuth(auth.Logout))

	// Catégories
	mux.HandleFunc("GET /categories", auth.RequireAuth(categories.List))
	mux.HandleFunc("GET /admin/categories", auth.RequireAdmin(categories.List))
	mux.HandleFunc("POST /admin/categories", auth.RequireAdmin(categories.Create))
	mux.HandleFunc("PATCH /admin/categories/{id}", auth.RequireAdmin(categories.Rename))
	mux.HandleFunc("DELETE /admin/categories/{id}", auth.RequireAdmin(categories.Delete))

	return mux
}
