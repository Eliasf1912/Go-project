package httpserver

import (
	"context"
	"net/http"
	"strings"

	"github.com/Eliasf1912/go-project/internal/domain"
)

// contextKey est un type privé, pour qu'aucun autre package ne puisse
// utiliser par erreur la même clé dans le context.
type contextKey string

const userKey contextKey = "user"

// RequireAuth laisse passer la requête seulement si elle porte un jeton de session valide.
func (h *AuthHandler) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			writeDomainError(w, domain.ErrUnauthorized)
			return
		}

		user, err := h.auth.Authenticate(r.Context(), token)
		if err != nil {
			writeDomainError(w, err)
			return
		}

		ctx := context.WithValue(r.Context(), userKey, user)
		next(w, r.WithContext(ctx))
	}
}

// UserFromContext renvoie l'utilisateur authentifié par RequireAuth.
func UserFromContext(ctx context.Context) (domain.User, bool) {
	user, ok := ctx.Value(userKey).(domain.User)
	return user, ok
}
