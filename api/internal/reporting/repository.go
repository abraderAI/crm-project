package reporting

import (
	"context"
	"database/sql"
	"fmt"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// Repository handles database operations for reporting queries.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new reporting Repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// supportBaseJoin is the common FROM/JOIN/WHERE clause for support-space queries.
const supportBaseJoin = `
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'support'
  AND t.deleted_at IS NULL`

// appendAssigneeFilter appends the optional assignee filter to a query string
// and returns the updated args slice.
func appendAssigneeFilter(query string, args []any, assignee string) (string, []any) {
	if assignee != "" {
		query += " AND json_extract(t.metadata, '$.assigned_to') = ?"
		args = append(args, assignee)
	}
	return query, args
}

// GetStatusBreakdown returns thread counts grouped by status.
func (r *Repository) GetStatusBreakdown(ctx context.Context, orgID string, params ReportParams) (map[string]int64, error) {
	q := `SELECT
  COALESCE(json_extract(t.metadata, '$.status'), 'unknown') AS status,
  COUNT(*) AS count` + supportBaseJoin + `
  AND t.created_at BETWEEN ? AND ?`
	args := []any{orgID, params.From, params.To}
	q, args = appendAssigneeFilter(q, args, params.Assignee)
	q += " GROUP BY status"

	type row struct {
		Status string
		Count  int64
	}
	var rows []row
	if err := r.db.WithContext(ctx).Raw(q, args...).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("status breakdown: %w", err)
	}

	result := make(map[string]int64, len(rows))
	for _, r := range rows {
		result[r.Status] = r.Count
	}
	return result, nil
}

// GetVolumeOverTime returns daily ticket creation counts.
func (r *Repository) GetVolumeOverTime(ctx context.Context, orgID string, params ReportParams) ([]DailyCount, error) {
	q := `SELECT DATE(t.created_at) AS date, COUNT(*) AS count` + supportBaseJoin + `
  AND t.created_at BETWEEN ? AND ?`
	args := []any{orgID, params.From, params.To}
	q, args = appendAssigneeFilter(q, args, params.Assignee)
	q += " GROUP BY DATE(t.created_at) ORDER BY date ASC"

	var rows []DailyCount
	if err := r.db.WithContext(ctx).Raw(q, args...).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("volume over time: %w", err)
	}
	if rows == nil {
		rows = []DailyCount{}
	}
	return rows, nil
}

// GetAvgResolutionHours returns the mean hours from created_at to updated_at
// for resolved/closed threads. Returns nil when no rows match.
func (r *Repository) GetAvgResolutionHours(ctx context.Context, orgID string, params ReportParams) (*float64, error) {
	q := `SELECT AVG((JULIANDAY(t.updated_at) - JULIANDAY(t.created_at)) * 24) AS avg_hours` + supportBaseJoin + `
  AND t.created_at BETWEEN ? AND ?
  AND json_extract(t.metadata, '$.status') IN ('resolved', 'closed')`
	args := []any{orgID, params.From, params.To}
	q, args = appendAssigneeFilter(q, args, params.Assignee)

	var avg sql.NullFloat64
	if err := r.db.WithContext(ctx).Raw(q, args...).Row().Scan(&avg); err != nil {
		return nil, fmt.Errorf("avg resolution hours: %w", err)
	}
	if !avg.Valid {
		return nil, nil
	}
	v := avg.Float64
	return &v, nil
}

// GetTicketsByAssignee returns open ticket counts per assigned user.
func (r *Repository) GetTicketsByAssignee(ctx context.Context, orgID string, params ReportParams) ([]AssigneeCount, error) {
	q := `SELECT
  json_extract(t.metadata, '$.assigned_to') AS user_id,
  COALESCE(u.display_name, json_extract(t.metadata, '$.assigned_to')) AS name,
  COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
LEFT JOIN user_shadows u ON u.clerk_user_id = json_extract(t.metadata, '$.assigned_to')
WHERE s.org_id = ? AND s.type = 'support'
  AND t.deleted_at IS NULL
  AND json_extract(t.metadata, '$.status') IN ('open', 'in_progress')
  AND json_extract(t.metadata, '$.assigned_to') IS NOT NULL`
	args := []any{orgID}
	q, args = appendAssigneeFilter(q, args, params.Assignee)
	q += " GROUP BY user_id ORDER BY count DESC"

	var rows []AssigneeCount
	if err := r.db.WithContext(ctx).Raw(q, args...).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("tickets by assignee: %w", err)
	}
	if rows == nil {
		rows = []AssigneeCount{}
	}
	return rows, nil
}

