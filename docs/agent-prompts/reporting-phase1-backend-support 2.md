Implement Reporting Phase 1 (Backend — Support Reporting) of the DEFT Evolution CRM project (abraderAI/crm-project).

START BY reading AGENTS.md in the repo root — it contains all required framework reading and
coding standards you MUST follow before writing any code.

---

## CONTEXT

This is the first phase of the Reporting Module (see SPECIFICATION-reporting.md and
PRD-reporting.md in the repo root for full requirements).

The main spec (all 16 phases) and IO Channels add-on (all 7 phases) are fully merged to
`main`. The `feat/reporting` collection branch has been created from `main` — all reporting
phase PRs target this branch.

**YOUR WORKING BRANCH**: check out `feat/reporting`, then create
`feat/reporting-phase1-backend-support` from it.

---

## YOUR TASK

Create the `api/internal/reporting/` package with:
1. The full package scaffold (used by all subsequent phases)
2. All 7 support ticket metric queries
3. Org-scoped support API and CSV export endpoints
4. Full test suite

### 1. Package Scaffold (`api/internal/reporting/`)

Follow the handler → service → repository pattern used by every other domain package
(e.g. `api/internal/org/`, `api/internal/thread/`). Create these files:

**`models.go`** — Request/response structs:
```go
// ReportParams — common query params for all report endpoints
type ReportParams struct {
    From     time.Time
    To       time.Time
    Assignee string // empty = all assignees
}

// SupportMetrics — response for GET /v1/orgs/{org}/reports/support
type SupportMetrics struct {
    StatusBreakdown      map[string]int64        `json:"status_breakdown"`
    VolumeOverTime       []DailyCount            `json:"volume_over_time"`
    AvgResolutionHours   *float64                `json:"avg_resolution_hours"`
    TicketsByAssignee    []AssigneeCount         `json:"tickets_by_assignee"`
    TicketsByPriority    map[string]int64        `json:"tickets_by_priority"`
    AvgFirstResponseHours *float64               `json:"avg_first_response_hours"`
    OverdueCount         int64                   `json:"overdue_count"`
}

type DailyCount struct {
    Date  string `json:"date"`  // "2026-03-01"
    Count int64  `json:"count"`
}

type AssigneeCount struct {
    UserID string `json:"user_id"`
    Name   string `json:"name"`
    Count  int64  `json:"count"`
}
```

**`repository.go`** — `ReportingRepository` interface + GORM/raw SQL implementation.
All methods accept `orgID string` and `params ReportParams`.

**`service.go`** — `ReportingService` that calls the repository and formats results.

**`handler.go`** — Chi HTTP handlers, param parsing, JSON/CSV responses.

**`reporting_test.go`** — All tests (unit + live API).

### 2. Support Metric Queries

All support queries scope to threads in boards in spaces where `s.org_id = orgID AND s.type = 'support'`.
When `params.Assignee` is non-empty, add `AND json_extract(t.metadata, '$.assigned_to') = ?`.

Use raw SQL via `db.Raw(...).Scan(...)` for aggregation queries (GORM is poor at GROUP BY).

**(1) Status breakdown** — count threads grouped by `json_extract(metadata, '$.status')`:
```sql
SELECT
  COALESCE(json_extract(t.metadata, '$.status'), 'unknown') AS status,
  COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'support'
  AND t.created_at BETWEEN ? AND ?
  AND t.deleted_at IS NULL
GROUP BY status
```

**(2) Volume over time** — daily ticket creation counts:
```sql
SELECT DATE(t.created_at) AS date, COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'support'
  AND t.created_at BETWEEN ? AND ?
  AND t.deleted_at IS NULL
GROUP BY DATE(t.created_at)
ORDER BY date ASC
```

**(3) Average resolution time** — mean hours from created_at to updated_at for
resolved/closed threads (updated_at is used as proxy for resolution time):
```sql
SELECT AVG((JULIANDAY(t.updated_at) - JULIANDAY(t.created_at)) * 24) AS avg_hours
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'support'
  AND t.created_at BETWEEN ? AND ?
  AND json_extract(t.metadata, '$.status') IN ('resolved', 'closed')
  AND t.deleted_at IS NULL
```
Return `nil` pointer if no rows match.

**(4) Tickets by assignee** — open ticket count per assigned user, joined to users_shadow
for display name:
```sql
SELECT
  json_extract(t.metadata, '$.assigned_to') AS user_id,
  COALESCE(u.name, json_extract(t.metadata, '$.assigned_to')) AS name,
  COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
LEFT JOIN users_shadow u ON u.user_id = json_extract(t.metadata, '$.assigned_to')
WHERE s.org_id = ? AND s.type = 'support'
  AND json_extract(t.metadata, '$.status') IN ('open', 'in_progress')
  AND json_extract(t.metadata, '$.assigned_to') IS NOT NULL
  AND t.deleted_at IS NULL
GROUP BY user_id
ORDER BY count DESC
```

