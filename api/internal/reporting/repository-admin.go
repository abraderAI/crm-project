package reporting

import "context"

// --- Platform-Wide Aggregate Queries (no org_id filter, no assignee filter) ---

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

// GetPlatformSalesMetrics gathers all 8 sales metrics across all orgs.
func (r *repository) GetPlatformSalesMetrics(ctx context.Context, params ReportParams) (*SalesMetrics, error) {
	funnel, err := r.getPlatformPipelineFunnel(ctx)
	if err != nil {
		return nil, err
	}

	velocity, err := r.getPlatformLeadVelocity(ctx, params)
	if err != nil {
		return nil, err
	}

	won, lost, err := r.getPlatformWinLossCounts(ctx, params)
	if err != nil {
		return nil, err
	}
	winRate, lossRate := computeWinLossRates(won, lost)

	avgDeal, err := r.getPlatformAvgDealValue(ctx, params)
	if err != nil {
		return nil, err
	}

	byAssignee, err := r.getPlatformLeadsByAssignee(ctx)
	if err != nil {
		return nil, err
	}

	scoreDist, err := r.getPlatformScoreDistribution(ctx, params)
	if err != nil {
		return nil, err
	}

	transitions, err := r.getPlatformStageTransitions(ctx, params)
	if err != nil {
		return nil, err
	}
	convRates := computeConversionRates(transitions)

	avgTime, err := r.getPlatformAvgTimeInStage(ctx, params)
	if err != nil {
		return nil, err
	}

	return &SalesMetrics{
		PipelineFunnel:       funnel,
		LeadVelocity:         velocity,
		WinRate:              winRate,
		LossRate:             lossRate,
		AvgDealValue:         avgDeal,
		LeadsByAssignee:      byAssignee,
		ScoreDistribution:    scoreDist,
		StageConversionRates: convRates,
		AvgTimeInStage:       avgTime,
	}, nil
}

func (r *repository) getPlatformPipelineFunnel(ctx context.Context) ([]StageCount, error) {
	var rows []StageCount
	q := `SELECT
  COALESCE(json_extract(t.metadata, '$.stage'), 'unknown') AS stage,
  COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.type = 'crm'
  AND t.deleted_at IS NULL
GROUP BY stage
ORDER BY count DESC`

	if err := r.db.WithContext(ctx).Raw(q).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []StageCount{}
	}
	return rows, nil
}

func (r *repository) getPlatformLeadVelocity(ctx context.Context, params ReportParams) ([]DailyCount, error) {
	var rows []DailyCount
	q := `SELECT DATE(t.created_at) AS date, COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.type = 'crm'
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

func (r *repository) getPlatformWinLossCounts(ctx context.Context, params ReportParams) (won, lost int64, err error) {
	type row struct {
		Stage string
		Count int64
	}
	var rows []row
	q := `SELECT
  json_extract(t.metadata, '$.stage') AS stage,
  COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.type = 'crm'
  AND json_extract(t.metadata, '$.stage') IN ('closed_won', 'closed_lost')
  AND t.created_at BETWEEN ? AND ?
  AND t.deleted_at IS NULL
GROUP BY stage`

	if err := r.db.WithContext(ctx).Raw(q, params.From, params.To).Scan(&rows).Error; err != nil {
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

func (r *repository) getPlatformAvgDealValue(ctx context.Context, params ReportParams) (*float64, error) {
	var avg *float64
	q := `SELECT AVG(CAST(json_extract(t.metadata, '$.deal_value') AS REAL)) AS avg_value
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.type = 'crm'
  AND t.created_at BETWEEN ? AND ?
  AND json_extract(t.metadata, '$.deal_value') IS NOT NULL
  AND t.deleted_at IS NULL`

	if err := r.db.WithContext(ctx).Raw(q, params.From, params.To).Row().Scan(&avg); err != nil {
		return nil, err
	}
	return avg, nil
}

func (r *repository) getPlatformLeadsByAssignee(ctx context.Context) ([]AssigneeCount, error) {
	var rows []AssigneeCount
	q := `SELECT
  json_extract(t.metadata, '$.assigned_to') AS user_id,
  COALESCE(u.display_name, json_extract(t.metadata, '$.assigned_to')) AS name,
  COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
LEFT JOIN user_shadows u ON u.clerk_user_id = json_extract(t.metadata, '$.assigned_to')
WHERE s.type = 'crm'
  AND json_extract(t.metadata, '$.stage') NOT IN ('closed_won', 'closed_lost')
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

func (r *repository) getPlatformScoreDistribution(ctx context.Context, params ReportParams) ([]BucketCount, error) {
	var rows []BucketCount
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
WHERE s.type = 'crm'
  AND t.created_at BETWEEN ? AND ?
  AND json_extract(t.metadata, '$.score') IS NOT NULL
  AND t.deleted_at IS NULL
GROUP BY range
ORDER BY range ASC`

	if err := r.db.WithContext(ctx).Raw(q, params.From, params.To).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []BucketCount{}
	}
	return rows, nil
}

func (r *repository) getPlatformStageTransitions(ctx context.Context, params ReportParams) ([]stageTransitionRow, error) {
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
    WHERE s.type = 'crm'
      AND t.deleted_at IS NULL
  )
GROUP BY from_stage, to_stage`

	if err := r.db.WithContext(ctx).Raw(q, params.From, params.To).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []stageTransitionRow{}
	}
	return rows, nil
}

