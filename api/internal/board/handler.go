package board

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for Board operations.
type Handler struct {
	service *Service
}

// NewHandler creates a new Board handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Create handles POST /v1/orgs/{org}/spaces/{space}/boards.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "space")
	if spaceID == "" {
		apierrors.BadRequest(w, "space identifier is required")
		return
	}

	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	b, err := h.service.Create(r.Context(), spaceID, input)
	if err != nil {
		apierrors.ValidationError(w, err.Error(), nil)
		return
	}

	response.Created(w, b)
}

// List handles GET /v1/orgs/{org}/spaces/{space}/boards.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "space")
	params := pagination.Parse(r)

	boards, pageInfo, err := h.service.List(r.Context(), spaceID, params)
	if err != nil {
		apierrors.InternalError(w, "failed to list boards")
		return
	}

	response.JSON(w, http.StatusOK, response.ListResponse{
		Data:     boards,
		PageInfo: pageInfo,
	})
}

// Get handles GET /v1/orgs/{org}/spaces/{space}/boards/{board}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "space")
	idOrSlug := chi.URLParam(r, "board")

	b, err := h.service.Get(r.Context(), spaceID, idOrSlug)
	if err != nil {
		apierrors.InternalError(w, "failed to get board")
		return
	}
	if b == nil {
		apierrors.NotFound(w, "board not found")
		return
	}

	response.JSON(w, http.StatusOK, b)
}

// Update handles PATCH /v1/orgs/{org}/spaces/{space}/boards/{board}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "space")
	idOrSlug := chi.URLParam(r, "board")

	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	b, err := h.service.Update(r.Context(), spaceID, idOrSlug, input)
	if err != nil {
		apierrors.ValidationError(w, err.Error(), nil)
		return
	}
	if b == nil {
		apierrors.NotFound(w, "board not found")
		return
	}

	response.JSON(w, http.StatusOK, b)
}

// Delete handles DELETE /v1/orgs/{org}/spaces/{space}/boards/{board}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "space")
	idOrSlug := chi.URLParam(r, "board")

	if err := h.service.Delete(r.Context(), spaceID, idOrSlug); err != nil {
		if err.Error() == "not found" {
			apierrors.NotFound(w, "board not found")
			return
		}
		apierrors.InternalError(w, "failed to delete board")
		return
	}

	response.NoContent(w)
}

// Lock handles POST /v1/orgs/{org}/spaces/{space}/boards/{board}/lock.
func (h *Handler) Lock(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "space")
	idOrSlug := chi.URLParam(r, "board")

	b, err := h.service.SetLock(r.Context(), spaceID, idOrSlug, true)
	if err != nil {
		apierrors.InternalError(w, "failed to lock board")
		return
	}
	if b == nil {
		apierrors.NotFound(w, "board not found")
		return
	}

	response.JSON(w, http.StatusOK, b)
}

// Unlock handles POST /v1/orgs/{org}/spaces/{space}/boards/{board}/unlock.
func (h *Handler) Unlock(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "space")
	idOrSlug := chi.URLParam(r, "board")

	b, err := h.service.SetLock(r.Context(), spaceID, idOrSlug, false)
	if err != nil {
		apierrors.InternalError(w, "failed to unlock board")
		return
	}
	if b == nil {
		apierrors.NotFound(w, "board not found")
		return
	}

	response.JSON(w, http.StatusOK, b)
}
