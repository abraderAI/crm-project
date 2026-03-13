package admin

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/abraderAI/crm-project/api/internal/audit"
	"github.com/abraderAI/crm-project/api/internal/auth"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// --- Platform-wide Audit Log ---

// ListAuditLog handles GET /v1/admin/audit-log.
func (h *Handler) ListAuditLog(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := AuditListParams{
		Params:     pagination.Parse(r),
		OrgID:      q.Get("org"),
		UserID:     q.Get("user"),
		Action:     q.Get("action"),
		EntityType: q.Get("entity_type"),
		IPAddress:  q.Get("ip"),
	}

	if v := q.Get("after"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			params.After = &t
		}
	}
	if v := q.Get("before"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			params.Before = &t
		}
	}

	logs, pageInfo, err := h.service.ListAuditLogs(r.Context(), params)
	if err != nil {
		apierrors.InternalError(w, "failed to list audit logs")
		return
	}

	response.JSON(w, http.StatusOK, response.ListResponse{
		Data:     logs,
		PageInfo: pageInfo,
	})
}

// --- Platform Admin Management ---

// ListPlatformAdmins handles GET /v1/admin/platform-admins.
func (h *Handler) ListPlatformAdmins(w http.ResponseWriter, r *http.Request) {
	admins, err := h.service.ListPlatformAdmins(r.Context())
	if err != nil {
		apierrors.InternalError(w, "failed to list platform admins")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"data": admins})
}

// AddPlatformAdmin handles POST /v1/admin/platform-admins.
func (h *Handler) AddPlatformAdmin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	if body.UserID == "" {
		apierrors.ValidationError(w, "user_id is required", nil)
		return
	}

	uc := auth.GetUserContext(r.Context())
	grantedBy := ""
	if uc != nil {
		grantedBy = uc.UserID
	}

	admin, err := h.service.AddPlatformAdmin(r.Context(), body.UserID, grantedBy)
	if err != nil {
		if strings.Contains(err.Error(), "already") {
			apierrors.Conflict(w, err.Error())
			return
		}
		apierrors.InternalError(w, "failed to add platform admin")
		return
	}

	// Audit log.
	audit.CreateAuditEntry(r.Context(), h.auditService, "create", "platform_admin", body.UserID, nil,
		map[string]string{"granted_by": grantedBy})

	response.Created(w, admin)
}

// RemovePlatformAdmin handles DELETE /v1/admin/platform-admins/{user_id}.
func (h *Handler) RemovePlatformAdmin(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		apierrors.BadRequest(w, "user_id is required")
		return
	}

	if err := h.service.RemovePlatformAdmin(r.Context(), userID); err != nil {
		if strings.Contains(err.Error(), "last platform admin") {
			apierrors.ValidationError(w, err.Error(), nil)
			return
		}
		if strings.Contains(err.Error(), "not found") {
			apierrors.NotFound(w, "platform admin not found")
			return
		}
		apierrors.InternalError(w, "failed to remove platform admin")
		return
	}

	// Audit log.
	audit.CreateAuditEntry(r.Context(), h.auditService, "delete", "platform_admin", userID, nil, nil)

	response.NoContent(w)
}
