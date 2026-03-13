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
