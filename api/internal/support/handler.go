package support

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Ensure models import is used (for ListUnclaimedTickets empty-slice init).
var _ = models.Thread{}

// Handler provides HTTP handlers for support ticket entry endpoints.
type Handler struct {
	service *Service
}

// NewHandler creates a new support Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// resolveCallerVisibility determines whether the authenticated user is a DEFT
// member and returns the boolean. On error it writes a 500 and returns false, true.
func (h *Handler) resolveCallerVisibility(w http.ResponseWriter, r *http.Request) (isDeft bool, abort bool) {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return false, true
	}
	deft, err := h.service.IsDeftMember(r.Context(), uc.UserID)
	if err != nil {
		apierrors.InternalError(w, "failed to check permissions")
		return false, true
	}
	return deft, false
}

// ListEntries handles GET /v1/support/tickets/{slug}/entries.
// Returns entries filtered to what the caller is permitted to see.
// Non-DEFT callers also receive their own draft entries so they can edit
// and publish them later.
func (h *Handler) ListEntries(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	isDeft, err := h.service.IsDeftMember(r.Context(), uc.UserID)
	if err != nil {
		apierrors.InternalError(w, "failed to check permissions")
		return
	}

	entries, err := h.service.ListEntries(r.Context(), slug, isDeft, uc.UserID)
	if err != nil {
		apierrors.InternalError(w, "failed to list entries")
		return
	}
	if entries == nil {
		apierrors.NotFound(w, "ticket not found")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"data": entries})
}

// createEntryRequest is the request body for creating a new ticket entry.
type createEntryRequest struct {
	Type       models.MessageType `json:"type"`
	Body       string             `json:"body"`
	IsDeftOnly bool               `json:"is_deft_only"`
}

// CreateEntry handles POST /v1/support/tickets/{slug}/entries.
func (h *Handler) CreateEntry(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	isDeft, abort := h.resolveCallerVisibility(w, r)
	if abort {
		return
	}

	var req createEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	msg, err := h.service.CreateEntry(r.Context(), slug, uc.UserID, isDeft, CreateEntryInput(req))
	if err != nil {
		switch {
		case errors.Is(err, ErrForbidden):
			apierrors.Forbidden(w, "only DEFT members may create this entry type")
		default:
			apierrors.ValidationError(w, err.Error(), nil)
		}
		return
	}
	if msg == nil {
		apierrors.NotFound(w, "ticket not found")
		return
	}

	response.Created(w, msg)
}

// updateEntryRequest is the request body for updating a draft entry body.
type updateEntryRequest struct {
	Body string `json:"body"`
}

// UpdateEntry handles PATCH /v1/support/tickets/{slug}/entries/{id}.
// Only mutable (draft) entries may be updated.
func (h *Handler) UpdateEntry(w http.ResponseWriter, r *http.Request) {
	entryID := chi.URLParam(r, "id")

	var req updateEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	msg, err := h.service.UpdateEntryBody(r.Context(), entryID, req.Body)
	if err != nil {
		if errors.Is(err, ErrImmutable) {
			apierrors.Forbidden(w, err.Error())
			return
		}
		apierrors.InternalError(w, "failed to update entry")
		return
	}
	if msg == nil {
		apierrors.NotFound(w, "entry not found")
		return
	}

	response.JSON(w, http.StatusOK, msg)
}

// PublishEntry handles POST /v1/support/tickets/{slug}/entries/{id}/publish.
// Promotes a draft entry to agent_reply, making it visible to the customer.
func (h *Handler) PublishEntry(w http.ResponseWriter, r *http.Request) {
	entryID := chi.URLParam(r, "id")

	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	// Only DEFT members can publish.
	isDeft, abort := h.resolveCallerVisibility(w, r)
	if abort {
		return
	}
	if !isDeft {
		apierrors.Forbidden(w, "only DEFT members may publish entries")
		return
	}

	msg, err := h.service.PublishDraft(r.Context(), entryID, uc.UserID)
	if err != nil {
		if errors.Is(err, ErrNotDraft) {
			apierrors.ValidationError(w, err.Error(), nil)
			return
		}
		apierrors.InternalError(w, "failed to publish entry")
		return
	}
	if msg == nil {
		apierrors.NotFound(w, "entry not found")
		return
	}

	response.JSON(w, http.StatusOK, msg)
}

