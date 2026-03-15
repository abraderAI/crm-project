Implement Reporting Phase 4 (Frontend — Shared Reporting Infrastructure) of the DEFT Evolution CRM project (abraderAI/crm-project).

START BY reading AGENTS.md in the repo root — it contains all required framework reading and
coding standards you MUST follow before writing any code.

---

## CONTEXT

Phases 1 and 2 have been merged into `feat/reporting`. The backend exposes:
- `GET /v1/orgs/{org}/reports/support[/export]` — support metrics + CSV
- `GET /v1/orgs/{org}/reports/sales[/export]` — sales metrics + CSV

**Phase 3 is being implemented in parallel** and will add:
- `GET /v1/admin/reports/support[/export]` — platform admin support + CSV
- `GET /v1/admin/reports/sales[/export]` — platform admin sales + CSV

All accept `?from=YYYY-MM-DD&to=YYYY-MM-DD&assignee=<user_id>`.

Your task is to build the frontend infrastructure for **all** endpoints (org-scoped and
admin) so that Phases 5, 6, and 7 can build on it immediately after both Phase 3 and
Phase 4 are merged. The admin API client functions and types should be implemented even
though the Phase 3 backend endpoints are not yet merged — the type contracts are known.

**YOUR WORKING BRANCH**: check out `feat/reporting` (which contains Phases 1 + 2), then
create `feat/reporting-phase4-frontend-infra` from it.

---

## YOUR TASK

Build all shared reporting infrastructure in the Next.js frontend. No dashboard pages yet —
this phase creates the reusable components, API client, layout, and nav wiring that Phases
5, 6, and 7 will build on.

Match existing patterns in `web/src/components/admin/` and `web/src/app/admin/` exactly
(shadcn/ui, Tailwind, `"use client"` where needed, no `any` types).

### 1. Install Recharts

```bash
cd web && npm install recharts
```

Confirm TypeScript types are available (recharts ships its own types as of v2.x — no
`@types/recharts` needed). Add to `package.json` under `dependencies`.

### 2. TypeScript Types (`web/src/lib/types.ts` or `web/src/lib/reporting-types.ts`)

Add interfaces matching the backend response shapes exactly:

```typescript
export interface ReportParams {
  from?: string;   // "YYYY-MM-DD"
  to?: string;
  assignee?: string;
}

export interface DailyCount { date: string; count: number; }
export interface AssigneeCount { user_id: string; name: string; count: number; }
export interface StageCount { stage: string; count: number; }
export interface BucketCount { range: string; count: number; }
export interface StageConversion { from_stage: string; to_stage: string; rate: number; }
export interface StageAvgTime { stage: string; avg_hours: number | null; }

export interface SupportMetrics {
  status_breakdown: Record<string, number>;
  volume_over_time: DailyCount[];
  avg_resolution_hours: number | null;
  tickets_by_assignee: AssigneeCount[];
  tickets_by_priority: Record<string, number>;
  avg_first_response_hours: number | null;
  overdue_count: number;
}

export interface SalesMetrics {
  pipeline_funnel: StageCount[];
  lead_velocity: DailyCount[];
  win_rate: number;
  loss_rate: number;
  avg_deal_value: number | null;
  leads_by_assignee: AssigneeCount[];
  score_distribution: BucketCount[];
  stage_conversion_rates: StageConversion[];
  avg_time_in_stage: StageAvgTime[];
}

export interface OrgSupportSummary {
  org_id: string; org_name: string; org_slug: string;
  open_count: number; overdue_count: number;
  avg_resolution_hours: number | null;
  avg_first_response_hours: number | null;
  total_in_range: number;
}

export interface OrgSalesSummary {
  org_id: string; org_name: string; org_slug: string;
  total_leads: number; win_rate: number;
  avg_deal_value: number | null; open_pipeline_count: number;
}

export interface AdminSupportMetrics extends SupportMetrics {
  org_breakdown: OrgSupportSummary[];
}

export interface AdminSalesMetrics extends SalesMetrics {
  org_breakdown: OrgSalesSummary[];
}
```

### 3. API Client Functions

Add to the existing API client (wherever other typed fetch functions live — likely
`web/src/lib/api.ts`). Follow the exact same fetch/error pattern as existing functions
(check how billing or webhook API calls are made):

```typescript
// Org-scoped
export async function getSupportMetrics(orgId: string, params: ReportParams): Promise<SupportMetrics>
export async function getSalesMetrics(orgId: string, params: ReportParams): Promise<SalesMetrics>

// Admin
export async function getAdminSupportMetrics(params: ReportParams): Promise<AdminSupportMetrics>
export async function getAdminSalesMetrics(params: ReportParams): Promise<AdminSalesMetrics>

// Export URLs (used by ExportButton — returns URL string, not fetch)
export function getSupportExportUrl(orgId: string, params: ReportParams): string
export function getSalesExportUrl(orgId: string, params: ReportParams): string
export function getAdminSupportExportUrl(params: ReportParams): string
export function getAdminSalesExportUrl(params: ReportParams): string
```

