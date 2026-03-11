# DEFT Evolution — Unified CRM & Community Platform

*Generated from `vbrief/specification.vbrief.json` — 2026-03-11*

## Overview

DEFT Evolution is a unified CRM and community platform built on a hierarchical threaded content model (Org → Space → Board → Thread → Message). Pre-sales teams manage leads, pipelines, and opportunities in dedicated Spaces/Boards. Converted customers receive their own Org with Spaces for support, feature requests, and documentation. Developers and customers collaborate in community Spaces. All interactions share the same authentication, permissions, metadata, search, and activity timeline.

The **Spaces API** is the foundational layer — built first — with all domain-specific behavior (sales CRM, billing, support, community) layered on top via metadata conventions, custom board rules, and UI views.

The backend MUST be complete with full test coverage before any frontend work begins.

## Requirements

### Functional

- Hierarchical content model: Org → Space → Board → Thread → Message with JSONB metadata at every level
- Clerk-based authentication (auth only; Orgs/memberships/RBAC managed in SQLite)
- Granular RBAC with configurable resolution strategy (explicit override with parent fallback)
- RESTful API (v1) with metadata filtering, full-text search, file uploads, webhooks
- Sales CRM: configurable pipeline stages, rule-based + LLM lead scoring, automated lead-to-customer provisioning
- Billing: FlexPoint integration behind provider abstraction
- Community: weighted voting, role-based moderation with flagging
- Real-time: WebSocket channels with RBAC-scoped subscriptions
- Notifications: in-app + email (Resend) + digests, behind provider abstraction
- Voice support: deferred — stubbed interface with Thread-based logging/escalation

### Non-Functional

- API/UI response < 2s
- ≥85% test coverage (lines, functions, branches, statements) — all code
- ≥50 fuzzing tests per input point
- Soft delete everywhere; GDPR hard-purge admin endpoint
- Full audit log (every mutation: who, what, when, before/after diff, IP, request ID)
- PII encrypted at rest/transit; PCI via FlexPoint (no card data in platform)
- SQLite WAL mode; Litestream-ready architecture for future backups

## Architecture

### Technology Stack

- **Backend**: Go 1.22+, Chi router, GORM, modernc.org/sqlite (pure Go, no CGo)
- **Frontend**: Next.js 14+ (App Router, RSC, TypeScript), shadcn/ui, Tailwind CSS
- **Auth**: Clerk (JWT validation only)
- **Database**: SQLite (WAL mode, FTS5, JSON + generated columns for hot paths)
- **Real-time**: nhooyr.io/websocket
- **Email**: Resend + React Email templates
- **Billing**: FlexPoint (behind BillingProvider interface)
- **Observability**: slog + OpenTelemetry (traces, metrics)
- **Testing**: Go: testify + httptest | Frontend: Vitest + Playwright
- **CI/CD**: GitHub Actions
- **Deployment**: Docker Compose (local), Vercel (frontend prod), Fly.io (backend prod, persistent volume)

### Repository Structure

```
/
├── api/                    # Go backend
│   ├── cmd/server/         # Entry point
│   ├── config/             # rbac-policy.yaml, pipeline config
│   ├── internal/           # Domain packages
│   │   ├── org/            # Org handler/service/repository
│   │   ├── space/
│   │   ├── board/
│   │   ├── thread/
│   │   ├── message/
│   │   ├── auth/           # JWT, API key, RBAC engine
│   │   ├── middleware/     # Logging, recovery, CORS, permission
│   │   ├── webhook/
│   │   ├── billing/
│   │   ├── search/
│   │   ├── notification/
│   │   ├── vote/
│   │   ├── moderation/
│   │   ├── upload/
│   │   └── audit/
│   └── pkg/                # Shared utilities (pagination, errors, slugs)
├── web/                    # Next.js frontend
├── docker/                 # Dockerfiles, docker-compose.yml
└── docs/
```

### Key Interfaces (Provider Abstractions)

