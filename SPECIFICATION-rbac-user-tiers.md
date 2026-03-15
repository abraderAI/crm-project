# DEFT Evolution — User Tiers & Role-Based Access SPECIFICATION

## Overview

Implements 6 user tiers with tier-specific home screens, global content spaces, a
rule-based access control enforcement layer, an AI chatbot with live agent handoff,
and home screen widget customization. See `PRD-rbac-user-tiers.md` for requirements.

## Architecture

### Tier Resolution (Server-Side)

A `TierResolver` service runs on every authenticated (and anonymous) request. It
checks in order:

1. Is `platform_admin` flag set on the user record? → **Tier 6**
2. Is the user a member of the `deft` org? → **Tier 4** (sub-type determined by space membership)
3. Is the user an `admin` or `owner` of any paying customer org? → **Tier 5**
4. Is the user a member of any paying customer org? → **Tier 3** (sub-type: owner if `owner` role)
5. Is the user authenticated (has Clerk account)? → **Tier 2**
6. Anonymous Clerk token only → **Tier 1**

Result is cached in the session/JWT and re-validated on sensitive operations.

### Global Spaces

Four system-seeded spaces owned by no customer org:

| Space slug | Purpose |
|---|---|
| `global-docs` | Public documentation, wiki, tutorials |
| `global-forum` | Public community forum |
| `global-support` | Support tickets (org-filtered for customers) |
| `global-leads` | Lead records (DEFT-internal) |

Threads gain two new fields: `thread_type` (enum) and `org_id` (nullable FK for
support ticket org scoping).

### Home Screen Widget System

A `HomeLayout` component renders a grid of `Widget` components. Each widget has a
stable `widget_id`. User preferences (`UserHomePreferences`) are stored server-side
and loaded at SSR time. Default layouts are defined per tier in a static config.
Authenticated users can toggle visibility and reorder widgets; preferences are saved
via `PUT /api/me/home-preferences`.

### Chatbot

A `ChatbotWidget` client component loads asynchronously. It holds a Clerk anonymous
session token for unauthenticated users. It calls a new `/api/chat/message` endpoint
that proxies to an AI provider (stubbed initially) with RAG context from `global-docs`.
On escalation, it opens an IO channel chat session with a DEFT support agent.
On user registration, a Clerk webhook triggers anonymous session promotion, linking
the prior `anon_session_id` to the new `user_id` in the leads table.

---

## Implementation Plan

### Phase 1: Data Model & Backend Foundation
**Dependencies**: none
**Agents**: 1

#### Subphase 1.1: Schema Extensions

- **Task 1.1.1**: Add `thread_type` column to `threads` table
  - Enum: `wiki | documentation | tutorial | forum | support | lead`
  - Default: `forum`
  - Acceptance: migration runs cleanly; existing threads default to `forum`

- **Task 1.1.2**: Add `visibility` column to `threads` table
  - Enum: `public | org-only | deft-only`
  - Default: `org-only`
  - Acceptance: existing threads default to `org-only`; `global-docs` and `global-forum` threads default to `public`

- **Task 1.1.3**: Add `org_id` column (nullable FK) to `threads` table
  - Used to scope `global-support` tickets to a customer org
  - Acceptance: nullable; no constraint breakage on existing threads

- **Task 1.1.4**: Add `user_home_preferences` table
  - Columns: `user_id` (PK), `tier`, `layout` (JSON: ordered list of `{widget_id, visible}`)
  - Acceptance: table created; can store and retrieve per-user layout JSON

- **Task 1.1.5**: Add `anon_session_id` column to `leads` table
  - Nullable string; populated when a lead is created from an anonymous chatbot session
  - Acceptance: column exists; index on `anon_session_id`

#### Subphase 1.2: Global Space Seeding

- **Task 1.2.1**: Seed the four global spaces in the database
  - Slugs: `global-docs`, `global-forum`, `global-support`, `global-leads`
  - Owned by the system (no `org_id`); type field set appropriately
  - Acceptance: spaces exist in DB after migration; idempotent seed

- **Task 1.2.2**: Seed the `deft` org and its three department spaces
  - Org slug: `deft`; spaces: `deft-sales`, `deft-support`, `deft-finance`
  - Acceptance: org and spaces exist after seed; idempotent

#### Subphase 1.3: Tier Resolution Service (Backend)

- **Task 1.3.1**: Implement `ResolveTier(userID string) (Tier, error)` in Go
  - Checks platform_admin flag → deft org membership → customer org admin → customer org member → registered → anonymous
  - Returns tier enum and optional sub-type (e.g., DEFT department, org owner flag)
  - Acceptance: unit tests cover all 6 paths; edge cases (multi-org, deft+customer) resolve correctly

