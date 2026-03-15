Implement Reporting Phase 2 (Backend — Sales Reporting) of the DEFT Evolution CRM project (abraderAI/crm-project).

START BY reading AGENTS.md in the repo root — it contains all required framework reading and
coding standards you MUST follow before writing any code.

---

## CONTEXT

Phase 1 (Backend Support Reporting) has been merged into `feat/reporting`. It created the
`api/internal/reporting/` package with the full handler/service/repository/models scaffold
and all 7 support metric queries.

**YOUR WORKING BRANCH**: check out `feat/reporting` (which contains Phase 1), then create
`feat/reporting-phase2-backend-sales` from it.

---

## YOUR TASK

Extend the existing `api/internal/reporting/` package with all 8 sales lead metric queries
and the org-scoped sales API and CSV export endpoints. Do NOT modify Phase 1's support code —
only add new methods and wire new routes.

### 1. Add to `models.go`

```go
// SalesMetrics — response for GET /v1/orgs/{org}/reports/sales
type SalesMetrics struct {
    PipelineFunnel       []StageCount          `json:"pipeline_funnel"`
    LeadVelocity         []DailyCount          `json:"lead_velocity"`
    WinRate              float64               `json:"win_rate"`
    LossRate             float64               `json:"loss_rate"`
    AvgDealValue         *float64              `json:"avg_deal_value"`
    LeadsByAssignee      []AssigneeCount       `json:"leads_by_assignee"`
    ScoreDistribution    []BucketCount         `json:"score_distribution"`
    StageConversionRates []StageConversion     `json:"stage_conversion_rates"`
    AvgTimeInStage       []StageAvgTime        `json:"avg_time_in_stage"`
}

type StageCount struct {
    Stage string `json:"stage"`
    Count int64  `json:"count"`
}

type BucketCount struct {
    Range string `json:"range"` // "0-20", "20-40", etc.
    Count int64  `json:"count"`
}

type StageConversion struct {
    FromStage string  `json:"from_stage"`
    ToStage   string  `json:"to_stage"`
    Rate      float64 `json:"rate"` // 0.0–1.0
}

type StageAvgTime struct {
    Stage    string   `json:"stage"`
    AvgHours *float64 `json:"avg_hours"` // nil if no data
}
```

`DailyCount` and `AssigneeCount` are already defined in Phase 1.

### 2. Sales Metric Queries

All sales queries scope to threads in boards in spaces where
`s.org_id = orgID AND s.type = 'crm'`.
When `params.Assignee` is non-empty, add `AND json_extract(t.metadata, '$.assigned_to') = ?`.

Use raw SQL via `db.Raw(...).Scan(...)`.

**(1) Pipeline funnel** — current funnel state; intentionally NO date range filter:
```sql
SELECT
  COALESCE(json_extract(t.metadata, '$.stage'), 'unknown') AS stage,
  COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'crm'
  AND t.deleted_at IS NULL
GROUP BY stage
ORDER BY count DESC
```

**(2) Lead velocity** — new leads per day in date range:
```sql
SELECT DATE(t.created_at) AS date, COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'crm'
  AND t.created_at BETWEEN ? AND ?
  AND t.deleted_at IS NULL
GROUP BY DATE(t.created_at)
ORDER BY date ASC
```

**(3) Win/loss counts** — count closed_won and closed_lost in range; compute rates in Go:
```sql
SELECT
  json_extract(t.metadata, '$.stage') AS stage,
  COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'crm'
  AND json_extract(t.metadata, '$.stage') IN ('closed_won', 'closed_lost')
  AND t.created_at BETWEEN ? AND ?
  AND t.deleted_at IS NULL
GROUP BY stage
```
In the service layer: `win_rate = closed_won / (closed_won + closed_lost)`.
Return `0.0` for both if denominator is zero.

**(4) Average deal value**:
```sql
SELECT AVG(CAST(json_extract(t.metadata, '$.deal_value') AS REAL)) AS avg_value
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'crm'
  AND t.created_at BETWEEN ? AND ?
  AND json_extract(t.metadata, '$.deal_value') IS NOT NULL
  AND t.deleted_at IS NULL
```
Return `nil` pointer if no rows have `deal_value`.

**(5) Leads by assignee** — active (non-closed) leads only:
```sql
SELECT
  json_extract(t.metadata, '$.assigned_to') AS user_id,
  COALESCE(u.name, json_extract(t.metadata, '$.assigned_to')) AS name,
  COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
LEFT JOIN users_shadow u ON u.user_id = json_extract(t.metadata, '$.assigned_to')
WHERE s.org_id = ? AND s.type = 'crm'
  AND json_extract(t.metadata, '$.stage') NOT IN ('closed_won', 'closed_lost')
  AND json_extract(t.metadata, '$.assigned_to') IS NOT NULL
  AND t.deleted_at IS NULL
GROUP BY user_id
ORDER BY count DESC
```

