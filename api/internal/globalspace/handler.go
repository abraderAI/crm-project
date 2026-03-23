package globalspace

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/abraderAI/crm-project/api/internal/auth"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for global space thread endpoints.
type Handler struct {
	service *Service
}

// NewHandler creates a new global space Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// resolveVisibility resolves the caller's visibility tier for support spaces.
// Returns nil when the space is not global-support or the user is unauthenticated.
// On error it writes a 500 and returns nil, true.
func (h *Handler) resolveVisibility(w http.ResponseWriter, r *http.Request, spaceSlug string) (cv *CallerVisibility, abort bool) {
	if spaceSlug != "global-support" {
		return nil, false
	}
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return nil, true
	}
	resolved, err := h.service.ResolveVisibility(r.Context(), uc.UserID)
	if err != nil {
		apierrors.InternalError(w, "failed to check permissions")
		return nil, true
	}
	return resolved, false
}

// ListThreads handles GET /v1/global-spaces/{space}/threads.
// Accepts optional query params: mine=true, org_id, limit, cursor.
// Authentication is required for support spaces; visibility scoping is enforced
// server-side for global-support.
func (h *Handler) ListThreads(w http.ResponseWriter, r *http.Request) {
	spaceSlug := chi.URLParam(r, "space")
	params := pagination.Parse(r)
	q := r.URL.Query()

	uc := auth.GetUserContext(r.Context())
	userID := ""
	if uc != nil {
		userID = uc.UserID
	}

	// Resolve visibility tier for support spaces.
	cv, abort := h.resolveVisibility(w, r, spaceSlug)
	if abort {
		return
	}

	input := ListInput{
		SpaceSlug:  spaceSlug,
		Params:     params,
		UserID:     userID,
		Mine:       q.Get("mine") == "true",
		OrgID:      q.Get("org_id"),
		Visibility: cv,
	}

	threads, pageInfo, err := h.service.ListThreads(r.Context(), input)
	if err != nil {
		apierrors.InternalError(w, "failed to list threads")
		return
	}

	response.JSON(w, http.StatusOK, response.ListResponse{
		Data:     threads,
		PageInfo: pageInfo,
	})
}

// createThreadRequest is the request body for POST /v1/global-spaces/{space}/threads.
type createThreadRequest struct {
	Title string  `json:"title"`
	Body  string  `json:"body"`
	OrgID *string `json:"org_id"`
}

