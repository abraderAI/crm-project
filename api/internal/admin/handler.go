package admin

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/abraderAI/crm-project/api/internal/audit"
	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/gdpr"
	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for admin endpoints.
type Handler struct {
	service      *Service
	auditService *audit.Service
	gdprService  *gdpr.Service
}

// NewHandler creates a new admin handler.
func NewHandler(service *Service, auditService *audit.Service, gdprService *gdpr.Service) *Handler {
	return &Handler{
		service:      service,
		auditService: auditService,
		gdprService:  gdprService,
	}
}

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

// --- Org Management ---

// ListOrgs handles GET /v1/admin/orgs.
func (h *Handler) ListOrgs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := OrgListParams{
		Params:        pagination.Parse(r),
		Slug:          q.Get("slug"),
		Name:          q.Get("name"),
		BillingTier:   q.Get("billing_tier"),
		PaymentStatus: q.Get("payment_status"),
	}

	if v := q.Get("created_after"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			params.CreatedAfter = &t
		}
	}
	if v := q.Get("created_before"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			params.CreatedBefore = &t
		}
	}

	orgs, pageInfo, err := h.service.ListOrgs(r.Context(), params)
	if err != nil {
		apierrors.InternalError(w, "failed to list orgs")
		return
	}

	response.JSON(w, http.StatusOK, response.ListResponse{
		Data:     orgs,
		PageInfo: pageInfo,
	})
}

// GetOrg handles GET /v1/admin/orgs/{org}.
func (h *Handler) GetOrg(w http.ResponseWriter, r *http.Request) {
	orgIDOrSlug := chi.URLParam(r, "org")
	if orgIDOrSlug == "" {
		apierrors.BadRequest(w, "org identifier is required")
		return
	}

	detail, err := h.service.GetOrgDetail(r.Context(), orgIDOrSlug)
	if err != nil {
		apierrors.InternalError(w, "failed to get org")
		return
	}
	if detail == nil {
		apierrors.NotFound(w, "org not found")
		return
	}

	response.JSON(w, http.StatusOK, detail)
}

// SuspendOrg handles POST /v1/admin/orgs/{org}/suspend.
func (h *Handler) SuspendOrg(w http.ResponseWriter, r *http.Request) {
	orgIDOrSlug := chi.URLParam(r, "org")
	if orgIDOrSlug == "" {
		apierrors.BadRequest(w, "org identifier is required")
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
	suspendedBy := ""
	if uc != nil {
		suspendedBy = uc.UserID
	}

	if err := h.service.SuspendOrg(r.Context(), orgIDOrSlug, body.Reason, suspendedBy); err != nil {
		if strings.Contains(err.Error(), "not found") {
			apierrors.NotFound(w, "org not found")
			return
		}
		apierrors.InternalError(w, "failed to suspend org")
		return
	}

	// Audit log.
	audit.CreateAuditEntry(r.Context(), h.auditService, "suspend", "org", orgIDOrSlug, nil,
		map[string]string{"reason": body.Reason, "suspended_by": suspendedBy})

	response.JSON(w, http.StatusOK, map[string]string{
		"status": "suspended",
		"org":    orgIDOrSlug,
	})
}

// UnsuspendOrg handles POST /v1/admin/orgs/{org}/unsuspend.
func (h *Handler) UnsuspendOrg(w http.ResponseWriter, r *http.Request) {
	orgIDOrSlug := chi.URLParam(r, "org")
	if orgIDOrSlug == "" {
		apierrors.BadRequest(w, "org identifier is required")
		return
	}

	if err := h.service.UnsuspendOrg(r.Context(), orgIDOrSlug); err != nil {
		if strings.Contains(err.Error(), "not found") {
			apierrors.NotFound(w, "org not found")
			return
		}
		apierrors.InternalError(w, "failed to unsuspend org")
		return
	}

	// Audit log.
	audit.CreateAuditEntry(r.Context(), h.auditService, "unsuspend", "org", orgIDOrSlug, nil, nil)

	response.JSON(w, http.StatusOK, map[string]string{
		"status": "unsuspended",
		"org":    orgIDOrSlug,
	})
}

// TransferOwnership handles POST /v1/admin/orgs/{org}/transfer-ownership.
func (h *Handler) TransferOwnership(w http.ResponseWriter, r *http.Request) {
	orgIDOrSlug := chi.URLParam(r, "org")
	if orgIDOrSlug == "" {
		apierrors.BadRequest(w, "org identifier is required")
		return
	}

	var body struct {
		NewOwnerUserID string `json:"new_owner_user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	if body.NewOwnerUserID == "" {
		apierrors.ValidationError(w, "new_owner_user_id is required", nil)
		return
	}

	if err := h.service.TransferOrgOwnership(r.Context(), orgIDOrSlug, body.NewOwnerUserID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			apierrors.NotFound(w, "org not found")
			return
		}
		apierrors.InternalError(w, "failed to transfer ownership")
		return
	}

	// Audit log.
	audit.CreateAuditEntry(r.Context(), h.auditService, "transfer_ownership", "org", orgIDOrSlug, nil,
		map[string]string{"new_owner_user_id": body.NewOwnerUserID})

	response.JSON(w, http.StatusOK, map[string]string{
		"status":            "transferred",
		"org":               orgIDOrSlug,
		"new_owner_user_id": body.NewOwnerUserID,
	})
}

// PurgeOrg handles DELETE /v1/admin/orgs/{org}/purge.
func (h *Handler) PurgeOrg(w http.ResponseWriter, r *http.Request) {
	orgIDOrSlug := chi.URLParam(r, "org")
	if orgIDOrSlug == "" {
		apierrors.BadRequest(w, "org identifier is required")
		return
	}

	var body struct {
		Confirm string `json:"confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	expected := "purge " + orgIDOrSlug
	if body.Confirm != expected {
		apierrors.ValidationError(w, "confirmation required: set confirm to \""+expected+"\"", nil)
		return
	}

	if err := h.gdprService.PurgeOrg(r.Context(), orgIDOrSlug); err != nil {
		apierrors.InternalError(w, "failed to purge org")
		return
	}

	// Audit log.
	audit.CreateAuditEntry(r.Context(), h.auditService, "purge", "org", orgIDOrSlug, nil, nil)

	response.JSON(w, http.StatusOK, map[string]string{
		"status":  "purged",
		"org":     orgIDOrSlug,
		"message": "org and all associated data has been permanently deleted",
	})
}

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