**(6) Score distribution** — 5 equal-width buckets (0–100 scale):
```sql
SELECT
  CASE
    WHEN CAST(json_extract(t.metadata, '$.score') AS REAL) < 20  THEN '0-20'
    WHEN CAST(json_extract(t.metadata, '$.score') AS REAL) < 40  THEN '20-40'
    WHEN CAST(json_extract(t.metadata, '$.score') AS REAL) < 60  THEN '40-60'
    WHEN CAST(json_extract(t.metadata, '$.score') AS REAL) < 80  THEN '60-80'
    ELSE '80-100'
  END AS bucket,
  COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'crm'
  AND t.created_at BETWEEN ? AND ?
  AND json_extract(t.metadata, '$.score') IS NOT NULL
  AND t.deleted_at IS NULL
GROUP BY bucket
ORDER BY bucket ASC
```

**(7) Stage conversion rates** — derived from `audit_log` table:
```sql
SELECT
  json_extract(al.before_state, '$.stage') AS from_stage,
  json_extract(al.after_state,  '$.stage') AS to_stage,
  COUNT(*) AS transition_count
FROM audit_log al
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
GROUP BY from_stage, to_stage
```
In the service layer, compute rate per from_stage:
`rate = count(from→to) / SUM(count(from→*))`.
If the audit_log has no matching records, return an empty slice (not an error).

**(8) Average time in stage** — audit_log self-join to find time between consecutive
stage changes per thread:
```sql
SELECT
  json_extract(a1.after_state, '$.stage') AS stage,
  AVG((JULIANDAY(a2.created_at) - JULIANDAY(a1.created_at)) * 24) AS avg_hours
FROM audit_log a1
JOIN audit_log a2 ON a2.entity_id = a1.entity_id
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
GROUP BY stage
```
Return `nil` for `avg_hours` on stages with no matching pairs. Return empty slice (not
error) if no audit data exists.

### 3. API Endpoints

Add to the Chi router (alongside the Phase 1 support routes):

```
GET /v1/orgs/{org}/reports/sales        → handler.GetSalesMetrics
GET /v1/orgs/{org}/reports/sales/export → handler.GetSalesExport
```

Same admin/owner RBAC enforcement as Phase 1.

**`GetSalesMetrics`**: same param parsing as `GetSupportMetrics`. Return `SalesMetrics` JSON.

**`GetSalesExport`**: stream CSV with columns:
`id`, `title`, `stage`, `assigned_to`, `deal_value`, `score`, `created_at`.

### 4. Tests

**Unit tests**:
- `TestGetPipelineFunnel` — seed CRM threads at 3 stages, assert counts
- `TestGetLeadVelocity` — seed leads over 3 days, assert shape
- `TestGetWinLossRate` — seed closed_won + closed_lost, assert rates; test zero-division case
- `TestGetAvgDealValue` — seed threads with/without deal_value, assert avg ignores nulls
- `TestGetLeadsByAssignee` — seed assigned active leads, assert list
- `TestGetScoreDistribution` — seed scored leads, assert bucket counts
- `TestGetStageConversionRates` — seed audit_log stage-change records, assert rates; test empty audit log returns empty slice
- `TestGetAvgTimeInStage` — seed paired audit_log records, assert hours; test missing data returns nil avg
- `TestSalesAssigneeFilter` — verify filter narrows results

**Live API tests** (start real server, real SQLite):
- Seed: org → CRM-type space → board → 15 threads at various stages with deal_value,
  score metadata, and assignees; seed audit_log stage-change records for at least 3 threads.
- `GET /v1/orgs/{org}/reports/sales?from=X&to=Y` → assert HTTP 200, all 8 fields present.
- `GET /v1/orgs/{org}/reports/sales/export` → assert HTTP 200, Content-Type: text/csv.
- `GET /v1/orgs/{org}/reports/sales` with viewer-role JWT → assert HTTP 403.
- Empty audit log case: org with CRM threads but no audit records → assert
  `stage_conversion_rates: []` and `avg_time_in_stage: []`.

Fuzzing (≥50 cases each):
- `FuzzStageName` — random stage name strings in audit_log before/after
- `FuzzDealValue` — random deal_value metadata strings

Coverage ≥85%.

---

## REQUIREMENTS CHECKLIST

- [ ] Read AGENTS.md and all deft files it references BEFORE writing any code
- [ ] Branch `feat/reporting-phase2-backend-sales` created from `feat/reporting`
- [ ] `SalesMetrics` and related structs added to `models.go`
- [ ] All 8 sales queries implemented with correct SQL
- [ ] Audit log queries return empty slices (not errors) when no data exists
- [ ] Routes registered on Chi router with admin/owner RBAC enforcement
- [ ] CSV export streams rows without buffering
- [ ] All unit tests pass including edge cases
- [ ] All live API tests pass
- [ ] Fuzzing ≥50 cases per input
- [ ] `task check` MUST fully pass before creating the PR
- [ ] Conventional commits format
- [ ] PR targets `feat/reporting`
- [ ] PR title: `feat(reporting): add backend sales reporting (Phase 2)`
