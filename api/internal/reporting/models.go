// Package reporting provides support reporting metrics, CSV export, and API handlers.
package reporting

import "time"

// ReportParams holds common query parameters for all report endpoints.
type ReportParams struct {
	From     time.Time
	To       time.Time
	Assignee string // empty = all assignees
}

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

// DailyCount represents a single day's ticket count.
type DailyCount struct {
	Date  string `json:"date"` // "2026-03-01"
	Count int64  `json:"count"`
}

// AssigneeCount represents ticket count for a single assignee.
type AssigneeCount struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Count  int64  `json:"count"`
}

// ExportRow represents a single CSV export row.
type ExportRow struct {
	ID         string
	Title      string
	Status     string
	Priority   string
	AssignedTo string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
