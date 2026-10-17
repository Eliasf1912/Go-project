package httpserver

import "net/http"

// NewRouter crée le routeur de l'API et enregistre toutes les routes.
//
// Pour ajouter une fonctionnalité : écrire ses handlers dans un fichier
// dédié (auth.go, products.go...) puis enregistrer ses routes ici, avec
// la méthode dans le motif ("GET /products", "POST /cart/items"...).
func NewRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)

	return mux
}
