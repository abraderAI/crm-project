# DEFT Evolution — IO Channels Add-on Module

*Generated from `vbrief/specification-io-channels.vbrief.json` — 2026-03-14*

## Overview

IO Channels is an add-on module for DEFT Evolution that adds four communication capabilities: (1) inbound email-to-lead conversion via Gmail IMAP IDLE, (2) AI-powered voice calls via LiveKit Agents with LiveKit Phone Numbers, (3) AI web chat via an embeddable widget, and (4) an agentic AI CLI for natural language CRM operations. All inbound channels feed through a unified ChannelGateway service that normalizes events into a common InboundEvent structure before routing to the CRM's thread/message model.

This add-on MUST NOT begin until the main spec's backend phases (1–10) are complete. It replaces the stubbed `VoiceProvider` interface from Phase 10 of the main spec with a full LiveKit-based implementation and adds email, chat, and CLI channels. All channels leverage existing provider interfaces (`StorageProvider`, `LLMProvider`, `NotificationProvider`) and the thread/message content model.

## Requirements

### Functional

- Unified `ChannelGateway` service MUST normalize all inbound events into a common `InboundEvent` structure
- Inbound email via Gmail IMAP IDLE MUST create lead threads or match existing threads via In-Reply-To/References headers
- Voice calls via LiveKit MUST provide AI-first interaction with human escalation
- AI web chat MUST be delivered as an embeddable `<script>` tag with anonymous session tokens and browser fingerprinting
- Agentic AI CLI MUST support natural language queries via LLM function calling against the existing REST API
- Per-org channel configuration MUST be stored in a `channel_configs` table with polymorphic JSONB settings
- Email attachments MUST be stored via existing `StorageProvider` and linked as `Upload` records
- Voice call recordings MUST be exported to `StorageProvider` with transcripts stored as thread messages
- Chat widget MUST support lead capture when visitors provide contact information
- CLI MUST support read-heavy CRUD and search operations (MUST NOT include admin ops)

### Non-Functional

- ≥85% test coverage on all new code (matching main spec requirement)
- External services MUST be behind interfaces with mock implementations for unit tests
- Integration tests MUST use `//go:build integration` tags for real-service testing
- Exponential backoff + dead letter queue for all channel error handling
- Channel health tracking per org (healthy/degraded/down)
- ≥50 fuzzing tests per input point

## Architecture

### Technology Stack Additions

- **Voice Orchestration**: LiveKit Agents SDK (Node.js/TypeScript sidecar)
- **Voice Infrastructure**: LiveKit Cloud + LiveKit Phone Numbers (inbound US, no Twilio)
- **Voice Models**: LiveKit Inference (provider-swappable STT/TTS via config)
- **Voice Go SDK**: `github.com/livekit/server-sdk-go` (room management, phone provisioning)
- **Email Inbound**: Gmail IMAP IDLE (`go-imap` library)
- **Email Auth**: Google OAuth 2.0 Service Account
- **Chat Widget**: Embeddable `<script>` tag (TypeScript → single IIFE JS bundle, Shadow DOM)
- **Chat Fingerprint**: FingerprintJS (browser fingerprinting for returning visitors)
- **CLI**: Go binary (`cmd/cli/`), Cobra, LLM function calling, Lipgloss output

### Repository Structure Additions

```
/
├── api/
│   ├── cmd/cli/              # Agentic AI CLI binary
│   └── internal/
│       ├── channel/          # ChannelGateway, InboundEvent, channel_configs
│       │   ├── gateway/      # Gateway service, retry engine, DLQ
│       │   ├── email/        # Gmail IMAP provider, parser, thread matcher
│       │   ├── voice/        # LiveKit provider, call lifecycle, recording
│       │   └── chat/         # Chat session auth, AI responder, escalation
│       └── ...
├── agent/                    # LiveKit Agent sidecar (Node.js/TypeScript)
│   ├── src/
│   │   ├── agent.ts          # AgentSession, voice pipeline
│   │   ├── tools.ts          # CRM data lookup functions
│   │   └── config.ts         # LiveKit + CRM bridge config
│   ├── package.json
│   └── tsconfig.json
├── widget/                   # Embeddable chat widget
│   ├── src/
│   │   ├── widget.ts         # Main widget entry (IIFE bundle)
│   │   ├── chat.ts           # WebSocket chat client
│   │   ├── fingerprint.ts    # FingerprintJS integration
│   │   └── ui.ts             # Shadow DOM UI components
│   ├── package.json
│   └── tsconfig.json
└── ...
```