Export URL functions build the query string from params and return the full API URL
(e.g. `${API_URL}/v1/orgs/${orgId}/reports/support/export?from=...`).

### 4. Shared Components (`web/src/components/reports/`)

Create this directory and these four components:

**`metric-card.tsx`**:
```typescript
interface MetricCardProps {
  label: string;
  value: string | number;
  subLabel?: string;
  href?: string;        // if set, wraps in Next.js <Link>
  loading?: boolean;    // shows skeleton
}
```
- Uses shadcn/ui `Card` + `CardContent`.
- When `href` is provided, the entire card is wrapped in `<Link href={href}>` with
  hover state.
- When `loading` is true, show a `Skeleton` in place of value.

**`date-range-picker.tsx`**:
```typescript
interface DateRangePickerProps {
  from: Date;
  to: Date;
  onChange: (range: { from: Date; to: Date }) => void;
}
```
- Uses shadcn/ui `Popover` + `Calendar` (date range mode).
- Trigger button shows formatted range: "Mar 1 – Mar 31, 2026".
- Default range passed in from parent (last 30 days).

**`assignee-filter.tsx`**:
```typescript
interface AssigneeFilterProps {
  orgId: string;
  value: string | null;           // null = "All"
  onChange: (userId: string | null) => void;
}
```
- Uses shadcn/ui `Select`.
- On mount, fetches org members from existing `GET /v1/orgs/{org}/members` endpoint.
- First option: "All assignees" (value = null).
- Remaining options: member display name + avatar (use existing member avatar pattern
  if available in other components).

**`export-button.tsx`**:
```typescript
interface ExportButtonProps {
  url: string;        // full export URL including query params
  filename: string;   // e.g. "support-report.csv"
}
```
- Renders a shadcn/ui `Button` with a download icon.
- On click: fetch the URL, create a `Blob`, trigger download via
  `URL.createObjectURL` + `<a>` click. Show loading spinner during fetch.
- Handle fetch errors with a toast notification.

### 5. Reports Layout & Navigation

**`web/src/app/reports/layout.tsx`**:
- Renders a page shell with a tab nav (Support | Sales) using shadcn/ui `Tabs` or
  plain styled `<Link>` tabs that highlight the active route.
- Wraps `{children}`.
- The active tab is determined by the current pathname.

**`web/src/app/reports/page.tsx`**:
- Simple redirect to `/reports/support` (use Next.js `redirect()`).

**Sidebar nav update**:
Find where the main `AppLayout` sidebar nav items are defined (likely
`web/src/components/layout/` — check existing code). Add a **Reports** nav item:
- Icon: a chart/bar-graph icon from lucide-react
- Label: "Reports"
- Href: `/reports`
- Visibility: only render this nav item when the current user has `owner` or `admin`
  org role. Check how other role-gated nav items are handled (e.g. the Admin nav item).

### 6. Tests (`web/src/components/reports/`)

- `metric-card.test.tsx`:
  - Renders value and label
  - Renders as `<Link>` when `href` is provided
  - Renders skeleton when `loading` is true
- `date-range-picker.test.tsx`:
  - Renders trigger button with formatted date range
  - Calls `onChange` with correct dates when range selected
- `assignee-filter.test.tsx`:
  - Renders "All assignees" option
  - Fetches and renders member list (mock fetch)
  - Calls `onChange` with user ID on selection
- `export-button.test.tsx`:
  - Renders button
  - Triggers download on click (mock fetch + createObjectURL)
  - Shows loading state during fetch

Coverage ≥85%.

---

## REQUIREMENTS CHECKLIST

- [ ] Read AGENTS.md and all deft files it references BEFORE writing any code
- [ ] Branch `feat/reporting-phase4-frontend-infra` created from `feat/reporting`
- [ ] `recharts` added to `web/package.json`
- [ ] All TypeScript interfaces added (no `any` types)
- [ ] All 4 API client functions + 4 export URL helpers implemented
- [ ] `MetricCard`, `DateRangePicker`, `AssigneeFilter`, `ExportButton` components created
- [ ] Reports layout + redirect page created
- [ ] **Reports** nav item added to sidebar (role-gated to owner/admin)
- [ ] All component tests pass with ≥85% coverage
- [ ] `task check` MUST fully pass before creating the PR
- [ ] File names use hyphens (not underscores)
- [ ] Conventional commits format
- [ ] PR targets `feat/reporting`
- [ ] PR title: `feat(reporting): add frontend shared reporting infrastructure (Phase 4)`