All external integrations MUST be behind interfaces for testability and swappability:

- `StorageProvider` — file storage (local → S3/R2)
- `BillingProvider` — billing (FlexPoint → Stripe)
- `LLMProvider` — AI enrichment (Grok → OpenAI/Anthropic)
- `NotificationProvider` — notifications (in-app, email → Slack, Teams, SMS)
- `VoiceProvider` — voice support (stub → Bland.ai/Retell/Twilio)

### RBAC Model

Roles (ordered): `viewer` < `commenter` < `contributor` < `moderator` < `admin` < `owner`

Resolution strategy (configurable via `/api/config/rbac-policy.yaml`): **explicit override with parent fallback**. Check board membership → space membership → org membership → no access. Lower-level membership completely replaces inherited role.

### API Design

- Base path: `/v1`
- Error format: RFC 7807 Problem Details (`Content-Type: application/problem+json`)
- Pagination: cursor-based (encoded UUIDv7 timestamps), default 50, max 100
- IDs: UUIDv7 (time-ordered); slugs auto-generated, unique per parent
- Auth: Clerk JWT (`Authorization: Bearer`) or API key (`X-API-Key: deft_live_xxx`)
- Versioning: URL path (`/v1/`, `/v2/`) with deprecation periods

---

## Implementation Plan

### Phase 1: Foundation & Infrastructure

*No dependencies — start here.*

Establish monorepo, Go module, Docker Compose, Taskfile, CI skeleton, config system, HTTP server foundation with Chi, and core helpers.

- **phase1.repo** — Initialize monorepo structure (`/api`, `/web`, `/docker`, `/docs`, `/api/internal/{domain}/`, `/api/config`, `/api/cmd/server`)
- **phase1.gomod** — Initialize Go module with dependencies (chi, gorm, modernc sqlite, testify, nhooyr websocket, otel, google/uuid)
- **phase1.docker** — Docker Compose for local dev (Go API with hot reload, SQLite volume, web placeholder)
- **phase1.taskfile** — Taskfile with `build`, `test`, `test:coverage`, `lint`, `fmt`, `check`, `dev` tasks
- **phase1.ci** — GitHub Actions CI skeleton (run `task check` on push/PR, fail on coverage < 85%)
- **phase1.config** — Environment config loading (server port, SQLite path, Clerk keys, log level, CORS, upload dir/max, OTel endpoint)
- **phase1.rbac-policy** — RBAC policy YAML config (`/api/config/rbac-policy.yaml`): resolution strategy, role hierarchy, defaults
- **phase1.router** — Chi router with middleware stack (slog logging, request ID, panic recovery, CORS, content-type enforcement, `/v1` subrouter)
- **phase1.errors** — RFC 7807 error response helpers (`not_found`, `unauthorized`, `forbidden`, `validation_error`, `conflict`, `internal_error`)
- **phase1.pagination** — Cursor pagination helpers (encode/decode UUIDv7 cursor, parse `?cursor&limit`, default 50 max 100)
- **phase1.health** — Health check endpoints (`GET /healthz`, `GET /readyz` with SQLite connectivity check)
- **phase1.tests** — Unit tests for config, RBAC policy, RFC 7807, pagination, health. Fuzzing ≥50 per input (cursor decode, config). **Live API tests**: start real server, `curl` health endpoints, verify 404 returns RFC 7807, verify CORS/request-ID headers. Coverage ≥85%.

Parallelism: `phase1.repo` and `phase1.docker` can start in parallel. `phase1.gomod` depends on `phase1.repo`. `phase1.taskfile` and `phase1.ci` depend on `phase1.gomod`. All others depend on `phase1.gomod`.

---

### Phase 2: Database & Models (depends on: Phase 1)

Define all GORM models, SQLite schema with JSON metadata, generated columns, FTS5 virtual tables, and all supporting models.

