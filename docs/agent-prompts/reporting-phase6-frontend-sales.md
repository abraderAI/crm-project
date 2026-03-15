Implement Reporting Phase 6 (Frontend — Sales Dashboard) of the DEFT Evolution CRM project (abraderAI/crm-project).

START BY reading AGENTS.md in the repo root — it contains all required framework reading and
coding standards you MUST follow before writing any code.

---

## CONTEXT

Phase 4 has been merged into `feat/reporting`. It provides:
- `web/src/components/reports/` — `MetricCard`, `DateRangePicker`, `AssigneeFilter`,
  `ExportButton`
- `web/src/lib/reporting-types.ts` — all TypeScript interfaces
- `getSalesMetrics()`, `getSalesExportUrl()` API client functions
- `web/src/app/reports/layout.tsx` — reports shell with Support/Sales tab nav

**YOUR WORKING BRANCH**: check out `feat/reporting` (which contains Phase 4), then create
`feat/reporting-phase6-frontend-sales` from it.

**Run in parallel with Phase 5** — they touch entirely different files.

---

## YOUR TASK

Build the sales dashboard page and all its chart components.

### 1. Chart Components (`web/src/components/reports/sales/`)

All charts are Client Components (`"use client"`). Use Recharts.

**`pipeline-funnel-chart.tsx`**:
```typescript
interface PipelineFunnelChartProps {
  data: StageCount[];
  onBarClick?: (stage: string) => void;
}
```
- Recharts `BarChart` with one bar per stage.
- Display stages in pipeline order: `new_lead`, `contacted`, `qualified`, `proposal`,
  `negotiation`, `closed_won`, `closed_lost` (any unknown stages appended after).
- Colour: `closed_won`=green, `closed_lost`=red, others=blue gradient.
- Call `onBarClick(stage)` on bar click (link-out to CRM list filtered by stage).
- Show count labels on top of each bar.

**`lead-velocity-chart.tsx`**:
```typescript
interface LeadVelocityChartProps {
  data: DailyCount[];
}
```
- Recharts `AreaChart` — same pattern as `VolumeOverTimeChart` in Phase 5.
- X-axis: date formatted as "Mar 1". Y-axis: count.
- Show empty state when data is empty.

**`leads-by-assignee-chart.tsx`**:
```typescript
interface LeadsByAssigneeChartProps {
  data: AssigneeCount[];
  onBarClick?: (userId: string) => void;
}
```
- Recharts horizontal `BarChart` — same pattern as `TicketsByAssigneeChart`.
- Cap at top 10 assignees.

**`score-distribution-chart.tsx`**:
```typescript
interface ScoreDistributionChartProps {
  data: BucketCount[];  // ranges: "0-20", "20-40", "40-60", "60-80", "80-100"
}
```
- Recharts `BarChart` histogram.
- X-axis: range labels. Y-axis: count.
- Colour gradient: low scores = red, high scores = green (5-step gradient).
- Show "No scored leads" empty state when data is empty.

**`stage-conversion-chart.tsx`**:
```typescript
interface StageConversionChartProps {
  data: StageConversion[];
}
```
- Recharts `BarChart` showing conversion rate (%) per `from_stage`.
- X-axis: from_stage labels. Y-axis: percentage (0–100%).
- Only show the dominant `to_stage` per `from_stage` (highest rate transition).
- Show "No conversion data" empty state when data is empty array.

**`time-in-stage-chart.tsx`**:
```typescript
interface TimeInStageChartProps {
  data: StageAvgTime[];
}
```
- Recharts horizontal `BarChart`.
- X-axis: avg_hours. Y-axis: stage label.
- Skip stages where `avg_hours` is null.
- Format tooltip values as "X.X hrs".
- Show "No stage timing data" when all values are null or array is empty.

### 2. Sales Dashboard Page (`web/src/app/reports/sales/page.tsx`)

Client Component (`"use client"`). Same state/fetch pattern as the Phase 5 support page.

