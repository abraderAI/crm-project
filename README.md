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
- **Voice** — Stubbed `VoiceProvider` interface with thread-based logging and escalation
- **App shell** — Sidebar navigation, topbar with search + user menu, breadcrumbs, dark/light theme toggle
- **File preview** — Rich staged file previews (image thumbnails, icons, size) with upload progress indicators
- **IO Channel Gateway** — Pluggable inbound channel processing with per-channel config, dead-letter queue (DLQ), and exponential-backoff retry engine
- **Inbound Email** — MIME parsing, thread deduplication by Message-ID / In-Reply-To, lead-thread auto-creation
- **Voice / LiveKit** — LiveKit room management, webhook ingestion, phone call bridging, transcript storage, and escalation
- **AI Web Chat Widget** — Embeddable shadow-DOM widget (plain JS/TS, zero deps) with JWT session auth, WebSocket streaming, and LLM-powered responses
- **Agentic CLI** — `deft` terminal client for natural-language CRM queries via LLM function-calling, interactive REPL, and one-shot mode

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
| CLI | [Cobra](https://github.com/spf13/cobra), [lipgloss](https://github.com/charmbracelet/lipgloss), tablewriter |
| Observability | slog + OpenTelemetry |
| Testing | Go: testify + httptest · Frontend: Vitest + Playwright |
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
├── docs/                       # Environment variables, architecture notes
├── SPECIFICATION.md            # Full implementation spec
└── Taskfile.yml                # All build/test/lint commands
```

---

## Frontend Routes

| Route | Description |
|---|---|
| `/` | Home / dashboard |
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
| `/crm/leads/[org]/[space]/[board]/[thread]` | Lead detail — enrichment, scoring breakdown, metadata sidebar |
| `/search` | Full-text search with filters |
| `/notifications` | Notification feed |
| `/notifications/preferences` | Notification channel preferences |
| `/admin` | Admin dashboard |
| `/admin/users` | User management (ban, purge, impersonate) |
| `/admin/billing` | Billing dashboard (FlexPoint) |
| `/admin/webhooks` | Webhook management + delivery log |
| `/admin/members` | Organization membership manager |
| `/admin/moderation` | Content moderation queue |
| `/admin/audit-log` | Platform-wide audit log |
| `/admin/feature-flags` | Feature flag management |

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

### IO Channel Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/orgs/{org}/channels/{type}` | Get channel config (`email` \| `voice` \| `chat`) |
| `PUT` | `/v1/orgs/{org}/channels/{type}` | Upsert channel config |
| `GET` | `/v1/orgs/{org}/channels/{type}/health` | Channel health check |
| `GET` | `/v1/orgs/{org}/channels/{type}/dlq` | List dead-letter queue events |
| `POST` | `/v1/orgs/{org}/channels/{type}/dlq/{id}/retry` | Retry a DLQ event |
| `POST` | `/v1/orgs/{org}/channels/{type}/dlq/{id}/dismiss` | Dismiss a DLQ event |
| `POST` | `/v1/chat/session` | Create anonymous chat session (JWT issued) |
| `POST` | `/v1/chat/message` | Send a chat message (WebSocket streaming) |
| `POST` | `/v1/webhooks/livekit` | LiveKit webhook ingestion (token-verified) |
| `GET` | `/v1/internal/contacts/lookup` | Internal bridge: contact lookup by phone |
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
fly secrets set CLERK_SECRET_KEY=... CLERK_ISSUER_URL=... CORS_ORIGINS=... PLATFORM_ADMIN_USER_ID=...
fly deploy
```

SQLite is persisted on a Fly volume at `/data/deft.db`. The architecture is Litestream-ready for future streaming backups.

Health checks: `GET /healthz` (liveness) and `GET /readyz` (readiness with DB connectivity).

### Frontend — Vercel

Connect the repo to Vercel and set the root directory to `web/`. Required environment variables:

```
NEXT_PUBLIC_API_URL=https://deft-evolution-api.fly.dev
NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY=pk_live_...
CLERK_SECRET_KEY=sk_live_...
```

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

Platform admins exist outside the Org-scoped RBAC hierarchy. They are stored in a separate `platform_admins` table and resolved before normal RBAC checks.

### Database

SQLite in WAL mode with FTS5 virtual tables for full-text search, JSON-extracted generated columns for hot filter paths, and soft delete on all entities. No CGo — uses `modernc.org/sqlite`.

Admin-specific tables: `platform_admins`, `users_shadow`, `system_settings`, `feature_flags`, `admin_exports`, `api_usage_stats`, `login_events`.

IO channel tables: `channel_configs` (per-org, per-type config with masked secrets), `dead_letter_events` (DLQ with status, retry count, next-retry timestamp).

### IO Channel Gateway

All inbound channels share a common processing pipeline:

1. **Normalise** — each channel implements `Normalizer` to produce a typed `InboundEvent`
2. **Process** — the `Gateway` resolves or creates a lead thread, appends the message, and fires domain events
3. **Retry** — `RetryEngine` wraps processing with exponential backoff (`1s → 2s → 4s … 30s max`); on exhaustion the event is written to the DLQ
4. **DLQ** — operators can inspect, retry, or dismiss failed events via the channel health API

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

- **907 frontend tests** across 62 test files (Vitest) — 94% statement coverage
- **37 Go test packages** with race detector enabled — 86% coverage
- ≥ 85% test coverage enforced on every PR
- ≥ 50 fuzz test cases per input entry point
- `task check` must pass fully before any merge (fmt + lint + typecheck + tests + coverage)
- All errors follow RFC 7807 Problem Details
- All admin destructive actions require confirmation and are audit-logged with reason

---

## License

MIT
