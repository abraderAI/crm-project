Implement Reporting Phase 3 (Backend — Platform Admin Reporting Endpoints) of the DEFT Evolution CRM project (abraderAI/crm-project).

START BY reading AGENTS.md in the repo root — it contains all required framework reading and
coding standards you MUST follow before writing any code.

---

## CONTEXT

Phases 1 and 2 have been merged into `feat/reporting`:
- Phase 1: `api/internal/reporting/` package with support metrics (7 queries, 2 endpoints)
- Phase 2: Sales metrics added to the same package (8 queries, 2 endpoints)

**YOUR WORKING BRANCH**: check out `feat/reporting` (which contains Phases 1 + 2), then
create `feat/reporting-phase3-backend-admin` from it.

---

## YOUR TASK

Add platform-wide aggregate endpoints and per-org breakdown queries to the existing
`api/internal/reporting/` package. Platform admin routes follow the same pattern as the
existing `/v1/admin/*` endpoints — check `api/internal/admin/` for the exact pattern.

### 1. Add to `models.go`

```go
// AdminSupportMetrics — response for GET /v1/admin/reports/support
type AdminSupportMetrics struct {
    SupportMetrics                      // embed all platform-wide fields
    OrgBreakdown []OrgSupportSummary `json:"org_breakdown"`
}

// AdminSalesMetrics — response for GET /v1/admin/reports/sales
type AdminSalesMetrics struct {
    SalesMetrics                      // embed all platform-wide fields
    OrgBreakdown []OrgSalesSummary `json:"org_breakdown"`
}

type OrgSupportSummary struct {
    OrgID                string   `json:"org_id"`
    OrgName              string   `json:"org_name"`
    OrgSlug              string   `json:"org_slug"`
    OpenCount            int64    `json:"open_count"`
    OverdueCount         int64    `json:"overdue_count"`
    AvgResolutionHours   *float64 `json:"avg_resolution_hours"`
    AvgFirstResponseHours *float64 `json:"avg_first_response_hours"`
    TotalInRange         int64    `json:"total_in_range"`
}

type OrgSalesSummary struct {
    OrgID              string   `json:"org_id"`
    OrgName            string   `json:"org_name"`
    OrgSlug            string   `json:"org_slug"`
    TotalLeads         int64    `json:"total_leads"`
    WinRate            float64  `json:"win_rate"`
    AvgDealValue       *float64 `json:"avg_deal_value"`
    OpenPipelineCount  int64    `json:"open_pipeline_count"`
}
```

### 2. Platform-Wide Aggregate Queries

These reuse the exact same SQL as Phases 1 and 2 but **remove the `s.org_id = ?`
constraint** so they aggregate across all orgs. Add new repository methods:
`GetPlatformSupportMetrics(params ReportParams) (SupportMetrics, error)`
`GetPlatformSalesMetrics(params ReportParams) (SalesMetrics, error)`

The `Assignee` field in params is ignored for platform-wide queries (too ambiguous
cross-org). Do not apply it.

### 3. Per-Org Breakdown Queries

**Support breakdown per org**:
```sql
SELECT
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
ORDER BY total_in_range DESC
```

For `avg_first_response_hours` per org, run a separate query that mirrors the Phase 1
first-response query but groups by `o.id`. Join the results to the breakdown in the
service layer (not SQL).

**Sales breakdown per org**:
```sql
SELECT
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
ORDER BY total_leads DESC
```

Compute `win_rate = won_count / (won_count + lost_count)` in Go (return 0.0 if denominator
is 0).

### 4. API Endpoints

Register on the existing admin router (inside the `/v1/admin/` subrouter that already
enforces platform admin checks):

```
GET /v1/admin/reports/support        → handler.GetAdminSupportMetrics
GET /v1/admin/reports/support/export → handler.GetAdminSupportExport
GET /v1/admin/reports/sales          → handler.GetAdminSalesMetrics
GET /v1/admin/reports/sales/export   → handler.GetAdminSalesExport
```

**`GetAdminSupportMetrics`** and **`GetAdminSalesMetrics`**:
- Parse `from` / `to` params (same defaults as Phase 1).
- Run platform-wide aggregate + per-org breakdown **concurrently** using goroutines +
  `sync.WaitGroup` or `errgroup`. Merge results.
- Return `AdminSupportMetrics` / `AdminSalesMetrics` JSON.

**`GetAdminSupportExport`** and **`GetAdminSalesExport`**:
- Stream CSV of row-level data across all orgs.
- Prepend `org_id` and `org_slug` as first two columns before the existing per-phase
  column sets.

### 5. Tests

**Unit tests**:
- `TestGetPlatformSupportMetrics` — seed 2 orgs with support threads, assert platform
  totals = sum of per-org totals
- `TestGetPlatformSalesMetrics` — seed 2 orgs with CRM threads, assert platform totals
- `TestGetOrgSupportBreakdown` — assert breakdown has correct per-org values, ordered by
  total_in_range DESC
- `TestGetOrgSalesBreakdown` — assert per-org win_rate computed correctly, zero-division safe
- `TestConcurrentAggregation` — goroutine-based aggregation doesn't race (use `-race` flag)

**Live API tests**:
- Seed 3 orgs with varied support + CRM data.
- `GET /v1/admin/reports/support` as platform admin → assert HTTP 200, `org_breakdown`
  length = 3, platform totals sum correctly.
- `GET /v1/admin/reports/sales` as platform admin → assert HTTP 200, `org_breakdown` present.
- `GET /v1/admin/reports/support` as org admin (non-platform-admin) → assert HTTP 403.
- `GET /v1/admin/reports/support/export` → assert CSV has `org_id` as first column.

Fuzzing: reuse Phase 1 `FuzzDateParams` — no new fuzz inputs unique to this phase.

Coverage ≥85%.

---

## REQUIREMENTS CHECKLIST

- [ ] Read AGENTS.md and all deft files it references BEFORE writing any code
- [ ] Branch `feat/reporting-phase3-backend-admin` created from `feat/reporting`
- [ ] Admin response structs added to `models.go`
- [ ] Platform-wide aggregate queries implemented (no org_id filter)
- [ ] Per-org breakdown queries implemented and ordered correctly
- [ ] Goroutine-based concurrent aggregation (no data race)
- [ ] 4 admin routes registered on admin subrouter
- [ ] CSV export includes org_id/org_slug prefix columns
- [ ] 403 enforced for non-platform-admins
- [ ] All tests pass including concurrent race test
- [ ] `task check` MUST fully pass before creating the PR
- [ ] Conventional commits format
- [ ] PR targets `feat/reporting`
- [ ] PR title: `feat(reporting): add platform admin reporting endpoints (Phase 3)`
