# DEFT Evolution — Reporting Module

*Generated from `vbrief/specification-reporting.vbrief.json` — 2026-03-15*
*PRD: `PRD-reporting.md`*

## Overview

The Reporting Module adds two self-service analytics dashboards — one for support tickets and
one for sales leads — accessible to org admins at `/reports/support` and `/reports/sales`.
Platform admins get cross-org aggregate views with per-org breakdown tables at
`/admin/reports/support` and `/admin/reports/sales`. All metrics are computed at query time
from existing SQLite data (no caching layer). Charts use Recharts. CSV export is server-side
streamed. Data is scoped by Space type: `support` spaces feed the support dashboard; `crm`
spaces feed the sales dashboard.

This module MUST NOT begin until the main spec's backend AND frontend phases are fully merged.
It depends on: threads, boards, spaces, messages, memberships, audit_log, users_shadow tables
and the existing RBAC middleware.

---

## Architecture

### New Backend Package

```
api/internal/reporting/
├── handler.go          # HTTP handlers (org-scoped + admin)
├── service.go          # Business logic, query orchestration
├── repository.go       # All SQL queries
├── models.go           # Request/response structs
└── reporting_test.go   # Unit + live API tests
```

Registered on the existing Chi router:
- `GET /v1/orgs/{org}/reports/support` — org support metrics (owner/admin only)
- `GET /v1/orgs/{org}/reports/support/export` — org support CSV export
- `GET /v1/orgs/{org}/reports/sales` — org sales metrics (owner/admin only)
- `GET /v1/orgs/{org}/reports/sales/export` — org sales CSV export
- `GET /v1/admin/reports/support` — platform support metrics + per-org breakdown
- `GET /v1/admin/reports/support/export` — platform support CSV export
- `GET /v1/admin/reports/sales` — platform sales metrics + per-org breakdown
- `GET /v1/admin/reports/sales/export` — platform sales CSV export

**Common query parameters** (all report endpoints):
- `from` — ISO 8601 date string, start of range (default: 30 days ago)
- `to` — ISO 8601 date string, end of range (default: today)
- `assignee` — optional user ID; if present, filter all metrics to that assignee

### Response Shapes

**Support metrics** (`SupportMetrics`):
```json
{
  "status_breakdown":        { "open": 14, "in_progress": 7, "resolved": 23, "closed": 45 },
  "volume_over_time":        [{ "date": "2026-03-01", "count": 5 }],
  "avg_resolution_hours":    18.5,
  "tickets_by_assignee":     [{ "user_id": "...", "name": "...", "count": 8 }],
  "tickets_by_priority":     { "low": 10, "medium": 25, "high": 12, "urgent": 3 },
  "avg_first_response_hours": 2.3,
  "overdue_count":           4
}
```

**Sales metrics** (`SalesMetrics`):
```json
{
  "pipeline_funnel":         [{ "stage": "new_lead", "count": 45 }],
  "lead_velocity":           [{ "date": "2026-03-01", "count": 3 }],
  "win_rate":                0.32,
  "loss_rate":               0.18,
  "avg_deal_value":          12500.00,
  "leads_by_assignee":       [{ "user_id": "...", "name": "...", "count": 6 }],
  "score_distribution":      [{ "range": "0-20", "count": 5 }],
  "stage_conversion_rates":  [{ "from_stage": "new_lead", "to_stage": "qualified", "rate": 0.65 }],
  "avg_time_in_stage":       [{ "stage": "new_lead", "avg_hours": 48.2 }]
}
```

**Admin aggregate response** wraps the above with an additional `org_breakdown` array:
```json
{
  // ...all platform-wide aggregates (same shape as org response)...
  "org_breakdown": [
    {
      "org_id": "...", "org_name": "...", "org_slug": "...",
      // support: open_count, overdue_count, avg_resolution_hours, avg_first_response_hours, total_in_range
      // sales:   total_leads, win_rate, avg_deal_value, open_pipeline_count
    }
  ]
}
```

### New Frontend Structure

