# DEFT Evolution — Admin UI Completeness & User-Facing Gaps

*Generated from `vbrief/specification-admin-ui-completeness.vbrief.json` — 2026-03-16*

## Overview

Fills 13 UI gaps identified by cross-referencing all four spec modules against the current
Next.js frontend. **All backend endpoints already exist** — this module adds only frontend
pages and components to close the specified surface area.

This module MUST NOT add new Go code unless a backend bug is discovered during implementation.
It depends on: all previously merged phases (main spec, IO channels, reporting, RBAC user tiers).

---

## Requirements

### Functional

- Admin user detail page MUST expose ban/unban, GDPR purge, and impersonation per user
- Impersonation token MUST be stored in `sessionStorage` only — MUST NOT use `localStorage`
- Phone number purchase flow MUST require a confirmation modal labeled as a billable action
- Self-service upgrade page MUST call `POST /v1/me/upgrade` and invalidate the tier cache on success
- User settings page MUST expose personal API key management (create, list masked, revoke)
- Chat widget preview MUST reflect live form state without requiring a save
- All new pages under `/admin/*` MUST enforce platform_admin check via existing middleware
- ChatbotWidget MUST be present on all pages (authenticated and public) via AppLayoutWrapper

### Non-Functional

- ≥85% test coverage (lines, functions, branches, statements) on all new TypeScript code
- All new components MUST have Vitest unit tests
- Critical flows (ban user, impersonate, upgrade, purge) MUST have Playwright E2E tests
- `task check` MUST pass fully before PR

---

## Architecture

### New Routes

```
web/src/app/
├── admin/
│   ├── users/[user_id]/page.tsx       # User detail (Task 1)
│   ├── settings/page.tsx              # System settings (Task 2)
│   ├── security/page.tsx              # Security monitoring (Task 3)
│   ├── rbac-policy/page.tsx           # RBAC policy editor (Task 6)
│   ├── api-usage/page.tsx             # API usage stats (Task 7)
│   ├── llm-usage/page.tsx             # LLM usage log (Task 7)
│   ├── exports/page.tsx               # Async data exports (Task 8)
│   └── channels/
│       └── voice/
│           └── numbers/page.tsx       # Phone number management (Task 11)
├── upgrade/page.tsx                   # Self-service upgrade flow (Task 4)
└── settings/page.tsx                  # User profile & account settings (Task 5)
```

### New Components

```
web/src/components/
├── admin/
│   ├── user-detail.tsx                # Profile card, memberships table, actions (Task 1)
│   ├── system-settings.tsx            # Key-value form, PATCH submit (Task 2)
│   ├── security-log.tsx               # Shared paginated table (Task 3)
│   ├── platform-stats.tsx             # KPI row from /v1/admin/stats (Task 9)
│   ├── rbac-policy-editor.tsx         # Policy form + save (Task 6)
│   ├── rbac-policy-preview.tsx        # Dry-run role resolution (Task 6)
│   ├── export-manager.tsx             # Trigger + poll + download (Task 8)
│   ├── chat-widget-preview.tsx        # Live widget preview panel (Task 10)
│   └── phone-number-manager.tsx       # LiveKit number CRUD (Task 11)
├── settings/
│   └── api-keys.tsx                   # Personal API key management (Task 5)
└── home/
    └── upgrade-page.tsx               # Plan comparison + upgrade button (Task 4)
```

---

## Implementation Plan

### Phase 1: Admin UI Completeness & User-Facing Gaps
*Single phase — all tasks are independent and can be parallelised across agents.*
*Depends on: all previously merged phases (backend complete, RBAC tiers complete).*

---

#### Task 1: Admin User Detail Page (priority: high)

Route `/admin/users/[user_id]` — full per-user management surface.

- **Task 1.1** — Create `app/admin/users/[user_id]/page.tsx` as a Server Component. Fetch `GET /v1/admin/users/{user_id}`. Pass data to `UserDetail` client component.
- **Task 1.2** — Create `components/admin/user-detail.tsx`: profile card (name, email, joined date, last seen), cross-org memberships table (org name, role, joined date), ban status badge.
- **Task 1.3** — Ban/unban toggle with optional reason textarea. Confirmation dialog before submit. Calls `POST /v1/admin/users/{user_id}/ban` or `.../unban`. Reflects new status immediately.
- **Task 1.4** — GDPR purge button. Requires typing the user's email to confirm. Calls `DELETE /v1/admin/users/{user_id}/purge`. Redirects to `/admin/users` on success.
- **Task 1.5** — Impersonate button. Calls `POST /v1/admin/users/{user_id}/impersonate`. Stores token in `sessionStorage['impersonation_token']`. Shows countdown timer (max 2h). Clear button removes token.
- **Task 1.6** — Add `href` to `/admin/users/[user_id]` on each row in the existing user list page.
- **Task 1.7** — Tests: Vitest unit tests for `user-detail` component (membership table renders, ban toggle, purge dialog). Playwright E2E: navigate list → detail → ban → verify badge updates. Coverage ≥85%.

