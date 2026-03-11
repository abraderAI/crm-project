package message

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/abraderAI/crm-project/api/internal/auth"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for Message operations.
type Handler struct {
	service *Service
}

// NewHandler creates a new Message handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Create handles POST .../threads/{thread}/messages.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "thread")

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

	// threadLocked is passed as false; the router checks board/thread locks upstream.
	msg, err := h.service.Create(r.Context(), threadID, authorID, false, input)
	if err != nil {
		if err.Error() == "thread is locked" {
			apierrors.Forbidden(w, "thread is locked; new messages cannot be created")
			return
		}
		apierrors.ValidationError(w, err.Error(), nil)
		return
	}

	response.Created(w, msg)
}

// List handles GET .../threads/{thread}/messages.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "thread")
	params := pagination.Parse(r)

	messages, pageInfo, err := h.service.List(r.Context(), threadID, params)
	if err != nil {
		apierrors.InternalError(w, "failed to list messages")
		return
	}

	response.JSON(w, http.StatusOK, response.ListResponse{
		Data:     messages,
		PageInfo: pageInfo,
	})
}

// Get handles GET .../threads/{thread}/messages/{message}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "thread")
	msgID := chi.URLParam(r, "message")

	msg, err := h.service.Get(r.Context(), threadID, msgID)
	if err != nil {
		apierrors.InternalError(w, "failed to get message")
		return
	}
	if msg == nil {
		apierrors.NotFound(w, "message not found")
		return
	}

	response.JSON(w, http.StatusOK, msg)
}

// Update handles PATCH .../threads/{thread}/messages/{message}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "thread")
	msgID := chi.URLParam(r, "message")

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

	msg, err := h.service.Update(r.Context(), threadID, msgID, editorID, input)
	if err != nil {
		if err.Error() == "only the author can update this message" {
			apierrors.Forbidden(w, err.Error())
			return
		}
		apierrors.ValidationError(w, err.Error(), nil)
		return
	}
	if msg == nil {
		apierrors.NotFound(w, "message not found")
		return
	}

	response.JSON(w, http.StatusOK, msg)
}

// Delete handles DELETE .../threads/{thread}/messages/{message}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "thread")
	msgID := chi.URLParam(r, "message")

	if err := h.service.Delete(r.Context(), threadID, msgID); err != nil {
		if err.Error() == "not found" {
			apierrors.NotFound(w, "message not found")
			return
		}
		apierrors.InternalError(w, "failed to delete message")
		return
	}

	response.NoContent(w)
}