```
web/src/app/
├── reports/
│   ├── layout.tsx                # Reports shell: tab nav (Support / Sales)
│   ├── support/page.tsx          # Support dashboard page
│   └── sales/page.tsx            # Sales dashboard page
└── admin/reports/
    ├── support/page.tsx          # Admin support dashboard
    └── sales/page.tsx            # Admin sales dashboard

web/src/components/reports/
├── DateRangePicker.tsx           # Shared date range picker (shadcn/ui Popover + Calendar)
├── AssigneeFilter.tsx            # Shared assignee dropdown
├── MetricCard.tsx                # KPI card with value, label, link-out href
├── ExportButton.tsx              # CSV download trigger
├── support/
│   ├── StatusBreakdownChart.tsx  # Recharts BarChart / PieChart
│   ├── VolumeOverTimeChart.tsx   # Recharts LineChart
│   ├── TicketsByAssigneeChart.tsx # Recharts HorizontalBarChart
│   └── TicketsByPriorityChart.tsx # Recharts BarChart
└── sales/
    ├── PipelineFunnelChart.tsx   # Recharts BarChart (staged)
    ├── LeadVelocityChart.tsx     # Recharts LineChart
    ├── LeadsByAssigneeChart.tsx  # Recharts HorizontalBarChart
    ├── ScoreDistributionChart.tsx # Recharts BarChart (histogram)
    ├── StageConversionChart.tsx  # Recharts BarChart (% labels)
    └── TimeInStageChart.tsx      # Recharts HorizontalBarChart
```

The sidebar `AppLayout` gains a **Reports** nav item (visible to org owner/admin only).

---

## Key SQL Patterns

All org-scoped support queries share this base join:
```sql
FROM threads t
JOIN boards b ON t.board_id = b.id
JOIN spaces s ON b.space_id = s.id
WHERE s.org_id = ? AND s.type = 'support'
  AND t.created_at BETWEEN ? AND ?
  AND t.deleted_at IS NULL
  [AND json_extract(t.metadata, '$.assigned_to') = ?]  -- if assignee filter
```

All org-scoped sales queries use `s.type = 'crm'` instead.

Stage conversion and time-in-stage query the `audit_log` table for records where
`entity_type = 'thread'`, `action = 'thread.updated'`, and the `stage` field differs between
`before_state` and `after_state`. Missing audit records return `null` / `0` (NFR-5).

Overdue count intentionally omits the date range filter — it always reflects current state:
`status IN ('open','in_progress') AND created_at < datetime('now','-72 hours')`.

---

## Implementation Plan

### Phase 1: Backend — Support Reporting (no IO channel dependencies)

*Depends on: main spec merged to `main`. Start here alongside Phase 2.*

Implements the `reporting` package scaffold plus all support ticket metric queries and endpoints.

- **reporting.scaffold** — Create `api/internal/reporting/` package. Define `ReportingRepository`
  interface, `ReportingService`, HTTP handler. Register all 8 routes on the Chi router under
  the existing auth + RBAC middleware. Add `owner`/`admin` role check middleware for
  org-scoped routes; platform admin check for `/v1/admin/reports/*`.

- **reporting.support-queries** — Implement all 7 support metric SQL queries in
  `repository.go`:
  (1) status breakdown (`json_extract` + GROUP BY),
  (2) volume over time (DATE + GROUP BY),
  (3) avg resolution time (JULIANDAY diff for resolved/closed threads),
  (4) tickets by assignee (open tickets grouped by `assigned_to`, joined to `users_shadow`
      for display names),
  (5) tickets by priority (GROUP BY `priority` metadata field),
  (6) avg first response time (subquery: MIN message created_at where author ≠ thread author),
  (7) overdue count (no date range; `created_at < NOW − 72h`, status open/in_progress).
  All queries MUST accept `from`, `to`, and optional `assignee` params.

- **reporting.support-api** — Wire `GetSupportMetrics` handler: parse + validate query params
  (default `from` = 30 days ago, default `to` = today), call service, return
  `SupportMetrics` JSON. Return RFC 7807 on invalid date format.

- **reporting.support-export** — `GetSupportExport` handler: stream row-level CSV to response
  writer (`Content-Type: text/csv`, `Content-Disposition: attachment`). Columns: `id`,
  `title`, `status`, `priority`, `assigned_to`, `created_at`, `updated_at`. MUST NOT buffer
  entire result set in memory — write rows as they are scanned.

- **reporting.support-tests** — Unit tests: each query function with mock data (status counts,
  volume shape, resolution calc, first response calc, overdue logic). Live API tests: start
  real server, seed support-type space + threads with varied statuses/priorities/assignees,
  `GET /v1/orgs/{org}/reports/support`, assert all 7 metric fields present with correct values.
  Test 403 for non-admin role. Test CSV export returns valid CSV with correct headers. Test
  assignee filter narrows results. Fuzzing ≥50 per (date range params, assignee param).
  Coverage ≥85%.

