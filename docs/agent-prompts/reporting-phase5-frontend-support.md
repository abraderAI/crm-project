Implement Reporting Phase 5 (Frontend — Support Dashboard) of the DEFT Evolution CRM project (abraderAI/crm-project).

START BY reading AGENTS.md in the repo root — it contains all required framework reading and
coding standards you MUST follow before writing any code.

---

## CONTEXT

Phase 4 has been merged into `feat/reporting`. It provides:
- `web/src/components/reports/` — `MetricCard`, `DateRangePicker`, `AssigneeFilter`,
  `ExportButton`
- `web/src/lib/reporting-types.ts` — all TypeScript interfaces
- `getSupportMetrics()`, `getSupportExportUrl()` API client functions
- `web/src/app/reports/layout.tsx` — reports shell with Support/Sales tab nav
- **Reports** nav item in the sidebar

**YOUR WORKING BRANCH**: check out `feat/reporting` (which contains Phase 4), then create
`feat/reporting-phase5-frontend-support` from it.

**Run in parallel with Phase 6** — they touch entirely different files.

---

## YOUR TASK

Build the support dashboard page and all its chart components.

### 1. Chart Components (`web/src/components/reports/support/`)

All charts are Client Components (`"use client"`). Use Recharts. Follow the same
component structure as existing admin components.

**`status-breakdown-chart.tsx`**:
```typescript
interface StatusBreakdownChartProps {
  data: Record<string, number>;   // e.g. { open: 14, in_progress: 7 }
  onSegmentClick?: (status: string) => void;
}
```
- Recharts `PieChart` with `Pie` + `Cell` components.
- Colour map: open=blue, in_progress=yellow, resolved=green, closed=gray.
- Show legend below chart.
- Call `onSegmentClick(status)` when a segment is clicked (used for link-out drilldown).

**`volume-over-time-chart.tsx`**:
```typescript
interface VolumeOverTimeChartProps {
  data: DailyCount[];
}
```
- Recharts `AreaChart` with `XAxis` (date label), `YAxis` (count), `Tooltip`, `Area`.
- Format x-axis dates as "Mar 1" (short month + day).
- Show empty state message "No data for this period" when data is empty.

**`tickets-by-assignee-chart.tsx`**:
```typescript
interface TicketsByAssigneeChartProps {
  data: AssigneeCount[];
  onBarClick?: (userId: string) => void;
}
```
- Recharts horizontal `BarChart` (`layout="vertical"`), sorted by count descending.
- Y-axis shows assignee name, X-axis shows count.
- Call `onBarClick(user_id)` on bar click.
- Cap display at top 10 assignees.

**`tickets-by-priority-chart.tsx`**:
```typescript
interface TicketsByPriorityChartProps {
  data: Record<string, number>;
}
```
- Recharts `BarChart`.
- Fixed order: urgent, high, medium, low, none.
- Colour map: urgent=red, high=orange, medium=yellow, low=blue, none=gray.

### 2. Support Dashboard Page (`web/src/app/reports/support/page.tsx`)

This is a **Client Component** (`"use client"`) — it manages filter state and
re-fetches on filter change.

```typescript
"use client";
// State:
//   from: Date (default: 30 days ago)
//   to: Date (default: today)
//   assignee: string | null (default: null)
//   data: SupportMetrics | null
//   loading: boolean
//   error: string | null
```

Layout (top to bottom):
1. **Page header**: title "Support Tickets" + `ExportButton` (top right)
2. **Filter bar**: `DateRangePicker` + `AssigneeFilter` side by side
3. **KPI row** (3 `MetricCard` components in a grid):
   - Avg Resolution Time: `"${data.avg_resolution_hours?.toFixed(1) ?? '–'} hrs"`
     (no link-out — not a filterable list view)
   - Avg First Response: same pattern
   - Overdue Tickets: value = `data.overdue_count`; `href` = link to thread list
     filtered by `status=open` and a flag indicating overdue (use existing thread list
     URL with query params — check how the existing CRM/thread list accepts filters)
4. **Charts grid** (2 columns on desktop, 1 on mobile):
   - `StatusBreakdownChart` — on segment click, navigate to thread list filtered by
     that status (use `router.push` with appropriate query params)
   - `VolumeOverTimeChart`
   - `TicketsByAssigneeChart` — on bar click, navigate to thread list filtered by assignee
   - `TicketsByPriorityChart`

On `DateRangePicker` or `AssigneeFilter` change: re-fetch `getSupportMetrics()`.
Sync filter values to URL search params (`useSearchParams` + `router.push`) so the
dashboard is shareable/bookmarkable. On mount, read initial values from URL params
(defaulting to last 30 days / no assignee).

Show `Skeleton` placeholders in each chart area while `loading` is true.
Show a shadcn/ui `Alert` with error message if fetch fails.

`ExportButton` url = `getSupportExportUrl(orgId, { from, to, assignee })`.

### 3. Tests

**Vitest component tests** (`web/src/components/reports/support/`):
- `status-breakdown-chart.test.tsx`:
  - Renders correct number of pie segments
  - Calls `onSegmentClick` with correct status when clicked
  - Shows all statuses in legend
- `volume-over-time-chart.test.tsx`:
  - Renders chart with data
  - Shows "No data" empty state when data is empty array
- `tickets-by-assignee-chart.test.tsx`:
  - Renders bars for each assignee
  - Calls `onBarClick` with user_id when bar clicked
  - Caps at 10 bars
- `tickets-by-priority-chart.test.tsx`:
  - Renders in correct priority order
  - Uses correct colour for urgent

**Vitest page test** (`web/src/app/reports/support/`):
- `page.test.tsx`:
  - Renders all 4 chart sections with mocked `getSupportMetrics`
  - Shows skeleton while loading
  - Shows error alert on fetch failure
  - Re-fetches when date range changes

**Playwright E2E** (`web/e2e/reporting-support.spec.ts`):
- Mock `GET /v1/orgs/*/reports/support` via `page.route()`
- Navigate to `/reports/support`
- Assert page title "Support Tickets" is visible
- Assert all 4 chart sections are rendered
- Assert 3 KPI metric cards are visible
- Change date range → assert fetch is called with new params
- Click "Overdue Tickets" card → assert navigation to thread list

Coverage ≥85%.

---

## REQUIREMENTS CHECKLIST

- [ ] Read AGENTS.md and all deft files it references BEFORE writing any code
- [ ] Branch `feat/reporting-phase5-frontend-support` created from `feat/reporting`
- [ ] All 4 chart components created in `web/src/components/reports/support/`
- [ ] Support dashboard page at `web/src/app/reports/support/page.tsx`
- [ ] Filter state synced to URL search params
- [ ] Skeleton loading state shown during fetch
- [ ] Error state shown on fetch failure
- [ ] Link-out drilldowns work for status and assignee charts + overdue card
- [ ] ExportButton wired with correct URL
- [ ] All Vitest component + page tests pass ≥85% coverage
- [ ] Playwright E2E test passes
- [ ] `task check` MUST fully pass before creating the PR
- [ ] File names use hyphens (not underscores), no `any` types
- [ ] Conventional commits format
- [ ] PR targets `feat/reporting`
- [ ] PR title: `feat(reporting): add support dashboard (Phase 5)`