- **phase2.base-model** — BaseModel: UUIDv7 PK, CreatedAt, UpdatedAt, DeletedAt (soft delete). GORM hooks for auto UUIDv7.
- **phase2.org** — Org model: Name, Slug (unique), Description, Metadata JSON, generated columns (`billing_tier`, `payment_status`)
- **phase2.space** — Space model: OrgID FK, Name, Slug (unique within org), Description, Metadata, Type enum (`general`/`crm`/`support`/`community`/`knowledge_base`)
- **phase2.board** — Board model: SpaceID FK, Name, Slug (unique within space), Description, Metadata, IsLocked
- **phase2.thread** — Thread model: BoardID FK, Title, Body (markdown), Slug, Metadata JSON, AuthorID, IsPinned, IsLocked, VoteScore. Generated columns: `status`, `priority`, `stage`, `assigned_to`.
- **phase2.message** — Message model: ThreadID FK, Body (markdown), AuthorID, Metadata JSON, Type enum (`note`/`email`/`call_log`/`comment`/`system`)
- **phase2.memberships** — Membership models: OrgMembership, SpaceMembership, BoardMembership. Role enum (`viewer`→`owner`). Composite unique (`UserID`, `EntityID`).
- **phase2.api-key** — API key model: OrgID FK, Name, KeyHash (SHA-256), KeyPrefix, Permissions JSON, LastUsedAt, ExpiresAt
- **phase2.audit** — Audit log model: UserID, Action enum, EntityType, EntityID, BeforeState JSON, AfterState JSON, IPAddress, RequestID. Immutable.
- **phase2.revision** — Revision model: EntityType, EntityID, Version (auto-increment per entity), PreviousContent JSON, EditorID
- **phase2.webhook** — Webhook models: Subscription (polymorphic scope, URL, Secret encrypted, EventFilter JSON). Delivery (SubscriptionID, EventType, Payload, StatusCode, Attempts, NextRetryAt).
- **phase2.notification** — Notification models: Notification (UserID, Type, Title, Body, EntityRef, IsRead). NotificationPreference (per channel per event). DigestSchedule.
- **phase2.vote** — Vote model: ThreadID, UserID, Weight (computed). Unique (ThreadID, UserID). Thread.VoteScore aggregated.
- **phase2.upload** — Upload model: OrgID, EntityType, EntityID, Filename, ContentType, Size, StoragePath, UploaderID
- **phase2.fts5** — FTS5 virtual tables for orgs, spaces, boards, threads, messages with sync triggers on insert/update/delete
- **phase2.indexes** — Database indexes: FKs, slugs, generated columns, membership composites, audit entity refs, webhook delivery, notifications
- **phase2.tests** — All model CRUD, FTS5 sync, fuzzing ≥50 per (metadata JSON, slug generation, UUIDv7). **Live API tests**: start server, verify DB migrations ran (health check returns 200 with valid SQLite), verify FTS5 tables exist via search endpoint stub. Coverage ≥85%.

Parallelism: `phase2.org` through `phase2.upload` can proceed in parallel after `phase2.base-model`. `phase2.fts5` and `phase2.indexes` depend on all models.

---

### Phase 3: Authentication & Authorization (depends on: Phase 2)

Clerk JWT validation, RBAC engine with configurable policy-based resolution, API key authentication, dual-auth middleware.

- **phase3.jwt** — Clerk JWT validation middleware: JWKS fetch/cache, signature/expiry/issuer validation, `user_id` extraction to context
- **phase3.rbac-engine** — RBAC resolution engine: load policy from YAML, resolve effective role (board → space → org fallback), role hierarchy permissions
- **phase3.permission-mw** — Permission check middleware: extract entity IDs from URL, resolve role, check required permission, 403 RFC 7807 on denial
- **phase3.api-key-auth** — API key CRUD (create/list/revoke) and auth middleware (`X-API-Key` header, hash lookup, permission/expiry check)
- **phase3.dual-auth** — Dual-auth middleware: accept either Clerk JWT or API key, set unified user context
- **phase3.tests** — JWT (valid/expired/wrong issuer/malformed), RBAC (all resolution paths), permissions (every endpoint × role), API keys. Fuzzing ≥50 per (JWT, API key). **Live API tests**: start server, send requests with valid/expired/missing JWT, verify 401/403 responses are RFC 7807, verify API key auth via `X-API-Key` header, verify CORS preflight with `Authorization`. Coverage ≥85%.

