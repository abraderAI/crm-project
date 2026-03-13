package admin

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/abraderAI/crm-project/api/internal/audit"
	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// --- User Management ---

// ListUsers handles GET /v1/admin/users.
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := UserListParams{
		Params:  pagination.Parse(r),
		Email:   q.Get("email"),
		Name:    q.Get("name"),
		UserID:  q.Get("user_id"),
		OrgSlug: q.Get("org_slug"),
	}

	if q.Get("is_banned") == "true" {
		v := true
		params.IsBanned = &v
	} else if q.Get("is_banned") == "false" {
		v := false
		params.IsBanned = &v
	}
	if v := q.Get("seen_after"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			params.SeenAfter = &t
		}
	}
	if v := q.Get("seen_before"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			params.SeenBefore = &t
		}
	}

	users, pageInfo, err := h.service.ListUsers(r.Context(), params)
	if err != nil {
		apierrors.InternalError(w, "failed to list users")
		return
	}

	response.JSON(w, http.StatusOK, response.ListResponse{
		Data:     users,
		PageInfo: pageInfo,
	})
}

// GetUser handles GET /v1/admin/users/{user_id}.
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		apierrors.BadRequest(w, "user_id is required")
		return
	}

	detail, err := h.service.GetUser(r.Context(), userID)
	if err != nil {
		apierrors.InternalError(w, "failed to get user")
		return
	}
	if detail == nil {
		apierrors.NotFound(w, "user not found")
		return
	}

	response.JSON(w, http.StatusOK, detail)
}

// BanUser handles POST /v1/admin/users/{user_id}/ban.
func (h *Handler) BanUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		apierrors.BadRequest(w, "user_id is required")
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	uc := auth.GetUserContext(r.Context())
	bannedBy := ""
	if uc != nil {
		bannedBy = uc.UserID
	}

	if err := h.service.BanUser(r.Context(), userID, body.Reason, bannedBy); err != nil {
		apierrors.InternalError(w, "failed to ban user")
		return
	}

	// Audit log.
	audit.CreateAuditEntry(r.Context(), h.auditService, "ban", "user", userID, nil,
		map[string]string{"reason": body.Reason, "banned_by": bannedBy})

	response.JSON(w, http.StatusOK, map[string]string{
		"status":  "banned",
		"user_id": userID,
	})
}

// UnbanUser handles POST /v1/admin/users/{user_id}/unban.
func (h *Handler) UnbanUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		apierrors.BadRequest(w, "user_id is required")
		return
	}

	if err := h.service.UnbanUser(r.Context(), userID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			apierrors.NotFound(w, "user not found")
			return
		}
		apierrors.InternalError(w, "failed to unban user")
		return
	}

	// Audit log.
	audit.CreateAuditEntry(r.Context(), h.auditService, "unban", "user", userID, nil, nil)

	response.JSON(w, http.StatusOK, map[string]string{
		"status":  "unbanned",
		"user_id": userID,
	})
}

// PurgeUser handles DELETE /v1/admin/users/{user_id}/purge.
func (h *Handler) PurgeUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		apierrors.BadRequest(w, "user_id is required")
		return
	}

	// Delegate to GDPR purge.
	if err := h.gdprService.PurgeUser(r.Context(), userID); err != nil {
		apierrors.InternalError(w, "failed to purge user data")
		return
	}

	// Also anonymize/delete the user shadow.
	h.service.db.Where("clerk_user_id = ?", userID).Delete(&models.UserShadow{})

	// Audit log.
	audit.CreateAuditEntry(r.Context(), h.auditService, "purge", "user", userID, nil, nil)

	response.JSON(w, http.StatusOK, map[string]string{
		"status":  "purged",
		"user_id": userID,
		"message": "all user PII has been removed and audit logs anonymized",
	})
}
