// Package reporting provides reporting and analytics endpoints for support and sales metrics.
package reporting

import "time"

// ReportParams holds common query params for all report endpoints.
type ReportParams struct {
	From     time.Time
	To       time.Time
	Assignee string // empty = all assignees
}

// --- Support Metrics ---

// SupportMetrics is the response for GET /v1/orgs/{org}/reports/support.
type SupportMetrics struct {
	StatusBreakdown       map[string]int64 `json:"status_breakdown"`
	VolumeOverTime        []DailyCount     `json:"volume_over_time"`
	AvgResolutionHours    *float64         `json:"avg_resolution_hours"`
	TicketsByAssignee     []AssigneeCount  `json:"tickets_by_assignee"`
	TicketsByPriority     map[string]int64 `json:"tickets_by_priority"`
	AvgFirstResponseHours *float64         `json:"avg_first_response_hours"`
	OverdueCount          int64            `json:"overdue_count"`
}

// --- Sales Metrics ---

// SalesMetrics is the response for GET /v1/orgs/{org}/reports/sales.
type SalesMetrics struct {
	PipelineFunnel       []StageCount      `json:"pipeline_funnel"`
	LeadVelocity         []DailyCount      `json:"lead_velocity"`
	WinRate              float64           `json:"win_rate"`
	LossRate             float64           `json:"loss_rate"`
	AvgDealValue         *float64          `json:"avg_deal_value"`
	LeadsByAssignee      []AssigneeCount   `json:"leads_by_assignee"`
	ScoreDistribution    []BucketCount     `json:"score_distribution"`
	StageConversionRates []StageConversion `json:"stage_conversion_rates"`
	AvgTimeInStage       []StageAvgTime    `json:"avg_time_in_stage"`
}

// --- Shared Types ---

// DailyCount holds a date string and a count, used for time-series data.
type DailyCount struct {
	Date  string `json:"date"` // "2026-03-01"
	Count int64  `json:"count"`
}

// AssigneeCount holds a user ID, display name, and count.
type AssigneeCount struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Count  int64  `json:"count"`
}

// --- Sales-specific Types ---

// StageCount holds a pipeline stage name and count.
type StageCount struct {
	Stage string `json:"stage"`
	Count int64  `json:"count"`
}

// BucketCount holds a score distribution bucket range and count.
type BucketCount struct {
	Range string `json:"range"` // "0-20", "20-40", etc.
	Count int64  `json:"count"`
}

// StageConversion holds a from→to stage conversion rate.
type StageConversion struct {
	FromStage string  `json:"from_stage"`
	ToStage   string  `json:"to_stage"`
	Rate      float64 `json:"rate"` // 0.0–1.0
}

// StageAvgTime holds the average hours a lead spends in a pipeline stage.
type StageAvgTime struct {
	Stage    string   `json:"stage"`
	AvgHours *float64 `json:"avg_hours"` // nil if no data
}

// --- Admin Response Types ---

// AdminSupportMetrics is the response for GET /v1/admin/reports/support.
type AdminSupportMetrics struct {
	SupportMetrics
	OrgBreakdown []OrgSupportSummary `json:"org_breakdown"`
}

// AdminSalesMetrics is the response for GET /v1/admin/reports/sales.
type AdminSalesMetrics struct {
	SalesMetrics
	OrgBreakdown []OrgSalesSummary `json:"org_breakdown"`
}

// OrgSupportSummary holds per-org support metrics for the admin breakdown.
type OrgSupportSummary struct {
	OrgID                 string   `json:"org_id"`
	OrgName               string   `json:"org_name"`
	OrgSlug               string   `json:"org_slug"`
	OpenCount             int64    `json:"open_count"`
	OverdueCount          int64    `json:"overdue_count"`
	AvgResolutionHours    *float64 `json:"avg_resolution_hours"`
	AvgFirstResponseHours *float64 `json:"avg_first_response_hours"`
	TotalInRange          int64    `json:"total_in_range"`
}

// OrgSalesSummary holds per-org sales metrics for the admin breakdown.
type OrgSalesSummary struct {
	OrgID             string   `json:"org_id"`
	OrgName           string   `json:"org_name"`
	OrgSlug           string   `json:"org_slug"`
	TotalLeads        int64    `json:"total_leads"`
	WinRate           float64  `json:"win_rate"`
	AvgDealValue      *float64 `json:"avg_deal_value"`
	OpenPipelineCount int64    `json:"open_pipeline_count"`
}

// --- Export Row Types ---

// SupportExportRow represents one row of the support CSV export.
type SupportExportRow struct {
	ID         string
	Title      string
	Status     string
	Priority   string
	AssignedTo string
	CreatedAt  string
	UpdatedAt  string
}

// SalesExportRow represents one row of the sales CSV export.
type SalesExportRow struct {
	ID         string
	Title      string
	Stage      string
	AssignedTo string
	DealValue  string
	Score      string
	CreatedAt  string
}

// AdminSupportExportRow extends SupportExportRow with org context.
type AdminSupportExportRow struct {
	OrgID   string
	OrgSlug string
	SupportExportRow
}

// AdminSalesExportRow extends SalesExportRow with org context.
type AdminSalesExportRow struct {
	OrgID   string
	OrgSlug string
	SalesExportRow
}
