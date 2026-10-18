package httpserver

import (
	"net/http"

	"github.com/Eliasf1912/go-project/internal/domain"
	"github.com/Eliasf1912/go-project/internal/dto"
	"github.com/Eliasf1912/go-project/internal/usecase"
)

type UserHandler struct {
	users *usecase.UserService
}

func NewUserHandler(users *usecase.UserService) *UserHandler {
	return &UserHandler{users: users}
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.users.List(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}

	resp := make([]dto.UserResponse, len(users))
	for i, u := range users {
		resp[i] = dto.UserResponse{
			ID:        u.ID,
			Email:     u.Email,
			Role:      string(u.Role),
			Confirmed: u.Confirmed,
			Deleted:   u.DeletedAt != nil,
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	admin, ok := UserFromContext(r.Context())
	if !ok {
		writeDomainError(w, domain.ErrUnauthorized)
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	anonymized, err := h.users.Delete(r.Context(), admin.ID, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.DeleteUserResponse{Anonymized: anonymized})
}