Sequential: `phase3.jwt` → `phase3.rbac-engine` → `phase3.permission-mw` → `phase3.api-key-auth` → `phase3.dual-auth`.

---

### Phase 4: Core Spaces API CRUD (depends on: Phase 3)

Full CRUD for the hierarchical model. Each domain follows handler → service → repository pattern within `/api/internal/{domain}/`.

- **phase4.org** — Org repository/service/handlers: `POST/GET/PATCH/DELETE /v1/orgs`. Slug+ID URLs, metadata deep-merge PATCH, cursor pagination, RBAC.
- **phase4.space** — Space CRUD nested under org: `/v1/orgs/{org}/spaces`. Slug unique within org. Type enum.
- **phase4.board** — Board CRUD nested under space. Lock/unlock endpoints. Locked boards reject new threads.
- **phase4.thread** — Thread CRUD: metadata filtering (`?metadata[status]=open&metadata[priority][gt]=3`), deep-merge PATCH, pin/unpin, lock/unlock, revision on update.
- **phase4.message** — Message CRUD nested under thread. Author-only update (creates revision). Type enum. Locked threads reject create.
- **phase4.membership** — Membership CRUD at org/space/board levels. Add/remove/update role. List members. Invitation flow. Cannot remove last owner.
- **phase4.tests** — Full lifecycle integration (org→space→board→thread→message), RBAC every endpoint × role, metadata filtering, pagination, slug, soft delete. Fuzzing ≥50 per (request body, metadata filters). **Live API tests**: start server, create full hierarchy via real HTTP (POST org → POST space → POST board → POST thread → POST message), verify each response status/body/headers, verify metadata filter queries return correct results, verify pagination cursor continuity across real requests, verify slug-based URLs resolve, verify soft-deleted entities return 404. Coverage ≥85%.

Sequential: `phase4.org` → `phase4.space` → `phase4.board` → `phase4.thread` → `phase4.message`. `phase4.membership` can parallel with `phase4.space` onward.

---

### Phase 5: Advanced API Features (depends on: Phase 4)

Search, file uploads, webhook system, audit logging, revision history.

- **phase5.search** — `GET /v1/search` with FTS5 + metadata filters, RBAC filtering, scope/type filters, ranked results with snippets, cursor pagination.
- **phase5.storage** — `StorageProvider` interface (`Store`/`Get`/`Delete`) + `LocalStorage` implementation. File type + size (100MB) validation.
- **phase5.upload-api** — Upload endpoints: `POST /v1/uploads` (multipart), `GET /v1/uploads/{id}`, `DELETE`. RBAC on parent entity.
- **phase5.event-bus** — Internal event bus (in-process pub/sub). All mutations publish typed events. Webhook + notification engines subscribe.
- **phase5.webhook-sub** — Webhook subscription CRUD at org/space/board scope. URL + encrypted secret + event filter. HMAC-SHA256.
- **phase5.webhook-delivery** — Webhook delivery engine: match subs, sign HMAC, POST, retry 3x exponential backoff, log all attempts.
- **phase5.webhook-dashboard** — Webhook delivery dashboard API (`GET` deliveries) and manual replay endpoint.
- **phase5.audit** — Audit middleware: capture before/after state on every mutation, async write. `GET /v1/orgs/{org}/audit-log` with filters.
- **phase5.revisions** — Revision history endpoints: `GET` revisions list and specific version for threads and messages.
- **phase5.tests** — Search integration, upload lifecycle, webhook e2e (create sub → event → verify delivery), audit captures all mutations. Fuzzing ≥50 per (search query, webhook URL, file type). **Live API tests**: start server, upload a real file via multipart POST, download it via GET and verify content matches, search via real GET with query params and verify ranked results, register a webhook subscription and trigger an event — verify delivery hits a local test HTTP server with valid HMAC signature, query audit log and verify entries match mutations performed. Coverage ≥85%.