func (r *repository) getPlatformAvgTimeInStage(ctx context.Context, params ReportParams) ([]StageAvgTime, error) {
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
    WHERE s.type = 'crm'
      AND t.deleted_at IS NULL
  )
GROUP BY stage`

	if err := r.db.WithContext(ctx).Raw(q, params.From, params.To).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []StageAvgTime{}
	}
	return rows, nil
}

// --- Per-Org Breakdown Queries ---

// orgSupportRow holds raw results from the per-org support breakdown query.
type orgSupportRow struct {
	OrgID              string
	OrgName            string
	OrgSlug            string
	OpenCount          int64
	OverdueCount       int64
	AvgResolutionHours *float64
	TotalInRange       int64
}

// GetOrgSupportBreakdown returns per-org support metrics ordered by total_in_range DESC.
func (r *repository) GetOrgSupportBreakdown(ctx context.Context, params ReportParams) ([]OrgSupportSummary, error) {
	var rows []orgSupportRow
	q := `SELECT
  o.id AS org_id,
  o.name AS org_name,
  o.slug AS org_slug,
  COUNT(CASE WHEN json_extract(t.metadata,'$.status') IN ('open','in_progress') THEN 1 END) AS open_count,
  COUNT(CASE WHEN json_extract(t.metadata,'$.status') IN ('open','in_progress')
              AND t.created_at < datetime('now','-72 hours') THEN 1 END) AS overdue_count,
  AVG(CASE WHEN json_extract(t.metadata,'$.status') IN ('resolved','closed')
           THEN (JULIANDAY(t.updated_at) - JULIANDAY(t.created_at)) * 24 END) AS avg_resolution_hours,
  COUNT(*) AS total_in_range
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
JOIN orgs o ON s.org_id = o.id
WHERE s.type = 'support'
  AND t.created_at BETWEEN ? AND ?
  AND t.deleted_at IS NULL
  AND o.deleted_at IS NULL
GROUP BY o.id
ORDER BY total_in_range DESC`

	if err := r.db.WithContext(ctx).Raw(q, params.From, params.To).Scan(&rows).Error; err != nil {
		return nil, err
	}

	// Build summaries (avg_first_response_hours will be joined from a separate query).
	summaries := make([]OrgSupportSummary, len(rows))
	for i, row := range rows {
		summaries[i] = OrgSupportSummary{
			OrgID:              row.OrgID,
			OrgName:            row.OrgName,
			OrgSlug:            row.OrgSlug,
			OpenCount:          row.OpenCount,
			OverdueCount:       row.OverdueCount,
			AvgResolutionHours: row.AvgResolutionHours,
			TotalInRange:       row.TotalInRange,
		}
	}
	return summaries, nil
}

// orgFirstResponseRow holds per-org first response hours.
type orgFirstResponseRow struct {
	OrgID    string
	AvgHours *float64
}

// GetOrgFirstResponseBreakdown returns avg first response hours grouped by org.
func (r *repository) GetOrgFirstResponseBreakdown(ctx context.Context, params ReportParams) (map[string]*float64, error) {
	var rows []orgFirstResponseRow
	q := `SELECT
  o.id AS org_id,
  AVG((JULIANDAY(fr.first_reply_at) - JULIANDAY(t.created_at)) * 24) AS avg_hours
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
JOIN orgs o ON s.org_id = o.id
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
  AND t.deleted_at IS NULL
  AND o.deleted_at IS NULL