- **Task 1.3.2**: Add `GET /api/me/tier` endpoint
  - Returns `{tier, sub_type, org_id?, deft_department?}`
  - Auth: requires valid Clerk token (anonymous token returns tier 1)
  - Acceptance: integration test covering each tier; anonymous token returns `tier: 1`

- **Task 1.3.3**: Add `GET /api/me/home-preferences` and `PUT /api/me/home-preferences` endpoints
  - GET returns stored layout or null (frontend uses default)
  - PUT validates and stores layout JSON
  - Acceptance: round-trip test; invalid widget IDs rejected with 400

---

### Phase 2: Frontend Infrastructure
**Dependencies**: Phase 1
**Agents**: 1

#### Subphase 2.1: Tier Context & Route Protection

- **Task 2.1.1**: Create `useTier()` React hook
  - Calls `GET /api/me/tier`; caches result in context
  - Returns `{tier, subType, isLoading}`
  - Acceptance: unit tested; anonymous (no Clerk session) returns tier 1

- **Task 2.1.2**: Update Next.js `middleware.ts` for route protection
  - `/admin/*`: redirect non-`platform_admin` users to home; log attempt
  - `global-leads` API routes: reject non-DEFT-org users with 403
  - Public routes (`/docs`, `/forum`): explicitly excluded from auth redirect
  - Acceptance: e2e tests confirm redirect behavior for each protected route

- **Task 2.1.3**: Create public layout for unauthenticated routes
  - New `(public)` route group in Next.js
  - Routes: `/docs/[...slug]`, `/forum/[...slug]`
  - No Clerk `<SignedIn>` wrapper; renders chatbot widget
  - Acceptance: routes render without auth; chatbot widget present

#### Subphase 2.2: Widget Framework

- **Task 2.2.1**: Implement `Widget` base component and `HomeLayout` grid
  - `Widget` accepts `{id, title, children, visible}`
  - `HomeLayout` renders a responsive CSS grid of widgets
  - Acceptance: renders correctly with 1–9 widgets; responsive at sm/md/lg

- **Task 2.2.2**: Implement `HomeLayoutEditor` — show/hide and reorder controls
  - Toggle visibility per widget; drag-to-reorder (or up/down buttons)
  - Calls `PUT /api/me/home-preferences` on save
  - Acceptance: toggling visibility persists; reorder persists across page reload

- **Task 2.2.3**: Implement `useHomeLayout(tier)` hook
  - Fetches `GET /api/me/home-preferences`; falls back to static default layout for tier
  - Acceptance: fresh user gets default layout; returning user gets saved layout

---

### Phase 3: Tier 1 & Tier 2 Home Screens
**Dependencies**: Phase 2
**Agents**: 2 (Tier 1 and Tier 2 can be built in parallel)

#### Subphase 3.1: Tier 1 — Anonymous Home Screen

- **Task 3.1.1**: Build `DocsHighlightsWidget` — recent `global-docs` threads (public)
  - Acceptance: renders without auth; shows up to 5 recent wiki/tutorial threads

- **Task 3.1.2**: Build `ForumHighlightsWidget` — recent `global-forum` threads (public)
  - Acceptance: renders without auth; links to forum thread detail

- **Task 3.1.3**: Build `GetStartedWidget` — CTA card with sign-up link and feature summary
  - Acceptance: visible to Tier 1 only; hidden once user is Tier 2+

- **Task 3.1.4**: Compose Tier 1 home screen at `/` with default widget layout
  - Acceptance: anonymous visitor sees docs highlights, forum highlights, get started CTA, chatbot widget

#### Subphase 3.2: Tier 2 — Registered Developer Home Screen

- **Task 3.2.1**: Build `MyProfileWidget` — name, email, account status, edit link
  - Acceptance: shows authenticated user's Clerk profile data

- **Task 3.2.2**: Build `MyForumActivityWidget` — user's recent posts and replies in `global-forum`
  - Acceptance: shows threads authored or commented on by user; empty state handled

- **Task 3.2.3**: Build `MySupportTicketsWidget` — user's open tickets in `global-support`
  - Acceptance: filtered to current user; shows status badges; empty state handled

- **Task 3.2.4**: Build `UpgradeCTAWidget` — "Upgrade to Pro" card with conversion path
  - Acceptance: visible to Tier 2 only; hidden for Tier 3+; links to upgrade flow

- **Task 3.2.5**: Compose Tier 2 home screen with default widget layout
  - Acceptance: registered user sees profile, forum activity, support tickets, upgrade CTA, chatbot

- **Task 3.2.6**: Implement forum post creation (`global-forum` thread create)
  - Tier 2+ only; uses existing thread create flow with space set to `global-forum`
  - Acceptance: Tier 1 cannot POST; Tier 2 can; thread appears publicly