Acceptance: `/admin/users/[user_id]` loads user detail; ban/unban/purge/impersonate all call correct endpoints; impersonation token never in `localStorage`.

---

#### Task 2: Admin System Settings Page (priority: high)

Route `/admin/settings` — platform-wide configuration editor.

- **Task 2.1** — Create `app/admin/settings/page.tsx`. Fetch `GET /v1/admin/settings` server-side. Render `SystemSettings` client component.
- **Task 2.2** — Create `components/admin/system-settings.tsx`. Render settings as an editable key-value list with type-inferred inputs (string → text, number → number, boolean → toggle, object → JSON textarea). Save button calls `PATCH /v1/admin/settings`. Show success toast on save.
- **Task 2.3** — Add **Settings** link to admin sidebar under the Configuration section.
- **Task 2.4** — Tests: Vitest unit tests — form renders, save submits correct body. Coverage ≥85%.

Acceptance: `/admin/settings` displays current settings; edits persist via PATCH.

---

#### Task 3: Admin Security Monitoring Page (priority: high)

Route `/admin/security` — recent logins and failed authentication events.

- **Task 3.1** — Create `app/admin/security/page.tsx` with shadcn/ui `Tabs`: **Recent Logins** | **Failed Auths**.
- **Task 3.2** — Create `components/admin/security-log.tsx`. Reusable paginated table: columns: user ID (links to `/admin/users/[user_id]`), IP address, user agent, timestamp. Props: `endpoint` string.
- **Task 3.3** — Recent Logins tab: uses `GET /v1/admin/security/recent-logins`. Failed Auths tab: uses `GET /v1/admin/security/failed-auths`.
- **Task 3.4** — Add **Security** link to admin sidebar.
- **Task 3.5** — Tests: Vitest unit tests — table renders rows, pagination works. Coverage ≥85%.

Acceptance: Both tabs load data; user ID links navigate to user detail.

---

#### Task 4: Self-Service Upgrade / Conversion Page (priority: high)

Route `/upgrade` — Tier 2 → Tier 3 conversion flow.

- **Task 4.1** — Create `app/upgrade/page.tsx`. Requires authentication (middleware redirect if not signed in). Renders `UpgradePage` client component.
- **Task 4.2** — Create `components/home/upgrade-page.tsx`. Two plan comparison cards: **Developer** (Tier 2, current) vs **Customer** (Tier 3, target). Feature checklist per tier. **Activate Trial** button (stub — billing deferred). Button calls `POST /v1/me/upgrade`. On success: call `mutate()` on `useTier()` hook to invalidate cache, then `router.push('/')`.
- **Task 4.3** — Update `UpgradeCTAWidget` to set `href="/upgrade"` instead of `#`.
- **Task 4.4** — Tests: Vitest — plan cards render, button calls endpoint, redirect occurs on success. Coverage ≥85%.

Acceptance: Clicking Upgrade in `UpgradeCTAWidget` navigates to `/upgrade`; confirming calls API and redirects to home with updated tier.

---

#### Task 5: User Profile / Account Settings Page (priority: high)

Route `/settings` — authenticated user's profile, API keys, and tier status.

- **Task 5.1** — Create `app/settings/page.tsx` with four sections: Profile, Notifications, API Keys, Current Tier.
- **Task 5.2** — Profile section: embed Clerk `<UserProfile>` component (name, email, password, connected accounts).
- **Task 5.3** — Notifications section: link card to `/notifications/preferences`.
- **Task 5.4** — Create `components/settings/api-keys.tsx`. Lists existing keys (name, prefix `deft_live_...`, created date, last used). Create Key button: opens modal with name input → calls `POST /v1/auth/api-keys` → shows full key in modal once only with copy button → closes. Revoke button: confirmation dialog → `DELETE /v1/auth/api-keys/{key_id}`.
- **Task 5.5** — Current Tier section: renders tier badge (number + label) using `useTier()`.
- **Task 5.6** — Add **Settings** link to topbar user menu (custom `<UserButton>` menu item pointing to `/settings`).
- **Task 5.7** — Tests: Vitest — `api-keys` renders list, create modal shows key once, revoke removes row. Coverage ≥85%.

Acceptance: `/settings` accessible from topbar; API key create/revoke flows work; key shown exactly once after creation.

---

#### Task 6: Admin RBAC Policy Management UI (priority: medium)

Route `/admin/rbac-policy` — live policy editor with dry-run role preview.

