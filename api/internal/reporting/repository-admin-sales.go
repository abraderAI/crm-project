package reporting

import "context"

// --- Platform-Wide Sales Queries (no org_id filter) ---

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
