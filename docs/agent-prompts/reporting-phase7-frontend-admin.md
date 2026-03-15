Implement Reporting Phase 7 (Frontend — Platform Admin Dashboards) of the DEFT Evolution CRM project (abraderAI/crm-project).

START BY reading AGENTS.md in the repo root — it contains all required framework reading and
coding standards you MUST follow before writing any code.

---

## CONTEXT

All prior reporting phases have been merged into `feat/reporting`:
- Phase 1: Backend support metrics (7 queries, org-scoped endpoints)
- Phase 2: Backend sales metrics (8 queries, org-scoped endpoints)
- Phase 3: Backend admin endpoints (platform-wide aggregates + per-org breakdown)
- Phase 4: Frontend shared components (MetricCard, DateRangePicker, AssigneeFilter, ExportButton,
  reports layout, sidebar nav, TypeScript types, API client)
- Phase 5: Frontend support dashboard (`/reports/support`, 4 chart components)
- Phase 6: Frontend sales dashboard (`/reports/sales`, 6 chart components)

**YOUR WORKING BRANCH**: check out `feat/reporting` (containing all prior phases), then
create `feat/reporting-phase7-frontend-admin` from it.

---

## YOUR TASK

Build the platform admin reporting dashboards by reusing all chart components from Phases 5
and 6, adding an `OrgBreakdownTable` component, and wiring up the admin API endpoints.

### 1. Org Breakdown Table Component (`web/src/components/reports/org-breakdown-table.tsx`)

```typescript
type OrgBreakdownVariant = "support" | "sales";

interface OrgBreakdownTableProps {
  variant: OrgBreakdownVariant;
  data: OrgSupportSummary[] | OrgSalesSummary[];
}
```

Use shadcn/ui `Table` (with `TableHeader`, `TableBody`, `TableRow`, `TableHead`,
`TableCell`). Add client-side sortable columns (click column header to toggle
asc/desc sort). Use a `ChevronUp`/`ChevronDown` icon in the active sort column header.

**Support variant columns** (using `OrgSupportSummary`):
| Column | Field | Format |
|---|---|---|
| Org | org_name (links to `/orgs/${org_slug}/reports/support`) | Link |
| Open Tickets | open_count | number |
| Overdue | overdue_count | number (red badge if > 0) |
| Avg Resolution | avg_resolution_hours | "X.X hrs" or "–" |
| Avg First Response | avg_first_response_hours | "X.X hrs" or "–" |
| Total (in range) | total_in_range | number |

**Sales variant columns** (using `OrgSalesSummary`):
| Column | Field | Format |
|---|---|---|
| Org | org_name (links to `/orgs/${org_slug}/reports/sales`) | Link |
| Total Leads | total_leads | number |
| Win Rate | win_rate | "X.X%" |
| Avg Deal Value | avg_deal_value | "$X,XXX" or "–" |
| Open Pipeline | open_pipeline_count | number |

Show a "No data" empty row when `data` is empty.

### 2. Admin Support Dashboard (`web/src/app/admin/reports/support/page.tsx`)

Client Component (`"use client"`). Uses `getAdminSupportMetrics()` and
`getAdminSupportExportUrl()` from the API client.

Layout (top to bottom):
1. **Page header**: "Platform Support Overview" + `ExportButton`
2. **Filter bar**: `DateRangePicker` only (no AssigneeFilter — platform-wide)
3. **KPI row** (3 `MetricCard` components, same as org support page but platform-wide):
   - Total Open Tickets: `data.status_breakdown.open ?? 0` (no link-out for admin view)
   - Platform Overdue: `data.overdue_count`
   - Avg Resolution: `data.avg_resolution_hours`
4. **Charts section**: reuse `StatusBreakdownChart`, `VolumeOverTimeChart`,
   `TicketsByPriorityChart` from `web/src/components/reports/support/`
   (import directly). No `onSegmentClick` or `onBarClick` needed for admin view.