Parallelism: `phase5.search`, `phase5.storage`+`phase5.upload-api`, `phase5.event-bus`+`phase5.webhook-*`, `phase5.audit`, `phase5.revisions` can all proceed in parallel.

---

### Phase 6: Real-time & Notifications (depends on: Phase 4; informed by: Phase 5)

WebSocket hub, notification provider abstraction, in-app + email + digest implementations.

- **phase6.ws-hub** — WebSocket upgrade (`GET /v1/ws`), JWT auth, hub/channel manager (`board:{id}`, `thread:{id}`), RBAC on subscribe, ping/pong keepalive.
- **phase6.ws-broadcast** — Event broadcasting: subscribe to event bus, broadcast `message.created`/`thread.updated`/`typing` to scoped channels, permission filtered.
- **phase6.notif-interface** — `NotificationProvider` interface + InApp implementation: DB storage, WS push, CRUD endpoints (list/mark-read/mark-all-read).
- **phase6.email** — Email provider (Resend): transactional templates (React Email) for new message, @mention, stage change, assignment, invite. User preferences.
- **phase6.digest** — Digest email: background goroutine, aggregate unreads, render summary, send via Resend. Daily/weekly configurable.
- **phase6.trigger** — Notification trigger engine: event bus → map to notification types → determine recipients (@mentions, thread participants) → route per preferences.
- **phase6.tests** — WS lifecycle integration, notification routing all event types, digest generation, mock provider. Fuzzing ≥50 per (WS message, notification payload). **Live API tests**: start server, open a real WebSocket connection to `ws://localhost:{port}/v1/ws`, authenticate, subscribe to a channel, create a message via POST, verify the WS client receives the broadcast event, verify GET /v1/notifications returns the in-app notification. Coverage ≥85%.

Sequential: `phase6.ws-hub` → `phase6.ws-broadcast` → `phase6.notif-interface` → `phase6.email` → `phase6.digest` → `phase6.trigger`.

---

### Phase 7: CRM Application Layer (depends on: Phase 5, Phase 6)

Sales pipeline, lead scoring, LLM enrichment, automated provisioning.

- **phase7.pipeline** — Pipeline config: default stages (`new_lead`→`closed_won`/`closed_lost` + `nurturing`), transition validation, per-org customization, stage change events.
- **phase7.scoring** — Rule-based scoring engine: config-driven rules (metadata conditions → points), score on metadata change, per-org rules.
- **phase7.llm** — LLM enrichment: `LLMProvider` interface (`Summarize`, `SuggestNextAction`), `GrokProvider` impl, `POST .../threads/{thread}/enrich`, results in metadata.
- **phase7.provision** — Automated provisioning on `closed_won`: create Org, default Spaces/Boards, Clerk invite, CRM thread link, FlexPoint customer. Transaction with rollback. Confirmation message.
- **phase7.tests** — Full sales lifecycle integration (lead→score→enrich→close→provision), LLM mocked. Fuzzing ≥50 per (stage transitions, scoring). **Live API tests**: start server, create a lead thread via POST, PATCH metadata through each pipeline stage, verify score updates in GET response, POST to enrich endpoint (LLM mocked), PATCH to closed_won — then verify customer Org was provisioned by GET-ing the new org/spaces/boards, verify CRM thread metadata contains `customer_org_id`. Coverage ≥85%.

Sequential: `phase7.pipeline` → `phase7.scoring` → `phase7.llm` → `phase7.provision`.

---

### Phase 8: Billing Module (depends on: Phase 4; informed by: Phase 5, Phase 7)