- **Task 6.1** — Create `app/admin/rbac-policy/page.tsx`. Fetch `GET /v1/admin/rbac-policy`. Render editor and preview components.
- **Task 6.2** — Create `components/admin/rbac-policy-editor.tsx`. Displays policy as structured form fields (resolution strategy dropdown, role hierarchy ordering, defaults). Save calls `PATCH /v1/admin/rbac-policy`.
- **Task 6.3** — Create `components/admin/rbac-policy-preview.tsx`. Form: user ID, entity type (org/space/board), entity ID. Submit calls `POST /v1/admin/rbac-policy/preview`. Displays resolved role + resolution path (which level it came from).
- **Task 6.4** — Add **RBAC Policy** link to admin sidebar.
- **Task 6.5** — Tests: Vitest — editor renders strategy dropdown, preview form calls correct endpoint and displays result. Coverage ≥85%.

Acceptance: Policy editable and saveable; dry-run shows resolved role for any user+entity combination without persisting.

---

#### Task 7: Admin API Usage & LLM Usage Stats Pages (priority: medium)

Routes `/admin/api-usage` and `/admin/llm-usage` — monitoring dashboards.

- **Task 7.1** — Create `app/admin/api-usage/page.tsx`. Time-window toggle (24h / 7d / 30d). Table: endpoint path, method, request count, sorted by count descending. Fetches `GET /v1/admin/api-usage?window={window}`.
- **Task 7.2** — Create `app/admin/llm-usage/page.tsx`. Table: thread ID (links to thread), model, input tokens, output tokens, latency (ms), timestamp. Fetches `GET /v1/admin/llm-usage`.
- **Task 7.3** — Add **API Usage** and **LLM Usage** links to admin sidebar under a **Monitoring** grouping.
- **Task 7.4** — Tests: Vitest — both tables render with fixture data, time-window toggle on api-usage changes fetch param. Coverage ≥85%.

Acceptance: Both pages load data; API usage table refreshes on window toggle.

---

#### Task 8: Admin Async Data Exports UI (priority: medium)

Route `/admin/exports` — trigger and download platform data exports.

- **Task 8.1** — Create `app/admin/exports/page.tsx`. Renders `ExportManager` client component.
- **Task 8.2** — Create `components/admin/export-manager.tsx`. Trigger form: type selector (users / orgs / audit), format selector (csv / json), **Create Export** button → `POST /v1/admin/exports`. Export history table: ID, type, format, status badge, created at, download button (active when `status === "ready"`). Status polling: `setInterval` every 3s for any export with `status === "pending"`. Auto-stop polling when all complete.
- **Task 8.3** — Add **Exports** link to admin sidebar.
- **Task 8.4** — Tests: Vitest — form submits, polling updates status badge, download button appears on ready. Coverage ≥85%.

Acceptance: Export triggered, status transitions from pending → ready shown in real time; download link works.

---

#### Task 9: Admin Platform Stats Enhancement (priority: medium)

Enhance existing `/admin` overview page with live platform KPIs.

- **Task 9.1** — Create `components/admin/platform-stats.tsx`. Fetches `GET /v1/admin/stats`. Renders 5 `MetricCard`-style tiles: Total Orgs, Total Users, Total Threads, DB Size, API Uptime. Uses same `MetricCard` component as reporting module.
- **Task 9.2** — Add `<PlatformStats />` above the existing nav card grid in `app/admin/page.tsx`.
- **Task 9.3** — Tests: Vitest — `platform-stats` renders all 5 metric values from fixture data. Coverage ≥85%.

Acceptance: Admin dashboard shows live platform KPIs on load.

---

#### Task 10: Chat Widget Preview & Embed Code Copy (priority: medium)

Enhance `/admin/channels/chat` config page with live preview and embed code.

- **Task 10.1** — Create `components/admin/chat-widget-preview.tsx`. Accepts `theme`, `greeting`, `logoUrl` as props. Renders a visual simulation of the chat widget (floating button + expanded panel) using the configured values. Updates live as form state changes (no save required).
- **Task 10.2** — Add embed code snippet to `channel-config-form.tsx` for the `chat` channel type. Snippet: `<script src="..." data-org-key="{embed_key}"></script>`. Copy button uses `navigator.clipboard.writeText`. Shows "Copied!" confirmation for 2s.
- **Task 10.3** — Integrate preview panel and embed code section into `/admin/channels/chat` page layout as a right-side panel alongside the config form.
- **Task 10.4** — Tests: Vitest — preview renders with theme props, copy button calls clipboard API with correct code. Coverage ≥85%.

Acceptance: Chat channel config page shows live widget preview + copyable embed code.

---

#### Task 11: Phone Number Management UI (priority: medium)

