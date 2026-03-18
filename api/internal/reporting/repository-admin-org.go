package reporting

import "context"

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