- **phase8.billing** — `BillingProvider` interface (`CreateCustomer`, `CreateInvoice`, `GetPaymentStatus`, `HandleWebhook`) + FlexPoint implementation. Billing metadata on Org. Webhook endpoint for FlexPoint events.
- **phase8.tests** — Billing lifecycle integration, FlexPoint mocked, webhook handling, metadata updates. **Live API tests**: start server, simulate FlexPoint webhook POST to billing endpoint, verify org metadata updated with payment status, verify billing metadata visible in GET /v1/orgs/{org} response. Coverage ≥85%.

---

### Phase 9: Community Features (depends on: Phase 5)

- **phase9.voting** — Vote CRUD: `POST` toggle, weight from role+billing tier, atomic VoteScore update, sort by votes, configurable weight table.
- **phase9.moderation** — Moderation actions: pin/lock/hide/move/merge threads. Flag system: any user flags, moderator queue (`GET /v1/orgs/{org}/flags`), resolve/dismiss. All audit logged.
- **phase9.tests** — Voting lifecycle, moderation e2e, thread move/merge integrity, concurrent voting, flag workflow. **Live API tests**: start server, POST vote on thread, verify VoteScore in GET response, POST flag, verify GET flag queue returns it, POST resolve, POST move thread to different board — GET thread at new location succeeds and old location returns 404. Coverage ≥85%.

Parallelism: `phase9.voting` and `phase9.moderation` can proceed in parallel.

---

### Phase 10: Voice Stubs & Operations (depends on: Phase 4)

- **phase10.voice** — `VoiceProvider` interface (`LogCall`, `GetTranscript`, `Escalate`) + stub. Call log model. Support thread creation/update on call events.
- **phase10.gdpr** — GDPR: `DELETE /v1/admin/users/{user}/purge` (remove PII, anonymize audit). `GET /v1/admin/users/{user}/export` (JSON archive). `DELETE /v1/admin/orgs/{org}/purge` (cascade). Owner-only.
- **phase10.otel** — OpenTelemetry: traces for HTTP requests + DB queries, custom metrics (request count, latency, WS connections, webhook success rate), configurable exporter.
- **phase10.tests** — GDPR purge across all tables, data export completeness, voice stub, OTel spans. **Live API tests**: start server, create user data (org, threads, messages), GET /v1/admin/users/{user}/export and verify JSON archive contains all created data, DELETE /v1/admin/users/{user}/purge, verify subsequent GETs return 404 and audit log entries are anonymized. Coverage ≥85%.

Parallelism: All three tasks can proceed in parallel.

---

### Phase 11: Frontend Foundation (depends on: ALL backend phases 1–10)

**MUST NOT begin until all backend phases are complete and tested with ≥85% coverage.**

- **phase11.nextjs** — Initialize Next.js 14+ in `/web`: App Router, strict TypeScript, Tailwind, shadcn/ui, ESLint+Prettier, Vitest (≥85% threshold).
- **phase11.clerk** — Clerk frontend: `@clerk/nextjs`, `ClerkProvider`, middleware for protected routes, sign-in/sign-up pages, JWT forwarded to Go API.
- **phase11.api-client** — Typed API client: RSC direct fetch with JWT, Client Component fetch wrapper for mutations, types matching Go API responses.
- **phase11.theming** — Theming: CSS custom properties, dark/light via Tailwind `dark:` + system pref + manual toggle (localStorage), per-org overrides from org metadata.
- **phase11.layout** — Base layout: sidebar (org/space/board nav), top bar (search, notification bell, user menu, theme toggle), responsive, breadcrumbs.
- **phase11.tests** — Layout, theme toggle, auth redirect components. Vitest + RTL. Coverage ≥85%.

Sequential: `phase11.nextjs` → `phase11.clerk` → `phase11.api-client`. `phase11.theming` can parallel with `phase11.clerk`. `phase11.layout` depends on `phase11.api-client` and `phase11.theming`.

---

### Phase 12: Core UI Views (depends on: Phase 11)

