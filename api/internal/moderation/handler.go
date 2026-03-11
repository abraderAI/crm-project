package moderation

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/abraderAI/crm-project/api/internal/auth"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for moderation operations.
type Handler struct {
	service *Service
}

// NewHandler creates a new Moderation handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// CreateFlag handles POST /v1/orgs/{org}/flags.
func (h *Handler) CreateFlag(w http.ResponseWriter, r *http.Request) {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	var input FlagInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	flag, err := h.service.CreateFlag(r.Context(), uc.UserID, input)
	if err != nil {
		switch err.Error() {
		case "thread not found":
			apierrors.NotFound(w, "thread not found")
		case "thread_id is required", "reason is required":
			apierrors.ValidationError(w, err.Error(), nil)
		default:
			apierrors.InternalError(w, "failed to create flag")
		}
		return
	}

	response.Created(w, flag)
}

// ListFlags handles GET /v1/orgs/{org}/flags.
func (h *Handler) ListFlags(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	params := pagination.Parse(r)

	flags, pageInfo, err := h.service.ListOrgFlags(r.Context(), orgID, params)
	if err != nil {
		apierrors.InternalError(w, "failed to list flags")
		return
	}

	response.JSON(w, http.StatusOK, response.ListResponse{
		Data:     flags,
		PageInfo: pageInfo,
	})
}

// ResolveFlag handles POST /v1/orgs/{org}/flags/{flag}/resolve.
func (h *Handler) ResolveFlag(w http.ResponseWriter, r *http.Request) {
	flagID := chi.URLParam(r, "flag")

	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	flag, err := h.service.ResolveFlag(r.Context(), flagID, uc.UserID)
	if err != nil {
		switch err.Error() {
		case "flag not found":
			apierrors.NotFound(w, "flag not found")
		default:
			apierrors.BadRequest(w, err.Error())
		}
		return
	}

	response.JSON(w, http.StatusOK, flag)
}

// DismissFlag handles POST /v1/orgs/{org}/flags/{flag}/dismiss.
func (h *Handler) DismissFlag(w http.ResponseWriter, r *http.Request) {
	flagID := chi.URLParam(r, "flag")

	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	flag, err := h.service.DismissFlag(r.Context(), flagID, uc.UserID)
	if err != nil {
		switch err.Error() {
		case "flag not found":
			apierrors.NotFound(w, "flag not found")
		default:
			apierrors.BadRequest(w, err.Error())
		}
		return
	}

	response.JSON(w, http.StatusOK, flag)
}

// MoveThread handles POST .../threads/{thread}/move.
func (h *Handler) MoveThread(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "thread")

	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	var input MoveInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	thread, err := h.service.MoveThread(r.Context(), threadID, uc.UserID, input)
	if err != nil {
		switch err.Error() {
		case "thread not found", "target board not found":
			apierrors.NotFound(w, err.Error())
		case "target_board_id is required", "thread is already in the target board":
			apierrors.ValidationError(w, err.Error(), nil)
		default:
			apierrors.InternalError(w, "failed to move thread")
		}
		return
	}

	response.JSON(w, http.StatusOK, thread)
}

// MergeThread handles POST .../threads/{thread}/merge.
func (h *Handler) MergeThread(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "thread")

	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	var input MergeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	thread, err := h.service.MergeThread(r.Context(), threadID, uc.UserID, input)
	if err != nil {
		switch err.Error() {
		case "source thread not found", "target thread not found":
			apierrors.NotFound(w, err.Error())
		case "target_thread_id is required", "cannot merge a thread into itself":
			apierrors.ValidationError(w, err.Error(), nil)
		default:
			apierrors.InternalError(w, "failed to merge thread")
		}
		return
	}

	response.JSON(w, http.StatusOK, thread)
}

// HideThread handles POST .../threads/{thread}/hide.
func (h *Handler) HideThread(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "thread")

	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	thread, err := h.service.HideThread(r.Context(), threadID, uc.UserID)
	if err != nil {
		if err.Error() == "thread not found" {
			apierrors.NotFound(w, "thread not found")
			return
		}
		apierrors.InternalError(w, "failed to hide thread")
		return
	}

	response.JSON(w, http.StatusOK, thread)
}

// UnhideThread handles POST .../threads/{thread}/unhide.
func (h *Handler) UnhideThread(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "thread")

	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	thread, err := h.service.UnhideThread(r.Context(), threadID, uc.UserID)
	if err != nil {
		if err.Error() == "thread not found" {
			apierrors.NotFound(w, "thread not found")
			return
		}
		apierrors.InternalError(w, "failed to unhide thread")
		return
	}

	response.JSON(w, http.StatusOK, thread)
}
