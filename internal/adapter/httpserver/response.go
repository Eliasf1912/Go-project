package httpserver

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/Eliasf1912/go-project/internal/domain"
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

// writeDomainError traduit une erreur du domaine en réponse HTTP.
// Les erreurs inconnues donnent un 500, avec le détail uniquement dans les logs.
func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrPasswordTooShort),
		errors.Is(err, domain.ErrInvalidCode),
		errors.Is(err, domain.ErrCodeExpired),
		errors.Is(err, domain.ErrAccountAlreadyConfirmed),
		errors.Is(err, domain.ErrInvalidCardNumber),
		errors.Is(err, domain.ErrInvalidCardExpiry),
		errors.Is(err, domain.ErrCardExpired),
		errors.Is(err, domain.ErrInvalidCVC),
		errors.Is(err, domain.ErrCancelReasonRequired),
		errors.Is(err, domain.ErrInvalidPriceRange),
		errors.Is(err, domain.ErrInvalidQuantity):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrEmailTaken),
		errors.Is(err, domain.ErrOrderNotPending),
		errors.Is(err, domain.ErrOrderStatusChanged),
		errors.Is(err, domain.ErrOrderNotCancellable):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials),
		errors.Is(err, domain.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrAccountNotConfirmed):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, domain.ErrInvalidCategoryName):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrCategoryNameTaken), errors.Is(err, domain.ErrCategoryInUse):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrCannotDeleteSelf):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		log.Printf("erreur interne : %v", err)
		writeError(w, http.StatusInternalServerError, "erreur interne")
	}
}