Route `/admin/channels/voice/numbers` — LiveKit phone number provisioning.

- **Task 11.1** — Create `app/admin/channels/voice/numbers/page.tsx`. Renders `PhoneNumberManager` component. Requires org context.
- **Task 11.2** — Create `components/admin/phone-number-manager.tsx`. Sections: **Owned Numbers** table (number, status, dispatch rule, purchased date); **Search Available** form (area code input, country selector) → calls `POST /v1/orgs/{org}/channels/voice/numbers/search` → results table with **Purchase** button per row. Purchase button opens confirmation modal: "This will add a phone number to your account and may incur charges." Confirm calls `POST /v1/orgs/{org}/channels/voice/numbers/purchase`.
- **Task 11.3** — Add **Manage Numbers** link from `/admin/channels/voice` config page.
- **Task 11.4** — Tests: Vitest — owned numbers table renders, search triggers API call, confirmation modal appears before purchase. Coverage ≥85%.

Acceptance: Phone numbers listed; search and purchase flow requires confirmation before calling API.

---

#### Task 12: In-App Chatbot Widget Verification & Fix (priority: medium)

Verify and complete the `ChatbotWidget` component specified in RBAC tiers Phase 6.

- **Task 12.1** — Audit `web/src/components/` for `chatbot-widget.tsx`. If missing or incomplete, implement: floating chat bubble → expandable panel, Clerk anonymous token support, POST `/v1/chat/message` with streaming response via WebSocket, conversation history in component state.
- **Task 12.2** — Add `ChatbotWidget` (via `dynamic(() => import(...), { ssr: false })`) to `AppLayoutWrapper` so it mounts on all pages without SSR.
- **Task 12.3** — Tests: Vitest — ChatbotWidget renders bubble, opens panel on click, sends message to correct endpoint. Coverage ≥85%.

Acceptance: Chat bubble visible on all pages; clicking opens panel; messages sent to `/v1/chat/message`.

---

#### Task 13: Minor Fixes — Naming Convention & Redirect (priority: low)

*Can proceed in parallel with all other tasks.*

- **Task 13.1** — Rename `components/home/tier4-home-screen.tsx` → `tier-4-home.tsx` and `tier6-home-screen.tsx` → `tier-6-home.tsx`. Rename corresponding test files. Update all imports (`grep -r "tier4-home-screen\|tier6-home-screen"` to find all usages).
- **Task 13.2** — Open `app/reports/page.tsx`. Verify it contains `redirect('/reports/support')`. If not, replace page body with `import { redirect } from 'next/navigation'; export default function Page() { redirect('/reports/support'); }`.
- **Task 13.3** — Run `task check` and confirm all existing tests still pass after rename. Coverage ≥85%.

Acceptance: No files named with underscores or mismatched tier naming. `/reports` redirects to `/reports/support`.

---

## Parallelism

All 13 tasks are **fully independent** — they touch different routes and components with no shared state changes. They can all run in parallel across up to 13 agents.

```
Task 1  (Admin User Detail)        ──┐
Task 2  (Admin Settings)           ──┤
Task 3  (Admin Security)           ──┤
Task 4  (Upgrade Page)             ──┤
Task 5  (User Settings)            ──┤
Task 6  (RBAC Policy UI)           ──┤──► task check passes → PR
Task 7  (API/LLM Usage)            ──┤
Task 8  (Exports UI)               ──┤
Task 9  (Platform Stats)           ──┤
Task 10 (Widget Preview)           ──┤
Task 11 (Phone Numbers)            ──┤
Task 12 (Chatbot Widget)           ──┤
Task 13 (Naming/Redirect Fixes)    ──┘
```

---

## Testing Strategy

### Per-Task Requirements

Every task MUST implement and pass tests before the PR is created.

### Test Levels

1. **Unit tests** — All new components tested with Vitest + Testing Library. Mock all API calls.
2. **E2E tests** — Playwright for: user detail (ban flow), upgrade flow (button → redirect), user settings (API key create/revoke).
3. **Gate tests** — `task check` MUST pass fully (fmt + lint + typecheck + tests + coverage ≥85%).

### Security Tests

- Verify `/admin/*` pages return 403 for non-platform-admin users
- Verify impersonation token is in `sessionStorage` not `localStorage`
- Verify phone number purchase confirmation modal cannot be bypassed

---

## Dependency Map

```
All previously merged phases (main spec, IO channels, reporting, RBAC tiers)
  └─► Phase 1: Admin UI Completeness & User-Facing Gaps (this spec)
       └─► All tasks run in parallel — no inter-task dependencies
```

---

*Generated from `vbrief/specification-admin-ui-completeness.vbrief.json` — Do not edit directly.*