- **phase12.entity-views** — Org/Space/Board list, create, and settings pages with name/description/metadata forms.
- **phase12.thread-views** — Thread list (filtering, sorting, cursor pagination) and detail (title, body, metadata sidebar, message timeline).
- **phase12.editor** — Tiptap editor for messages: GFM, mermaid (live preview), syntax-highlighted code, image upload, raw markdown toggle. Edit creates revision. Revision history viewable.
- **phase12.upload-ui** — File upload UI: drag-drop, progress indicator, client-side type/size validation, image preview, download links.
- **phase12.tests** — Component tests + Playwright E2E (sign in → org → space → board → thread → message → edit). Coverage ≥85%.

Sequential: `phase12.entity-views` → `phase12.thread-views` → `phase12.editor` → `phase12.upload-ui`.

---

### Phase 13: CRM UI Views (depends on: Phase 12)

- **phase13.kanban** — Kanban pipeline: columns = stages, cards = leads with metadata (company, value, assigned_to, score), drag-drop transitions, filters.
- **phase13.lead-detail** — Lead detail: metadata sidebar, activity timeline, score breakdown (contributing rules), AI enrichment section (summary, next action, Enrich button). Dashboard stats.
- **phase13.tests** — Kanban drag-drop, Playwright E2E sales lifecycle (lead→pipeline→close→provision visible). Coverage ≥85%.

---

### Phase 14: Community & Admin Views (depends on: Phase 12)

- **phase14.community** — Voting UI (toggle, weighted score, sort), moderation dashboard (flag queue, resolve/dismiss, pin/lock/hide/move/merge actions).
- **phase14.admin** — Admin views: webhook CRUD + delivery log + replay, billing dashboard (tier, status, invoices), audit log viewer with diff, membership management.
- **phase14.tests** — Component tests + Playwright E2E community flow (feature request → vote → flag → moderate). Coverage ≥85%.

Parallelism: `phase14.community` and `phase14.admin` can proceed in parallel.

---

### Phase 15: Real-time & Notifications UI (depends on: Phase 12)

- **phase15.realtime** — WS client (connect on auth, subscribe to channels), real-time messages in thread view, typing indicators, notification bell + unread badge, feed dropdown, preferences page, digest frequency.
- **phase15.tests** — Notification components, Playwright E2E (post message → verify real-time in another session), preferences save/load. Coverage ≥85%.

---

### Phase 16: Deployment & Final Integration (depends on: Phase 13, Phase 14, Phase 15)

- **phase16.deploy** — Dockerfile (multi-stage Go build), Docker Compose (full stack), `fly.toml` (persistent volume, health checks, scaling), Vercel config, env var documentation.
- **phase16.cicd** — GitHub Actions: PR → Go tests+lint+coverage + Next.js tests+lint+typecheck. Merge → deploy to Fly.io + Vercel. Coverage gates ≥85%. Rollback on failure.
- **phase16.smoke** — Playwright E2E smoke suite: all MVP acceptance criteria (org→thread→message, RBAC, sales flow, provisioning, billing, community). Run against local Docker + production. API/UI < 2s.

---

## Dependency Map

```
Phase 1 (Foundation)
  └─► Phase 2 (Database)
       └─► Phase 3 (Auth)
            └─► Phase 4 (Core API)
                 ├─► Phase 5 (Advanced)  ─► Phase 7 (CRM) ◄─ Phase 6
                 ├─► Phase 6 (Real-time) ─► Phase 7 (CRM)
                 ├─► Phase 8 (Billing)
                 ├─► Phase 9 (Community) ◄─ Phase 5
                 └─► Phase 10 (Operations)
                          │
         ALL (5-10) ──────┘
                 └─► Phase 11 (Frontend Foundation)
                      └─► Phase 12 (Core UI)
                           ├─► Phase 13 (CRM UI)    ─┐
                           ├─► Phase 14 (Admin UI)   ─┤─► Phase 16 (Deploy)
                           └─► Phase 15 (Realtime UI)─┘
```

