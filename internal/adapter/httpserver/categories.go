package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Eliasf1912/go-project/internal/domain"
	"github.com/Eliasf1912/go-project/internal/dto"
	"github.com/Eliasf1912/go-project/internal/usecase"
)

type CategoryHandler struct {
	categories *usecase.CategoryService
}

func NewCategoryHandler(categories *usecase.CategoryService) *CategoryHandler {
	return &CategoryHandler{categories: categories}
}

func toCategoryResponse(c domain.Category) dto.CategoryResponse {
	return dto.CategoryResponse{ID: c.ID, Name: c.Name}
}

// pathID lit le paramètre {id} de l'URL. En cas d'erreur, la réponse 400 est déjà écrite.
func pathID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return 0, false
	}
	return id, true
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	categories, err := h.categories.List(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}

	resp := make([]dto.CategoryResponse, len(categories))
	for i, c := range categories {
		resp[i] = toCategoryResponse(c)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalide")
		return
	}

	category, err := h.categories.Create(r.Context(), req.Name)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toCategoryResponse(category))
}

func (h *CategoryHandler) Rename(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req dto.CategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalide")
		return
	}

	category, err := h.categories.Rename(r.Context(), id, req.Name)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toCategoryResponse(category))
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	if err := h.categories.Delete(r.Context(), id); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
