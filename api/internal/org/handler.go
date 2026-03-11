package org

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// Handler provides HTTP handlers for org endpoints.
type Handler struct {
	service    *Service
	memberRepo MemberRepo
}

// MemberRepo is the interface for creating org membership from the handler.
type MemberRepo interface {
	CreateOrgMembership(ctx interface{ Value(any) any }, orgID, userID string, role models.Role) error
}

// NewHandler creates a new org handler.
func NewHandler(service *Service, memberRepo MemberRepo) *Handler {
	return &Handler{service: service, memberRepo: memberRepo}
}

// Create handles POST /v1/orgs.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	org, err := h.service.Create(r.Context(), input)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	// Auto-create owner membership for the creator.
	if h.memberRepo != nil {
		_ = h.memberRepo.CreateOrgMembership(r.Context(), org.ID, uc.UserID, models.RoleOwner)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(org)
}

// List handles GET /v1/orgs.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	params := pagination.Parse(r)
	cursorID := ""
	if params.Cursor != "" {
		decoded, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			apierrors.BadRequest(w, "invalid cursor")
			return
		}
		if decoded != uuid.Nil {
			cursorID = decoded.String()
		}
	}

	orgs, hasMore, err := h.service.List(r.Context(), cursorID, params.Limit, uc.UserID)
	if err != nil {
		apierrors.InternalError(w, "failed to list orgs")
		return
	}

	pageInfo := pagination.PageInfo{HasMore: hasMore}
	if hasMore && len(orgs) > 0 {
		lastID, _ := uuid.Parse(orgs[len(orgs)-1].ID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data":      orgs,
		"page_info": pageInfo,
	})
}

// Get handles GET /v1/orgs/{org}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	ref := chi.URLParam(r, "org")
	if ref == "" {
		apierrors.BadRequest(w, "org identifier is required")
		return
	}

	org, err := h.service.GetByRef(r.Context(), ref)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(org)
}

// Update handles PATCH /v1/orgs/{org}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ref := chi.URLParam(r, "org")
	if ref == "" {
		apierrors.BadRequest(w, "org identifier is required")
		return
	}

	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	org, err := h.service.Update(r.Context(), ref, input)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(org)
}

// Delete handles DELETE /v1/orgs/{org}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ref := chi.URLParam(r, "org")
	if ref == "" {
		apierrors.BadRequest(w, "org identifier is required")
		return
	}

	if err := h.service.Delete(r.Context(), ref); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		apierrors.NotFound(w, err.Error())
	case errors.Is(err, ErrNameRequired):
		apierrors.ValidationError(w, "validation failed", []apierrors.FieldError{
			{Field: "name", Message: err.Error()},
		})
	case errors.Is(err, ErrInvalidMeta):
		apierrors.ValidationError(w, "validation failed", []apierrors.FieldError{
			{Field: "metadata", Message: err.Error()},
		})
	case errors.Is(err, ErrSlugConflict):
		apierrors.Conflict(w, err.Error())
	default:
		apierrors.InternalError(w, "internal server error")
	}
}
