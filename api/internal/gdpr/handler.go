package gdpr

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for GDPR compliance operations.
type Handler struct {
	service *Service
}

// NewHandler creates a new GDPR handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ExportUserData handles GET /v1/admin/users/{user}/export.
func (h *Handler) ExportUserData(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user")
	if userID == "" {
		apierrors.BadRequest(w, "user identifier is required")
		return
	}

	data, err := h.service.ExportUserDataJSON(r.Context(), userID)
	if err != nil {
		apierrors.InternalError(w, "failed to export user data")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=user-export-"+userID+".json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// PurgeUser handles DELETE /v1/admin/users/{user}/purge.
func (h *Handler) PurgeUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user")
	if userID == "" {
		apierrors.BadRequest(w, "user identifier is required")
		return
	}

	if err := h.service.PurgeUser(r.Context(), userID); err != nil {
		apierrors.InternalError(w, "failed to purge user data")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"status":  "purged",
		"user_id": userID,
		"message": "all user PII has been removed and audit logs anonymized",
	})
}

// PurgeOrg handles DELETE /v1/admin/orgs/{org}/purge.
func (h *Handler) PurgeOrg(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	if orgID == "" {
		apierrors.BadRequest(w, "org identifier is required")
		return
	}

	if err := h.service.PurgeOrg(r.Context(), orgID); err != nil {
		apierrors.InternalError(w, "failed to purge org data")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"status":  "purged",
		"org_id":  orgID,
		"message": "org and all associated data has been permanently deleted",
	})
}
