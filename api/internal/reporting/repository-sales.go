package reporting

import "context"

// --- Sales Queries ---

// GetPipelineFunnel returns the current funnel state (no date range filter).
func (r *repository) GetPipelineFunnel(ctx context.Context, orgID string, params ReportParams) ([]StageCount, error) {
	var rows []StageCount
	args := []interface{}{orgID}
	q := `SELECT
  COALESCE(json_extract(t.metadata, '$.stage'), 'unknown') AS stage,
  COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'crm'
  AND t.deleted_at IS NULL`
	q, args = appendAssigneeFilter(q, args, params.Assignee)
	q += "\nGROUP BY stage\nORDER BY count DESC"

	if err := r.db.WithContext(ctx).Raw(q, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []StageCount{}
	}
	return rows, nil
}

// GetLeadVelocity returns new leads per day in the date range.
func (r *repository) GetLeadVelocity(ctx context.Context, orgID string, params ReportParams) ([]DailyCount, error) {
	var rows []DailyCount
	args := []interface{}{orgID, params.From, params.To}
	q := `SELECT DATE(t.created_at) AS date, COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'crm'
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

// GetWinLossCounts returns the count of closed_won and closed_lost threads.
func (r *repository) GetWinLossCounts(ctx context.Context, orgID string, params ReportParams) (won, lost int64, err error) {
	type row struct {
		Stage string
		Count int64
	}
	var rows []row
	args := []interface{}{orgID, params.From, params.To}
	q := `SELECT
  json_extract(t.metadata, '$.stage') AS stage,
  COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'crm'
  AND json_extract(t.metadata, '$.stage') IN ('closed_won', 'closed_lost')
  AND t.created_at BETWEEN ? AND ?
  AND t.deleted_at IS NULL`
	q, args = appendAssigneeFilter(q, args, params.Assignee)
	q += "\nGROUP BY stage"

	if err := r.db.WithContext(ctx).Raw(q, args...).Scan(&rows).Error; err != nil {
		return 0, 0, err
	}
	for _, r := range rows {
		switch r.Stage {
		case "closed_won":
			won = r.Count
		case "closed_lost":
			lost = r.Count
		}
	}
	return won, lost, nil
}

// GetAvgDealValue returns the average deal_value from thread metadata.
func (r *repository) GetAvgDealValue(ctx context.Context, orgID string, params ReportParams) (*float64, error) {
	var avg *float64
	args := []interface{}{orgID, params.From, params.To}
	q := `SELECT AVG(CAST(json_extract(t.metadata, '$.deal_value') AS REAL)) AS avg_value
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'crm'
  AND t.created_at BETWEEN ? AND ?
  AND json_extract(t.metadata, '$.deal_value') IS NOT NULL
  AND t.deleted_at IS NULL`
	q, args = appendAssigneeFilter(q, args, params.Assignee)

	if err := r.db.WithContext(ctx).Raw(q, args...).Row().Scan(&avg); err != nil {
		return nil, err
	}
	return avg, nil
}

// GetLeadsByAssignee returns active (non-closed) lead counts per assigned user.
func (r *repository) GetLeadsByAssignee(ctx context.Context, orgID string, params ReportParams) ([]AssigneeCount, error) {
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
WHERE s.org_id = ? AND s.type = 'crm'
  AND json_extract(t.metadata, '$.stage') NOT IN ('closed_won', 'closed_lost')
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

// GetScoreDistribution returns lead counts in 5 equal-width score buckets.
func (r *repository) GetScoreDistribution(ctx context.Context, orgID string, params ReportParams) ([]BucketCount, error) {
	var rows []BucketCount
	args := []interface{}{orgID, params.From, params.To}
	q := `SELECT
  CASE
    WHEN CAST(json_extract(t.metadata, '$.score') AS REAL) < 20  THEN '0-20'
    WHEN CAST(json_extract(t.metadata, '$.score') AS REAL) < 40  THEN '20-40'
    WHEN CAST(json_extract(t.metadata, '$.score') AS REAL) < 60  THEN '40-60'
    WHEN CAST(json_extract(t.metadata, '$.score') AS REAL) < 80  THEN '60-80'
    ELSE '80-100'
  END AS range,
  COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'crm'
  AND t.created_at BETWEEN ? AND ?
  AND json_extract(t.metadata, '$.score') IS NOT NULL
  AND t.deleted_at IS NULL`
	q, args = appendAssigneeFilter(q, args, params.Assignee)
	q += "\nGROUP BY range\nORDER BY range ASC"

	if err := r.db.WithContext(ctx).Raw(q, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []BucketCount{}
	}
	return rows, nil
}

// GetStageTransitions returns raw from→to transition counts from the audit_log.
func (r *repository) GetStageTransitions(ctx context.Context, orgID string, params ReportParams) ([]stageTransitionRow, error) {
	var rows []stageTransitionRow
	q := `SELECT
  json_extract(al.before_state, '$.stage') AS from_stage,
  json_extract(al.after_state,  '$.stage') AS to_stage,
  COUNT(*) AS transition_count
FROM audit_logs al
WHERE al.entity_type = 'thread'
  AND al.action = 'thread.updated'
  AND json_extract(al.before_state, '$.stage') IS NOT NULL
  AND json_extract(al.after_state,  '$.stage') IS NOT NULL
  AND json_extract(al.before_state, '$.stage') != json_extract(al.after_state, '$.stage')
  AND al.created_at BETWEEN ? AND ?
  AND al.entity_id IN (
    SELECT t.id FROM threads t
    JOIN boards b ON t.board_id = b.id
    JOIN spaces s ON b.space_id = s.id
    WHERE s.org_id = ? AND s.type = 'crm'
      AND t.deleted_at IS NULL
  )
GROUP BY from_stage, to_stage`
	args := []interface{}{params.From, params.To, orgID}

	if err := r.db.WithContext(ctx).Raw(q, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []stageTransitionRow{}
	}
	return rows, nil
}

// GetAvgTimeInStage returns the average hours between consecutive stage changes.
func (r *repository) GetAvgTimeInStage(ctx context.Context, orgID string, params ReportParams) ([]StageAvgTime, error) {
	var rows []StageAvgTime
	q := `SELECT
  json_extract(a1.after_state, '$.stage') AS stage,
  AVG((JULIANDAY(a2.created_at) - JULIANDAY(a1.created_at)) * 24) AS avg_hours
FROM audit_logs a1
JOIN audit_logs a2 ON a2.entity_id = a1.entity_id
  AND a2.entity_type = 'thread'
  AND a2.action = 'thread.updated'
  AND a2.created_at > a1.created_at
  AND json_extract(a2.before_state, '$.stage') = json_extract(a1.after_state, '$.stage')
WHERE a1.entity_type = 'thread'
  AND a1.action = 'thread.updated'
  AND json_extract(a1.after_state, '$.stage') IS NOT NULL
  AND a1.created_at BETWEEN ? AND ?
  AND a1.entity_id IN (
    SELECT t.id FROM threads t
    JOIN boards b ON t.board_id = b.id
    JOIN spaces s ON b.space_id = s.id
    WHERE s.org_id = ? AND s.type = 'crm'
      AND t.deleted_at IS NULL
  )
GROUP BY stage`
	args := []interface{}{params.From, params.To, orgID}

	if err := r.db.WithContext(ctx).Raw(q, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []StageAvgTime{}
	}
	return rows, nil
}

// GetSalesExportRows returns row-level data for CSV export.
func (r *repository) GetSalesExportRows(ctx context.Context, orgID string, params ReportParams) ([]SalesExportRow, error) {
	var rows []SalesExportRow
	args := []interface{}{orgID, params.From, params.To}
	q := `SELECT
  t.id,
  t.title,
  COALESCE(json_extract(t.metadata, '$.stage'), '') AS stage,
  COALESCE(json_extract(t.metadata, '$.assigned_to'), '') AS assigned_to,
  COALESCE(CAST(json_extract(t.metadata, '$.deal_value') AS TEXT), '') AS deal_value,
  COALESCE(CAST(json_extract(t.metadata, '$.score') AS TEXT), '') AS score,
  t.created_at
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'crm'
  AND t.created_at BETWEEN ? AND ?
  AND t.deleted_at IS NULL`
	q, args = appendAssigneeFilter(q, args, params.Assignee)
	q += "\nORDER BY t.created_at ASC"

	if err := r.db.WithContext(ctx).Raw(q, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []SalesExportRow{}
	}
	return rows, nil
}
