# Reporting Module — PRD

*Generated from interview — 2026-03-15*

## Problem Statement

The platform currently has no analytics or reporting layer. Org admins and platform admins lack
visibility into support ticket health (volume, resolution time, SLA compliance, workload
distribution) and sales pipeline performance (funnel shape, win rates, stage velocity, deal
values). Without this, teams operate blind and cannot identify bottlenecks, forecast pipeline,
or manage team workload effectively.

---

## Goals

**Primary:**
- Give org admins (owner + admin role) a self-service reporting dashboard for support tickets
- Give org admins a self-service reporting dashboard for sales leads and pipeline health
- Give platform admins a cross-org aggregate view with per-org breakdown for both dashboards

**Secondary:**
- Enable CSV data export from both dashboards
- Provide click-through navigation from metric cards to the underlying filtered record lists

**Non-Goals (explicitly out of scope):**
- Real-time push / streaming updates to dashboard data
- PDF export or chart image download
- Custom report builder (drag-and-drop metric selection)
- IO channel metrics (email volume, voice call logs, chat sessions)
- Scheduled report delivery (email digest of dashboard data)
- Embedded analytics in individual thread or board views
- Per-org SLA configuration (threshold is fixed platform-wide)

---

## User Stories

**US-1 — Org admin, support:**
As an org admin, I want to see status breakdown, volume trends, resolution times, workload by
assignee, and SLA breach count for my support tickets so I can identify bottlenecks and manage
my support team.

**US-2 — Org admin, sales:**
As an org admin, I want to see the pipeline funnel, win/loss rates, deal velocity, deal value
averages, and stage conversion rates for my sales leads so I can forecast revenue and identify
where leads are dropping off.

**US-3 — Platform admin:**
As a platform admin, I want to see aggregate support and sales metrics across all orgs plus a
per-org breakdown table so I can monitor platform-wide health and identify struggling customers.

**US-4 — Filtering:**
As any reporting user, I want to filter dashboards by a custom date range and by assignee so I
can analyse specific time windows and individual team members.

**US-5 — Export:**
As any reporting user, I want to download the underlying data as a CSV file so I can perform
further analysis in spreadsheet tools.

**US-6 — Drilldown:**
As any reporting user, I want to click a metric card to navigate to the existing filtered thread
list so I can investigate individual records.

---

## Requirements

### Functional Requirements

**FR-1: Support Ticket Dashboard**
Display the following metrics for threads in `support`-type Spaces within the org and date range:

- FR-1.1 Status breakdown — count by `status` metadata field (open / in_progress / resolved / closed)
- FR-1.2 Volume over time — tickets created per calendar day, line/bar chart
- FR-1.3 Average resolution time — mean hours from `created_at` to `updated_at` for threads with status `resolved` or `closed`
- FR-1.4 Tickets by assignee — open ticket count grouped by `assigned_to` metadata field
- FR-1.5 Tickets by priority — count grouped by `priority` metadata field
- FR-1.6 Average first response time — mean hours from thread `created_at` to the first message from a user other than the thread author
- FR-1.7 Overdue count — threads with status `open` or `in_progress` and `created_at` older than 72 hours

**FR-2: Sales Lead Dashboard**
Display the following metrics for threads in `crm`-type Spaces within the org and date range:

- FR-2.1 Pipeline funnel — lead count at each pipeline stage (bar or funnel chart)
- FR-2.2 Lead velocity — new leads created per calendar day, line/bar chart
- FR-2.3 Win/loss rate — ratio of `closed_won` to `closed_lost` threads created within the range
- FR-2.4 Average deal value — mean of `deal_value` metadata field across leads in range
- FR-2.5 Leads by assignee — open lead count per `assigned_to` user
- FR-2.6 Lead score distribution — histogram of `score` metadata field in 5 equal-width buckets
- FR-2.7 Stage conversion rates — % of leads that advanced from each stage to the next (derived from audit log stage-change events)
- FR-2.8 Average time in stage — mean hours a lead spent at each stage before moving (derived from audit log)

**FR-3: Date Range Filter**
All dashboards MUST include a custom date range picker (from/to dates). Default range: last 30 days.

**FR-4: Assignee Filter**
All dashboards MUST include an assignee dropdown populated from org members. Default: all assignees.

**FR-5: Link-out Drilldown**
Clicking a metric card MUST navigate to the existing thread list view pre-filtered to the
relevant subset (e.g. clicking "14 open tickets" opens the support board filtered by `status=open`).

**FR-6: CSV Export**
Both dashboards MUST provide a "Download CSV" button. A dedicated server-side endpoint streams
the underlying row-level data as a CSV file (not the aggregated summary).

**FR-7: Org-Scoped Routes**
- `/reports/support` — support dashboard (org context from the user's active org)
- `/reports/sales` — sales dashboard

**FR-8: Platform Admin Routes**
- `/admin/reports/support` — cross-org support metrics + per-org breakdown table
- `/admin/reports/sales` — cross-org sales metrics + per-org breakdown table

**FR-9: Overdue Threshold**
Overdue is defined as: `status IN ('open', 'in_progress') AND created_at < NOW − 72h`.
Threshold is hardcoded; not configurable per org.

**FR-10: Chart Library**
All charts MUST be implemented with **Recharts**.

**FR-11: RBAC**
Org-scoped report pages and API endpoints MUST be restricted to `owner` and `admin` org roles.
Platform admin report pages restricted to platform admins. All others receive 403.

---

### Non-Functional Requirements

**NFR-1: Test Coverage** — ≥85% lines/functions/branches/statements on all new Go and TypeScript code.

**NFR-2: Performance** — All reporting API endpoints MUST respond in <2s for up to ~10,000 threads per org. SQLite JSON-extracted generated columns and existing indexes MUST be used on hot filter paths.

**NFR-3: Security** — Org-scoped endpoints MUST enforce org membership and role. No cross-org data leakage. Platform admin endpoints require platform admin check.

**NFR-4: Export Streaming** — CSV export endpoints MUST stream rows to the response writer rather than buffering the entire dataset in memory.

**NFR-5: Audit Log Resilience** — Stage conversion and time-in-stage metrics derived from the audit log MUST handle missing or incomplete audit records gracefully (return `null` or `0` rather than erroring).

---

## Success Metrics

- Both dashboards load in <2s for typical org data volumes
- All 7 support metrics and all 8 sales metrics are visible and accurate
- CSV export completes successfully and contains correct row-level data
- Link-out drilldowns navigate to correctly pre-filtered thread lists
- Platform admin per-org breakdown lists all orgs with correct aggregate values
- ≥85% test coverage on all new code

---

## Open Questions

None — all major decisions resolved during interview.

---

*Interview answers log available in `vbrief/specification-reporting.vbrief.json`*
