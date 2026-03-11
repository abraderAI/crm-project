package thread

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/abraderAI/crm-project/api/internal/auth"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// BoardLookup is a function that checks if a board is locked.
type BoardLookup func(ctx interface{}, spaceID, boardID string) (bool, error)

// Handler provides HTTP handlers for Thread operations.
type Handler struct {
	service *Service
}

// NewHandler creates a new Thread handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Create handles POST .../boards/{board}/threads.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "board")

	uc := auth.GetUserContext(r.Context())
	authorID := ""
	if uc != nil {
		authorID = uc.UserID
	}

	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	// The handler receives boardLocked from the route context.
	// For simplicity, we pass false and let the handler wrapper check board lock.
	t, err := h.service.Create(r.Context(), boardID, authorID, false, input)
	if err != nil {
		if err.Error() == "board is locked" {
			apierrors.Forbidden(w, "board is locked; new threads cannot be created")
			return
		}
		apierrors.ValidationError(w, err.Error(), nil)
		return
	}

	response.Created(w, t)
}

// CreateWithBoardCheck creates a thread after checking if the board is locked.
func (h *Handler) CreateWithBoardCheck(boardLocked bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		boardID := chi.URLParam(r, "board")

		uc := auth.GetUserContext(r.Context())
		authorID := ""
		if uc != nil {
			authorID = uc.UserID
		}

		var input CreateInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			apierrors.BadRequest(w, "invalid request body")
			return
		}

		t, err := h.service.Create(r.Context(), boardID, authorID, boardLocked, input)
		if err != nil {
			if err.Error() == "board is locked" {
				apierrors.Forbidden(w, "board is locked; new threads cannot be created")
				return
			}
			apierrors.ValidationError(w, err.Error(), nil)
			return
		}

		response.Created(w, t)
	}
}

// List handles GET .../boards/{board}/threads with metadata filtering.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "board")
	params := pagination.Parse(r)

	// Parse metadata filters from query params: ?metadata[key]=value or ?metadata[key][op]=value
	filters := parseMetadataFilters(r)

	listParams := ListParams{
		Params:  params,
		Filters: filters,
	}

	threads, pageInfo, err := h.service.List(r.Context(), boardID, listParams)
	if err != nil {
		apierrors.InternalError(w, "failed to list threads")
		return
	}

	response.JSON(w, http.StatusOK, response.ListResponse{
		Data:     threads,
		PageInfo: pageInfo,
	})
}

// Get handles GET .../boards/{board}/threads/{thread}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "board")
	idOrSlug := chi.URLParam(r, "thread")

	t, err := h.service.Get(r.Context(), boardID, idOrSlug)
	if err != nil {
		apierrors.InternalError(w, "failed to get thread")
		return
	}
	if t == nil {
		apierrors.NotFound(w, "thread not found")
		return
	}

	response.JSON(w, http.StatusOK, t)
}

// Update handles PATCH .../boards/{board}/threads/{thread}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "board")
	idOrSlug := chi.URLParam(r, "thread")

	uc := auth.GetUserContext(r.Context())
	editorID := ""
	if uc != nil {
		editorID = uc.UserID
	}

	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	t, err := h.service.Update(r.Context(), boardID, idOrSlug, editorID, input)
	if err != nil {
		apierrors.ValidationError(w, err.Error(), nil)
		return
	}
	if t == nil {
		apierrors.NotFound(w, "thread not found")
		return
	}

	response.JSON(w, http.StatusOK, t)
}

// Delete handles DELETE .../boards/{board}/threads/{thread}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "board")
	idOrSlug := chi.URLParam(r, "thread")

	if err := h.service.Delete(r.Context(), boardID, idOrSlug); err != nil {
		if err.Error() == "not found" {
			apierrors.NotFound(w, "thread not found")
			return
		}
		apierrors.InternalError(w, "failed to delete thread")
		return
	}

	response.NoContent(w)
}

// Pin handles POST .../threads/{thread}/pin.
func (h *Handler) Pin(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "board")
	idOrSlug := chi.URLParam(r, "thread")

	t, err := h.service.SetPin(r.Context(), boardID, idOrSlug, true)
	if err != nil {
		apierrors.InternalError(w, "failed to pin thread")
		return
	}
	if t == nil {
		apierrors.NotFound(w, "thread not found")
		return
	}
	response.JSON(w, http.StatusOK, t)
}

// Unpin handles POST .../threads/{thread}/unpin.
func (h *Handler) Unpin(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "board")
	idOrSlug := chi.URLParam(r, "thread")

	t, err := h.service.SetPin(r.Context(), boardID, idOrSlug, false)
	if err != nil {
		apierrors.InternalError(w, "failed to unpin thread")
		return
	}
	if t == nil {
		apierrors.NotFound(w, "thread not found")
		return
	}
	response.JSON(w, http.StatusOK, t)
}

// Lock handles POST .../threads/{thread}/lock.
func (h *Handler) Lock(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "board")
	idOrSlug := chi.URLParam(r, "thread")

	t, err := h.service.SetLock(r.Context(), boardID, idOrSlug, true)
	if err != nil {
		apierrors.InternalError(w, "failed to lock thread")
		return
	}
	if t == nil {
		apierrors.NotFound(w, "thread not found")
		return
	}
	response.JSON(w, http.StatusOK, t)
}

// Unlock handles POST .../threads/{thread}/unlock.
func (h *Handler) Unlock(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "board")
	idOrSlug := chi.URLParam(r, "thread")

	t, err := h.service.SetLock(r.Context(), boardID, idOrSlug, false)
	if err != nil {
		apierrors.InternalError(w, "failed to unlock thread")
		return
	}
	if t == nil {
		apierrors.NotFound(w, "thread not found")
		return
	}
	response.JSON(w, http.StatusOK, t)
}

// parseMetadataFilters extracts metadata filters from query params.
// Supports: ?metadata[status]=open and ?metadata[priority][gt]=3
func parseMetadataFilters(r *http.Request) []MetadataFilter {
	var filters []MetadataFilter
	for key, values := range r.URL.Query() {
		if !strings.HasPrefix(key, "metadata[") {
			continue
		}
		if len(values) == 0 {
			continue
		}
		// Parse "metadata[key]" or "metadata[key][op]"
		inner := key[len("metadata["):]
		inner = strings.TrimSuffix(inner, "]")
		parts := strings.Split(inner, "][")

		if len(parts) == 1 {
			// Simple equality: ?metadata[status]=open
			filters = append(filters, MetadataFilter{
				Path:     "$." + parts[0],
				Operator: "eq",
				Value:    values[0],
			})
		} else if len(parts) == 2 {
			// Comparison: ?metadata[priority][gt]=3
			filters = append(filters, MetadataFilter{
				Path:     "$." + parts[0],
				Operator: parts[1],
				Value:    values[0],
			})
		}
	}
	return filters
}
