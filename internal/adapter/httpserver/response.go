package httpserver

import (
	"encoding/json"
	"log"
	"net/http"
)

// writeJSON envoie data encodé en JSON avec le code HTTP status.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status) // doit être appelé avant d'écrire le corps
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("écriture de la réponse JSON : %v", err)
	}
}

// writeError envoie une erreur au format {"error": "message"}.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
