Implement IO Phase 6 (Channel Admin UI) of the DEFT Evolution CRM project (abraderAI/crm-project).

START BY reading AGENTS.md in the repo root — it contains all required framework reading and
coding standards you MUST follow before writing any code.

---

## CONTEXT

The IO Channels add-on module lives on branch `feat/io-phase1-channel-gateway`.
Phases 1–5 are already merged into that branch:

- Phase 1: Channel Gateway & Infrastructure (ChannelConfig model, DeadLetterEvent model,
  Repository, Service, Handler, Gateway, RetryEngine with exponential backoff)
- Phase 2: Inbound Email (MIME parsing, thread deduplication, lead-thread creation)
- Phase 3: Voice / LiveKit (room management, webhook, phone bridge, transcripts)
- Phase 4: AI Web Chat Widget (embeddable IIFE bundle, JWT session, WebSocket streaming)
- Phase 5: Agentic CLI (`deft` binary with LLM function-calling)

**YOUR WORKING BRANCH**: check out `feat/io-phase1-channel-gateway`, then create
`feat/io-phase6-channel-admin-ui` from it.

---

## BACKEND API (already implemented — do not modify)

Routes registered at `GET/PUT /v1/orgs/{org}/channels/{type}`:
  GET  /v1/orgs/{org}/channels/{type}               → GetConfig
  PUT  /v1/orgs/{org}/channels/{type}               → PutConfig
  GET  /v1/orgs/{org}/channels/{type}/health        → GetHealth
  GET  /v1/orgs/{org}/channels/{type}/dlq           → ListDLQ  (?status=&cursor=&limit=)
  POST /v1/orgs/{org}/channels/{type}/dlq/{id}/retry    → RetryDLQ
  POST /v1/orgs/{org}/channels/{type}/dlq/{id}/dismiss  → DismissDLQ

Models (api/internal/models/channel-config.go):
  ChannelConfig  { id, org_id, channel_type, settings (JSON string), enabled }
  DeadLetterEvent { id, org_id, channel_type, event_payload, error_message,
                    attempts, last_attempt_at, status (failed|retrying|resolved|dismissed) }
  ChannelHealth   { channel_type, enabled, last_event_at, error_rate, status }

Channel types: "email" | "voice" | "chat"

---

## YOUR TASK

Implement the Channel Admin UI in the Next.js frontend (`web/`).
Match all existing patterns in `web/src/components/admin/` and `web/src/app/admin/`.

### 1. Types (`web/src/lib/api-types.ts`)
Add TypeScript interfaces:
  ChannelConfig, ChannelHealth, DeadLetterEvent, DLQStatus, ChannelType

### 2. API client functions (`web/src/lib/api.ts` or equivalent)
  getChannelConfig(org, type)    → GET /v1/orgs/{org}/channels/{type}
  putChannelConfig(org, type, body)
  getChannelHealth(org, type)    → GET /v1/orgs/{org}/channels/{type}/health
  listDLQEvents(org, type, params)
  retryDLQEvent(org, type, id)
  dismissDLQEvent(org, type, id)

### 3. Components (`web/src/components/admin/`)
Follow the pattern of `webhook-manager.tsx` / `billing-dashboard.tsx` exactly:
  - `"use client"` directive
  - Named exports with explicit PropTypes interface
  - shadcn/ui components (Card, Table, Badge, Button, Input, Select, Dialog, Form)
  - Tailwind CSS only — no inline styles

**channel-overview.tsx**
  - Grid of 3 cards, one per channel type (Email, Voice, Chat)
  - Each card shows: channel name, enabled toggle (Badge), health status badge
    (green=healthy / yellow=degraded / red=error / grey=unconfigured),
    last event time, error rate
  - "Configure" button → links to /admin/channels/[type]
  - "View DLQ" button with red badge count when failed events > 0

**channel-config-form.tsx**
  - Props: channelType, initialConfig, onSave
  - Dynamic fields per channel type:
      email: imap_host, imap_port, imap_user, imap_password (masked), mailbox
      voice: livekit_url, livekit_api_key, livekit_api_secret (masked), webhook_token (masked)
      chat:  jwt_secret (masked), allowed_origins, max_session_minutes
  - Masked fields: display "••••••••" for existing secrets; show a separate
    "Update Secret" input that only sends a new value when non-empty
  - Enabled toggle switch
  - Save / Reset buttons with loading state

**dlq-monitor.tsx**
  - Props: org, channelType, events, loading, onRetry, onDismiss, onRefresh
  - Table columns: Created, Error Message (truncated), Attempts, Last Attempt, Status, Actions
  - Status filter dropdown (all / failed / retrying / resolved / dismissed)
  - Row actions: "Retry" button (disabled when status=resolved|dismissed),
    "Dismiss" button with confirmation dialog
  - Empty state message when no events
  - Auto-refresh every 30 s via useEffect + setInterval; show "Last refreshed" timestamp

**channel-health-badge.tsx**
  - Small reusable badge component: colour-coded by health status string

### 4. Pages (`web/src/app/admin/channels/`)

**page.tsx** — /admin/channels
  - Server component that fetches health for all 3 channel types in parallel
  - Renders <ChannelOverview /> with the fetched data

**[type]/page.tsx** — /admin/channels/[type]
  - Server component; validates `type` param is email|voice|chat (404 otherwise)
  - Fetches config + health
  - Renders <ChannelConfigForm /> for config editing
  - Renders <DLQMonitor /> below the form (client component with its own fetch)
  - Breadcrumb: Admin → Channels → [Type]

### 5. Admin sidebar navigation
Add "Channels" link to the admin sidebar in `web/src/components/layout/` or wherever
the admin nav is defined. Place it between "Webhooks" and "Audit Log".

---

## TESTING REQUIREMENTS

- Vitest component tests for every new component (≥85% coverage):
  - channel-overview.test.tsx
  - channel-config-form.test.tsx
  - dlq-monitor.test.tsx
  - channel-health-badge.test.tsx
- Test happy path + loading state + empty state + error state for each component
- Add one Playwright E2E test in `web/e2e/`:
  - Navigate to /admin/channels
  - Verify all 3 channel type cards render
  - Click "Configure" on email card → verify config form renders

---

## REQUIREMENTS CHECKLIST

- [ ] Read AGENTS.md and all deft files it references BEFORE writing any code
- [ ] Branch `feat/io-phase6-channel-admin-ui` created from `feat/io-phase1-channel-gateway`
- [ ] All components follow existing `web/src/components/admin/` patterns
- [ ] TypeScript strict mode — no `any` types
- [ ] File names use hyphens (not underscores)
- [ ] Conventional commits format for all commits
- [ ] `task check` MUST fully pass before creating the PR
- [ ] PR targets `feat/io-phase1-channel-gateway`
- [ ] PR title: `feat(io): add channel admin UI (Phase 6)`