---

### Phase 2: Backend — Sales Reporting (parallel with Phase 1)

*Depends on: reporting.scaffold from Phase 1. Can start once scaffold is committed.*

Implements all sales lead metric queries and endpoints.

- **reporting.sales-queries** — Implement all 8 sales metric SQL queries:
  (1) pipeline funnel (GROUP BY `stage`, all non-deleted CRM threads — no date filter on
      current funnel state),
  (2) lead velocity (DATE + GROUP BY, date range applied),
  (3) win/loss rate (count `closed_won` vs `closed_lost` in range; compute ratio in Go;
      return 0 if denominator is 0),
  (4) avg deal value (`AVG(CAST(json_extract(metadata,'$.deal_value') AS REAL))` where not null),
  (5) leads by assignee (open/active leads grouped by `assigned_to`, joined to `users_shadow`),
  (6) score distribution (CASE bucketing into 5 × 20-point bands: 0–20, 20–40, 40–60, 60–80,
      80–100),
  (7) stage conversion rates (audit_log query: `entity_type='thread'`,
      `action='thread.updated'`, stage field changed; GROUP BY from_stage/to_stage; compute
      rate as `count(to_next) / count(total_from)` in Go; gracefully return empty slice if
      no audit data),
  (8) avg time in stage (audit_log self-join: pair consecutive stage-change events per thread,
      compute JULIANDAY diff, AVG by stage; return null per stage if insufficient data).

- **reporting.sales-api** — `GetSalesMetrics` handler: same param parsing as support. Return
  `SalesMetrics` JSON.

- **reporting.sales-export** — `GetSalesExport` handler: stream CSV. Columns: `id`, `title`,
  `stage`, `assigned_to`, `deal_value`, `score`, `created_at`.

- **reporting.sales-tests** — Unit tests for all 8 queries including edge cases (no closed
  deals → win_rate=0, no audit log → empty conversion/time arrays). Live API tests: seed CRM
  space + threads at various stages with deal_value and score metadata, assert all 8 fields
  correct. Test CSV export. Test assignee filter. Fuzzing ≥50 per (stage names, numeric
  metadata values). Coverage ≥85%.

---

### Phase 3: Backend — Platform Admin Reporting Endpoints (depends on: Phase 1 + Phase 2)

Adds cross-org aggregate queries and per-org breakdown for both dashboards.

- **reporting.admin-queries** — Implement platform-wide variants of all support and sales
  queries by removing the `s.org_id = ?` constraint. Implement two per-org breakdown queries:
  - Support breakdown: for each org, return `open_count`, `overdue_count`,
    `avg_resolution_hours`, `avg_first_response_hours`, `total_in_range`.
  - Sales breakdown: for each org, return `total_leads`, `win_rate`, `avg_deal_value`,
    `open_pipeline_count`. Results ordered by `total_in_range` DESC (support) or
    `total_leads` DESC (sales).

- **reporting.admin-api** — `GetAdminSupportMetrics` and `GetAdminSalesMetrics` handlers:
  run platform-wide aggregate + per-org breakdown concurrently (Go goroutines), merge into
  admin response shape. Platform admin RBAC check MUST be enforced (existing platform admin
  middleware).

- **reporting.admin-export** — `GetAdminSupportExport` and `GetAdminSalesExport`: same
  streaming CSV as org endpoints but include an `org_id` / `org_slug` column prepended.

- **reporting.admin-tests** — Seed 3+ orgs with varied data. Assert platform aggregate sums
  match per-org totals. Assert per-org breakdown is ordered correctly. Assert non-platform-admin
  receives 403. Live API tests. Coverage ≥85%.

---

### Phase 4: Frontend — Shared Reporting Infrastructure (depends on: Phase 1 + Phase 2)

*Builds all reusable reporting components and wires up the API client. No dashboard pages yet.*

- **reporting.recharts-setup** — Add `recharts` dependency to `web/package.json`. Confirm
  TypeScript types available (`@types/recharts` or built-in). Add a barrel
  `web/src/components/reports/index.ts`.