// GetTicketsByPriority returns thread counts grouped by priority.
func (r *Repository) GetTicketsByPriority(ctx context.Context, orgID string, params ReportParams) (map[string]int64, error) {
	q := `SELECT
  COALESCE(json_extract(t.metadata, '$.priority'), 'none') AS priority,
  COUNT(*) AS count` + supportBaseJoin + `
  AND t.created_at BETWEEN ? AND ?`
	args := []any{orgID, params.From, params.To}
	q, args = appendAssigneeFilter(q, args, params.Assignee)
	q += " GROUP BY priority"

	type row struct {
		Priority string
		Count    int64
	}
	var rows []row
	if err := r.db.WithContext(ctx).Raw(q, args...).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("tickets by priority: %w", err)
	}

	result := make(map[string]int64, len(rows))
	for _, r := range rows {
		result[r.Priority] = r.Count
	}
	return result, nil
}

// GetAvgFirstResponseHours returns the mean hours from thread creation to the
// first reply by someone other than the thread author. Returns nil when no rows match.
func (r *Repository) GetAvgFirstResponseHours(ctx context.Context, orgID string, params ReportParams) (*float64, error) {
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
	args := []any{orgID, params.From, params.To}
	q, args = appendAssigneeFilter(q, args, params.Assignee)

	var avg sql.NullFloat64
	if err := r.db.WithContext(ctx).Raw(q, args...).Row().Scan(&avg); err != nil {
		return nil, fmt.Errorf("avg first response hours: %w", err)
	}
	if !avg.Valid {
		return nil, nil
	}
	v := avg.Float64
	return &v, nil
}

// GetOverdueCount returns the number of open/in_progress threads older than 72 hours.
// Intentionally no date range filter — always current state.
func (r *Repository) GetOverdueCount(ctx context.Context, orgID string, params ReportParams) (int64, error) {
	q := `SELECT COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'support'
  AND json_extract(t.metadata, '$.status') IN ('open', 'in_progress')
  AND t.created_at < datetime('now', '-72 hours')
  AND t.deleted_at IS NULL`
	args := []any{orgID}
	q, args = appendAssigneeFilter(q, args, params.Assignee)

	var count int64
	if err := r.db.WithContext(ctx).Raw(q, args...).Row().Scan(&count); err != nil {
		return 0, fmt.Errorf("overdue count: %w", err)
	}
	return count, nil
}

// ScanExportRows streams thread rows for CSV export, calling fn for each row.
// This avoids buffering all rows in memory.
func (r *Repository) ScanExportRows(ctx context.Context, orgID string, params ReportParams, fn func(ExportRow) error) error {
	q := `SELECT
  t.id, t.title,
  COALESCE(json_extract(t.metadata, '$.status'), 'unknown') AS status,
  COALESCE(json_extract(t.metadata, '$.priority'), 'none') AS priority,
  COALESCE(json_extract(t.metadata, '$.assigned_to'), '') AS assigned_to,
  t.created_at, t.updated_at` + supportBaseJoin + `
  AND t.created_at BETWEEN ? AND ?`
	args := []any{orgID, params.From, params.To}
	q, args = appendAssigneeFilter(q, args, params.Assignee)
	q += " ORDER BY t.created_at ASC"

	rows, err := r.db.WithContext(ctx).Raw(q, args...).Rows()
	if err != nil {
		return fmt.Errorf("export query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var row ExportRow
		if err := rows.Scan(&row.ID, &row.Title, &row.Status, &row.Priority, &row.AssignedTo, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return fmt.Errorf("scanning export row: %w", err)
		}
		if err := fn(row); err != nil {
			return err
		}
	}
	return rows.Err()
}

// IsOrgAdmin returns true when the given user has admin or owner role in the org.
func (r *Repository) IsOrgAdmin(ctx context.Context, orgID, userID string) (bool, error) {
	var m models.OrgMembership
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND user_id = ?", orgID, userID).
		First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("checking org membership: %w", err)
	}
	return m.Role == models.RoleAdmin || m.Role == models.RoleOwner, nil
}