Layout (top to bottom):
1. **Page header**: title "Sales Pipeline" + `ExportButton`
2. **Filter bar**: `DateRangePicker` + `AssigneeFilter`
3. **KPI row** (3 `MetricCard` components):
   - Win Rate: `"${(data.win_rate * 100).toFixed(1)}%"` (no link-out)
   - Loss Rate: `"${(data.loss_rate * 100).toFixed(1)}%"` (no link-out)
   - Avg Deal Value: `data.avg_deal_value ? "$${data.avg_deal_value.toLocaleString()}" : "–"`
     (no link-out)
4. **Charts grid** (2 columns desktop, 1 mobile):
   - `PipelineFunnelChart` — on bar click, navigate to CRM thread list filtered by stage
   - `LeadVelocityChart`
   - `LeadsByAssigneeChart` — on bar click, navigate to CRM list filtered by assignee
   - `ScoreDistributionChart`
   - `StageConversionChart` (full width — spans 2 columns)
   - `TimeInStageChart` (full width)

Same URL search param sync, skeleton loading, error state, and ExportButton pattern as
Phase 5 (read that prompt for reference if needed).

`ExportButton` url = `getSalesExportUrl(orgId, { from, to, assignee })`.

### 3. Tests

**Vitest component tests** (`web/src/components/reports/sales/`):
- `pipeline-funnel-chart.test.tsx`:
  - Renders correct number of bars
  - Calls `onBarClick` with stage name
  - Renders closed_won bar in green
- `lead-velocity-chart.test.tsx`:
  - Renders with data
  - Shows empty state when data is empty
- `leads-by-assignee-chart.test.tsx`:
  - Renders bars; caps at 10
  - Calls `onBarClick` with user_id
- `score-distribution-chart.test.tsx`:
  - Renders all 5 buckets when data present
  - Shows empty state when data is empty
- `stage-conversion-chart.test.tsx`:
  - Renders bars for each from_stage
  - Shows "No conversion data" when data is empty array
- `time-in-stage-chart.test.tsx`:
  - Skips stages with null avg_hours
  - Shows empty state when all null

**Vitest page test**:
- `page.test.tsx`:
  - Renders all 6 chart sections with mocked `getSalesMetrics`
  - Shows 3 KPI metric cards
  - Shows skeleton while loading
  - Shows error alert on fetch failure

**Playwright E2E** (`web/e2e/reporting-sales.spec.ts`):
- Mock `GET /v1/orgs/*/reports/sales` via `page.route()`
- Navigate to `/reports/sales`
- Assert page title "Sales Pipeline" is visible
- Assert KPI cards (Win Rate, Loss Rate, Avg Deal Value) visible
- Assert all 6 chart section containers rendered
- Change assignee filter → assert fetch called with assignee param
- Click pipeline funnel bar → assert navigation to CRM list

Coverage ≥85%.

---

## REQUIREMENTS CHECKLIST

- [ ] Read AGENTS.md and all deft files it references BEFORE writing any code
- [ ] Branch `feat/reporting-phase6-frontend-sales` created from `feat/reporting`
- [ ] All 6 chart components created in `web/src/components/reports/sales/`
- [ ] Sales dashboard page at `web/src/app/reports/sales/page.tsx`
- [ ] Empty states implemented for all charts that can have no data
- [ ] Filter state synced to URL search params
- [ ] Skeleton + error states implemented
- [ ] Link-out drilldowns for pipeline funnel + assignee charts
- [ ] ExportButton wired with correct URL
- [ ] All Vitest tests pass ≥85% coverage
- [ ] Playwright E2E test passes
- [ ] `task check` MUST fully pass before creating the PR
- [ ] File names use hyphens (not underscores), no `any` types
- [ ] Conventional commits format
- [ ] PR targets `feat/reporting`
- [ ] PR title: `feat(reporting): add sales dashboard (Phase 6)`
