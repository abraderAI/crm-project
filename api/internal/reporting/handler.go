package reporting

import (
	"encoding/csv"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for reporting endpoints.
type Handler struct {
	service *ReportingService
	db      *gorm.DB
}

// NewHandler creates a new reporting Handler.
func NewHandler(service *ReportingService, db *gorm.DB) *Handler {
	return &Handler{service: service, db: db}
}

// RequireOrgAdminOrOwner is middleware that checks the authenticated user has
// admin or owner role on the org identified by the {org} URL param.
func RequireOrgAdminOrOwner(db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uc := auth.GetUserContext(r.Context())
			if uc == nil {
				apierrors.Unauthorized(w, "authentication required")
				return
			}

			orgID := chi.URLParam(r, "org")
			if orgID == "" {
				apierrors.BadRequest(w, "org ID is required")
				return
			}

			var membership models.OrgMembership
			err := db.Where("org_id = ? AND user_id = ?", orgID, uc.UserID).First(&membership).Error
			if err != nil {
				apierrors.Forbidden(w, "insufficient permissions for reports")
				return
			}

			if membership.Role != models.RoleAdmin && membership.Role != models.RoleOwner {
				apierrors.Forbidden(w, "insufficient permissions for reports")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetSupportMetrics handles GET /v1/orgs/{org}/reports/support.
func (h *Handler) GetSupportMetrics(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	params, err := parseReportParams(r)
	if err != nil {
		apierrors.BadRequest(w, err.Error())
		return
	}

	metrics, err := h.service.GetSupportMetrics(r.Context(), orgID, params)
	if err != nil {
		apierrors.InternalError(w, "failed to compute support metrics")
		return
	}

	response.JSON(w, http.StatusOK, metrics)
}

// GetSupportExport handles GET /v1/orgs/{org}/reports/support/export.
func (h *Handler) GetSupportExport(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	params, err := parseReportParams(r)
	if err != nil {
		apierrors.BadRequest(w, err.Error())
		return
	}

	rows, err := h.service.GetSupportExportRows(r.Context(), orgID, params)
	if err != nil {
		apierrors.InternalError(w, "failed to export support data")
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="support-report.csv"`)
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"id", "title", "status", "priority", "assigned_to", "created_at", "updated_at"})
	for _, row := range rows {
		_ = cw.Write([]string{row.ID, row.Title, row.Status, row.Priority, row.AssignedTo, row.CreatedAt, row.UpdatedAt})
	}
	cw.Flush()
}

// GetSalesMetrics handles GET /v1/orgs/{org}/reports/sales.
func (h *Handler) GetSalesMetrics(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	params, err := parseReportParams(r)
	if err != nil {
		apierrors.BadRequest(w, err.Error())
		return
	}

	metrics, err := h.service.GetSalesMetrics(r.Context(), orgID, params)
	if err != nil {
		apierrors.InternalError(w, "failed to compute sales metrics")
		return
	}

	response.JSON(w, http.StatusOK, metrics)
}

// GetSalesExport handles GET /v1/orgs/{org}/reports/sales/export.
func (h *Handler) GetSalesExport(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	params, err := parseReportParams(r)
	if err != nil {
		apierrors.BadRequest(w, err.Error())
		return
	}

	rows, err := h.service.GetSalesExportRows(r.Context(), orgID, params)
	if err != nil {
		apierrors.InternalError(w, "failed to export sales data")
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="sales-report.csv"`)
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"id", "title", "stage", "assigned_to", "deal_value", "score", "created_at"})
	for _, row := range rows {
		_ = cw.Write([]string{row.ID, row.Title, row.Stage, row.AssignedTo, row.DealValue, row.Score, row.CreatedAt})
	}
	cw.Flush()
}

// parseReportParams extracts from, to, and assignee query params from the request.
// Defaults: from = 30 days ago, to = end of today (UTC).
func parseReportParams(r *http.Request) (ReportParams, error) {
	now := time.Now().UTC()
	params := ReportParams{
		From: now.AddDate(0, 0, -30).Truncate(24 * time.Hour),
		To:   now.Truncate(24 * time.Hour).Add(24*time.Hour - time.Nanosecond),
	}

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		t, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			return params, &paramError{field: "from", msg: "invalid date format, expected YYYY-MM-DD"}
		}
		params.From = t
	}

	if toStr := r.URL.Query().Get("to"); toStr != "" {
		t, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			return params, &paramError{field: "to", msg: "invalid date format, expected YYYY-MM-DD"}
		}
		// Set to end of day to include full day.
		params.To = t.Add(24*time.Hour - time.Nanosecond)
	}

	params.Assignee = r.URL.Query().Get("assignee")

	return params, nil
}

// paramError represents a query parameter validation error.
type paramError struct {
	field string
	msg   string
}

func (e *paramError) Error() string {
	return e.field + ": " + e.msg
}
