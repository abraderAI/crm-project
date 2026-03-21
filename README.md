# DEFT Evolution — Unified CRM & Community Platform

A full-stack CRM and community platform built on a hierarchical threaded content model. Pre-sales teams manage leads and pipelines; converted customers get their own org with spaces for support and documentation; developers and customers collaborate in community spaces. All interactions share the same auth, permissions, search, and activity timeline.

---

## Features

- **Hierarchical content model** — Org → Space → Board → Thread → Message, with JSONB metadata at every level
- **Granular RBAC** — `viewer` → `commenter` → `contributor` → `moderator` → `admin` → `owner`, with explicit-override + parent-fallback resolution
- **Dual authentication** — Clerk JWT or API key (`X-API-Key: deft_live_...`)
- **Sales CRM** — Configurable pipeline stages, rule-based + LLM lead scoring, automated lead-to-customer provisioning
- **Billing** — FlexPoint integration behind a swappable `BillingProvider` interface
- **Community** — Weighted voting, role-based moderation with content flagging
- **Real-time** — WebSocket channels with RBAC-scoped subscriptions
- **Notifications** — In-app + email (Resend) + digests, behind a `NotificationProvider` interface
- **Search** — FTS5 full-text search with metadata filters, RBAC scoping, ranked snippets
- **File uploads** — `StorageProvider` abstraction (local → S3/R2), 100 MB default limit
- **Webhooks** — Org/space/board-scoped subscriptions, HMAC-SHA256 signing, delivery retries
- **Audit log** — Every mutation logged: who, what, when, before/after diff, IP, request ID
- **Administration console** — Platform-level admin with user/org management, impersonation, system settings, feature flags, security monitoring
- **GDPR** — Hard-purge admin endpoints; soft delete everywhere else
- **Observability** — Structured `slog` logging + OpenTelemetry traces and metrics
- **App shell** — Sidebar navigation, topbar with search + user menu, breadcrumbs, dark/light theme toggle
- **File preview** — Rich staged file previews (image thumbnails, icons, size) with upload progress indicators
- **IO Channel Gateway** — Pluggable inbound channel processing with per-channel config, dead-letter queue (DLQ), and exponential-backoff retry engine
- **Inbound Email** — Multiple IMAP inboxes per org (e.g. support@, sales@), each with a configurable **routing action**: `support_ticket` → Support space, `sales_lead` → CRM space, `general` → General space. Live IMAP IDLE watcher starts on server boot, reconnects with exponential backoff, and delivers unread mail on reconnect. MIME parsing, thread deduplication by Message-ID / In-Reply-To, attachment storage, and dead-letter queue.
- **Voice / LiveKit** — LiveKit AI-first inbound calls: STT→LLM→TTS agent pipeline, human escalation, call recording, transcript stored as thread messages, phone number provisioning; swappable `VoiceProvider` interface (stub available for core CRM)
- **AI Web Chat Widget** — Embeddable shadow-DOM widget (plain JS/TS, zero deps) with JWT session auth, WebSocket streaming, and LLM-powered responses
- **Agentic CLI** — `deft` terminal client for natural-language CRM queries via LLM function-calling, interactive REPL, and one-shot mode
- **Reporting** — Org-scoped support-ticket and sales-pipeline analytics (15 query types): volume-over-time, status/priority breakdowns, assignee distribution, win/loss rates, funnel conversion, deal-value score distribution, avg time-in-stage, and CSV export with date + assignee filtering. Platform admins get cross-org aggregate dashboards with per-org sortable breakdown tables.
- **RBAC User Tiers** — Six-tier user classification (anonymous → authenticated → customer → customer admin → DEFT internal → platform admin) with tier-specific home screens, customisable widget layouts, global content spaces (docs, forum, support, leads), AI chatbot with live-agent escalation, and automated conversion flows (self-service upgrade, sales-led conversion, admin promotion).
- **Leads Management** — Dedicated `/crm/leads` page for DEFT sales staff: tier 5–6 see all leads with assignee filter; tier 4 sales reps see only their own and assigned leads. Status/search filters, load-more pagination, and detail view per lead.
- **Support Ticket Management** — Tier-aware `/support` page: tier 1 sees a sign-in prompt; tiers 2–3 see own tickets (org-scoped for org members); tier 4 DEFT staff and tier 5 DEFT support admins see all tickets; tier 5 customer org admins see org-scoped tickets; all elevated views include an open/pending/resolved stats strip. Inline create-ticket form with `org_id` passthrough. Each ticket row shows the **subject** and **creator** (email + org name). Clicking **Open** opens the full **multi-entry ticket editor**: a chronological timeline of entries (customer messages, agent replies, internal context notes, drafts, system events) with per-entry badges and timestamps. The ticket creator's initial body is automatically saved as the first customer entry on creation. Entries may be published, edited (drafts only), and toggled between public and DEFT-only visibility. DEFT members (tiers 4–6) can compose new entries, set DEFT-only visibility, and update ticket status (`open` → `pending` → `resolved`). Immutable entries (initial customer message, published entries) are locked. Full dark-mode support throughout.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.24+, [Chi](https://github.com/go-chi/chi), GORM, `modernc.org/sqlite` (pure Go, no CGo) |
| Frontend | Next.js 14+ (App Router, RSC), TypeScript, shadcn/ui, Tailwind CSS |
| Auth | [Clerk](https://clerk.com) (JWT validation only) |
| Database | SQLite — WAL mode, FTS5, JSON + generated columns |
| Real-time | `coder/websocket` |
| Email | [Resend](https://resend.com) + React Email |
| Billing | FlexPoint (`BillingProvider` interface) |
| Voice | [LiveKit](https://livekit.io) (room/webhook/bridge/transcript) |
| Chat Widget | esbuild IIFE bundle, shadow DOM, zero dependencies |
| Charts | [Recharts](https://recharts.org) (reporting dashboards) |
| CLI | [Cobra](https://github.com/spf13/cobra), [lipgloss](https://github.com/charmbracelet/lipgloss), tablewriter |
| Observability | slog + OpenTelemetry |
| Testing | Go: testify + httptest + native fuzz · Frontend: Vitest + Playwright |
| CI/CD | GitHub Actions |
| Deployment | Docker Compose (local) · [Fly.io](https://fly.io) (backend) · [Vercel](https://vercel.com) (frontend) |

---

## Repository Structure

```
/
├── api/                        # Go backend
│   ├── cmd/
│   │   ├── server/             # API server entry point
│   │   └── cli/                # deft CLI entry point
│   ├── config/                 # rbac-policy.yaml, pipeline config
│   ├── internal/
│   │   ├── admin/              # Administration console (platform admin)
│   │   ├── org/                # Handler → service → repository
│   │   ├── space/
│   │   ├── board/
│   │   ├── thread/
│   │   ├── message/
│   │   ├── auth/               # JWT, API key, RBAC engine
│   │   ├── billing/
│   │   ├── search/
│   │   ├── notification/
│   │   ├── vote/
│   │   ├── moderation/
│   │   ├── upload/
│   │   ├── webhook/
│   │   ├── audit/
│   │   ├── pipeline/           # CRM pipeline stages
│   │   ├── scoring/            # Lead scoring
│   │   ├── provision/          # Lead-to-customer provisioning
│   │   ├── telemetry/          # OpenTelemetry
│   │   ├── reporting/          # Support + sales metrics, admin aggregates, CSV export
│   │   ├── channel/            # IO Channel Gateway
│   │   │   ├── email/          # Inbound email (MIME parsing, threading)
│   │   │   ├── voice/          # LiveKit voice (rooms, webhooks, transcripts)
│   │   │   └── chat/           # AI chat session handler
│   │   ├── cli/                # CLI internals
│   │   │   ├── agent/          # LLM function-calling agent
│   │   │   ├── auth/           # Credential store (keyring)
│   │   │   ├── chat/           # REPL + one-shot session
│   │   │   ├── client/         # Typed HTTP client for the CRM API
│   │   │   ├── config/         # CLI config (file + env overrides)
│   │   │   └── output/         # Table / JSON formatter (lipgloss)
│   │   └── server/             # Router configuration
│   └── pkg/                    # Shared: pagination, errors, slugs, response
├── web/                        # Next.js frontend
│   ├── src/
│   │   ├── app/                # App Router pages
│   │   ├── components/
│   │   │   ├── admin/          # Billing, webhooks, audit log, membership
│   │   │   ├── community/      # Voting, moderation, flagging
│   │   │   ├── crm/            # Pipeline, kanban, lead detail, scoring
│   │   │   ├── reports/        # Shared reporting components + chart components (support/, sales/)
│   │   │   ├── editor/         # Message editor, toolbar, revision history
│   │   │   ├── entities/       # Entity card, form, list, create/settings views
│   │   │   ├── layout/         # App shell: sidebar, topbar, breadcrumbs
│   │   │   ├── realtime/       # WebSocket, notifications, typing indicators
│   │   │   ├── thread/         # Thread list, detail, filters, message timeline
│   │   │   └── upload/         # File upload, preview, progress
│   │   ├── hooks/              # useWebSocket, useNotifications, useTyping
│   │   └── lib/                # API client, types, utils
│   └── e2e/                    # Playwright smoke tests
├── widget/                     # Embeddable chat widget (esbuild, shadow DOM)
│   └── src/                    # widget.ts, chat.ts, ui.ts, fingerprint.ts
├── agent/                      # TypeScript AI agent (function-calling wrapper)
├── docker/                     # Dockerfiles + docker-compose.yml
├── docs/                       # Environment variables, architecture notes, PRDs, specs
├── secrets/                    # Local secret files (git-ignored); see secrets/README.md
├── CHANGELOG.md                # Project changelog (Keep a Changelog format)
└── Taskfile.yml                # All build/test/lint commands
```

---

## Frontend Routes

| Route | Description |
|---|---|
| `/` | Home / dashboard — tier-aware: renders the correct home screen for the current user's tier with customisable widget layout |
| `/docs/[...slug]` | Public documentation (global-docs space) — no auth required |
| `/forum/[...slug]` | Public community forum (global-forum space) — no auth required |
| `/sign-in`, `/sign-up` | Clerk authentication |
| `/orgs/create` | Create organization |
| `/orgs/[org]` | Org overview |
| `/orgs/[org]/settings` | Org settings |
| `/orgs/[org]/spaces/create` | Create space |
| `/orgs/[org]/spaces/[space]` | Space overview |
| `/orgs/[org]/spaces/[space]/settings` | Space settings |
| `/orgs/[org]/spaces/[space]/boards/create` | Create board |
| `/orgs/[org]/spaces/[space]/boards/[board]` | Board view (thread list) |
| `/orgs/[org]/spaces/[space]/boards/[board]/settings` | Board settings |
| `/orgs/[org]/spaces/[space]/boards/[board]/threads/create` | Create thread |
| `/orgs/[org]/spaces/[space]/boards/[board]/threads/[thread]` | Thread detail (messages, editor, attachments, voting, flags, revisions) |
| `/crm` | CRM pipeline — Kanban board with drag-and-drop stage management |
| `/crm/leads` | Leads management — tier-aware list: tier 5–6 see all leads with assignee filter; tier 4 `deft_sales` reps see own/assigned leads only; status, search, and load-more pagination |
| `/crm/leads/global/[thread_slug]` | Global lead detail — enrichment data, scoring breakdown, and metadata sidebar for leads in the `global-leads` space |
| `/crm/leads/[org]/[space]/[board]/[thread]` | Lead detail — enrichment, scoring breakdown, metadata sidebar |
| `/support` | Support tickets — tier-aware list with open/pending/resolved stats strip and inline create-ticket form; **Open** launches the multi-entry ticket editor: chronological timeline of customer messages, agent replies, drafts, internal notes, and system events; initial ticket body auto-saved as first entry; per-entry publish / edit / DEFT-visibility controls; status transitions (tiers 4–6) |
| `/search` | Full-text search with filters |
| `/notifications` | Notification feed |
| `/notifications/preferences` | Notification channel preferences |
| `/admin` | Admin dashboard |
| `/admin/users` | User management list |
| `/admin/users/[user_id]` | User detail — profile, cross-org memberships, ban/unban, GDPR purge, impersonation |
| `/admin/billing` | Billing dashboard (FlexPoint) |
| `/admin/webhooks` | Webhook management + delivery log |
| `/admin/members` | Organization membership manager |
| `/admin/moderation` | Content moderation queue |
| `/admin/audit-log` | Platform-wide audit log |
| `/admin/channels` | IO Channel configuration hub |
|| `/admin/channels/[type]` | Per-channel config, health, and DLQ monitor (`email` \| `voice` \| `chat`) |
|| `/admin/channels/email` | Email channel — includes **Email Inboxes** panel: add/edit/delete IMAP inboxes with routing action selector |
| `/admin/feature-flags` | Feature flag management |
| `/admin/settings` | System settings — editable key-value platform configuration |
| `/admin/security` | Security monitoring — recent logins and failed authentication events |
| `/admin/rbac-policy` | RBAC policy editor — resolution strategy, role hierarchy, and dry-run role preview |
| `/admin/api-usage` | API usage stats by endpoint (24 h / 7 d / 30 d windows) |
| `/admin/llm-usage` | LLM enrichment call log (thread, model, tokens, latency) |
| `/admin/exports` | Async data exports — trigger, poll status, and download (CSV / JSON) |
| `/admin/channels/voice/numbers` | Phone number provisioning — list owned numbers, search available, purchase |
| `/upgrade` | Self-service tier upgrade — plan comparison and activation |
| `/settings` | User profile (Clerk), personal API keys, notification preferences, tier status |
| `/reports` | Redirects to `/reports/support` |
| `/reports/support` | Support tickets dashboard — status breakdown, volume over time, assignee + priority charts, CSV export |
| `/reports/sales` | Sales pipeline dashboard — funnel, lead velocity, assignee, score distribution, conversion rates, time-in-stage, CSV export |
| `/admin/reports/support` | Platform-wide support overview + per-org sortable breakdown table |
| `/admin/reports/sales` | Platform-wide sales overview + per-org sortable breakdown table |

All routes use server components for data fetching with client wrappers for interactivity. The app shell (`AppLayoutWrapper`) provides sidebar navigation, topbar with search and Clerk `UserButton`, and route-aware active states.

---

## Getting Started

### Prerequisites

- [Go 1.24+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/)
- [Task](https://taskfile.dev/#/installation) (`brew install go-task/tap/go-task`)
- [Docker](https://www.docker.com/) (optional, for Compose)

### Local Development

**1. Clone and install dependencies**

```bash
git clone https://github.com/abraderAI/crm-project.git
cd crm-project
cd api && go mod download && cd ..
cd web && npm ci && cd ..
```

**2. Configure environment**

```bash
# API
cp api/.env.example api/.env   # fill in CLERK_* keys

# Web
cp web/.env.example web/.env.local   # fill in NEXT_PUBLIC_* keys
```

See [`docs/env-vars.md`](docs/env-vars.md) for the full reference.

> **Secrets**: place any local `.env` files you don't want tracked in the `secrets/` directory (git-ignored). See `secrets/README.md`.

**3. Bootstrap a platform admin**

Set the `PLATFORM_ADMIN_USER_ID` environment variable to your Clerk user ID before starting the server. This seeds the first platform admin on startup:

```bash
export PLATFORM_ADMIN_USER_ID=user_abc123
```

**4. Start the backend**

```bash
task dev          # Go API server on :8080
```

**5. Start the frontend**

```bash
cd web && npm run dev   # Next.js on :3000
```

**Or use Docker Compose**

```bash
docker compose -f docker/docker-compose.yml up
```

---

## Development Commands

All commands are defined in `Taskfile.yml` and run from the repo root.

```bash
task build             # Compile the API binary
task cli:build         # Compile the deft CLI binary (→ api/bin/deft)
task fmt               # Format Go code (gofmt)
task lint              # Lint Go code (golangci-lint)
task test              # Run all Go tests (race detector)
task test:coverage     # Go tests + enforce ≥85% coverage
task test:fuzz         # Run fuzz tests (5s each)
task check             # Full pre-commit suite (fmt + lint + test + coverage + web + widget checks)

# Go-namespaced aliases (equivalent to the above, preferred in CI)
task go:build
task go:fmt
task go:lint
task go:test
task go:test:coverage
task go:test:fuzz

task web:fmt           # Format frontend (Prettier)
task web:fmt:check     # Check frontend formatting
task web:lint          # ESLint
task web:typecheck     # TypeScript type check
task web:test          # Vitest unit tests
task web:test:coverage # Vitest + enforce ≥85% coverage
task web:build         # Next.js production build
task web:e2e           # Playwright E2E tests
task web:e2e:smoke     # Playwright smoke tests only (@smoke tag)

task widget:build         # esbuild → dist/widget.js (IIFE bundle)
task widget:test          # Vitest widget unit tests
task widget:test:coverage # Vitest + enforce ≥85% coverage
```

---

## API Overview

- **Base path**: `/v1`
- **Auth**: `Authorization: Bearer <clerk-jwt>` or `X-API-Key: deft_live_...`
- **Errors**: [RFC 7807 Problem Details](https://datatracker.ietf.org/doc/html/rfc7807) (`application/problem+json`)
- **Pagination**: Cursor-based (UUIDv7), default 50, max 100
- **IDs**: UUIDv7 (time-ordered); slugs auto-generated, unique per parent

### Core Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/healthz` | Liveness probe |
| `GET` | `/readyz` | Readiness probe (DB connectivity) |
| `GET` | `/v1/search` | FTS5 full-text search |
| `GET/POST` | `/v1/orgs` | Org management |
| `*` | `/v1/orgs/{org}/spaces` | Space CRUD |
| `*` | `/v1/orgs/{org}/spaces/{space}/boards` | Board CRUD |
| `*` | `/v1/orgs/{org}/.../{board}/threads` | Thread CRUD + pin/lock/vote/move |
| `GET` | `/v1/ws` | WebSocket upgrade (JWT via `?token=`) |
| `GET/PATCH` | `/v1/notifications` | Notifications + preferences |
| `POST/GET` | `/v1/uploads` | File uploads |
| `*` | `/v1/orgs/{org}/webhooks` | Webhook subscriptions |
| `GET` | `/v1/orgs/{org}/audit-log` | Audit log |
| `*` | `/v1/orgs/{org}/billing` | Billing management |

### User Tier & Conversion Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/me/tier` | Resolve current user's tier (1–6) and tier metadata |
| `GET` | `/v1/me/home-preferences` | Get home screen widget layout preferences |
| `PUT` | `/v1/me/home-preferences` | Save home screen widget layout preferences |
| `POST` | `/v1/me/upgrade` | Self-service tier upgrade (e.g. anonymous → registered, or registered → customer trial) |
| `POST` | `/v1/leads/{lead_id}/convert` | Sales-led conversion — promote a lead to a paying customer org |
| `POST` | `/v1/admin/users/{user_id}/promote` | Admin-force tier promotion (bypasses normal conversion flow) |

### Reporting Endpoints

All accept `?from=YYYY-MM-DD&to=YYYY-MM-DD&assignee=<user_id>`. Org-scoped routes require `admin` or `owner` org role; admin routes require platform admin.

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/orgs/{org}/reports/support` | Org support metrics (7 query types) |
| `GET` | `/v1/orgs/{org}/reports/support/export` | CSV export of support tickets |
| `GET` | `/v1/orgs/{org}/reports/sales` | Org sales metrics (8 query types) |
| `GET` | `/v1/orgs/{org}/reports/sales/export` | CSV export of sales leads |
| `GET` | `/v1/admin/reports/support` | Platform-wide support metrics + per-org breakdown |
| `GET` | `/v1/admin/reports/support/export` | Platform-wide support CSV export |
| `GET` | `/v1/admin/reports/sales` | Platform-wide sales metrics + per-org breakdown |
| `GET` | `/v1/admin/reports/sales/export` | Platform-wide sales CSV export |

### Global Space Endpoints

The `global-support` and `global-leads` spaces are served under `/v1/global-spaces/{space}/threads`. These endpoints use the same RBAC tier gating as the frontend.

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/global-spaces/{space}/threads` | List threads — enriched with `author_email`, `author_name`, `org_name`; cursor-paginated |
| `POST` | `/v1/global-spaces/{space}/threads` | Create a thread in a global space; a non-empty body is atomically saved as the first customer timeline entry |
| `GET` | `/v1/global-spaces/{space}/threads/{slug}` | Fetch a single thread with author enrichment |
| `PATCH` | `/v1/global-spaces/{space}/threads/{slug}` | Update thread body and/or status; status change is written as a metadata deep-merge and a revision is recorded |
| `GET` | `/v1/global-spaces/{space}/threads/{slug}/attachments` | List file attachments for a thread |
| `POST` | `/v1/global-spaces/{space}/threads/{slug}/attachments` | Upload a file attachment to a thread |

### Support Ticket Entry Endpoints

The multi-entry ticket editor exposes a sub-resource API under `/v1/support/tickets/{slug}`. Entry visibility is filtered by DEFT membership: non-DEFT callers only receive published entries and their own drafts.

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/support/tickets/{slug}/entries` | List entries — published entries visible to all; DEFT callers also see DEFT-only entries |
| `POST` | `/v1/support/tickets/{slug}/entries` | Create a new entry (`customer`, `agent_reply`, `draft`, `context`, `system_event`) |
| `PATCH` | `/v1/support/tickets/{slug}/entries/{id}` | Update body of a mutable (draft, not yet published) entry |
| `POST` | `/v1/support/tickets/{slug}/entries/{id}/publish` | Publish a draft entry (DEFT members or the draft's own author) |
| `PATCH` | `/v1/support/tickets/{slug}/entries/{id}/deft-visibility` | Toggle DEFT-only visibility on an entry (DEFT members only) |
| `PATCH` | `/v1/support/tickets/{slug}/notifications` | Update the caller's notification preference for this ticket |

---

### IO Channel Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/orgs/{org}/channels/{type}` | Get channel config (`email` \| `voice` \| `chat`) |
| `PUT` | `/v1/orgs/{org}/channels/{type}` | Upsert channel config (platform admin bypasses org-membership check) |
|| `GET` | `/v1/orgs/{org}/channels/{type}/health` | Channel health check |
|| `GET` | `/v1/orgs/{org}/channels/{type}/dlq` | List dead-letter queue events |
|| `POST` | `/v1/orgs/{org}/channels/{type}/dlq/{id}/retry` | Retry a DLQ event |
|| `POST` | `/v1/orgs/{org}/channels/{type}/dlq/{id}/dismiss` | Dismiss a DLQ event |
|| `GET` | `/v1/orgs/{org}/channels/email/inboxes` | List email inboxes |
|| `POST` | `/v1/orgs/{org}/channels/email/inboxes` | Create email inbox (IMAP credentials + routing action) |
|| `PUT` | `/v1/orgs/{org}/channels/email/inboxes/{id}` | Update email inbox |
|| `DELETE` | `/v1/orgs/{org}/channels/email/inboxes/{id}` | Delete email inbox |
|| `GET` | `/v1/orgs/{org}/channels/voice/numbers` | List owned phone numbers |
| `POST` | `/v1/orgs/{org}/channels/voice/numbers/search` | Search available numbers by area code / country |
| `POST` | `/v1/orgs/{org}/channels/voice/numbers/purchase` | Purchase a phone number (confirmation required) |
| `POST` | `/v1/chat/session` | Create anonymous chat session (JWT issued) |
| `POST` | `/v1/chat/message` | Send a chat message (WebSocket streaming) |
| `POST` | `/v1/webhooks/livekit` | LiveKit webhook ingestion (token-verified) |
| `GET` | `/v1/internal/contacts/lookup` | Internal bridge: contact lookup by email / phone |
| `GET` | `/v1/internal/threads/{id}/summary` | Internal bridge: thread summary for voice |

### Administration Console (`/v1/admin/*`)

All admin endpoints require platform admin privileges. The first platform admin is bootstrapped via the `PLATFORM_ADMIN_USER_ID` environment variable; additional admins are managed through the API.

**User Management**

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/admin/users` | List users (filter by email, name, org, ban status, last seen) |
| `GET` | `/v1/admin/users/{user_id}` | User detail with cross-org memberships |
| `POST` | `/v1/admin/users/{user_id}/ban` | Ban user (blocks all API access) |
| `POST` | `/v1/admin/users/{user_id}/unban` | Unban user |
| `DELETE` | `/v1/admin/users/{user_id}/purge` | GDPR hard-purge user data |
| `POST` | `/v1/admin/users/{user_id}/impersonate` | Time-limited impersonation token (max 2h) |

**Organization Management**

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/admin/orgs` | List orgs (filter by slug, billing tier, payment status) |
| `GET` | `/v1/admin/orgs/{org}` | Org detail with member/space/thread counts |
| `POST` | `/v1/admin/orgs/{org}/suspend` | Suspend org (blocks all write operations) |
| `POST` | `/v1/admin/orgs/{org}/unsuspend` | Unsuspend org |
| `POST` | `/v1/admin/orgs/{org}/transfer-ownership` | Transfer org ownership |
| `DELETE` | `/v1/admin/orgs/{org}/purge` | GDPR hard-purge org (requires confirmation) |

**Platform Admin Management**

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/admin/platform-admins` | List platform admins |
| `POST` | `/v1/admin/platform-admins` | Add platform admin |
| `DELETE` | `/v1/admin/platform-admins/{user_id}` | Remove platform admin (cannot remove last) |

**Configuration & Monitoring**

| Method | Path | Description |
|---|---|---|
| `GET/PATCH` | `/v1/admin/settings` | System settings (deep-merge update) |
| `GET/PATCH` | `/v1/admin/rbac-policy` | RBAC policy overrides |
| `POST` | `/v1/admin/rbac-policy/preview` | Dry-run RBAC role resolution |
| `GET/PATCH` | `/v1/admin/feature-flags/{key}` | Feature flag management |
| `GET` | `/v1/admin/stats` | Platform stats (org/user/thread counts, DB size) |
| `GET` | `/v1/admin/webhooks/deliveries` | Platform-wide webhook delivery log |
| `GET` | `/v1/admin/integrations/status` | Integration health (Clerk, Resend, FlexPoint) |
| `GET` | `/v1/admin/audit-log` | Platform-wide audit log with filters |

**Advanced Features**

| Method | Path | Description |
|---|---|---|
| `POST/GET` | `/v1/admin/exports` | Async data exports (users, orgs, audit → CSV/JSON) |
| `GET` | `/v1/admin/exports/{id}` | Export status and download |
| `GET` | `/v1/admin/api-usage` | Per-endpoint request counts (24h/7d/30d) |
| `GET` | `/v1/admin/llm-usage` | LLM enrichment call log |
| `GET` | `/v1/admin/security/recent-logins` | Login event log |
| `GET` | `/v1/admin/security/failed-auths` | Failed authentication tracking |

**Admin Middleware** (applied globally to authenticated routes):
- **BanCheck** — Banned users receive 403 on all requests
- **OrgSuspensionCheck** — Suspended orgs reject write operations with 503
- **MaintenanceMode** — When enabled via feature flag, all non-GET requests return 503
- **UserShadowSync** — Syncs Clerk user data to local shadow table on each request
- **APIUsageCounter** — Tracks per-endpoint request counts (async, non-blocking)
- **LoginEventRecorder** — Records login events (debounced, 1 per user per hour)

---

## Deployment

### Backend — Fly.io

```bash
fly apps create deft-evolution-api
fly volumes create deft_data --region iad --size 3
fly secrets set \
  CLERK_SECRET_KEY=... \
  CLERK_ISSUER_URL=... \
  CORS_ORIGINS=... \
  PLATFORM_ADMIN_USER_ID=... \
  CHAT_JWT_SECRET=... \
  INTERNAL_API_KEY=... \
  LIVEKIT_URL=... \
  LIVEKIT_API_KEY=... \
  LIVEKIT_API_SECRET=... \
  LIVEKIT_WEBHOOK_TOKEN=...
fly deploy
```

SQLite is persisted on a Fly volume at `/data/deft.db`. The architecture is Litestream-ready for future streaming backups.

Health checks: `GET /healthz` (liveness) and `GET /readyz` (readiness with DB connectivity).

### Frontend — Fly.io

The frontend is deployed as a Docker container to Fly.io. A `fly.web.toml` config is included in the repo root.

```bash
fly apps create deft-evolution-web
fly secrets set CLERK_SECRET_KEY=sk_live_... --app deft-evolution-web
flyctl deploy --config fly.web.toml --app deft-evolution-web
```

`NEXT_PUBLIC_*` variables (Clerk publishable key, API URL) are baked into the image at build time via Docker build args — update them in `fly.web.toml` before deploying.

---

## Architecture Notes

### Provider Abstractions

All external integrations are behind interfaces, making them testable and swappable without code changes:

| Interface | Default | Swap-in |
|---|---|---|
| `StorageProvider` | Local filesystem | S3 / Cloudflare R2 |
| `BillingProvider` | FlexPoint | Stripe |
| `NotificationProvider` | In-app + Resend | Slack, Teams, SMS |
| `LLMProvider` | Grok | OpenAI / Anthropic |
| `VoiceProvider` | Stub | Bland.ai / Retell / Twilio |
| `LiveKitProvider` | Mock (testing) | LiveKit Cloud / self-hosted |
| `Keyring` (CLI) | In-memory (testing) | OS keyring (`go-keyring`) |

### RBAC

Roles are resolved bottom-up: **board membership → space membership → org membership → no access**. A lower-level role completely replaces the inherited one (explicit override). Strategy is configurable in `api/config/rbac-policy.yaml` and can be overridden at runtime via `/v1/admin/rbac-policy`.

Platform admins exist outside the Org-scoped RBAC hierarchy. They are stored in a separate `platform_admins` table and resolved before normal RBAC checks. Channel config mutations and email inbox management bypass the org-membership check when the caller is a platform admin.

### Database

SQLite in WAL mode with FTS5 virtual tables for full-text search, JSON-extracted generated columns for hot filter paths, and soft delete on all entities. No CGo — uses `modernc.org/sqlite`.

Admin-specific tables: `platform_admins`, `users_shadow`, `system_settings`, `feature_flags`, `admin_exports`, `api_usage_stats`, `login_events`.

IO channel tables: `channel_configs` (per-org, per-type enable/disable toggle with masked secrets), `email_inboxes` (per-org IMAP credentials, routing action, enabled flag — one row per inbox address), `dead_letter_events` (DLQ with status, retry count, next-retry timestamp).

### RBAC User Tiers

Tier resolution runs server-side on every request via `TierResolver`, checking in priority order:

| Tier | Label | Condition |
|---|---|---|
| 6 | Platform Admin | `platform_admin` record exists |
| 5 | Customer Admin | `admin` or `owner` role in any paying customer org |
| 4 | DEFT Internal | Member of the `deft` org |
| 3 | Customer | Member of any paying customer org |
| 2 | Authenticated | Has a Clerk account |
| 1 | Anonymous | Clerk anonymous token only |

The resolved tier is cached in the session. Each tier gets a distinct home screen (`HomeLayout`) composed of `Widget` components. Authenticated users can toggle visibility and reorder widgets; preferences are persisted via `PUT /v1/me/home-preferences` and loaded at SSR time. Default layouts per tier are defined in a static config.

Four system-seeded **global spaces** exist outside any customer org: `global-docs`, `global-forum`, `global-support`, and `global-leads`. The public docs and forum spaces are accessible without authentication.

### IO Channel Gateway

All inbound channels share a common processing pipeline:

1. **Normalise** — each channel implements `Normalizer` to produce a typed `InboundEvent`
2. **Route** — for email, the inbox's `routing_action` selects the target space type: `support_ticket` → `SpaceTypeSupport`, `sales_lead` → `SpaceTypeCRM`, `general` → `SpaceTypeGeneral`; falls back to any available space
3. **Process** — the `Gateway` resolves or creates a thread (deduplicating by Message-ID / In-Reply-To for email), appends the message, stores attachments, and fires domain events
4. **Retry** — `RetryEngine` wraps processing with exponential backoff (`1s → 2s → 4s … 30s max`); on exhaustion the event is written to the DLQ
5. **DLQ** — operators can inspect, retry, or dismiss failed events via the channel health API

**Email Inbox Watcher** (`InboxWatcher`): On server startup all enabled `EmailInbox` records are loaded and one `IDLEManager` is started per inbox. Each manager connects with implicit TLS, authenticates with username + App Password, selects the mailbox, processes any existing unread mail, then enters IMAP IDLE. New arrivals are fetched and processed with the inbox's routing action. Managers reconnect with exponential backoff on disconnection. Adding, editing, or deleting an inbox via the API restarts its manager immediately — no server restart needed.

### Embeddable Chat Widget

The widget (`widget/dist/widget.js`) is a single self-contained IIFE bundle built with esbuild. It attaches a shadow DOM to avoid CSS leakage and has zero external runtime dependencies. Embed with:

```html
<script src="/widget.js"></script>
<script>CRMChatWidget.init({ orgId: "my-org", apiUrl: "https://api.example.com" });</script>
```

### Agentic CLI

The `deft` CLI ships as a single binary. It uses LLM function-calling to map natural-language queries to typed CRM API calls, with multi-step tool-use chains and a full conversation history.

```bash
# Build
task cli:build        # → api/bin/deft

# One-shot query
deft ask "show me all leads from last week"

# Interactive REPL
deft chat

# Auth
deft login --token <api-key-or-jwt>
deft whoami
```

Config is read from `~/.deft-cli.yaml` with env var overrides (`DEFT_API_URL`, `DEFT_API_KEY`, `DEFT_ORG`).

---

## Quality

- **2,050 frontend tests** across 159 test files (Vitest) — 91% statement, 85% branch coverage
- **1,878 Go tests** across 37 packages with race detector enabled — 85.8% coverage
- **15 fuzz functions** across 8 Go packages (`board`, `thread`, `message`, `membership`, `conversion`, `gdpr`, `tier`, `notification`), each with ≥ 40 diverse seeds
- ≥ 85% test coverage enforced on every PR (statements + branches)
- `task check` must pass fully before any merge (fmt + lint + typecheck + tests + coverage)
- All errors follow RFC 7807 Problem Details
- All admin destructive actions require confirmation and are audit-logged with reason

---

## License

MIT
