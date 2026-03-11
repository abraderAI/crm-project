package space

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for Space operations.
type Handler struct {
	service *Service
}

// NewHandler creates a new Space handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Create handles POST /v1/orgs/{org}/spaces.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	if orgID == "" {
		apierrors.BadRequest(w, "org identifier is required")
		return
	}

	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	sp, err := h.service.Create(r.Context(), orgID, input)
	if err != nil {
		apierrors.ValidationError(w, err.Error(), nil)
		return
	}

	response.Created(w, sp)
}

// List handles GET /v1/orgs/{org}/spaces.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	params := pagination.Parse(r)

	spaces, pageInfo, err := h.service.List(r.Context(), orgID, params)
	if err != nil {
		apierrors.InternalError(w, "failed to list spaces")
		return
	}

	response.JSON(w, http.StatusOK, response.ListResponse{
		Data:     spaces,
		PageInfo: pageInfo,
	})
}

// Get handles GET /v1/orgs/{org}/spaces/{space}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	idOrSlug := chi.URLParam(r, "space")

	sp, err := h.service.Get(r.Context(), orgID, idOrSlug)
	if err != nil {
		apierrors.InternalError(w, "failed to get space")
		return
	}
	if sp == nil {
		apierrors.NotFound(w, "space not found")
		return
	}

	response.JSON(w, http.StatusOK, sp)
}

// Update handles PATCH /v1/orgs/{org}/spaces/{space}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	idOrSlug := chi.URLParam(r, "space")

	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	sp, err := h.service.Update(r.Context(), orgID, idOrSlug, input)
	if err != nil {
		apierrors.ValidationError(w, err.Error(), nil)
		return
	}
	if sp == nil {
		apierrors.NotFound(w, "space not found")
		return
	}

	response.JSON(w, http.StatusOK, sp)
}

// Delete handles DELETE /v1/orgs/{org}/spaces/{space}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	idOrSlug := chi.URLParam(r, "space")

	if err := h.service.Delete(r.Context(), orgID, idOrSlug); err != nil {
		if err.Error() == "not found" {
			apierrors.NotFound(w, "space not found")
			return
		}
		apierrors.InternalError(w, "failed to delete space")
		return
	}

	response.NoContent(w)
}
