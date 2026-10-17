package httpserver

import "net/http"

// NewRouter crée le routeur de l'API et enregistre toutes les routes.
//
// Pour ajouter une fonctionnalité : écrire ses handlers dans un fichier
// dédié (auth.go, products.go...) puis enregistrer ses routes ici, avec
// la méthode dans le motif ("GET /products", "POST /cart/items"...).
func NewRouter(auth *AuthHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)

	// Authentification
	mux.HandleFunc("POST /auth/register", auth.Register)
	mux.HandleFunc("POST /auth/confirm", auth.Confirm)
	mux.HandleFunc("POST /auth/login", auth.Login)
	mux.HandleFunc("GET /auth/me", auth.RequireAuth(auth.Me))

	return mux
}
