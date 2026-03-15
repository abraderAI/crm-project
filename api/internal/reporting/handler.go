package reporting

import (
	"encoding/csv"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/abraderAI/crm-project/api/internal/auth"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for reporting endpoints.
type Handler struct {
	service *Service
}

// NewHandler creates a new reporting Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// requireOrgAdmin checks that the authenticated user holds admin or owner role
// in the org. Writes an error response and returns false when the check fails.
func (h *Handler) requireOrgAdmin(w http.ResponseWriter, r *http.Request, orgID string) bool {
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return false
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

// parseReportParams extracts common report query params from the request.
// Returns false and writes an RFC 7807 error on invalid input.
func parseReportParams(w http.ResponseWriter, r *http.Request) (ReportParams, bool) {
	now := time.Now().UTC()
	params := ReportParams{
		From:     now.AddDate(0, 0, -30),
		To:       now,
		Assignee: r.URL.Query().Get("assignee"),
	}

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		t, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			apierrors.WriteProblem(w, apierrors.ProblemDetail{
				Type:   "https://httpstatuses.com/400",
				Title:  "Bad Request",
				Status: http.StatusBadRequest,
				Detail: "invalid 'from' date format, expected ISO 8601 date (YYYY-MM-DD)",
			})
			return params, false
		}
		params.From = t
	}

	if toStr := r.URL.Query().Get("to"); toStr != "" {
		t, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			apierrors.WriteProblem(w, apierrors.ProblemDetail{
				Type:   "https://httpstatuses.com/400",
				Title:  "Bad Request",
				Status: http.StatusBadRequest,
				Detail: "invalid 'to' date format, expected ISO 8601 date (YYYY-MM-DD)",
			})
			return params, false
		}
		// Include the entire "to" day.
		params.To = t.Add(24*time.Hour - time.Nanosecond)
	}

	return params, true
}

// GetSupportMetrics handles GET /v1/orgs/{org}/reports/support.
func (h *Handler) GetSupportMetrics(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	if !h.requireOrgAdmin(w, r, orgID) {
		return
	}

	params, ok := parseReportParams(w, r)
	if !ok {
		return
	}

	metrics, err := h.service.GetSupportMetrics(r.Context(), orgID, params)
	if err != nil {
		apierrors.InternalError(w, "failed to generate support metrics")
		return
	}

	response.JSON(w, http.StatusOK, metrics)
}

// GetSupportExport handles GET /v1/orgs/{org}/reports/support/export.
// Streams CSV rows directly to the response writer without buffering.
func (h *Handler) GetSupportExport(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")
	if !h.requireOrgAdmin(w, r, orgID) {
		return
	}

	params, ok := parseReportParams(w, r)
	if !ok {
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="support-report.csv"`)
	w.WriteHeader(http.StatusOK)

	csvWriter := csv.NewWriter(w)
	// Write header row.
	_ = csvWriter.Write([]string{"id", "title", "status", "priority", "assigned_to", "created_at", "updated_at"})

	err := h.service.ScanExportRows(r.Context(), orgID, params, func(row ExportRow) error {
		return csvWriter.Write([]string{
			row.ID,
			row.Title,
			row.Status,
			row.Priority,
			row.AssignedTo,
			row.CreatedAt.Format(time.RFC3339),
			row.UpdatedAt.Format(time.RFC3339),
		})
	})
	if err != nil {
		// Headers already sent — cannot write RFC 7807. Best-effort flush.
		csvWriter.Flush()
		return
	}

	csvWriter.Flush()
}
