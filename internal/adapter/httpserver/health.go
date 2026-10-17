package httpserver

import "net/http"

// handleHealth indique que le serveur est démarré et répond.
func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
