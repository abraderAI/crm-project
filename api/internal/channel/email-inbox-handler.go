package channel

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// InboxRestarter is satisfied by email.InboxWatcher. Using an interface here
// prevents an import cycle between the channel and channel/email packages.
type InboxRestarter interface {
	// RestartInbox stops the IDLE manager for the inbox (if running) and starts
	// a new one when inbox.Enabled is true.
	RestartInbox(inbox models.EmailInbox)
}

// EmailInboxHandler provides HTTP handlers for managing email inboxes.
// All routes require the caller to hold admin or owner role in the org,
// or be a platform admin.
type EmailInboxHandler struct {
	service    *EmailInboxService
	channelSvc *Service // used for IsPlatformAdmin / IsOrgAdmin checks
	watcher    InboxRestarter
}

// NewEmailInboxHandler creates a new EmailInboxHandler.
// channelSvc provides the auth checks; watcher may be nil in tests.
func NewEmailInboxHandler(service *EmailInboxService, channelSvc *Service, watcher InboxRestarter) *EmailInboxHandler {
	return &EmailInboxHandler{service: service, channelSvc: channelSvc, watcher: watcher}
}

// ListInboxes handles GET /v1/orgs/{org}/channels/email/inboxes.
func (h *EmailInboxHandler) ListInboxes(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	if !h.requireAdmin(w, r, orgID) {
		return
	}

	inboxes, err := h.service.List(r.Context(), orgID)
	if err != nil {
		apierrors.InternalError(w, "failed to list email inboxes")
		return
	}
	response.JSON(w, http.StatusOK, response.ListResponse{Data: inboxes})
}

// CreateInbox handles POST /v1/orgs/{org}/channels/email/inboxes.
func (h *EmailInboxHandler) CreateInbox(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	if !h.requireAdmin(w, r, orgID) {
		return
	}

	var input CreateInboxInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	inbox, err := h.service.Create(r.Context(), orgID, input)
	if err != nil {
		apierrors.ValidationError(w, err.Error(), nil)
		return
	}

	if h.watcher != nil {
		h.watcher.RestartInbox(*inbox)
	}
	response.JSON(w, http.StatusCreated, inbox)
}

// UpdateInbox handles PUT /v1/orgs/{org}/channels/email/inboxes/{id}.
func (h *EmailInboxHandler) UpdateInbox(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	id := chi.URLParam(r, "id")
	if !h.requireAdmin(w, r, orgID) {
		return
	}

	var input UpdateInboxInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	inbox, err := h.service.Update(r.Context(), orgID, id, input)
	if err != nil {
		apierrors.ValidationError(w, err.Error(), nil)
		return
	}
	if inbox == nil {
		apierrors.NotFound(w, "email inbox not found")
		return
	}

	if h.watcher != nil {
		// Fetch the full inbox (with real password) to pass to the watcher.
		raw, _ := h.service.repo.FindByID(r.Context(), id)
		if raw != nil {
			h.watcher.RestartInbox(*raw)
		}
	}
	response.JSON(w, http.StatusOK, inbox)
}

// DeleteInbox handles DELETE /v1/orgs/{org}/channels/email/inboxes/{id}.
func (h *EmailInboxHandler) DeleteInbox(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	id := chi.URLParam(r, "id")
	if !h.requireAdmin(w, r, orgID) {
		return
	}

	deleted, err := h.service.Delete(r.Context(), orgID, id)
	if err != nil {
		apierrors.InternalError(w, "failed to delete email inbox")
		return
	}
	if !deleted {
		apierrors.NotFound(w, "email inbox not found")
		return
	}

	if h.watcher != nil {
		// Deregister the deleted inbox by passing a disabled stub.
		h.watcher.RestartInbox(models.EmailInbox{BaseModel: models.BaseModel{ID: id}, Enabled: false})
	}
	w.WriteHeader(http.StatusNoContent)
}

// requireAdmin enforces that the caller is a platform admin or holds admin/owner
// role in the specified org. Returns false and writes an error response on failure.
func (h *EmailInboxHandler) requireAdmin(w http.ResponseWriter, r *http.Request, orgID string) bool {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return false
	}

	isPlatform, err := h.channelSvc.IsPlatformAdmin(r.Context(), uc.UserID)
	if err != nil {
		apierrors.InternalError(w, "failed to check permissions")
		return false
	}
	if isPlatform {
		return true
	}

	isAdmin, err := h.channelSvc.IsOrgAdmin(r.Context(), orgID, uc.UserID)
	if err != nil {
		apierrors.InternalError(w, "failed to check permissions")
		return false
	}
	if !isAdmin {
		apierrors.Forbidden(w, "org admin or owner role required")
		return false
	}
	return true
}
