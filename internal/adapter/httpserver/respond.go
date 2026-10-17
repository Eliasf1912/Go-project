package httpserver

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/Eliasf1912/go-project/internal/domain"
	"github.com/Eliasf1912/go-project/internal/dto"
)

// writeJSON envoie une réponse JSON avec le code HTTP donné.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

// writeError traduit une erreur du domaine en réponse HTTP.
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "erreur interne"

	switch {
	case errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrPasswordTooShort),
		errors.Is(err, domain.ErrInvalidCode),
		errors.Is(err, domain.ErrCodeExpired),
		errors.Is(err, domain.ErrAccountAlreadyConfirmed):
		status = http.StatusBadRequest
		message = err.Error()
	case errors.Is(err, domain.ErrEmailTaken):
		status = http.StatusConflict
		message = err.Error()
	case errors.Is(err, domain.ErrInvalidCredentials),
		errors.Is(err, domain.ErrUnauthorized):
		status = http.StatusUnauthorized
		message = err.Error()
	case errors.Is(err, domain.ErrAccountNotConfirmed):
		status = http.StatusForbidden
		message = err.Error()
	default:
		log.Printf("erreur interne : %v", err)
	}

	writeJSON(w, status, dto.ErrorResponse{Error: message})
}
