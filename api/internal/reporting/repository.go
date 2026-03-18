package reporting

import (
	"context"

	"gorm.io/gorm"
)

// ReportingRepository defines the data-access interface for reporting queries.
type ReportingRepository interface {
	// Support queries.
	GetStatusBreakdown(ctx context.Context, orgID string, params ReportParams) (map[string]int64, error)
	GetVolumeOverTime(ctx context.Context, orgID string, params ReportParams) ([]DailyCount, error)
	GetAvgResolutionHours(ctx context.Context, orgID string, params ReportParams) (*float64, error)
	GetTicketsByAssignee(ctx context.Context, orgID string, params ReportParams) ([]AssigneeCount, error)
	GetTicketsByPriority(ctx context.Context, orgID string, params ReportParams) (map[string]int64, error)
	GetAvgFirstResponseHours(ctx context.Context, orgID string, params ReportParams) (*float64, error)
	GetOverdueCount(ctx context.Context, orgID string, params ReportParams) (int64, error)
	GetSupportExportRows(ctx context.Context, orgID string, params ReportParams) ([]SupportExportRow, error)

	// Sales queries.
	GetPipelineFunnel(ctx context.Context, orgID string, params ReportParams) ([]StageCount, error)
	GetLeadVelocity(ctx context.Context, orgID string, params ReportParams) ([]DailyCount, error)
	GetWinLossCounts(ctx context.Context, orgID string, params ReportParams) (won, lost int64, err error)
	GetAvgDealValue(ctx context.Context, orgID string, params ReportParams) (*float64, error)
	GetLeadsByAssignee(ctx context.Context, orgID string, params ReportParams) ([]AssigneeCount, error)
	GetScoreDistribution(ctx context.Context, orgID string, params ReportParams) ([]BucketCount, error)
	GetStageTransitions(ctx context.Context, orgID string, params ReportParams) ([]stageTransitionRow, error)
	GetAvgTimeInStage(ctx context.Context, orgID string, params ReportParams) ([]StageAvgTime, error)
	GetSalesExportRows(ctx context.Context, orgID string, params ReportParams) ([]SalesExportRow, error)
}

// stageTransitionRow holds a single row from the stage transition query.
type stageTransitionRow struct {
	FromStage       string
	ToStage         string
	TransitionCount int64
}

// repository is the concrete implementation of ReportingRepository backed by GORM.
type repository struct {
	db *gorm.DB
}

// NewRepository creates a new ReportingRepository.
func NewRepository(db *gorm.DB) ReportingRepository {
	return &repository{db: db}
}

// --- Support Queries ---

// GetStatusBreakdown returns thread counts grouped by status metadata.
func (r *repository) GetStatusBreakdown(ctx context.Context, orgID string, params ReportParams) (map[string]int64, error) {
	type row struct {
		Status string
		Count  int64
	}
	var rows []row
	args := []interface{}{orgID, params.From, params.To}
	q := `SELECT
  COALESCE(json_extract(t.metadata, '$.status'), 'unknown') AS status,
  COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'support'
  AND t.created_at BETWEEN ? AND ?
  AND t.deleted_at IS NULL`
	q, args = appendAssigneeFilter(q, args, params.Assignee)
	q += "\nGROUP BY status"

	if err := r.db.WithContext(ctx).Raw(q, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(rows))
	for _, row := range rows {
		result[row.Status] = row.Count
	}
	return result, nil
}

// GetVolumeOverTime returns daily ticket creation counts.
func (r *repository) GetVolumeOverTime(ctx context.Context, orgID string, params ReportParams) ([]DailyCount, error) {
	var rows []DailyCount
	args := []interface{}{orgID, params.From, params.To}
	q := `SELECT DATE(t.created_at) AS date, COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'support'
  AND t.created_at BETWEEN ? AND ?
  AND t.deleted_at IS NULL`
	q, args = appendAssigneeFilter(q, args, params.Assignee)
	q += "\nGROUP BY DATE(t.created_at)\nORDER BY date ASC"

	if err := r.db.WithContext(ctx).Raw(q, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []DailyCount{}
	}
	return rows, nil
}

// GetAvgResolutionHours returns the mean hours from created_at to updated_at
// for resolved/closed threads.
func (r *repository) GetAvgResolutionHours(ctx context.Context, orgID string, params ReportParams) (*float64, error) {
	var avg *float64
	args := []interface{}{orgID, params.From, params.To}
	q := `SELECT AVG((JULIANDAY(t.updated_at) - JULIANDAY(t.created_at)) * 24) AS avg_hours
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'support'
  AND t.created_at BETWEEN ? AND ?
  AND json_extract(t.metadata, '$.status') IN ('resolved', 'closed')
  AND t.deleted_at IS NULL`
	q, args = appendAssigneeFilter(q, args, params.Assignee)

	if err := r.db.WithContext(ctx).Raw(q, args...).Row().Scan(&avg); err != nil {
		return nil, err
	}
	return avg, nil
}

