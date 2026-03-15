package reporting

import (
	"encoding/csv"
	"net/http"

	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// GetAdminSupportMetrics handles GET /v1/admin/reports/support.
func (h *Handler) GetAdminSupportMetrics(w http.ResponseWriter, r *http.Request) {
	params, err := parseReportParams(r)
	if err != nil {
		apierrors.BadRequest(w, err.Error())
		return
	}

	metrics, err := h.service.GetAdminSupportMetrics(r.Context(), params)
	if err != nil {
		apierrors.InternalError(w, "failed to compute admin support metrics")
		return
	}

	response.JSON(w, http.StatusOK, metrics)
}

// GetAdminSupportExport handles GET /v1/admin/reports/support/export.
func (h *Handler) GetAdminSupportExport(w http.ResponseWriter, r *http.Request) {
	params, err := parseReportParams(r)
	if err != nil {
		apierrors.BadRequest(w, err.Error())
		return
	}

	rows, err := h.service.GetAdminSupportExportRows(r.Context(), params)
	if err != nil {
		apierrors.InternalError(w, "failed to export admin support data")
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="admin-support-report.csv"`)
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"org_id", "org_slug", "id", "title", "status", "priority", "assigned_to", "created_at", "updated_at"})
	for _, row := range rows {
		_ = cw.Write([]string{row.OrgID, row.OrgSlug, row.ID, row.Title, row.Status, row.Priority, row.AssignedTo, row.CreatedAt, row.UpdatedAt})
	}
	cw.Flush()
}

// GetAdminSalesMetrics handles GET /v1/admin/reports/sales.
func (h *Handler) GetAdminSalesMetrics(w http.ResponseWriter, r *http.Request) {
	params, err := parseReportParams(r)
	if err != nil {
		apierrors.BadRequest(w, err.Error())
		return
	}

	metrics, err := h.service.GetAdminSalesMetrics(r.Context(), params)
	if err != nil {
		apierrors.InternalError(w, "failed to compute admin sales metrics")
		return
	}

	response.JSON(w, http.StatusOK, metrics)
}

// GetAdminSalesExport handles GET /v1/admin/reports/sales/export.
func (h *Handler) GetAdminSalesExport(w http.ResponseWriter, r *http.Request) {
	params, err := parseReportParams(r)
	if err != nil {
		apierrors.BadRequest(w, err.Error())
		return
	}

	rows, err := h.service.GetAdminSalesExportRows(r.Context(), params)
	if err != nil {
		apierrors.InternalError(w, "failed to export admin sales data")
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="admin-sales-report.csv"`)
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"org_id", "org_slug", "id", "title", "stage", "assigned_to", "deal_value", "score", "created_at"})
	for _, row := range rows {
		_ = cw.Write([]string{row.OrgID, row.OrgSlug, row.ID, row.Title, row.Stage, row.AssignedTo, row.DealValue, row.Score, row.CreatedAt})
	}
	cw.Flush()
}