- **Task 3.2.7**: Implement support ticket creation (`global-support` thread create)
  - Tier 2+ only; `org_id` set to user's org (if any) or null
  - Acceptance: created ticket visible to user; visible to DEFT support; not visible to other orgs

---

### Phase 4: Tier 3 & Tier 5 Home Screens
**Dependencies**: Phase 3
**Agents**: 2 (Tier 3 and Tier 5 can be built in parallel)

#### Subphase 4.1: Tier 3 — Paying Customer Home Screen

- **Task 4.1.1**: Build `OrgOverviewWidget` — org name, member count, plan status
  - Acceptance: shows current org data; org owner sees billing status stub

- **Task 4.1.2**: Build `OrgSupportTicketsWidget` — support tickets filtered to `org_id`
  - Acceptance: only shows tickets where `org_id` matches user's org; no cross-org leakage

- **Task 4.1.3**: Build `BillingStatusWidget` (stub) — current plan, renewal date, upgrade link
  - Acceptance: visible to org owner only; shows placeholder data; links to billing page stub

- **Task 4.1.4**: Build `OrgSupportDashboardWidget` — ticket volume, open/closed counts (org owner only)
  - Acceptance: visible to org owner; correct counts from `global-support` filtered by org

- **Task 4.1.5**: Compose Tier 3 home screen (member vs. owner variants)
  - Member: org overview, org support tickets, forum activity, chatbot
  - Owner: org support dashboard, billing status, org member summary, chatbot
  - Acceptance: correct variant shown based on `owner` role flag

#### Subphase 4.2: Tier 5 — Customer Org Admin Home Screen

- **Task 4.2.1**: Build `OrgAccessControlWidget` — member list with role badges and edit controls
  - Calls existing membership API; allows role change and member removal
  - Acceptance: admin can change roles; changes persist; cannot elevate above own role

- **Task 4.2.2**: Build `OrgRBACEditorWidget` — space-level role overrides per member
  - Acceptance: can set space-level role overrides; existing RBAC resolution respected

- **Task 4.2.3**: Compose Tier 5 home screen
  - Org access controls, member list + roles, org support dashboard, billing placeholder
  - Acceptance: org admin sees RBAC controls; regular org member cannot access this view

---

### Phase 5: Tier 4 & Tier 6 Home Screens
**Dependencies**: Phase 2 (can run in parallel with Phase 4)
**Agents**: 2

#### Subphase 5.1: Tier 4 — DEFT Employee Home Screens

- **Task 5.1.1**: Build `LeadPipelineWidget` — leads from `global-leads`, grouped by status
  - Visible to DEFT sales space members only
  - Acceptance: shows lead count by status; links to lead detail thread

- **Task 5.1.2**: Build `RecentLeadsWidget` — last 10 leads with source and status
  - Acceptance: newest first; shows anon vs. registered vs. converted status

- **Task 5.1.3**: Build `ConversionMetricsWidget` — Tier 1→2→3 funnel counts
  - Acceptance: shows total anon sessions, registrations, and conversions (stub counts ok)

- **Task 5.1.4**: Build `TicketQueueWidget` — all open tickets in `global-support` (DEFT support)
  - Visible to DEFT support space members only
  - Acceptance: shows all org tickets; sortable by date/status; links to ticket

- **Task 5.1.5**: Build `TicketStatsWidget` — open/pending/resolved counts, avg response time stub
  - Acceptance: correct counts from `global-support`

- **Task 5.1.6**: Build `BillingOverviewWidget` — paying org count, MRR stub, recent payments stub
  - Visible to DEFT finance space members only
  - Acceptance: shows paying org count; revenue fields show placeholder

- **Task 5.1.7**: Compose Tier 4 home screen (department-specific default layouts)
  - Sales: lead pipeline, recent leads, conversion metrics
  - Support: ticket queue, ticket stats, recent escalations
  - Finance: billing overview, revenue summary
  - Acceptance: DEFT sales member sees sales widgets by default; support sees support widgets; etc.

#### Subphase 5.2: Tier 6 — Platform Admin Home Screen

- **Task 5.2.1**: Build `SystemHealthWidget` — API uptime, DB status, channel health summary
  - Pulls from existing `/admin` overview data
  - Acceptance: shows live health status

- **Task 5.2.2**: Build `RecentAuditLogWidget` — last 10 audit events with actor and action
  - Acceptance: links to full audit log; shows most recent events

- **Task 5.2.3**: Compose Tier 6 home screen
  - System health, recent audit log, user management quick link, feature flags quick link, all T4 widgets
  - Acceptance: platform admin sees full operational view; no widget is hidden by default