// GetTicketsByAssignee returns open ticket counts per assigned user.
func (r *repository) GetTicketsByAssignee(ctx context.Context, orgID string, params ReportParams) ([]AssigneeCount, error) {
	var rows []AssigneeCount
	args := []interface{}{orgID}
	q := `SELECT
  json_extract(t.metadata, '$.assigned_to') AS user_id,
  COALESCE(u.display_name, json_extract(t.metadata, '$.assigned_to')) AS name,
  COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
LEFT JOIN user_shadows u ON u.clerk_user_id = json_extract(t.metadata, '$.assigned_to')
WHERE s.org_id = ? AND s.type = 'support'
  AND json_extract(t.metadata, '$.status') IN ('open', 'in_progress')
  AND json_extract(t.metadata, '$.assigned_to') IS NOT NULL
  AND t.deleted_at IS NULL`
	q, args = appendAssigneeFilter(q, args, params.Assignee)
	q += "\nGROUP BY user_id\nORDER BY count DESC"

	if err := r.db.WithContext(ctx).Raw(q, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []AssigneeCount{}
	}
	return rows, nil
}

// GetTicketsByPriority returns thread counts grouped by priority metadata.
func (r *repository) GetTicketsByPriority(ctx context.Context, orgID string, params ReportParams) (map[string]int64, error) {
	type row struct {
		Priority string
		Count    int64
	}
	var rows []row
	args := []interface{}{orgID, params.From, params.To}
	q := `SELECT
  COALESCE(json_extract(t.metadata, '$.priority'), 'none') AS priority,
  COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'support'
  AND t.created_at BETWEEN ? AND ?
  AND t.deleted_at IS NULL`
	q, args = appendAssigneeFilter(q, args, params.Assignee)
	q += "\nGROUP BY priority"

	if err := r.db.WithContext(ctx).Raw(q, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(rows))
	for _, row := range rows {
		result[row.Priority] = row.Count
	}
	return result, nil
}

// GetAvgFirstResponseHours returns the mean hours from thread creation to the
// first reply by someone other than the thread author.
func (r *repository) GetAvgFirstResponseHours(ctx context.Context, orgID string, params ReportParams) (*float64, error) {
	var avg *float64
	args := []interface{}{orgID, params.From, params.To}
	q := `SELECT AVG((JULIANDAY(fr.first_reply_at) - JULIANDAY(t.created_at)) * 24) AS avg_hours
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
JOIN (
  SELECT m.thread_id, MIN(m.created_at) AS first_reply_at
  FROM messages m
  JOIN threads t2 ON t2.id = m.thread_id
  WHERE m.author_id != t2.author_id
    AND m.deleted_at IS NULL
  GROUP BY m.thread_id
) fr ON fr.thread_id = t.id
WHERE s.org_id = ? AND s.type = 'support'
  AND t.created_at BETWEEN ? AND ?
  AND t.deleted_at IS NULL`
	q, args = appendAssigneeFilter(q, args, params.Assignee)

	if err := r.db.WithContext(ctx).Raw(q, args...).Row().Scan(&avg); err != nil {
		return nil, err
	}
	return avg, nil
}

// GetOverdueCount returns the count of open/in_progress tickets older than 72 hours.
// Intentionally NO date range filter — always reflects current state.
func (r *repository) GetOverdueCount(ctx context.Context, orgID string, params ReportParams) (int64, error) {
	var count int64
	args := []interface{}{orgID}
	q := `SELECT COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'support'
  AND json_extract(t.metadata, '$.status') IN ('open', 'in_progress')
  AND t.created_at < datetime('now', '-72 hours')
  AND t.deleted_at IS NULL`
	q, args = appendAssigneeFilter(q, args, params.Assignee)

	if err := r.db.WithContext(ctx).Raw(q, args...).Row().Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// GetSupportExportRows returns row-level data for CSV export.
func (r *repository) GetSupportExportRows(ctx context.Context, orgID string, params ReportParams) ([]SupportExportRow, error) {
	var rows []SupportExportRow
	args := []interface{}{orgID, params.From, params.To}
	q := `SELECT
  t.id,
  t.title,
  COALESCE(json_extract(t.metadata, '$.status'), '') AS status,
  COALESCE(json_extract(t.metadata, '$.priority'), '') AS priority,
  COALESCE(json_extract(t.metadata, '$.assigned_to'), '') AS assigned_to,
  t.created_at,
  t.updated_at
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'support'
  AND t.created_at BETWEEN ? AND ?
  AND t.deleted_at IS NULL`
	q, args = appendAssigneeFilter(q, args, params.Assignee)
	q += "\nORDER BY t.created_at ASC"

	if err := r.db.WithContext(ctx).Raw(q, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []SupportExportRow{}
	}
	return rows, nil
}

// appendAssigneeFilter appends an assignee filter clause if the assignee is set.
func appendAssigneeFilter(q string, args []interface{}, assignee string) (string, []interface{}) {
	if assignee != "" {
		q += "\n  AND json_extract(t.metadata, '$.assigned_to') = ?"
		args = append(args, assignee)
	}
	return q, args
}