### New Interfaces (Provider Abstractions)

All new external integrations MUST be behind interfaces:

- `IMAPProvider` — IMAP connection (Gmail → future providers)
- `LiveKitProvider` — LiveKit room/phone/recording management
- `FingerprintProvider` — browser fingerprint hashing (FingerprintJS → alternatives)

### New Models

- `ChannelConfig` — OrgID, ChannelType enum (`email`/`voice`/`chat`), Settings JSONB, Enabled bool
- `DeadLetterEvent` — OrgID, ChannelType, EventPayload JSONB, ErrorMessage, Attempts, Status enum

### Key Data Flows

**Inbound Email:**
```
Gmail IMAP IDLE → new mail notification → fetch message → parse MIME →
normalize to InboundEvent → ChannelGateway → match thread (headers/email) or create lead →
create message (type: email) → extract attachments → StorageProvider → event bus
```

**Inbound Voice Call:**
```
Phone call → LiveKit Phone Number → dispatch rule → LiveKit Agent room →
Agent sidecar (STT→LLM→TTS) → LiveKit webhook → Go backend →
create thread + call_log message → on call end: recording → StorageProvider,
transcript → thread message → event bus
```

**Web Chat:**
```
Visitor loads page → widget <script> → POST /v1/chat/session (embed key + fingerprint) →
JWT → WebSocket connect → chat messages → LLMProvider generates response →
stream tokens via WS → store messages on thread → lead capture on contact info
```

**CLI Query:**
```
User: "show me all leads from this week" → LLM function calling →
search_leads(created_after: "2026-03-08") → REST API call → format table output
```

### API Endpoints (New)

**Channel Config:**
- `GET /v1/orgs/{org}/channels/{type}` — Get channel config
- `PUT /v1/orgs/{org}/channels/{type}` — Update channel config (admin-only)

**Dead Letter Queue:**
- `GET /v1/orgs/{org}/channels/dlq` — List failed events
- `POST /v1/orgs/{org}/channels/dlq/{id}/retry` — Retry failed event
- `POST /v1/orgs/{org}/channels/dlq/{id}/dismiss` — Dismiss failed event

**Channel Health:**
- `GET /v1/orgs/{org}/channels/health` — Per-channel status

**Chat:**
- `POST /v1/chat/session` — Create anonymous chat session (returns JWT)

**Voice Phone Numbers:**
- `GET /v1/orgs/{org}/channels/voice/numbers` — List org's numbers
- `POST /v1/orgs/{org}/channels/voice/numbers/search` — Search available
- `POST /v1/orgs/{org}/channels/voice/numbers/purchase` — Purchase number

**Internal (Agent Bridge):**
- `GET /v1/internal/contacts/lookup` — Contact lookup by email/phone
- `GET /v1/internal/threads/{id}/summary` — Thread summary for agent context

---

## Implementation Plan

### Phase 1: Channel Gateway & Infrastructure (depends on: main spec Phases 1–10)

*No IO channel dependencies — start here after main spec backend is complete.*

Establish the unified ChannelGateway service, `channel_configs` table, InboundEvent normalization, error handling with exponential backoff and dead letter queue, and channel health tracking.

