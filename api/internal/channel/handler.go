package channel

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for the channel config, DLQ, and health APIs.
type Handler struct {
	service *Service
}

// NewHandler creates a new channel Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// requireOrgAdmin checks that the authenticated user holds admin or owner role in the org,
// OR is an active platform admin. It writes an error response and returns false when the
// check fails.
func (h *Handler) requireOrgAdmin(w http.ResponseWriter, r *http.Request, orgID string) bool {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return false
	}

	// Platform admins can manage channel config for any org.
	isPlatformAdmin, err := h.service.IsPlatformAdmin(r.Context(), uc.UserID)
	if err != nil {
		apierrors.InternalError(w, "failed to check permissions")
		return false
	}
	if isPlatformAdmin {
		return true
	}

	isAdmin, err := h.service.IsOrgAdmin(r.Context(), orgID, uc.UserID)
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

// GetConfig handles GET /v1/orgs/{org}/channels/{type}.
// Returns the channel config for the org, or a default empty config when none exists.
func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	channelType := models.ChannelType(chi.URLParam(r, "type"))
	if !channelType.IsValid() {
		apierrors.NotFound(w, "unknown channel type")
		return
	}

	cfg, err := h.service.GetConfig(r.Context(), orgID, channelType)
	if err != nil {
		apierrors.InternalError(w, "failed to get channel config")
		return
	}
	if cfg == nil {
		// Return a default empty config so callers always get a valid structure.
		response.JSON(w, http.StatusOK, &models.ChannelConfig{
			OrgID:       orgID,
			ChannelType: channelType,
			Settings:    "{}",
			Enabled:     false,
		})
		return
	}
	response.JSON(w, http.StatusOK, cfg)
}

// PutConfig handles PUT /v1/orgs/{org}/channels/{type}. Admin-only.
// Creates or updates the channel configuration for the org.
func (h *Handler) PutConfig(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	channelType := models.ChannelType(chi.URLParam(r, "type"))
	if !channelType.IsValid() {
		apierrors.NotFound(w, "unknown channel type")
		return
	}
	if !h.requireOrgAdmin(w, r, orgID) {
		return
	}

	var input PutConfigInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	cfg, err := h.service.UpsertConfig(r.Context(), orgID, channelType, input)
	if err != nil {
		apierrors.ValidationError(w, err.Error(), nil)
		return
	}
	response.JSON(w, http.StatusOK, cfg)
}

// ListDLQ handles GET /v1/orgs/{org}/channels/dlq. Admin-only.
// Supports optional query params: channel_type, status.
func (h *Handler) ListDLQ(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	if !h.requireOrgAdmin(w, r, orgID) {
		return
	}

	channelType := models.ChannelType(r.URL.Query().Get("channel_type"))
	status := models.DLQStatus(r.URL.Query().Get("status"))
	params := pagination.Parse(r)

	evts, pageInfo, err := h.service.ListDLQEvents(r.Context(), orgID, ListDLQEventsInput{
		ChannelType: channelType,
		Status:      status,
		Params:      params,
	})
	if err != nil {
		apierrors.InternalError(w, "failed to list DLQ events")
		return
	}
	response.JSON(w, http.StatusOK, response.ListResponse{Data: evts, PageInfo: pageInfo})
}

// RetryDLQ handles POST /v1/orgs/{org}/channels/dlq/{id}/retry. Admin-only.
// Marks the DLQ event as retrying.
func (h *Handler) RetryDLQ(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	id := chi.URLParam(r, "id")
	if !h.requireOrgAdmin(w, r, orgID) {
		return
	}

	evt, err := h.service.RetryDLQEvent(r.Context(), orgID, id)
	if err != nil {
		apierrors.InternalError(w, "failed to retry DLQ event")
		return
	}
	if evt == nil {
		apierrors.NotFound(w, "DLQ event not found")
		return
	}
	response.JSON(w, http.StatusOK, evt)
}

// DismissDLQ handles POST /v1/orgs/{org}/channels/dlq/{id}/dismiss. Admin-only.
// Marks the DLQ event as dismissed.
func (h *Handler) DismissDLQ(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	id := chi.URLParam(r, "id")
	if !h.requireOrgAdmin(w, r, orgID) {
		return
	}

	evt, err := h.service.DismissDLQEvent(r.Context(), orgID, id)
	if err != nil {
		apierrors.InternalError(w, "failed to dismiss DLQ event")
		return
	}
	if evt == nil {
		apierrors.NotFound(w, "DLQ event not found")
		return
	}
	response.JSON(w, http.StatusOK, evt)
}

// GetHealth handles GET /v1/orgs/{org}/channels/health.
// Returns per-channel health status, last event time, and 24-hour error count.
func (h *Handler) GetHealth(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")

	health, err := h.service.GetHealth(r.Context(), orgID)
	if err != nil {
		apierrors.InternalError(w, "failed to get channel health")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"channels": health})
}