// deftVisibilityRequest is the request body for toggling DEFT-only visibility.
type deftVisibilityRequest struct {
	IsDeftOnly bool `json:"is_deft_only"`
}

// SetDeftVisibility handles PATCH /v1/support/tickets/{slug}/entries/{id}/deft-visibility.
// Toggles the DEFT-only flag, instantly hiding or showing the entry to customers.
func (h *Handler) SetDeftVisibility(w http.ResponseWriter, r *http.Request) {
	entryID := chi.URLParam(r, "id")

	isDeft, abort := h.resolveCallerVisibility(w, r)
	if abort {
		return
	}

	var req deftVisibilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	msg, err := h.service.SetDeftVisibility(r.Context(), entryID, req.IsDeftOnly, isDeft)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			apierrors.Forbidden(w, err.Error())
			return
		}
		apierrors.InternalError(w, "failed to update visibility")
		return
	}
	if msg == nil {
		apierrors.NotFound(w, "entry not found")
		return
	}

	response.JSON(w, http.StatusOK, msg)
}

// ListDeftMembers handles GET /v1/support/deft-members.
// Returns all active DEFT org members with display name and email.
func (h *Handler) ListDeftMembers(w http.ResponseWriter, r *http.Request) {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	// Only DEFT members may list the DEFT team.
	isDeft, abort := h.resolveCallerVisibility(w, r)
	if abort {
		return
	}
	if !isDeft {
		apierrors.Forbidden(w, "only DEFT members may list the team")
		return
	}

	members, err := h.service.ListDeftMembers(r.Context())
	if err != nil {
		apierrors.InternalError(w, "failed to list DEFT members")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"data": members})
}

// ListUnclaimedTickets handles GET /v1/support/unclaimed-tickets.
// Returns tickets where contact_email matches the caller's email but
// author_id does not (orphaned tickets awaiting claim).
func (h *Handler) ListUnclaimedTickets(w http.ResponseWriter, r *http.Request) {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	tickets, err := h.service.ListUnclaimedTickets(r.Context(), uc.UserID)
	if err != nil {
		apierrors.InternalError(w, "failed to list unclaimed tickets")
		return
	}
	if tickets == nil {
		tickets = []models.Thread{}
	}

	response.JSON(w, http.StatusOK, map[string]any{"data": tickets})
}

// claimTicketsRequest is the request body for POST /v1/support/claim-tickets.
type claimTicketsRequest struct {
	TicketIDs []string `json:"ticket_ids"`
}

// ClaimTickets handles POST /v1/support/claim-tickets.
// Transfers ownership of orphaned tickets to the authenticated caller.
func (h *Handler) ClaimTickets(w http.ResponseWriter, r *http.Request) {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	var req claimTicketsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	if len(req.TicketIDs) == 0 {
		apierrors.BadRequest(w, "ticket_ids is required")
		return
	}

	claimed, err := h.service.ClaimTickets(r.Context(), uc.UserID, req.TicketIDs)
	if err != nil {
		if errors.Is(err, ErrEmailMismatch) {
			apierrors.Forbidden(w, err.Error())
			return
		}
		apierrors.InternalError(w, "failed to claim tickets")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"claimed": claimed})
}

// notifPrefRequest is the request body for updating notification detail level.
type notifPrefRequest struct {
	NotificationDetailLevel string `json:"notification_detail_level"`
}

// SetNotificationPref handles PATCH /v1/support/tickets/{slug}/notifications.
// Lets the ticket owner set whether notification emails include agent reply content.
func (h *Handler) SetNotificationPref(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var req notifPrefRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	if err := h.service.SetNotificationDetailLevel(r.Context(), slug, req.NotificationDetailLevel); err != nil {
		apierrors.ValidationError(w, err.Error(), nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