---

### Phase 6: Chatbot
**Dependencies**: Phase 3 (anonymous session model), Phase 1 (global-docs for RAG)
**Agents**: 1

- **Task 6.1**: Build `ChatbotWidget` client component
  - Loads asynchronously (dynamic import, no SSR block)
  - Issues Clerk anonymous token for unauthenticated users
  - Renders a floating chat bubble; opens into a panel
  - Acceptance: renders on all pages; does not delay LCP; anonymous users get a session token

- **Task 6.2**: Implement `POST /api/chat/message` endpoint (stubbed AI)
  - Accepts `{session_id, message, anon_token?}`
  - Returns a canned response initially; RAG integration is a follow-on task
  - Acceptance: anonymous and authenticated calls both succeed; session_id persists across messages

- **Task 6.3**: Implement lead capture in chatbot
  - If user provides name/email in chat, create/update lead record in `global-leads`
  - Link `anon_session_id` on the lead record
  - Acceptance: lead record created with `anon_session_id`; duplicate sessions do not create duplicate leads

- **Task 6.4**: Implement live agent handoff
  - Chatbot sends "connect me to support" intent → opens IO channels chat session with DEFT support
  - Acceptance: handoff creates a new IO chat thread; DEFT support space member receives it

- **Task 6.5**: Implement Clerk anonymous session promotion on registration
  - Clerk `user.created` webhook handler links `anon_session_id` from prior sessions to new `user_id`
  - Updates lead record: set `user_id`, change status from `anonymous` to `registered`
  - Acceptance: after sign-up, prior chatbot sessions are linked to user; lead record updated

- **Task 6.6**: Implement RAG over `global-docs` (stub → real)
  - Index `global-docs` threads; on chat message, retrieve top-k relevant chunks as context
  - Acceptance: chatbot answers questions about published documentation content

---

### Phase 7: Conversion Flows & Home Screen Customization
**Dependencies**: Phases 3–6
**Agents**: 2 (conversion and customization can run in parallel)

#### Subphase 7.1: Tier 2 → Tier 3 Conversion

- **Task 7.1.1**: Self-service upgrade flow (stub)
  - "Upgrade" button → billing stub page (OQ-1 deferred); creates org, promotes user to Tier 3
  - Acceptance: user org created; tier re-resolved to Tier 3 on next login

- **Task 7.1.2**: Sales-assisted conversion in DEFT CRM
  - DEFT sales member can mark a lead as "converted" from the `global-leads` view
  - Triggers org creation and Tier 3 promotion via API
  - Acceptance: DEFT sales member can convert; converted user's tier updates on next login

- **Task 7.1.3**: Admin override in platform admin panel
  - Platform admin can set a user's org and mark them as paying from `/admin/users`
  - Acceptance: admin can promote; immediate effect; audit-logged

#### Subphase 7.2: Full Home Screen Customization

- **Task 7.2.1**: Polish `HomeLayoutEditor` — drag-and-drop reorder, accessible keyboard controls
  - Acceptance: reorder works via mouse drag and keyboard; ARIA labels present

- **Task 7.2.2**: Add "Reset to defaults" button in layout editor
  - Clears saved preferences; reverts to tier default layout
  - Acceptance: reset confirmed by dialog; default layout restored immediately

- **Task 7.2.3**: E2E tests for customization persistence across tiers
  - Acceptance: saved layout survives page reload and re-login for Tier 2, 3, 4, 5, 6

---

## Testing Strategy

- **Unit tests**: TierResolver logic (all 6 paths + edge cases); widget visibility logic;
  home-preferences API (CRUD, invalid input)
- **Integration tests**: All new API endpoints (`/me/tier`, `/me/home-preferences`, `/chat/message`);
  global space permission enforcement; support ticket org-scoping
- **E2E tests**: One full happy-path per tier (anonymous browse → chatbot → sign up → tier 2 home;
  conversion to tier 3; DEFT employee home by department; platform admin home)
- **Security tests**: Unauthenticated access to `/admin/*` returns redirect; cross-org ticket
  leakage cannot occur; `global-leads` inaccessible to non-DEFT users

≥85% test coverage required on all new code per project standards.

---

## Deployment

No new services required. Changes deploy as:

1. **API** (`deft-evolution-api`): DB migrations (Phase 1) → new endpoints → updated middleware
2. **Web** (`deft-evolution-web`): new route group, updated `middleware.ts`, new components

Migration order: run Phase 1 migrations before deploying Phase 2+ code.
Seed scripts for global spaces and `deft` org must be idempotent (safe to re-run).

New environment variables required: none in this phase (AI provider config deferred to Phase 6.6).