- **io-phase1.channel-config-model** — `channel_configs` model: OrgID FK, ChannelType enum (`email`/`voice`/`chat`), Settings JSONB, Enabled bool, CreatedAt, UpdatedAt, DeletedAt. Composite unique (OrgID, ChannelType). Go structs per channel type for JSONB validation (`EmailConfig`, `VoiceConfig`, `ChatConfig`).
- **io-phase1.channel-config-api** — Channel config CRUD: `GET/PUT /v1/orgs/{org}/channels/{type}`. Admin-only. Validates Settings JSONB against channel-specific Go struct. Returns masked secrets.
- **io-phase1.inbound-event** — `InboundEvent` struct: ID (UUIDv7), ChannelType, OrgID, ExternalID, SenderIdentifier, Subject, Body, Metadata JSONB, Attachments `[]AttachmentRef`, ReceivedAt. Normalization interface per channel.
- **io-phase1.channel-gateway** — `ChannelGateway` service: accepts `InboundEvent` from any channel adapter, resolves org from channel config, matches to existing thread or creates new lead thread, creates message (type per channel: `email`/`call_log`/`comment`), publishes to event bus.
- **io-phase1.dlq-model** — Dead letter queue model: ID, OrgID, ChannelType, EventPayload JSONB, ErrorMessage, Attempts int, LastAttemptAt, Status enum (`failed`/`retrying`/`resolved`/`dismissed`), CreatedAt. Index on (OrgID, Status).
- **io-phase1.retry-engine** — Retry engine: exponential backoff (1s, 2s, 4s, 8s, 16s cap) with jitter. Max 5 retries. On exhaustion, insert into DLQ. Channel health status tracked per org (`healthy`/`degraded`/`down`) based on recent failure rate.
- **io-phase1.dlq-api** — DLQ admin API: `GET /v1/orgs/{org}/channels/dlq` (list, filter by channel/status), `POST .../dlq/{id}/retry`, `POST .../dlq/{id}/dismiss`. Admin-only.
- **io-phase1.health-api** — Channel health API: `GET /v1/orgs/{org}/channels/health` — returns per-channel status, last event time, error count.
- **io-phase1.tests** — channel_configs CRUD, InboundEvent normalization, gateway routing (match existing thread, create new lead), retry engine (verify backoff timing, DLQ insertion after max retries), DLQ API lifecycle. Fuzzing ≥50 per (Settings JSONB, InboundEvent payload). Coverage ≥85%.

Parallelism: `io-phase1.channel-config-model` and `io-phase1.inbound-event` can start in parallel. `io-phase1.dlq-model` can parallel with `io-phase1.channel-gateway`. `io-phase1.channel-config-api` depends on `io-phase1.channel-config-model`. `io-phase1.channel-gateway` depends on `io-phase1.inbound-event` and `io-phase1.channel-config-model`.

---

### Phase 2: Inbound Email Channel (depends on: Phase 1)

Gmail IMAP IDLE integration for real-time inbound email processing. Emails are parsed, matched to existing threads via In-Reply-To/References headers and sender email, attachments extracted to `StorageProvider`, and unmatched emails create new lead threads.

- **io-phase2.imap-provider** — `IMAPProvider` interface: `Connect(config)`, `StartIDLE(mailbox, handler)`, `FetchMessage(uid)`, `Close()`. `GmailIMAPProvider` implementation using `go-imap` library. Connection pooling per org.
- **io-phase2.google-auth** — Google OAuth 2.0 Service Account integration: load credentials from `channel_configs`, obtain IMAP XOAUTH2 token, auto-refresh before expiry. Store refresh token encrypted in `channel_configs` Settings JSONB.
- **io-phase2.idle-manager** — IMAP IDLE connection manager: per-org goroutine, IDLE on INBOX, reconnect with exponential backoff on disconnect, graceful shutdown on org config change or server stop. Health reporting to channel gateway.
- **io-phase2.email-parser** — Email parser: extract From, To, Subject, plain text + HTML body (prefer plain, fallback HTML→text), In-Reply-To, References, Message-ID headers. Handle multipart MIME. Normalize to `InboundEvent`.
- **io-phase2.thread-matcher** — Thread matcher: (1) check In-Reply-To/References against stored Message-IDs on thread metadata, (2) fallback to sender email address match against existing contact/lead threads, (3) no match → create new lead thread with sender email in metadata.
- **io-phase2.attachment-handler** — Attachment handler: extract MIME attachments, upload each to `StorageProvider`, create `Upload` records linked to the message. Size limit per main spec (100MB). Skip inline images unless >10KB.
- **io-phase2.email-metadata** — Email metadata storage: store Message-ID, In-Reply-To, References, original From/To/CC in message metadata JSONB. Store thread-level `email_address` and `message_ids` array for future matching.
- **io-phase2.tests** — Mock `IMAPProvider` with realistic email fixtures (plain, HTML, multipart, attachments, reply chains). Thread matching (by headers, by email, no match → new lead). Attachment extraction lifecycle. IDLE reconnect behavior. Google auth token refresh. Fuzzing ≥50 per (email body, MIME parsing, headers). Integration tests behind `//go:build integration` tag for real Gmail. Coverage ≥85%.