- **reporting.api-client** — Add typed fetch functions to the existing API client:
  `getSupportMetrics(orgId, params)`, `getSalesMetrics(orgId, params)`,
  `getAdminSupportMetrics(params)`, `getAdminSalesMetrics(params)`. Return types match
  `SupportMetrics` and `SalesMetrics` interfaces defined in `lib/types.ts`.

- **reporting.shared-components** — Implement:
  - `DateRangePicker` — shadcn/ui `Popover` + `Calendar` (or date-range variant); emits
    `{ from: Date, to: Date }` via `onChange`. Shows formatted range label on trigger button.
  - `AssigneeFilter` — shadcn/ui `Select` populated from org members API; emits user ID or
    `null` for "All". Shows member avatar + name.
  - `MetricCard` — displays a KPI value + label + optional sub-label; wraps in a Next.js
    `Link` when `href` prop is provided (link-out drilldown, FR-5).
  - `ExportButton` — triggers a fetch to the export endpoint and initiates browser download
    via `URL.createObjectURL`; shows loading spinner during fetch.

- **reporting.layout** — Create `web/src/app/reports/layout.tsx`: reports shell with tab nav
  (Support | Sales) using shadcn/ui `Tabs` or plain `Link` tabs. Add **Reports** to the
  `AppLayout` sidebar under the existing nav items — visible only to org owner/admin role.

- **reporting.shared-tests** — Vitest unit tests: `MetricCard` renders href as link when
  provided; `DateRangePicker` emits correct range; `AssigneeFilter` renders correct options;
  `ExportButton` calls correct endpoint. Coverage ≥85%.

---

### Phase 5: Frontend — Support Dashboard (depends on: Phase 4)

*Can run in parallel with Phase 6 once Phase 4 is merged.*

- **reporting.support-charts** — Implement support-specific chart components in
  `web/src/components/reports/support/`:
  - `StatusBreakdownChart` — Recharts `PieChart` (or `BarChart`) showing open / in_progress /
    resolved / closed counts. Each segment links out to thread list filtered by that status.
  - `VolumeOverTimeChart` — Recharts `AreaChart` / `LineChart` with daily tick marks.
  - `TicketsByAssigneeChart` — Recharts horizontal `BarChart` sorted by count descending.
    Each bar links out to thread list filtered by `assigned_to`.
  - `TicketsByPriorityChart` — Recharts `BarChart` with priority-based colour coding.
  - Standalone `MetricCard` instances for: avg resolution time (hours), avg first response
    time (hours), overdue count (links to list filtered by `status=open&overdue=true`).

- **reporting.support-page** — `web/src/app/reports/support/page.tsx`: Server Component that
  reads org context (from session/cookie), renders `DateRangePicker` + `AssigneeFilter` as
  Client Components with URL search-param sync (`useSearchParams`), fetches
  `getSupportMetrics` with current params, passes data to chart components. Shows skeleton
  loaders (`Suspense`) while fetching. `ExportButton` points to org support export endpoint
  with current filter params.

- **reporting.support-tests** — Vitest component tests: each chart renders without error with
  fixture data; `StatusBreakdownChart` slice count matches input. Playwright E2E: sign in as
  org admin → navigate to `/reports/support` → assert all 7 metric sections visible → change
  date range → assert page re-fetches → click overdue count card → assert navigation to
  filtered thread list. Coverage ≥85%.

---

### Phase 6: Frontend — Sales Dashboard (depends on: Phase 4, parallel with Phase 5)

- **reporting.sales-charts** — Implement sales-specific chart components in
  `web/src/components/reports/sales/`:
  - `PipelineFunnelChart` — Recharts `BarChart` with one bar per stage, ordered by pipeline
    progression. Each bar links out to CRM thread list filtered by that stage.
  - `LeadVelocityChart` — Recharts `AreaChart` / `LineChart` with daily tick marks.
  - `LeadsByAssigneeChart` — Recharts horizontal `BarChart`.
  - `ScoreDistributionChart` — Recharts `BarChart` histogram (5 buckets, x-axis = range label).
  - `StageConversionChart` — Recharts `BarChart` with percentage labels; stages on x-axis,
    conversion rate on y-axis (0–100%). Shows "No data" state if audit log is empty.
  - `TimeInStageChart` — Recharts horizontal `BarChart` (avg hours per stage).
  - Standalone `MetricCard` instances for: win rate (%), loss rate (%), avg deal value ($).

