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
- **GDPR** — Hard-purge admin endpoints; soft delete everywhere else
- **Observability** — Structured `slog` logging + OpenTelemetry traces and metrics
- **Voice** — Stubbed `VoiceProvider` interface with thread-based logging and escalation

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
| Observability | slog + OpenTelemetry |
| Testing | Go: testify + httptest · Frontend: Vitest + Playwright |
| CI/CD | GitHub Actions |
| Deployment | Docker Compose (local) · [Fly.io](https://fly.io) (backend) · [Vercel](https://vercel.com) (frontend) |

---

## Repository Structure

```
/
├── api/                        # Go backend
│   ├── cmd/server/             # Entry point
│   ├── config/                 # rbac-policy.yaml, pipeline config
│   ├── internal/
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
│   │   └── server/             # Router configuration
│   └── pkg/                    # Shared: pagination, errors, slugs, response
├── web/                        # Next.js frontend
│   ├── src/
│   │   ├── app/                # App Router pages
│   │   ├── components/
│   │   │   ├── crm/            # Pipeline, kanban, lead views
│   │   │   ├── community/      # Voting, moderation
│   │   │   ├── admin/          # Billing, webhooks, audit log
│   │   │   ├── realtime/       # WebSocket, notifications
│   │   │   ├── threads/        # Thread and message UI
│   │   │   └── upload/         # File upload components
│   │   └── lib/                # API client, types, hooks
│   └── e2e/                    # Playwright smoke tests
├── docker/                     # Dockerfiles + docker-compose.yml
├── docs/                       # Environment variables, architecture notes
├── SPECIFICATION.md            # Full implementation spec
└── Taskfile.yml                # All build/test/lint commands
```

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

**3. Start the backend**

```bash
task dev          # Go API server on :8080
```

**4. Start the frontend**

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
task fmt               # Format Go code (gofmt)
task lint              # Lint Go code (golangci-lint)
task test              # Run all Go tests (race detector)
task test:coverage     # Go tests + enforce ≥85% coverage
task test:fuzz         # Run fuzz tests (5s each)
task check             # Full pre-commit suite (fmt + lint + test + coverage + web checks)

task web:fmt           # Format frontend (Prettier)
task web:fmt:check     # Check frontend formatting
task web:lint          # ESLint
task web:typecheck     # TypeScript type check
task web:test          # Vitest unit tests
task web:test:coverage # Vitest + enforce ≥85% coverage
task web:build         # Next.js production build
task web:e2e           # Playwright E2E tests
task web:e2e:smoke     # Playwright smoke tests only (@smoke tag)
```

---

## API Overview

- **Base path**: `/v1`
- **Auth**: `Authorization: Bearer <clerk-jwt>` or `X-API-Key: deft_live_...`
- **Errors**: [RFC 7807 Problem Details](https://datatracker.ietf.org/doc/html/rfc7807) (`application/problem+json`)
- **Pagination**: Cursor-based (UUIDv7), default 50, max 100
- **IDs**: UUIDv7 (time-ordered); slugs auto-generated, unique per parent

Key endpoints:

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

---

## Deployment

### Backend — Fly.io

```bash
fly deploy          # deploys docker/Dockerfile → deft-evolution-api
```

SQLite is persisted on a Fly volume at `/data/deft.db`. The architecture is Litestream-ready for future streaming backups.

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

### RBAC

Roles are resolved bottom-up: **board membership → space membership → org membership → no access**. A lower-level role completely replaces the inherited one (explicit override). Strategy is configurable in `api/config/rbac-policy.yaml`.

### Database

SQLite in WAL mode with FTS5 virtual tables for full-text search, JSON-extracted generated columns for hot filter paths, and soft delete on all entities. No CGo — uses `modernc.org/sqlite`.

---

## Quality

- ≥ 85% test coverage enforced on every PR
- Race detector enabled on all test runs (`-race`)
- ≥ 50 fuzz test cases per input entry point
- `task check` must pass fully before any merge
- All errors follow RFC 7807 Problem Details

---

## License

MIT
