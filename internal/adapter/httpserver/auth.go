package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/Eliasf1912/go-project/internal/domain"
	"github.com/Eliasf1912/go-project/internal/dto"
	"github.com/Eliasf1912/go-project/internal/usecase"
)

type AuthHandler struct {
	auth *usecase.AuthService
}

func NewAuthHandler(auth *usecase.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalide")
		return
	}

	if err := h.auth.Register(r.Context(), req.Email, req.Password); err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.MessageResponse{
		Message: "compte créé, un code de confirmation a été envoyé",
	})
}

func (h *AuthHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	var req dto.ConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalide")
		return
	}

	if err := h.auth.Confirm(r.Context(), req.Email, req.Code); err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.MessageResponse{Message: "compte confirmé"})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalide")
		return
	}

	token, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.LoginResponse{Token: token})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeDomainError(w, domain.ErrUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, dto.MeResponse{
		ID:    user.ID,
		Email: user.Email,
		Role:  string(user.Role),
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token, ok := TokenFromContext(r.Context())
	if !ok {
		writeDomainError(w, domain.ErrUnauthorized)
		return
	}

	if err := h.auth.Logout(r.Context(), token); err != nil {
		writeDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req dto.ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalide")
		return
	}

	if err := h.auth.ForgotPassword(r.Context(), req.Email); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.MessageResponse{
		Message: "si ce compte existe, un code de réinitialisation a été envoyé",
	})
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req dto.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalide")
		return
	}

	if err := h.auth.ResetPassword(r.Context(), req.Email, req.Code, req.Password); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.MessageResponse{Message: "mot de passe modifié, reconnecte-toi"})
}