- **reporting.sales-page** — `web/src/app/reports/sales/page.tsx`: same pattern as support
  page. Fetches `getSalesMetrics`. `ExportButton` points to sales export endpoint.

- **reporting.sales-tests** — Vitest component tests: each chart renders correctly with
  fixture data; empty audit log renders `StageConversionChart` with "No data" state.
  Playwright E2E: navigate to `/reports/sales` → assert all 8 metric sections visible →
  change assignee filter → assert metric cards update → click pipeline funnel bar → assert
  navigation to CRM filtered list. Coverage ≥85%.

---

### Phase 7: Frontend — Platform Admin Dashboards (depends on: Phase 3 + Phase 5 + Phase 6)

Reuses all chart components from Phases 5–6; adds the org breakdown table and admin pages.

- **reporting.org-breakdown-table** — `OrgBreakdownTable` component: shadcn/ui `Table` with
  sortable columns. Support columns: Org, Open Tickets, Overdue, Avg Resolution (h), Avg
  First Response (h), Total in Range. Sales columns: Org, Total Leads, Win Rate, Avg Deal
  Value ($), Open Pipeline. Each org name links to that org's org-scoped report page.

- **reporting.admin-pages** — Create `web/src/app/admin/reports/support/page.tsx` and
  `web/src/app/admin/reports/sales/page.tsx`. Both pages: fetch admin aggregate endpoint,
  render platform-wide KPI cards + charts (reusing support/sales chart components with
  platform-wide data), then render `OrgBreakdownTable` below. Add **Reports** nav item to
  the admin sidebar section. `ExportButton` points to admin export endpoint.

- **reporting.admin-tests** — Vitest: `OrgBreakdownTable` renders correct column count and
  sorts correctly. Playwright E2E: sign in as platform admin → `/admin/reports/support` →
  assert platform KPIs visible → assert org breakdown table has ≥1 row → click org name →
  assert navigation to org-scoped report. Coverage ≥85%.

---

## Dependency Map

```
Main Spec (fully merged)
  └─► Phase 1 (Backend Support)  ──┐
  └─► Phase 2 (Backend Sales)    ──┤
                                   ├─► Phase 3 (Backend Admin)
                                   └─► Phase 4 (Frontend Infrastructure)
                                            ├─► Phase 5 (Frontend Support Dashboard)  ──┐
                                            └─► Phase 6 (Frontend Sales Dashboard)    ──┤
                                                                                         └─► Phase 7 (Frontend Admin Dashboards)
```

**Parallel execution opportunities:**
- Phases 1 and 2 can run in parallel (both backend, independent query sets)
- Phases 5 and 6 can run in parallel (both frontend, independent chart sets)
- Phase 3 and Phase 4 can run in parallel once Phases 1 + 2 are merged

---

## Testing Strategy

### Per-Phase Requirements

Every phase MUST pass its tests before the next dependent phase begins.

### Test Levels

1. **Unit tests** — All repository functions, service logic, and React components tested in
   isolation with mocked dependencies. Testify (Go), Vitest + Testing Library (TS).
2. **Fuzzing** — ≥50 fuzzing tests per input point: date range params, assignee UUID, numeric
   metadata values (deal_value, score), stage name strings.
3. **Live API tests** — Start real compiled server, seed realistic data, exercise all report
   endpoints via actual HTTP. Verify response shape, RBAC enforcement (403 for wrong role),
   and CSV streaming (chunked response).
4. **E2E tests** — Playwright for each dashboard: filter interactions, CSV download, link-out
   navigation.
5. **Gate tests** — `task check` MUST pass fully (fmt + lint + typecheck + tests + coverage).

### Coverage Requirements

- ≥85% lines, functions, branches, statements on all new Go and TypeScript code
- CI MUST block merge on coverage drop below threshold

### Security Tests

- Verify org-scoped endpoints return 403 for viewer/commenter/contributor roles
- Verify org-scoped endpoints return 403 for members of a different org
- Verify admin endpoints return 403 for non-platform-admin users
- Verify `assignee` filter cannot be used to expose data from other orgs

---

## Deployment Notes

No new infrastructure required. The reporting package is added to the existing Go API binary.
`recharts` is a frontend-only dependency — no CDN or external service.

New Fly.io secrets: none.
New Vercel environment variables: none.

---

*Generated from `vbrief/specification-reporting.vbrief.json` — Do not edit directly.*