**Parallel execution opportunities after Phase 4:**
- Phase 5, 6, 8, 9, 10 can all run in parallel
- Phase 13, 14, 15 can all run in parallel after Phase 12

---

## Testing Strategy

### Per-Phase Requirements

Every phase/subphase MUST implement and run tests until they pass before proceeding.

### Test Levels

1. **Unit tests** — All functions/methods/components. Fast, no external dependencies. Testify (Go), Vitest (TS).
2. **Fuzzing** — ≥50 fuzzing tests per input point. Random/malformed inputs. Go `testing.F` or equivalent.
3. **Integration tests** — Full workflows with real SQLite. HTTP lifecycle via `httptest`.
4. **Live API tests** — Start the real compiled server binary on a random port with real config/SQLite. Exercise every endpoint via actual HTTP requests (Go `net/http` client or `curl`-style calls). Verify response status, headers (`Content-Type`, `CORS`, `X-Request-ID`), body schema, pagination cursors, error format (RFC 7807). Tear down after. These MUST run at the end of every backend phase (1–10), not just Phase 16.
5. **E2E tests** — Playwright for frontend flows and smoke tests.
6. **Smoke tests** — MVP acceptance criteria against deployed environment.
7. **Gate tests** — CI gates: coverage ≥85%, all tests green, lint clean.

### Live API Test Requirements (Phases 1–10)

Every backend phase test gate MUST include a live API test suite that:

- Compiles and starts the real server binary (`go build ./cmd/server && ./server`)
- Uses a temporary SQLite file (cleaned up after)
- Makes real HTTP requests over TCP to `localhost:{random_port}`
- Validates the full request/response cycle including middleware (CORS, auth, logging, request ID, content-type enforcement)
- Tests both happy paths and error paths (malformed input, unauthorized, not found)
- Verifies response headers, not just body (e.g., `Content-Type: application/problem+json` on errors, `X-Request-ID` present, CORS headers on preflight)
- For authenticated endpoints: obtains a real or mock JWT and sends it in `Authorization: Bearer` header; also tests API key auth via `X-API-Key`
- Runs as part of `task test` alongside unit/integration tests
- Managed via Go `TestMain` (start server in setup, shut down in teardown) or a dedicated `_live_test.go` file per phase

### Coverage Requirements

- ≥85% lines, functions, branches, statements
- Exclude: entry points (`main`), generated code, test files
- CI MUST block merge on coverage drop

### Security Tests

- SQL injection, XSS, auth bypass, path traversal for all untrusted input handlers
- RBAC verification: unauthorized access returns 403 for every endpoint

---

## Deployment

### Local Development

- Docker Compose: Go API (hot reload) + SQLite (volume mount) + Next.js dev server
- `task dev` starts full local stack

### Production

- **Frontend**: Vercel (automatic deploy on merge to main)
- **Backend**: Fly.io with persistent volume for SQLite, health checks at `/healthz` and `/readyz`
- **Backups**: Deferred; SQLite WAL mode enabled, Litestream integration point documented

### CI/CD (GitHub Actions)

- **On PR**: Go lint + test + coverage + Next.js lint + typecheck + test + coverage
- **On merge to main**: Deploy Go to Fly.io, deploy frontend to Vercel
- **Gates**: ≥85% coverage, all tests green, lint clean

---

## MVP Acceptance Criteria

1. Create Org → Space → Board → Thread → Message via API and UI
2. Clerk login → user sees only permitted content (RBAC enforced)
3. Full sales flow: lead Thread → opportunity → Closed-Won → customer Org provisioned automatically
4. Post-conversion: invoice metadata, voice stub interaction logged as Message
5. Dashboard views for sales pipeline (Kanban) and community activity
6. Real-time message updates via WebSocket
7. Notification bell with unread count, email notifications, digest emails
8. Search across all levels with metadata filtering
9. API/UI response < 2s
10. ≥85% test coverage across all code

---

*Generated from `vbrief/specification.vbrief.json` — Do not edit directly.*