GROUP BY o.id`

	if err := r.db.WithContext(ctx).Raw(q, params.From, params.To).Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]*float64, len(rows))
	for _, row := range rows {
		result[row.OrgID] = row.AvgHours
	}
	return result, nil
}

// orgSalesRow holds raw results from the per-org sales breakdown query.
type orgSalesRow struct {
	OrgID             string
	OrgName           string
	OrgSlug           string
	TotalLeads        int64
	OpenPipelineCount int64
	AvgDealValue      *float64
	WonCount          int64
	LostCount         int64
}

// GetOrgSalesBreakdown returns per-org sales metrics ordered by total_leads DESC.
func (r *repository) GetOrgSalesBreakdown(ctx context.Context, params ReportParams) ([]OrgSalesSummary, error) {
	var rows []orgSalesRow
	q := `SELECT
  o.id AS org_id,
  o.name AS org_name,
  o.slug AS org_slug,
  COUNT(*) AS total_leads,
  COUNT(CASE WHEN json_extract(t.metadata,'$.stage') NOT IN ('closed_won','closed_lost')
             THEN 1 END) AS open_pipeline_count,
  AVG(CAST(json_extract(t.metadata,'$.deal_value') AS REAL)) AS avg_deal_value,
  COUNT(CASE WHEN json_extract(t.metadata,'$.stage') = 'closed_won' THEN 1 END) AS won_count,
  COUNT(CASE WHEN json_extract(t.metadata,'$.stage') = 'closed_lost' THEN 1 END) AS lost_count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
JOIN orgs o ON s.org_id = o.id
WHERE s.type = 'crm'
  AND t.created_at BETWEEN ? AND ?
  AND t.deleted_at IS NULL
  AND o.deleted_at IS NULL
GROUP BY o.id
ORDER BY total_leads DESC`

	if err := r.db.WithContext(ctx).Raw(q, params.From, params.To).Scan(&rows).Error; err != nil {
		return nil, err
	}

	summaries := make([]OrgSalesSummary, len(rows))
	for i, row := range rows {
		winRate := float64(0)
		denom := row.WonCount + row.LostCount
		if denom > 0 {
			winRate = float64(row.WonCount) / float64(denom)
		}
		summaries[i] = OrgSalesSummary{
			OrgID:             row.OrgID,
			OrgName:           row.OrgName,
			OrgSlug:           row.OrgSlug,
			TotalLeads:        row.TotalLeads,
			WinRate:           winRate,
			AvgDealValue:      row.AvgDealValue,
			OpenPipelineCount: row.OpenPipelineCount,
		}
	}
	return summaries, nil
}

// --- Admin Export Queries ---

// GetAdminSupportExportRows returns row-level support data across all orgs with org context.
func (r *repository) GetAdminSupportExportRows(ctx context.Context, params ReportParams) ([]AdminSupportExportRow, error) {
	var rows []AdminSupportExportRow
	q := `SELECT
  o.id AS org_id,
  o.slug AS org_slug,
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
JOIN orgs o ON s.org_id = o.id
WHERE s.type = 'support'
  AND t.created_at BETWEEN ? AND ?
  AND t.deleted_at IS NULL
  AND o.deleted_at IS NULL
ORDER BY t.created_at ASC`

	if err := r.db.WithContext(ctx).Raw(q, params.From, params.To).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []AdminSupportExportRow{}
	}
	return rows, nil
}

// GetAdminSalesExportRows returns row-level sales data across all orgs with org context.
func (r *repository) GetAdminSalesExportRows(ctx context.Context, params ReportParams) ([]AdminSalesExportRow, error) {
	var rows []AdminSalesExportRow
	q := `SELECT
  o.id AS org_id,
  o.slug AS org_slug,
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
JOIN orgs o ON s.org_id = o.id
WHERE s.type = 'crm'
  AND t.created_at BETWEEN ? AND ?
  AND t.deleted_at IS NULL
  AND o.deleted_at IS NULL
ORDER BY t.created_at ASC`

	if err := r.db.WithContext(ctx).Raw(q, params.From, params.To).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []AdminSalesExportRow{}
	}
	return rows, nil
}
