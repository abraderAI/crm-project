package reporting

import "context"

// --- Platform-Wide Support Queries (no org_id filter) ---

// GetPlatformSupportMetrics gathers all 7 support metrics across all orgs.
func (r *repository) GetPlatformSupportMetrics(ctx context.Context, params ReportParams) (*SupportMetrics, error) {
	statusBreakdown, err := r.getPlatformStatusBreakdown(ctx, params)
	if err != nil {
		return nil, err
	}

	volume, err := r.getPlatformVolumeOverTime(ctx, params)
	if err != nil {
		return nil, err
	}

	avgRes, err := r.getPlatformAvgResolutionHours(ctx, params)
	if err != nil {
		return nil, err
	}

	byAssignee, err := r.getPlatformTicketsByAssignee(ctx)
	if err != nil {
		return nil, err
	}

	byPriority, err := r.getPlatformTicketsByPriority(ctx, params)
	if err != nil {
		return nil, err
	}

	avgFirst, err := r.getPlatformAvgFirstResponseHours(ctx, params)
	if err != nil {
		return nil, err
	}

	overdue, err := r.getPlatformOverdueCount(ctx)
	if err != nil {
		return nil, err
	}

	return &SupportMetrics{
		StatusBreakdown:       statusBreakdown,
		VolumeOverTime:        volume,
		AvgResolutionHours:    avgRes,
		TicketsByAssignee:     byAssignee,
		TicketsByPriority:     byPriority,
		AvgFirstResponseHours: avgFirst,
		OverdueCount:          overdue,
	}, nil
}

func (r *repository) getPlatformStatusBreakdown(ctx context.Context, params ReportParams) (map[string]int64, error) {
	type row struct {
		Status string
		Count  int64
	}
	var rows []row
	q := `SELECT
  COALESCE(json_extract(t.metadata, '$.status'), 'unknown') AS status,
  COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.type = 'support'
  AND t.created_at BETWEEN ? AND ?
  AND t.deleted_at IS NULL
GROUP BY status`

	if err := r.db.WithContext(ctx).Raw(q, params.From, params.To).Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(rows))
	for _, row := range rows {
		result[row.Status] = row.Count
	}
	return result, nil
}

func (r *repository) getPlatformVolumeOverTime(ctx context.Context, params ReportParams) ([]DailyCount, error) {
	var rows []DailyCount
	q := `SELECT DATE(t.created_at) AS date, COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.type = 'support'
  AND t.created_at BETWEEN ? AND ?
  AND t.deleted_at IS NULL
GROUP BY DATE(t.created_at)
ORDER BY date ASC`

	if err := r.db.WithContext(ctx).Raw(q, params.From, params.To).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []DailyCount{}
	}
	return rows, nil
}

func (r *repository) getPlatformAvgResolutionHours(ctx context.Context, params ReportParams) (*float64, error) {
	var avg *float64
	q := `SELECT AVG((JULIANDAY(t.updated_at) - JULIANDAY(t.created_at)) * 24) AS avg_hours
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.type = 'support'
  AND t.created_at BETWEEN ? AND ?
  AND json_extract(t.metadata, '$.status') IN ('resolved', 'closed')
  AND t.deleted_at IS NULL`

	if err := r.db.WithContext(ctx).Raw(q, params.From, params.To).Row().Scan(&avg); err != nil {
		return nil, err
	}
	return avg, nil
}

func (r *repository) getPlatformTicketsByAssignee(ctx context.Context) ([]AssigneeCount, error) {
	var rows []AssigneeCount
	q := `SELECT
  json_extract(t.metadata, '$.assigned_to') AS user_id,
  COALESCE(u.display_name, json_extract(t.metadata, '$.assigned_to')) AS name,
  COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
LEFT JOIN user_shadows u ON u.clerk_user_id = json_extract(t.metadata, '$.assigned_to')
WHERE s.type = 'support'
  AND json_extract(t.metadata, '$.status') IN ('open', 'in_progress')
  AND json_extract(t.metadata, '$.assigned_to') IS NOT NULL
  AND t.deleted_at IS NULL
GROUP BY user_id
ORDER BY count DESC`

	if err := r.db.WithContext(ctx).Raw(q).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []AssigneeCount{}
	}
	return rows, nil
}

func (r *repository) getPlatformTicketsByPriority(ctx context.Context, params ReportParams) (map[string]int64, error) {
	type row struct {
		Priority string
		Count    int64
	}
	var rows []row
	q := `SELECT
  COALESCE(json_extract(t.metadata, '$.priority'), 'none') AS priority,
  COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.type = 'support'
  AND t.created_at BETWEEN ? AND ?
  AND t.deleted_at IS NULL
GROUP BY priority`

	if err := r.db.WithContext(ctx).Raw(q, params.From, params.To).Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(rows))
	for _, row := range rows {
		result[row.Priority] = row.Count
	}
	return result, nil
}

func (r *repository) getPlatformAvgFirstResponseHours(ctx context.Context, params ReportParams) (*float64, error) {
	var avg *float64
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
WHERE s.type = 'support'
  AND t.created_at BETWEEN ? AND ?
  AND t.deleted_at IS NULL`

	if err := r.db.WithContext(ctx).Raw(q, params.From, params.To).Row().Scan(&avg); err != nil {
		return nil, err
	}
	return avg, nil
}

func (r *repository) getPlatformOverdueCount(ctx context.Context) (int64, error) {
	var count int64
	q := `SELECT COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.type = 'support'
  AND json_extract(t.metadata, '$.status') IN ('open', 'in_progress')
  AND t.created_at < datetime('now', '-72 hours')
  AND t.deleted_at IS NULL`

	if err := r.db.WithContext(ctx).Raw(q).Row().Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