5. **Per-org breakdown**: heading "By Organization" + `OrgBreakdownTable variant="support"`

### 3. Admin Sales Dashboard (`web/src/app/admin/reports/sales/page.tsx`)

Same pattern as admin support. Uses `getAdminSalesMetrics()`.

Layout:
1. **Page header**: "Platform Sales Overview" + `ExportButton`
2. **Filter bar**: `DateRangePicker` only
3. **KPI row**:
   - Total Leads (in range): `data.pipeline_funnel.reduce((s, x) => s + x.count, 0)`
   - Platform Win Rate: `"${(data.win_rate * 100).toFixed(1)}%"`
   - Avg Deal Value: `data.avg_deal_value`
4. **Charts section**: reuse `PipelineFunnelChart`, `LeadVelocityChart`,
   `ScoreDistributionChart` (no click handlers for admin view)
5. **Per-org breakdown**: `OrgBreakdownTable variant="sales"`

### 4. Admin Nav Update

Find the admin sidebar nav (check `web/src/components/layout/` or `web/src/app/admin/`).
Add a **Reports** section with two links:
- "Support Reports" → `/admin/reports/support`
- "Sales Reports" → `/admin/reports/sales`

Place it after the existing "Feature Flags" nav item (or at the end of the admin nav,
whichever is more consistent with existing ordering).

### 5. Tests

**Vitest component tests** (`web/src/components/reports/`):
- `org-breakdown-table.test.tsx`:
  - Renders support variant with correct column headers
  - Renders sales variant with correct column headers
  - Renders org name as link with correct href
  - Sort: clicking column header toggles asc/desc; data re-orders correctly
  - Shows "No data" row when data is empty
  - Shows red badge on overdue count > 0 (support variant)
  - Formats avg_hours as "X.X hrs" and null as "–"

**Vitest page tests**:
- `admin/reports/support/page.test.tsx`:
  - Renders KPI cards with mocked `getAdminSupportMetrics`
  - Renders `OrgBreakdownTable` with correct row count
  - Shows loading skeleton
- `admin/reports/sales/page.test.tsx`:
  - Same pattern for sales

**Playwright E2E** (`web/e2e/reporting-admin.spec.ts`):
- Mock `GET /v1/admin/reports/support` and `GET /v1/admin/reports/sales` via `page.route()`
- Sign in as platform admin
- Navigate to `/admin/reports/support`
  - Assert "Platform Support Overview" heading visible
  - Assert org breakdown table renders with ≥1 row
  - Assert org name is a clickable link
  - Click org link → assert navigation to org-scoped report
- Navigate to `/admin/reports/sales`
  - Assert "Platform Sales Overview" heading visible
  - Assert KPI cards (Total Leads, Win Rate, Avg Deal Value) visible

Coverage ≥85%.

---

## REQUIREMENTS CHECKLIST

- [ ] Read AGENTS.md and all deft files it references BEFORE writing any code
- [ ] Branch `feat/reporting-phase7-frontend-admin` created from `feat/reporting`
- [ ] `OrgBreakdownTable` component with both support/sales variants + sortable columns
- [ ] Admin support dashboard page at `web/src/app/admin/reports/support/page.tsx`
- [ ] Admin sales dashboard page at `web/src/app/admin/reports/sales/page.tsx`
- [ ] Reuses chart components from Phases 5 + 6 (no duplication)
- [ ] Admin nav updated with Reports links
- [ ] Loading skeleton + error state on both admin pages
- [ ] All Vitest tests pass ≥85% coverage
- [ ] Playwright E2E test passes
- [ ] `task check` MUST fully pass before creating the PR
- [ ] File names use hyphens (not underscores), no `any` types
- [ ] Conventional commits format
- [ ] PR targets `feat/reporting`
- [ ] PR title: `feat(reporting): add platform admin reporting dashboards (Phase 7)`