Sequential: `io-phase2.imap-provider` → `io-phase2.google-auth` → `io-phase2.idle-manager`. `io-phase2.email-parser` and `io-phase2.thread-matcher` can parallel after `io-phase2.imap-provider`. `io-phase2.attachment-handler` depends on `io-phase2.email-parser`.

---

### Phase 3: Voice Channel — LiveKit Integration (depends on: Phase 1; informed by: Phase 2)

Full voice call support via LiveKit. The Go backend manages LiveKit rooms and phone number provisioning via LiveKit's Go server SDK. A Node.js/TypeScript sidecar runs the LiveKit Agent with an STT→LLM→TTS voice pipeline. Calls are recorded, transcribed, and stored as thread messages with audio uploads. AI handles calls first with human escalation support.

**Constraint:** LiveKit Phone Numbers currently only supports inbound calling (US local + toll-free). Outbound calling deferred until LiveKit adds support.

- **io-phase3.livekit-go-sdk** — LiveKit Go server SDK integration: room creation, participant management, SIP dispatch rule management, phone number provisioning (search, purchase, assign dispatch rule), recording start/stop. Behind `LiveKitProvider` interface for testability.
- **io-phase3.voice-config** — Voice channel config (in `channel_configs` JSONB): LiveKit API key/secret, project URL, phone number IDs, agent deployment ID, default STT/TTS models, recording enabled flag, escalation phone number.
- **io-phase3.call-lifecycle** — Call lifecycle management: inbound call → LiveKit dispatch rule routes to agent room → Go backend notified via LiveKit webhook → create thread + `call_log` message → update metadata on call events (answered, transferred, ended). Store call duration, participant info.
- **io-phase3.agent-sidecar** — LiveKit Agent sidecar (Node.js/TypeScript): `AgentSession` with configurable STT (via LiveKit Inference), LLM (via CRM's `LLMProvider` API endpoint), TTS (via LiveKit Inference). System prompt loaded from org's voice channel config. Tool functions for CRM data lookup.
- **io-phase3.agent-crm-bridge** — Agent-CRM bridge API: `GET /v1/internal/contacts/lookup?email=&phone=`, `GET /v1/internal/threads/{id}/summary`. Authenticated via internal API key. Agent calls these during conversation for context.
- **io-phase3.escalation** — Human escalation: agent detects escalation intent (keyword or LLM decision) → LiveKit transfers participant to human agent room → Go backend updates thread metadata (`escalated=true`, `escalated_to`, `escalated_at`). WebSocket notification to CRM UI for human agent.
- **io-phase3.recording** — Call recording: LiveKit room-level recording (composite audio). On call end, export recording to `StorageProvider`, create `Upload` record linked to thread. Store recording URL in `call_log` message metadata.
- **io-phase3.transcript** — Transcript storage: LiveKit agent emits transcript events during call. On call end, compile full transcript as a message (type: `call_log`) on the thread. Include speaker labels (agent/caller) and timestamps.
- **io-phase3.phone-admin** — Phone number admin API: `GET/POST /v1/orgs/{org}/channels/voice/numbers/...`. Admin-only. Proxies to LiveKit Phone Number API.
- **io-phase3.tests** — Mock `LiveKitProvider` for all Go-side tests. Agent sidecar tested with LiveKit's console mode (local audio I/O). Call lifecycle (inbound → agent → transcript → recording → thread). Escalation flow. Phone number provisioning. Fuzzing ≥50 per (webhook payloads, transcript events). Integration tests (`//go:build integration`) with real LiveKit Cloud sandbox. Coverage ≥85%.

Parallelism: `io-phase3.livekit-go-sdk` and `io-phase3.agent-sidecar` can start in parallel. `io-phase3.voice-config`, `io-phase3.call-lifecycle`, and `io-phase3.agent-crm-bridge` depend on `io-phase3.livekit-go-sdk`. `io-phase3.escalation`, `io-phase3.recording`, and `io-phase3.transcript` depend on `io-phase3.call-lifecycle` and `io-phase3.agent-sidecar`.

---

### Phase 4: AI Web Chat (depends on: Phase 1; informed by: Phase 2, Phase 3)

Embeddable AI chat widget delivered via `<script>` tag. Anonymous visitors authenticated with session tokens and browser fingerprints for returning visitor recognition. Chat messages routed through the existing WebSocket hub. AI responses generated server-side via the existing `LLMProvider`. Human escalation via WebSocket channel handoff.

- **io-phase4.widget-bundle** — Chat widget: TypeScript bundle compiled to single IIFE JS file. Renders floating chat button → expandable chat panel. Styled with Shadow DOM (no CSS conflicts with host page). Configurable via data attributes on script tag (org embed key, theme, position).
- **io-phase4.session-auth** — Anonymous session auth: widget requests `POST /v1/chat/session` with org embed key → returns short-lived JWT (24h). Embed key validated against `channel_configs`. JWT stored in `localStorage`. Auto-refresh before expiry.
- **io-phase4.fingerprint** — Browser fingerprinting: integrate FingerprintJS in widget. Send fingerprint hash with session request. Server stores fingerprint → `visitor_id` mapping. On returning visit with same fingerprint, merge into existing lead/thread. Privacy notice displayed in widget.
- **io-phase4.chat-ws** — Chat WebSocket integration: widget connects to existing WS hub (`/v1/ws`) with chat session JWT. Subscribes to `chat:{session_id}` channel. Sends/receives chat messages. Server-side handler creates thread (if new session) and messages on the CRM thread.
- **io-phase4.ai-responder** — AI chat responder: on inbound chat message, invoke `LLMProvider` with conversation history + org-specific system prompt (from chat channel config). Stream response tokens back via WebSocket. Store both user and AI messages on the CRM thread.
- **io-phase4.lead-capture** — Lead capture: if visitor provides name/email during chat (detected by LLM or explicit form), update the thread's lead metadata (`contact_email`, `contact_name`). Merge with fingerprint-matched existing lead if applicable.
- **io-phase4.human-escalation** — Human escalation: AI detects escalation intent → server marks thread as escalated → WebSocket notification to CRM UI agent dashboard → human agent joins the WS channel → visitor sees 'connecting to agent' status. If no human available within timeout, AI continues with apology.
- **io-phase4.chat-config** — Chat channel config (in `channel_configs` JSONB): embed key (auto-generated UUID), widget theme (colors, logo URL, greeting message), AI system prompt, escalation timeout seconds, allowed domains (CORS), operating hours.
- **io-phase4.tests** — Widget bundle builds and renders (Vitest + jsdom). Session auth flow (valid/invalid/expired embed key). Fingerprint matching (new visitor, returning visitor, merge). WS chat message round-trip. AI responder with mocked LLM. Lead capture detection. Human escalation flow. Fuzzing ≥50 per (chat message, embed key, fingerprint). Coverage ≥85%.

Parallelism: `io-phase4.widget-bundle` (frontend) and `io-phase4.session-auth` + `io-phase4.fingerprint` (backend) can proceed in parallel. `io-phase4.chat-ws` depends on both. `io-phase4.ai-responder`, `io-phase4.lead-capture`, and `io-phase4.human-escalation` can parallel after `io-phase4.chat-ws`.

---

### Phase 5: Agentic AI CLI (depends on: Phase 1)

Go binary CLI tool (`cmd/cli/`) that provides a natural language interface to the CRM. Uses LLM function calling to translate user queries into REST API calls. Supports read-heavy CRUD and search across all CRM entities. Authenticated via API key or JWT.

**Constraint:** CLI MUST NOT include admin operations (org settings, user management, role assignments, billing). MUST use existing REST API endpoints — no direct database access.

- **io-phase5.cli-scaffold** — CLI scaffold: Go binary in `cmd/cli/` using Cobra. Config file (`~/.deft-cli.yaml`) for API URL, API key, default org. Environment variable overrides (`DEFT_API_URL`, `DEFT_API_KEY`). Version command.
- **io-phase5.auth** — CLI auth: `deft login` command — interactive API key entry (stored securely in OS keychain via `go-keyring`). `deft login --token` for JWT. Validate credentials via `GET /healthz` with auth header. `deft logout` to clear.
- **io-phase5.api-client** — CLI API client: typed Go HTTP client wrapping all REST endpoints used by CLI (leads, contacts, deals, threads, messages, search, activities). Handles pagination, error formatting, auth header injection.
- **io-phase5.llm-functions** — LLM function calling: define tool functions matching CLI operations (`search_leads`, `get_thread`, `create_lead`, `update_deal_stage`, `list_activities`, `search_contacts`, etc.). Send user's natural language query + tools to `LLMProvider`. Execute returned function calls against API client. Format results for terminal output.
- **io-phase5.interactive** — Interactive mode: `deft chat` enters REPL. Maintains conversation context across queries. Supports follow-up queries ("now show me their deals"). Exit with Ctrl+C or `exit`. Non-interactive mode: `deft ask "show me all leads from this week"`.
- **io-phase5.output-format** — Output formatting: table output (default for lists), JSON output (`--json` flag), single-record detail view. Colored terminal output via Lipgloss. Respects `NO_COLOR` env var. Pagination: auto-fetch all pages or `--limit` flag.
- **io-phase5.tests** — CLI scaffold (config loading, auth flow with mocked keyring). API client (all endpoints mocked). LLM function calling (mock LLM returns tool calls, verify correct API calls made). Interactive mode (simulated stdin/stdout). Output formatting. Fuzzing ≥50 per (user query, API response). Coverage ≥85%.

Sequential: `io-phase5.cli-scaffold` → `io-phase5.auth` → `io-phase5.api-client` → `io-phase5.llm-functions` → `io-phase5.interactive`. `io-phase5.output-format` can parallel with `io-phase5.llm-functions`.

---

### Phase 6: Channel Admin UI (depends on: Phase 2, Phase 3, Phase 4)

Frontend views for managing IO channel configurations, monitoring channel health, reviewing the dead letter queue, and configuring/previewing the chat widget. Extends the existing Next.js admin views.

- **io-phase6.channel-settings** — Channel settings page: `/settings/channels` — tabbed view (Email, Voice, Chat, CLI). Per-channel enable/disable toggle. Channel-specific config forms (Gmail OAuth connect flow, LiveKit API keys, chat widget theme editor). Save validates via API.
- **io-phase6.health-dashboard** — Channel health dashboard: `/settings/channels/health` — real-time status per channel (`healthy`/`degraded`/`down`). Last event timestamp. Error count (24h). Sparkline chart of event volume. Auto-refresh via WebSocket.
- **io-phase6.dlq-viewer** — Dead letter queue viewer: `/settings/channels/dlq` — table of failed events (channel, error, timestamp, attempts). Filter by channel/status. Retry and dismiss actions. Event payload detail modal.
- **io-phase6.widget-preview** — Chat widget preview: in chat channel settings, live preview of widget appearance (colors, logo, greeting). Copy embed code snippet. Test chat in preview mode (sends to test thread).
- **io-phase6.phone-management** — Phone number management UI: `/settings/channels/voice/numbers` — list org's numbers, search available numbers by area code, purchase flow, assign to dispatch rule. Status indicators.
- **io-phase6.tests** — Component tests for all settings forms, health dashboard, DLQ viewer. Playwright E2E: navigate to channel settings → configure email → verify saved. Widget preview renders. Coverage ≥85%.

Parallelism: `io-phase6.channel-settings` first, then all other views can proceed in parallel.

---

### Phase 7: Integration Testing & Deployment (depends on: Phase 2, Phase 3, Phase 4, Phase 5, Phase 6)

Cross-channel integration tests, LiveKit Agent sidecar deployment configuration, CI/CD pipeline updates, and end-to-end smoke tests covering all four IO channels.

- **io-phase7.cross-channel** — Cross-channel integration tests: email arrives → creates lead → same lead calls via phone → thread matched → chat from same contact → thread matched. Verify single unified thread with all message types.
- **io-phase7.agent-deploy** — LiveKit Agent sidecar deployment: Dockerfile for Node.js agent. Docker Compose service (agent alongside Go API). Fly.io deployment config (separate process group or machine). Environment variables for LiveKit credentials.
- **io-phase7.ci-update** — CI/CD updates: GitHub Actions workflow adds Node.js agent lint+test step. Widget bundle build step. CLI binary build (multi-arch). Integration test job with `//go:build integration` tag (requires LiveKit sandbox credentials in CI secrets).
- **io-phase7.smoke** — E2E smoke tests: (1) send test email → verify lead thread created with attachments, (2) inbound call → verify transcript + recording on thread, (3) chat widget session → verify AI response + lead capture, (4) CLI `deft ask` → verify correct data returned. Run against local Docker stack.
- **io-phase7.docs** — Documentation: update README with IO channels overview, setup guides (Gmail OAuth, LiveKit Cloud, widget embed), CLI installation instructions. Update `PROJECT.md` with new dependencies.

Parallelism: `io-phase7.agent-deploy` and `io-phase7.ci-update` can proceed in parallel. `io-phase7.smoke` depends on both. `io-phase7.cross-channel` and `io-phase7.docs` can parallel with everything.

---

## Dependency Map

```
Main Spec Phases 1–10 (Backend Complete)
  └─► IO Phase 1 (Channel Gateway & Infrastructure)
       ├─► IO Phase 2 (Inbound Email)        ──┐
       ├─► IO Phase 3 (Voice — LiveKit)       ──┤─► IO Phase 6 (Channel Admin UI)
       ├─► IO Phase 4 (AI Web Chat)           ──┤         │
       └─► IO Phase 5 (Agentic CLI)            │         │
                │                               │         │
                └───────────────────────────────┴─────────┘
                                                │
                                    IO Phase 7 (Integration & Deploy)
```

**Parallel execution opportunities after Phase 1:**
- Phases 2, 3, 4, 5 can ALL run in parallel (maximum 4-agent parallelism)
- Phase 6 starts after Phases 2, 3, 4 complete
- Phase 7 starts after ALL phases complete

---

## Testing Strategy

### Per-Phase Requirements

Every phase MUST implement and run tests until they pass before proceeding. This follows the same testing discipline as the main spec.

### Test Levels

1. **Unit tests** — All functions/methods/components behind mocked interfaces. Testify (Go), Vitest (TS). Fast, no external dependencies.
2. **Fuzzing** — ≥50 fuzzing tests per input point. Random/malformed inputs for: JSONB settings, InboundEvent payloads, email MIME, webhook payloads, chat messages, CLI queries.
3. **Integration tests** — Behind `//go:build integration` tags. Require real external service credentials (Gmail, LiveKit Cloud sandbox). Run in CI with secrets.
4. **Component tests** — Widget bundle (Vitest + jsdom), agent sidecar (LiveKit console mode), CLI (simulated stdin/stdout).
5. **E2E tests** — Playwright for channel admin UI flows.
6. **Cross-channel tests** — Full flow from external event through ChannelGateway to CRM thread.
7. **Gate tests** — CI gates: coverage ≥85%, all tests green, lint clean, typecheck clean.

### Coverage Requirements

- ≥85% lines, functions, branches, statements (all new Go, TypeScript, and widget code)
- CI MUST block merge on coverage drop
- LiveKit agent sidecar: Vitest with mocked LiveKit SDK

### Security Tests

- Chat widget: XSS prevention in Shadow DOM, embed key validation, JWT expiry enforcement
- IMAP: credential encryption at rest, OAuth token secure storage
- CLI: keychain storage for API keys, no plaintext credential logging
- Voice: LiveKit webhook signature validation, internal API key authentication

---

## Deployment

### Local Development

- Docker Compose adds: LiveKit Agent sidecar service (Node.js), widget dev server
- `task dev` starts full local stack including agent sidecar
- Chat widget: `task widget:dev` for hot-reload development
- CLI: `go run ./cmd/cli/` or `task cli:build` for binary

### Production

- **Go API**: Fly.io (existing) — adds channel gateway, email IMAP, voice lifecycle endpoints
- **LiveKit Agent**: Fly.io (separate process group or machine) or LiveKit Cloud managed deployment
- **Chat Widget**: CDN-hosted JS bundle (built in CI, deployed to Fly.io static or CDN)
- **CLI**: GitHub Releases (multi-arch binaries: linux/amd64, darwin/amd64, darwin/arm64)

### CI/CD Additions

- **On PR**: Go lint+test+coverage + Node.js agent lint+test + widget build+test + CLI build
- **On merge**: Deploy Go API, deploy agent sidecar, publish widget bundle, publish CLI binaries
- **Integration tests**: Separate CI job with `//go:build integration` tag, LiveKit sandbox credentials in GitHub secrets

---

## Scope Boundaries

### In Scope

- Inbound email → lead conversion (Gmail IMAP IDLE)
- AI voice calls with LiveKit (inbound only, US numbers)
- AI web chat widget (embeddable, anonymous visitors)
- Agentic AI CLI (natural language CRM queries)
- Unified channel gateway with error handling
- Channel admin UI (config, health, DLQ)

### Out of Scope (Deferred)

- Outbound email sending (use existing Resend/notification system)
- Outbound voice calls (blocked on LiveKit Phone Numbers adding support)
- Video calls
- SMS/WhatsApp channels
- International phone numbers (blocked on LiveKit availability)
- CLI admin operations (org settings, user management, billing)
- Multi-provider email (only Gmail initially; interface supports future providers)

---

## Appendix: Interview Questions and Answers

<details>
<summary>Complete Interview Log (23 Questions)</summary>

**Q1: Gateway Architecture**
A: Option 1 — Unified `ChannelGateway` service with normalized `InboundEvent` objects from all channels

**Q2: Email Provider**
A: Gmail IMAP (not Resend/SendGrid/Mailgun webhooks)

**Q3: Email Polling Strategy**
A: IMAP IDLE (push-like)

**Q4: Voice Provider**
A: Twilio Voice (later revised to LiveKit in Q14)

**Q5: Chat AI Orchestration**
A: Server-side LLM via existing `LLMProvider`

**Q6: Chat Widget Delivery**
A: Embeddable `<script>` tag

**Q7: Chat-to-Human Escalation**
A: Existing WebSocket hub

**Q8: CLI Location**
A: Go binary in same repo (`cmd/cli/`)

**Q9: CLI Natural Language**
A: LLM function calling

**Q10: Email Thread Matching**
A: Email address + In-Reply-To/References headers

**Q11: Call Flow**
A: AI-first with human escalation

**Q12: Gmail Authentication**
A: Google OAuth 2.0 (Service Account)

**Q13: STT/TTS Providers**
A: LiveKit Inference (unified gateway, provider-swappable via config) — revised with Q14

**Q14: Voice Architecture**
A: LiveKit Agents + LiveKit Phone Numbers (all-in LiveKit, no Twilio). Node.js/TypeScript sidecar. Go backend manages CRM integration via LiveKit's Go server SDK. LiveKit Phone Numbers for inbound US telephony. Outbound deferred.

**Q15: Chat Widget Authentication**
A: Anonymous session token + browser fingerprint (FingerprintJS for returning visitor recognition, merged with session token auth)

**Q16: Email Attachment Handling**
A: Store in existing `StorageProvider` + link to thread as `Upload` records

**Q17: Voice Call Recording**
A: Record via LiveKit + store in `StorageProvider`. Audio as `Upload`, transcript as thread message.

**Q18: CLI Scope**
A: Read-heavy CRUD + search (no admin ops)

**Q19: Per-Org Channel Configuration**
A: `channel_configs` table with polymorphic JSONB settings per org+channel_type, validated by Go structs

**Q20: Error Handling**
A: Exponential backoff + dead letter queue (DB-backed, admin UI with retry/dismiss)

**Q21: Testing Strategy**
A: Interface mocks + integration test tags (`//go:build integration` for real-service tests)

**Q22: LiveKit Agent Runtime**
A: Node.js/TypeScript (aligns with existing Next.js frontend tooling)

**Q23: Chat Transport**
A: Text-only via existing WebSocket hub (not LiveKit Rooms)

</details>

---

*Generated from `vbrief/specification-io-channels.vbrief.json` — Do not edit directly.*
