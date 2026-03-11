package notification

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

// Handler provides HTTP handlers for notification endpoints.
type Handler struct {
	repo *Repository
}

// NewHandler creates a new notification handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// List handles GET /v1/notifications — returns paginated notifications for the authenticated user.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	params := pagination.Parse(r)
	notifs, pageInfo, err := h.repo.ListByUser(r.Context(), uc.UserID, params)
	if err != nil {
		apierrors.InternalError(w, "failed to list notifications")
		return
	}

	// Include unread count in response.
	unread, _ := h.repo.CountUnread(r.Context(), uc.UserID)

	response.JSON(w, http.StatusOK, map[string]any{
		"data":         notifs,
		"page_info":    pageInfo,
		"unread_count": unread,
	})
}

// MarkRead handles PATCH /v1/notifications/{id}/read.
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	id := chi.URLParam(r, "id")
	if err := h.repo.MarkRead(r.Context(), id, uc.UserID); err != nil {
		if err == ErrNotFound {
			apierrors.NotFound(w, "notification not found")
			return
		}
		apierrors.InternalError(w, "failed to mark notification read")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"id": id, "is_read": true})
}

// MarkAllRead handles POST /v1/notifications/mark-all-read.
func (h *Handler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	count, err := h.repo.MarkAllRead(r.Context(), uc.UserID)
	if err != nil {
		apierrors.InternalError(w, "failed to mark all notifications read")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"marked_read": count})
}

// PreferenceInput represents a single notification preference update.
type PreferenceInput struct {
	EventType string `json:"event_type"`
	Channel   string `json:"channel"`
	Enabled   bool   `json:"enabled"`
}

// PreferencesInput represents a batch of notification preference updates.
type PreferencesInput struct {
	Preferences []PreferenceInput `json:"preferences"`
}

// GetPreferences handles GET /v1/notifications/preferences.
func (h *Handler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	prefs, err := h.repo.GetPreferences(r.Context(), uc.UserID)
	if err != nil {
		apierrors.InternalError(w, "failed to get preferences")
		return
	}

	// Also get digest schedule.
	digest, _ := h.repo.GetDigestSchedule(r.Context(), uc.UserID)

	response.JSON(w, http.StatusOK, map[string]any{
		"preferences": prefs,
		"digest":      digest,
	})
}

// UpdatePreferences handles PUT /v1/notifications/preferences.
func (h *Handler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	var input struct {
		Preferences []PreferenceInput `json:"preferences"`
		Digest      *struct {
			Frequency string `json:"frequency"`
			Enabled   bool   `json:"enabled"`
		} `json:"digest,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	// Update preferences.
	for _, p := range input.Preferences {
		if p.EventType == "" || p.Channel == "" {
			continue
		}
		pref := &models.NotificationPreference{
			UserID:    uc.UserID,
			EventType: p.EventType,
			Channel:   p.Channel,
			Enabled:   p.Enabled,
		}
		if err := h.repo.UpsertPreference(r.Context(), pref); err != nil {
			apierrors.InternalError(w, "failed to update preference")
			return
		}
	}

	// Update digest schedule if provided.
	if input.Digest != nil {
		freq := input.Digest.Frequency
		if freq != "daily" && freq != "weekly" {
			freq = "daily"
		}
		sched := &models.DigestSchedule{
			UserID:    uc.UserID,
			Frequency: freq,
			Enabled:   input.Digest.Enabled,
		}
		if err := h.repo.UpsertDigestSchedule(r.Context(), sched); err != nil {
			apierrors.InternalError(w, "failed to update digest schedule")
			return
		}
	}

	response.JSON(w, http.StatusOK, map[string]any{"status": "updated"})
}