// GetThread handles GET /v1/global-spaces/{space}/threads/{slug}.
// Returns the enriched thread including author email/name and org name.
// Visibility scoping is enforced for global-support.
func (h *Handler) GetThread(w http.ResponseWriter, r *http.Request) {
	spaceSlug := chi.URLParam(r, "space")
	threadSlug := chi.URLParam(r, "slug")

	cv, abort := h.resolveVisibility(w, r, spaceSlug)
	if abort {
		return
	}

	t, err := h.service.GetThread(r.Context(), spaceSlug, threadSlug, cv)
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

// updateThreadRequest is the request body for PATCH /v1/global-spaces/{space}/threads/{slug}.
type updateThreadRequest struct {
	Body       *string `json:"body"`
	Status     *string `json:"status"`
	AssignedTo *string `json:"assigned_to"`
	IsPinned   *bool   `json:"is_pinned"`
	IsHidden   *bool   `json:"is_hidden"`
	IsLocked   *bool   `json:"is_locked"`
}

// UpdateThread handles PATCH /v1/global-spaces/{space}/threads/{slug}.
// Requires authentication. Allows updating body and/or status.
// Visibility scoping is enforced for global-support.
func (h *Handler) UpdateThread(w http.ResponseWriter, r *http.Request) {
	spaceSlug := chi.URLParam(r, "space")
	threadSlug := chi.URLParam(r, "slug")

	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	cv, abort := h.resolveVisibility(w, r, spaceSlug)
	if abort {
		return
	}

	var req updateThreadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	input := UpdateInput(req)

	t, err := h.service.UpdateThread(r.Context(), spaceSlug, threadSlug, uc.UserID, input, cv)
	if err != nil {
		if err.Error() == "assignee must be a DEFT org member" {
			apierrors.ValidationError(w, err.Error(), nil)
			return
		}
		apierrors.InternalError(w, "failed to update thread")
		return
	}
	if t == nil {
		apierrors.NotFound(w, "thread not found")
		return
	}

	response.JSON(w, http.StatusOK, t)
}

// ListAttachments handles GET /v1/global-spaces/{space}/threads/{slug}/attachments.
// Returns uploads attached to the specified thread.
// Visibility scoping is enforced for global-support.
func (h *Handler) ListAttachments(w http.ResponseWriter, r *http.Request) {
	spaceSlug := chi.URLParam(r, "space")
	threadSlug := chi.URLParam(r, "slug")

	cv, abort := h.resolveVisibility(w, r, spaceSlug)
	if abort {
		return
	}

	uploads, err := h.service.ListAttachments(r.Context(), spaceSlug, threadSlug, cv)
	if err != nil {
		apierrors.InternalError(w, "failed to list attachments")
		return
	}
	if uploads == nil {
		apierrors.NotFound(w, "thread not found")
		return
	}

	response.JSON(w, http.StatusOK, uploads)
}

// maxAttachmentSize is the maximum file size accepted by the attachment upload handler.
const maxAttachmentSize = 100 << 20 // 100 MB

// UploadAttachment handles POST /v1/global-spaces/{space}/threads/{slug}/attachments.
// Accepts a multipart/form-data body with a single "file" field.
// Requires authentication. The org_id is resolved server-side.
func (h *Handler) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	spaceSlug := chi.URLParam(r, "space")
	threadSlug := chi.URLParam(r, "slug")

	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	if err := r.ParseMultipartForm(maxAttachmentSize); err != nil {
		apierrors.BadRequest(w, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		apierrors.BadRequest(w, "file field is required")
		return
	}
	defer func() { _ = file.Close() }()

	cv, abort := h.resolveVisibility(w, r, spaceSlug)
	if abort {
		return
	}

	uploaded, err := h.service.UploadAttachment(r.Context(), spaceSlug, threadSlug, uc.UserID, header.Filename, header.Size, file, cv)
	if err != nil {
		apierrors.InternalError(w, "failed to upload attachment")
		return
	}
	if uploaded == nil {
		apierrors.NotFound(w, "thread not found")
		return
	}

	response.Created(w, uploaded)
}

// ListMessages handles GET /v1/global-spaces/{space}/threads/{slug}/messages.
// Public for forum spaces; returns paginated messages.
func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	spaceSlug := chi.URLParam(r, "space")
	threadSlug := chi.URLParam(r, "slug")
	params := pagination.Parse(r)

	messages, pageInfo, err := h.service.ListMessages(r.Context(), spaceSlug, threadSlug, params)
	if err != nil {
		apierrors.InternalError(w, "failed to list messages")
		return
	}
	if messages == nil {
		apierrors.NotFound(w, "thread not found")
		return
	}

	response.JSON(w, http.StatusOK, response.ListResponse{
		Data:     messages,
		PageInfo: pageInfo,
	})
}

// CreateMessage handles POST /v1/global-spaces/{space}/threads/{slug}/messages.
// Requires authentication.
func (h *Handler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	spaceSlug := chi.URLParam(r, "space")
	threadSlug := chi.URLParam(r, "slug")

	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	var input CreateMessageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	msg, err := h.service.CreateMessage(r.Context(), spaceSlug, threadSlug, uc.UserID, input)
	if err != nil {
		switch err.Error() {
		case "body is required":
			apierrors.ValidationError(w, err.Error(), nil)
		case "thread is locked":
			apierrors.Forbidden(w, "thread is locked; new replies cannot be created")
		default:
			apierrors.InternalError(w, "failed to create message")
		}
		return
	}
	if msg == nil {
		apierrors.NotFound(w, "thread not found")
		return
	}

	response.Created(w, msg)
}

// CreateThread handles POST /v1/global-spaces/{space}/threads.
// Requires authentication. Tier enforcement is handled client-side.
func (h *Handler) CreateThread(w http.ResponseWriter, r *http.Request) {
	spaceSlug := chi.URLParam(r, "space")

	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	var req createThreadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	if req.Title == "" {
		apierrors.ValidationError(w, "title is required", nil)
		return
	}

	t, err := h.service.CreateThread(r.Context(), spaceSlug, uc.UserID, CreateInput(req))
	if err != nil {
		switch err.Error() {
		case "title is required":
			apierrors.ValidationError(w, err.Error(), nil)
		case "board is locked":
			apierrors.Forbidden(w, "board is locked; new threads cannot be created")
		default:
			apierrors.InternalError(w, "failed to create thread")
		}
		return
	}

	response.Created(w, t)
}
