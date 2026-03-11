package org

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for Org operations.
type Handler struct {
	service *Service
}

// NewHandler creates a new Org handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Create handles POST /v1/orgs.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	org, err := h.service.Create(r.Context(), input)
	if err != nil {
		apierrors.ValidationError(w, err.Error(), nil)
		return
	}

	response.Created(w, org)
}

// List handles GET /v1/orgs.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r)

	orgs, pageInfo, err := h.service.List(r.Context(), params)
	if err != nil {
		apierrors.InternalError(w, "failed to list orgs")
		return
	}

	response.JSON(w, http.StatusOK, response.ListResponse{
		Data:     orgs,
		PageInfo: pageInfo,
	})
}

// Get handles GET /v1/orgs/{org}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	idOrSlug := chi.URLParam(r, "org")
	if idOrSlug == "" {
		apierrors.BadRequest(w, "org identifier is required")
		return
	}

	org, err := h.service.Get(r.Context(), idOrSlug)
	if err != nil {
		apierrors.InternalError(w, "failed to get org")
		return
	}
	if org == nil {
		apierrors.NotFound(w, "org not found")
		return
	}

	response.JSON(w, http.StatusOK, org)
}

// Update handles PATCH /v1/orgs/{org}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	idOrSlug := chi.URLParam(r, "org")
	if idOrSlug == "" {
		apierrors.BadRequest(w, "org identifier is required")
		return
	}

	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	org, err := h.service.Update(r.Context(), idOrSlug, input)
	if err != nil {
		apierrors.ValidationError(w, err.Error(), nil)
		return
	}
	if org == nil {
		apierrors.NotFound(w, "org not found")
		return
	}

	response.JSON(w, http.StatusOK, org)
}

// Delete handles DELETE /v1/orgs/{org}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	idOrSlug := chi.URLParam(r, "org")
	if idOrSlug == "" {
		apierrors.BadRequest(w, "org identifier is required")
		return
	}

	if err := h.service.Delete(r.Context(), idOrSlug); err != nil {
		if err.Error() == "not found" {
			apierrors.NotFound(w, "org not found")
			return
		}
		apierrors.InternalError(w, "failed to delete org")
		return
	}

	response.NoContent(w)
}