**(5) Tickets by priority**:
```sql
SELECT
  COALESCE(json_extract(t.metadata, '$.priority'), 'none') AS priority,
  COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'support'
  AND t.created_at BETWEEN ? AND ?
  AND t.deleted_at IS NULL
GROUP BY priority
```

**(6) Average first response time** — mean hours from thread created_at to the first
message by someone other than the thread author:
```sql
SELECT AVG((JULIANDAY(fr.first_reply_at) - JULIANDAY(t.created_at)) * 24) AS avg_hours
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
  AND t.deleted_at IS NULL
```
Return `nil` pointer if no rows match.

**(7) Overdue count** — intentionally NO date range filter; always current state:
```sql
SELECT COUNT(*) AS count
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'support'
  AND json_extract(t.metadata, '$.status') IN ('open', 'in_progress')
  AND t.created_at < datetime('now', '-72 hours')
  AND t.deleted_at IS NULL
```

### 3. API Endpoints

Register on the Chi router in `api/internal/server/router.go` (or wherever routes are
registered — check existing patterns):

```
GET /v1/orgs/{org}/reports/support        → handler.GetSupportMetrics
GET /v1/orgs/{org}/reports/support/export → handler.GetSupportExport
```

Both routes MUST be behind the existing dual-auth middleware AND a role check that
requires `admin` or `owner` org role. Look at how `/v1/admin/*` and existing org routes
enforce roles — use the same pattern.

**`GetSupportMetrics`**:
- Parse `from`, `to` query params (ISO 8601 date, e.g. `2026-03-01`). Default: `from` = 30
  days ago, `to` = today (UTC). Return RFC 7807 on invalid date format.
- Parse optional `assignee` query param.
- Call `service.GetSupportMetrics(orgID, params)`.
- Return 200 with `SupportMetrics` JSON.

**`GetSupportExport`**:
- Same param parsing as above.
- Set response headers: `Content-Type: text/csv`,
  `Content-Disposition: attachment; filename="support-report.csv"`
- Stream row-level data (NOT aggregates) directly to the response writer using Go's
  `encoding/csv` package. Write one row at a time as you scan — do NOT buffer all rows.
- CSV columns: `id`, `title`, `status`, `priority`, `assigned_to`, `created_at`, `updated_at`
- Query: threads in support-type spaces with the same org/date/assignee filters.

### 4. Tests

**Unit tests** (mock DB or in-memory SQLite):
- `TestGetStatusBreakdown` — seed threads with 3 statuses, assert counts match
- `TestGetVolumeOverTime` — seed threads across 3 days, assert daily shape
- `TestGetAvgResolutionTime` — seed resolved threads with known durations, assert avg
- `TestGetTicketsByAssignee` — seed assigned threads, assert assignee list
- `TestGetTicketsByPriority` — seed threads with priorities, assert breakdown
- `TestGetAvgFirstResponseTime` — seed threads + replies, assert avg hours
- `TestGetOverdueCount` — seed old open threads, assert count
- `TestAssigneeFilterApplied` — verify assignee param narrows results
- `TestEmptyOrgReturnsZeros` — org with no support spaces returns valid zero-value response

**Live API tests** (start real server, real SQLite):
- Seed: create org → support-type space → board → 10 threads with varied
  status/priority/assignee metadata; add reply messages to some.
- `GET /v1/orgs/{org}/reports/support?from=X&to=Y` → assert HTTP 200, all 7 fields present,
  status_breakdown sums match seeded count.
- `GET /v1/orgs/{org}/reports/support/export` → assert HTTP 200, Content-Type: text/csv,
  body is valid CSV with correct header row.
- `GET /v1/orgs/{org}/reports/support` with viewer-role JWT → assert HTTP 403.
- `GET /v1/orgs/{org}/reports/support?from=invalid` → assert HTTP 400 RFC 7807.

Fuzzing (≥50 cases each):
- `FuzzDateParams` — random from/to strings
- `FuzzAssigneeParam` — random user ID strings

Coverage ≥85%.

---

## REQUIREMENTS CHECKLIST

- [ ] Read AGENTS.md and all deft files it references BEFORE writing any code
- [ ] Branch `feat/reporting-phase1-backend-support` created from `feat/reporting`
- [ ] `api/internal/reporting/` package created with handler/service/repository/models files
- [ ] All 7 support queries implemented with correct SQL
- [ ] Routes registered on Chi router with admin/owner RBAC enforcement
- [ ] CSV export streams rows without buffering
- [ ] All unit tests pass
- [ ] All live API tests pass (including 403 and 400 cases)
- [ ] Fuzzing ≥50 cases per input
- [ ] `task check` MUST fully pass before creating the PR
- [ ] File names use hyphens (not underscores)
- [ ] Conventional commits format for all commits
- [ ] PR targets `feat/reporting`
- [ ] PR title: `feat(reporting): add backend support reporting (Phase 1)`
