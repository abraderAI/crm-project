# Sales CRM Leads Interface — SPECIFICATION

> Generated from: `vbrief/specification-sales-crm.vbrief.json` (status: **draft**)
> PRD: `PRD.md` | Source spec: v1.4 (March 2026) | Branch: `feat/sales-crm`

---

## Overview

A purpose-built internal Sales CRM for the DEFT sales team, layered on top of the existing Thread/Message/Space/RBAC architecture with minimal structural disruption.

**Key architectural decisions:**
- Company, Contact, and Opportunity are **Threads** with a `crm_type` metadata field (`company` / `contact` / `opportunity`) in a single CRM-type Space per org, each in their own board (`companies`, `contacts`, `opportunities`)
- Record-level access control via a new **`thread_acl`** table, enforced at the repository query layer — not board/space RBAC
- Cross-entity relationships (Contact↔Company, Opportunity↔Company, Opportunity↔Contact) stored in a **`crm_links`** join table
- Follow-up tasks in a dedicated **`crm_tasks`** table
- Email integration built on the **existing `channel` package** (`EmailInbox` model + `ChannelGateway`). A new `sales_crm` routing action is added; admin registers each rep's `@deft.co` address in channel settings; the rep connects their Gmail credentials via the CRM profile
- All LLM features route through the existing **`LLMProvider`** abstraction (extended with 5 new methods)
- CEO identified by **`DEFT_CEO_USER_ID`** environment variable (no RBAC change)
- Stage probability = default per stage + per-opportunity override
- Custom fields via existing metadata JSON (no admin UI this phase)
- PDF export deferred; CSV export only
- USD only

See `PRD.md` for full functional requirements (FR-1–FR-49) and non-functional requirements (NFR-1–NFR-8).

---

## Architecture

### New Database Tables

**`thread_acl`** — Per-thread visibility grants
```sql
thread_acl (
  id          TEXT PRIMARY KEY,       -- UUIDv7
  thread_id   TEXT NOT NULL,          -- FK → threads.id
  user_id     TEXT NOT NULL,          -- Clerk user ID
  grant_type  TEXT NOT NULL,          -- owner | task_assignee | linked_opportunity_owner
  created_at  DATETIME NOT NULL,
  INDEX idx_thread_acl_user  (user_id, thread_id)
)
```

**`crm_links`** — Typed relationships between CRM entity Threads
```sql
crm_links (
  id              TEXT PRIMARY KEY,   -- UUIDv7
  from_thread_id  TEXT NOT NULL,      -- FK → threads.id
  to_thread_id    TEXT NOT NULL,      -- FK → threads.id
  link_type       TEXT NOT NULL,      -- contact_company | opportunity_company | opportunity_contact
  is_primary      BOOLEAN DEFAULT 0,
  created_at      DATETIME NOT NULL,
  UNIQUE (from_thread_id, to_thread_id, link_type),
  INDEX idx_crm_links_from (from_thread_id, link_type),
  INDEX idx_crm_links_to   (to_thread_id, link_type)
)
```

**`crm_tasks`** — Follow-up tasks
```sql
crm_tasks (
  id           TEXT PRIMARY KEY,      -- UUIDv7
  parent_type  TEXT NOT NULL,         -- company | contact | opportunity
  parent_id    TEXT NOT NULL,         -- Thread ID of parent entity
  title        TEXT NOT NULL,
  description  TEXT,
  assigned_to  TEXT NOT NULL,         -- Clerk user ID
  due_date     DATE,
  priority     TEXT NOT NULL,         -- low | medium | high | urgent
  status       TEXT NOT NULL,         -- open | in-progress | completed | cancelled
  created_by   TEXT NOT NULL,         -- Clerk user ID
  created_at   DATETIME NOT NULL,
  updated_at   DATETIME NOT NULL,
  deleted_at   DATETIME,
  INDEX idx_crm_tasks_assignee (assigned_to, status),
  INDEX idx_crm_tasks_parent   (parent_id),
  INDEX idx_crm_tasks_due      (due_date, status)
)
```

**`crm_import_history`** — CSV import log
```sql
crm_import_history (
  id            TEXT PRIMARY KEY,     -- UUIDv7
  org_id        TEXT NOT NULL,        -- FK → orgs.id
  imported_by   TEXT NOT NULL,        -- Clerk user ID
  entity_type   TEXT NOT NULL,        -- company | contact
  total_rows    INTEGER NOT NULL,
  success_count INTEGER NOT NULL,
  error_count   INTEGER NOT NULL,
  error_detail  TEXT,                 -- JSON array of {row, field, message}
  created_at    DATETIME NOT NULL
)
```

**`crm_unassigned_emails`** — Inbound emails that could not be auto-matched to a CRM entity
```sql
crm_unassigned_emails (
  id               TEXT PRIMARY KEY,  -- UUIDv7
  inbox_id         TEXT NOT NULL,     -- FK → email_inboxes.id
  assigned_user_id TEXT NOT NULL,     -- Clerk user ID of the inbox owner
  gmail_message_id TEXT NOT NULL,     -- Gmail message ID (for idempotency)
  sender_email     TEXT NOT NULL,
  subject          TEXT,
  body_snippet     TEXT,
  received_at      DATETIME NOT NULL,
  created_at       DATETIME NOT NULL,
  UNIQUE (gmail_message_id)
)
```

### EmailInbox Model Extensions

The existing `api/internal/models/email-inbox.go` `EmailInbox` model gains four new **nullable** fields (additive, no breaking changes):

| Field | Type | Purpose |
|---|---|---|
| `AssignedUserID` | `TEXT` | Clerk user ID of the rep who owns this inbox (set by admin; `sales_crm` routing only) |
| `OAuthRefreshToken` | `TEXT` | AES-256-GCM encrypted Gmail OAuth2 refresh token (provided by rep, masked `[REDACTED]` in API responses) |
| `PubSubSubscription` | `TEXT` | Gmail watch push subscription name (managed by system) |
| `LastHistoryID` | `TEXT` | Gmail sync checkpoint (managed by system) |

A new `sales_crm` value is added to the `RoutingAction` enum (`support_ticket` / `sales_lead` / `general` / **`sales_crm`**).

### Pipeline Extension

`api/internal/pipeline/config.go` — `StageInfo` gains `Probability int` (0–100). `DefaultStages()` updated with defaults. `WeightedForecast(amountCents int64, probability int) int64` helper added. Per-opportunity override stored as `metadata.probability_override` (integer) on the Opportunity Thread; takes precedence over stage default when set.

> **Disambiguation — `lead_score` vs `probability_override`:** The existing `api/internal/scoring/` package computes a `metadata.lead_score` (0–100 integer) representing **lead qualification rank** — how hot/qualified a lead is based on rules (stage, priority, company presence, deal value thresholds). This is a different concept from `metadata.probability_override`, which represents **deal close probability** used exclusively for weighted forecast calculations (`amount × probability / 100`). These two integers coexist in Thread metadata with no collision. In the UI, they must be labelled distinctly: **"Lead Score"** (AI-computed qualification, display-only) and **"Close Probability %"** (rep-overridable, used in forecasting).

### New API Packages

All follow the existing `handler → service → repository` pattern under `api/internal/`:

| Package | Responsibility |
|---|---|
| `crm/` | ACL engine, `EnsureCRMSpace`, `crm_links` CRUD, `FilterByACL` scope |
| `crm/company/` | Company repository, service, handler |
| `crm/contact/` | Contact repository, service, handler |
| `crm/opportunity/` | Opportunity repository, service, handler, transition endpoint |
| `crmtask/` | Task CRUD, assignment ACL updates, quality rules, reminder job |
| `crmimport/` | CSV parser, validator, apply (transactional), import history |
| `crmhandoff/` | Closed Won → support ticket creation, notifications |

**Extended packages (no new packages for email or reporting):**
- `api/internal/pipeline/` — `StageInfo.Probability`, `WeightedForecast` helper
- `api/internal/llm/` — 5 new `LLMProvider` interface methods added to existing interface; existing `Summarize`, `SuggestNextAction`, and `/enrich` endpoint preserved unchanged
- `api/internal/channel/` — `sales_crm` routing action, Gmail API + Pub/Sub push handler, OAuth2 flow, `EmailInbox` model extensions
- `api/internal/reporting/` — new CRM report queries added alongside existing sales reports (weighted forecast, activity/task reports, SMTM, ACL-scoped variants)
- `api/internal/search/` — extend existing FTS5 search to include `crm_type`-filtered Thread results and `crm_tasks` table for CRM global search (FR-41)

### New Environment Variables

```
DEFT_CEO_USER_ID          # Clerk user ID of the DEFT CEO
GMAIL_CLIENT_ID           # Google OAuth2 client ID
GMAIL_CLIENT_SECRET       # Google OAuth2 client secret
GMAIL_PUBSUB_PROJECT_ID   # Google Cloud project for Pub/Sub
```
All added to `secrets/secrets.example.env`. OAuth tokens encrypted using the app's existing master secret key — no new key needed.

### Frontend Routes (additions to `/crm/`)

| Route | Purpose |
|---|---|
| `/crm/companies` | Company list |
| `/crm/companies/new` | Create Company |
| `/crm/companies/[id]` | Company detail |
| `/crm/contacts` | Contact list |
| `/crm/contacts/new` | Create Contact |
| `/crm/contacts/[id]` | Contact detail |
| `/crm/pipeline` | Opportunity kanban + list |
| `/crm/pipeline/new` | Create Opportunity |
| `/crm/pipeline/[id]` | Opportunity detail |
| `/crm/inbox` | Unassigned email inbox |
| `/crm/import` | CSV import flow |
| `/reports/sales` | **Existing** sales reports page — extended with new tabs |
| `/reports/sales/show-me-the-money` | CEO executive report (new route added to existing reports section) |

### Dependency Graph

```
Phase 1 (Models)
    └── Phase 2 (Entity CRUD + ACL)
            ├── Phase 3 (Tasks + Quality)      ──→ Phase 5 (LLM)
            ├── Phase 4 (Gmail)                ──→ Phase 5 (LLM)
            ├── Phase 6 (Reports Backend)
            ├── Phase 7 (CSV Import)
            ├── Phase 8 (Closed Won Handoff)
            └── Phase 9 (Frontend — Entities)
                    ├── Phase 10 (Frontend — Email + Tasks)  [needs Phase 4]
                    ├── Phase 11 (Frontend — LLM)            [needs Phase 5]
                    ├── Phase 12 (Frontend — Reports)        [needs Phase 6]
                    └── Phase 13 (Frontend — CSV Import)     [needs Phase 7]
```

Phases 4, 6, 7, 8 may be developed in parallel after Phase 2 completes.

---

## Implementation Plan

---

### Phase 1: Data Models & Infrastructure
*All subsequent phases depend on this phase being complete and tested.*

#### Subphase 1.1: Database Schema
*Dependencies: none*

- **Task 1.1.1:** Define `thread_acl` GORM model with `BaseModel`, fields as specified above, composite index on `(user_id, thread_id)`. (traces: FR-14, NFR-1)
  - Acceptance: AutoMigrates in test DB; index present; unit tests for model CRUD

- **Task 1.1.2:** Define `crm_links` GORM model with `BaseModel`, `from_thread_id`, `to_thread_id`, `link_type` enum, `is_primary` bool, unique constraint on `(from_thread_id, to_thread_id, link_type)`. (traces: FR-2)
  - Acceptance: Unique constraint enforced; all link types unit tested

- **Task 1.1.3:** Define `crm_tasks` GORM model with `BaseModel` and all fields specified. Three indexes as specified. `deleted_at` for soft delete. (traces: FR-20)
  - Acceptance: All status transitions valid; indexes present

- **Task 1.1.4:** Define `crm_import_history` GORM model. `error_detail` stored as JSON text. (traces: FR-19)
  - Acceptance: JSON array stored and retrieved correctly

- **Task 1.1.5:** Define `crm_unassigned_emails` GORM model with unique index on `gmail_message_id`. (traces: FR-29)
  - Acceptance: AutoMigrates; uniqueness constraint prevents duplicate ingestion; unit test for CRUD

- **Task 1.1.6:** Phase 1.1 tests — all model CRUD, constraint enforcement, ACL query construction. Fuzz ≥50 inputs on ACL query construction. `task test:coverage` ≥85%. (traces: NFR-3)

#### Subphase 1.2: Pipeline Probability Extension
*Dependencies: none (can run in parallel with 1.1)*

- **Task 1.2.1:** Add `Probability int` to `StageInfo` struct in `api/internal/pipeline/config.go`. Update `DefaultStages()` with default probabilities: New Lead=5, Contacted=15, Qualified=30, Proposal=50, Negotiation=75, Closed Won=100, Closed Lost=0, Nurturing=10. Update `ParseConfigFromMetadata` to read/write probability. Add package-level comment distinguishing `Probability` (close likelihood for forecasting) from the `scoring` package's `lead_score` (qualification rank). (traces: FR-10)
  - Acceptance: Existing pipeline tests pass unchanged; probability defaults correct; per-org override via org metadata works; package comment present

- **Task 1.2.2:** Add `WeightedForecast(amountCents int64, probability int) int64` helper. Effective probability logic: use `metadata.probability_override` (0–100 int) if set, else use stage default. (traces: FR-9, FR-10)
  - Acceptance: Unit tests for override vs. default; edge cases 0%, 100%, nil override; integer arithmetic (no floating point rounding loss)

- **Task 1.2.3:** Verify the existing `scoring.Service.HandleStageChanged` continues to fire on `PipelineStageChanged` events as before — the scoring engine and probability extension are additive and independent. Add a test confirming both `lead_score` and `probability_override` coexist in Thread metadata without collision after a stage transition. (traces: FR-10)
  - Acceptance: `lead_score` still recalculated on stage change; `probability_override` set by transition endpoint; no metadata key collision

#### Subphase 1.3: Config & Environment
*Dependencies: none*

- **Task 1.3.1:** Add `DEFT_CEO_USER_ID string` to existing config struct. Add `CRMConfig` struct: `StaleOpportunityDays int` (default 30), `DataQualityCheckDays int` (default 14). Add all new env vars to `secrets/secrets.example.env` with placeholder values and comments. (traces: FR-39, FR-49)
  - Acceptance: Config loads from env; sensible defaults applied when env vars absent; unit test for missing/invalid values

---

### Phase 2: CRM Entity CRUD APIs
*Depends on: Phase 1 complete. This is the most critical phase — all other phases build on it.*

#### Subphase 2.1: CRM Space Initialization
*Dependencies: Phase 1.1*

- **Task 2.1.1:** `api/internal/crm/init.go` — `EnsureCRMSpace(ctx context.Context, db *gorm.DB, orgID string) error`. Idempotently creates: CRM Space (type=crm, slug=crm), and three Boards (`companies`, `contacts`, `opportunities`). Safe to call multiple times. Called during first CRM access for an org. (traces: FR-1)
  - Acceptance: Idempotent — second call produces no changes; integration test verifies Space and all three Boards exist after call

#### Subphase 2.2: ACL Engine
*Dependencies: Phase 1.1*

- **Task 2.2.1:** `api/internal/crm/acl.go` — `ACLEngine` struct with methods:
  - `GrantOwner(ctx, tx, threadID, userID string) error`
  - `RevokeOwner(ctx, tx, threadID, userID string) error`
  - `GrantTaskAssignee(ctx, tx, threadID, userID string) error`
  - `RevokeTaskAssignee(ctx, tx, threadID, userID string) error`
  - `RecalculateLinkedContactACL(ctx, tx, contactThreadID string) error` — adds grant for every current Opportunity owner linked to this Contact via `crm_links`
  - `HasAccess(ctx, threadID, userID string) (bool, error)`

  All grant/revoke mutations **must** accept an in-progress `*gorm.DB` transaction from the caller. Never open their own transaction. (traces: FR-14, FR-15, NFR-5)
  - Acceptance: Unit tests for every grant/deny path; RecalculateLinkedContactACL verified to cascade correctly; concurrent calls within same transaction produce no races

- **Task 2.2.2:** `api/internal/crm/repository.go` — `FilterByACL(db *gorm.DB, userID string, isAdmin bool) *gorm.DB` GORM scope. For non-admin: joins `thread_acl` on `thread_id = threads.id AND user_id = ?`. For admin: no filter applied. (traces: FR-15, NFR-1, NFR-2)
  - Acceptance: Non-admin query returns only ACL-granted threads; admin returns all; benchmark confirms `idx_thread_acl_user` index used (no full scans)

#### Subphase 2.3: Company API
*Dependencies: 2.1, 2.2*

- **Task 2.3.1:** `api/internal/crm/company/` — repository, service, handler for Company CRUD:
  - `POST /v1/orgs/{org}/crm/companies` — create Thread (crm_type=company), store attributes in metadata, grant ACL to creator as owner (atomic in transaction)
  - `GET /v1/orgs/{org}/crm/companies` — list with `FilterByACL`, cursor pagination, metadata filtering (`?metadata[status]=...`)
  - `GET /v1/orgs/{org}/crm/companies/{id}` — get with ACL check; 404 if no access
  - `PATCH /v1/orgs/{org}/crm/companies/{id}` — deep-merge metadata update; owner or admin only
  - `DELETE /v1/orgs/{org}/crm/companies/{id}` — soft delete; owner or admin only
  
  All mutations write audit log entry. (traces: FR-3, FR-5, FR-15, FR-44)
  - Acceptance: Non-member gets 404 on get; non-owner gets 403 on patch/delete; audit entry on every mutation; cursor pagination works

- **Task 2.3.2:** Duplicate detection: `GET /v1/orgs/{org}/crm/companies/check-duplicate?name=X` — returns matching Company threads by name similarity (exact match + case-insensitive contains). (traces: FR-17)
  - Acceptance: Returns matches; empty array for unique names; respects ACL (only returns visible Companies)

#### Subphase 2.4: Contact API
*Dependencies: 2.1, 2.2*

- **Task 2.4.1:** `api/internal/crm/contact/` — Contact CRUD mirroring Company pattern. Create endpoint accepts optional `company_id`: if provided, creates `contact_company` crm_links entry and calls `ACLEngine.RecalculateLinkedContactACL` atomically within the create transaction. (traces: FR-6, FR-8, FR-14, FR-44)
  - Acceptance: Contact invisible to non-owner/non-linked-opportunity-owner/non-admin (returns 404); company_id creates crm_links entry; ACL recalculates on company link

- **Task 2.4.2:** Duplicate detection: `GET /v1/orgs/{org}/crm/contacts/check-duplicate?email=X` — returns matching Contact by email. Returns match only if caller has ACL access to it. (traces: FR-17)
  - Acceptance: Caller without ACL to the matching Contact gets empty result (not an information leak)

#### Subphase 2.5: Opportunity API
*Dependencies: 2.1, 2.2, 2.3.1, 2.4.1*

- **Task 2.5.1:** `api/internal/crm/opportunity/` — Opportunity CRUD. Create: requires `company_id` (400 if absent), accepts `contact_ids[]` + `primary_contact_id`. Creates all `crm_links` entries and ACL grants atomically. Calculates and stores `metadata.weighted_forecast` on create and every update. (traces: FR-9, FR-11, FR-12, FR-14, FR-44)
  - Acceptance: Missing company_id returns 400; weighted_forecast recalculated on amount or probability_override change; crm_links entries created; Opportunity invisible to non-owner/non-admin

- **Task 2.5.2:** Stage transition endpoint: `POST /v1/orgs/{org}/crm/opportunities/{id}/transition` body: `{stage, comment?, reason?}`.
  - Wraps existing `pipeline.Service.TransitionStage`
  - Updates `metadata.probability_override` if empty (sets stage default probability)
  - Recalculates and persists `weighted_forecast`
  - Requires `reason` for backward stage movement
  - Requires `close_reason` for Closed Won / Closed Lost
  - Publishes `PipelineStageChanged` event (triggers handoff in Phase 8, scoring in existing scoring service)
  (traces: FR-10, FR-13, NFR-5)
  - Acceptance: Backward transition without reason returns 400; close without close_reason returns 400; weighted_forecast updated; event published

#### Subphase 2.6: Entity Linking API
*Dependencies: 2.3.1, 2.4.1, 2.5.1*

- **Task 2.6.1:** `POST /v1/orgs/{org}/crm/links` body: `{from_thread_id, to_thread_id, link_type, is_primary?}`. `DELETE /v1/orgs/{org}/crm/links/{id}`. Both operations: execute crm_links change + ACL recalculation atomically within single transaction.
  - `opportunity_contact` link create/delete: call `RecalculateLinkedContactACL` for the Contact
  - `contact_company` link create/delete: recalculate Contact ACL (company link doesn't change Contact visibility but may cascade in future)
  (traces: FR-2, FR-14, NFR-5)
  - Acceptance: ACL recalculation verified for all link type mutations; concurrent link mutations tested

#### Subphase 2.7: Ownership Reassignment
*Dependencies: 2.6.1*

- **Task 2.7.1:** `POST /v1/orgs/{org}/crm/{entity_type}/{id}/reassign` body: `{new_owner_id}`. Entity types: `companies`, `contacts`, `opportunities`. Flow: within transaction — revoke old owner `thread_acl` grant, grant new owner, if Opportunity: call `RecalculateLinkedContactACL` for all linked Contacts. Write audit log with `before_user_id` and `after_user_id`. (traces: FR-24, FR-25, NFR-4)
  - Acceptance: Old owner loses access if no other grants remain; new owner gains access; audit entry present; admin-only endpoint

- **Task 2.8:** Phase 2 tests — full CRUD lifecycle for all 3 entities, every ACL grant/deny scenario, stage transition validation (forward/backward/close), weighted forecast math with known inputs, concurrent ACL recalculation with parallel goroutines. Fuzz ≥50 on request bodies and metadata JSON. `task test:coverage` ≥85%. (traces: NFR-3)

---

### Phase 3: Tasks, Ownership & Data Quality
*Depends on: Phase 2 complete.*

#### Subphase 3.1: Task CRUD API

- **Task 3.1.1:** `api/internal/crmtask/` — handler, service, repository for `crm_tasks`:
  - `POST /v1/orgs/{org}/crm/tasks` — create task, within transaction: insert crm_tasks row + call `ACLEngine.GrantTaskAssignee(parentThreadID, assignedTo)`
  - `GET /v1/orgs/{org}/crm/tasks` — list with filters: `assigned_to`, `parent_id`, `status`, `due_date_from/to`
  - `PATCH /v1/orgs/{org}/crm/tasks/{id}` — update; if `assigned_to` changes: within transaction revoke old ACL grant (if no other tasks link them to parent), grant new assignee
  - `DELETE /v1/orgs/{org}/crm/tasks/{id}` — soft delete (status → cancelled); revoke ACL grant if no other tasks remain

  Visibility: assigned user, parent record owner, admins only. (traces: FR-20, FR-21, FR-23)
  - Acceptance: ACL grant created on task creation; revoked on delete if no other tasks link assignee to parent; audit logged; non-assignee/non-owner/non-admin gets 403/404

#### Subphase 3.2: Task Notifications & Reminders

- **Task 3.2.1:** Task event notification integration — publish `task.assigned` event on create/reassign. Subscribe in notification trigger engine. Template: "You've been assigned: {task title} on {entity name} — due {due_date}". Route via existing `NotificationProvider` (in-app + email per user preferences). (traces: FR-22)
  - Acceptance: In-app and email notification delivered on assignment; user preference respected; mock provider in tests

- **Task 3.2.2:** Reminder background goroutine in `crmtask` package startup. Nightly at configurable time (default 07:00 UTC). Queries:
  - Tasks with `due_date = tomorrow` AND `status IN (open, in-progress)` → fire `task.due_tomorrow` notification to assignee
  - Tasks with `due_date < today` AND `status IN (open, in-progress)` → fire `task.overdue` notification to assignee (throttled: once per task per day via a `last_notified_at` field)
  (traces: FR-22)
  - Acceptance: Integration test with time-mocked task records; no duplicate notifications; respects user preferences

#### Subphase 3.3: Data Quality Rules Engine

- **Task 3.3.1:** `api/internal/crmtask/quality.go` — `QualityRule` struct: `EntityType`, `FieldPath`, `Condition`, `GracePeriodHours`. `DefaultQualityRules()` returns:
  - Opportunity: `deal_amount` missing after 48h of creation
  - Opportunity: `expected_close_date` missing after 48h of creation
  - Opportunity: no Company link (immediate)
  - Opportunity: no activity (Messages) in last `DataQualityCheckDays` days
  - Contact: `email` missing (immediate)
  - Contact: `phone` missing (immediate)

  `EvaluateRecord(ctx, db, thread Thread) []QualityViolation` runs applicable rules. (traces: FR-49)
  - Acceptance: Each rule correctly identifies violations in unit tests with fixture data

- **Task 3.3.2:** Quality check trigger — after every Company/Contact/Opportunity `PATCH` or `POST`: publish `quality.check_requested` event with threadID (async, non-blocking). Nightly background goroutine scans all open (non-closed) CRM entities. Both trigger `EvaluateRecord` and publish `quality.violation` event if violations found. (traces: FR-49, NFR-8)
  - Acceptance: POST/PATCH handler returns immediately; check runs in background; nightly scan covers all open records

- **Task 3.3.3:** Quality violation handler — subscribes to `quality.violation` event. Calls `LLMProvider.QualityMessage(ctx, violations, thread)` asynchronously. Sends in-app notification to record owner with LLM-generated message. Accumulates violations for daily admin digest (sent via existing digest mechanism at end of day). (traces: FR-49, NFR-8)
  - Acceptance: LLM call non-blocking; admin digest sent once per day with all violations; mock LLM provider in tests

- **Task 3.4:** Phase 3 tests — task full lifecycle (create/assign/complete/cancel), ACL grant/revoke for all task mutations, notification firing for all events, reminder scan with time-mocked tasks (due tomorrow, overdue), quality rule evaluation for all rules, mock LLM provider. `task test:coverage` ≥85%. (traces: NFR-3)

---

### Phase 4: Gmail Integration via Existing Channel Infrastructure
*Extends the existing `channel` package and `EmailInbox` model. Depends on Phase 1.1.5, Phase 2. Can be developed in parallel with Phases 3, 6, 7, 8.*

**Design:** Admin registers each sales rep's `@deft.co` address as an `EmailInbox` with `routing_action = "sales_crm"` in system channel settings. The rep then connects their personal Gmail credentials via the CRM profile. `sales@deft.co` is configured as an `EmailInbox` with the existing `routing_action = "sales_lead"` (already supported by the Gateway). All email routing flows through the existing `ChannelGateway.Process` pipeline.

#### Subphase 4.1: EmailInbox Model & RoutingAction Extensions

- **Task 4.1.1:** Add `sales_crm` to `RoutingAction` enum in `api/internal/models/email-inbox.go`. Update `ValidRoutingActions()`. Add four nullable fields to `EmailInbox` (as detailed in Architecture section): `AssignedUserID`, `OAuthRefreshToken` (encrypted), `PubSubSubscription`, `LastHistoryID`. Update `EmailInboxService.Create/Update` inputs to accept `AssignedUserID`. (traces: FR-27, NFR-1)
  - Acceptance: `sales_crm` passes `IsValid()`; nullable fields AutoMigrate without breaking existing rows; existing `support_ticket` and `sales_lead` inboxes unaffected

- **Task 4.1.2:** Update `EmailInboxService` auth: allow the `AssignedUserID` user to call `PUT /v1/orgs/{org}/channels/email/inboxes/{id}` for their own inbox (in addition to org admins). This lets reps store their OAuth token without admin involvement. Encrypt `OAuthRefreshToken` at rest using AES-256-GCM with the app's existing master secret key; mask as `[REDACTED]` in all API responses. (traces: FR-27, NFR-1)
  - Acceptance: Assigned rep can PATCH credentials on their own inbox; admin can PATCH any; token encrypted in DB; never appears in logs or API responses

#### Subphase 4.2: Channel Gateway Extension for `sales_crm` Routing

- **Task 4.2.1:** Extend `Gateway.resolveThread` to handle `sales_crm` routing. Add `RoutingAction` to `InboundEvent` (set by the Pub/Sub webhook handler from the inbox's config). When `routing_action = sales_crm`: look up a CRM Contact Thread (`crm_type = "contact"`) where `metadata.email` matches `evt.SenderIdentifier`, scoped to threads the `AssignedUserID` rep has ACL access to (join `thread_acl`). If Contact matched: route Message to Contact Thread and, via `crm_links`, also persist on the most recent open linked Opportunity Thread if one exists. If no match: insert into `crm_unassigned_emails` for that rep. (traces: FR-29, FR-14)
  - Acceptance: `sales_crm` inbound routes to correct Contact/Opportunity Thread; unmatched goes to `crm_unassigned_emails`; existing `support_ticket` and `sales_lead` routing paths unchanged; unit tests for all routing paths

- **Task 4.2.2:** Unassigned CRM inbox API (extends existing channel package):
  - `GET /v1/orgs/{org}/crm/email/inbox` — lists `crm_unassigned_emails` scoped to requesting rep (`assigned_user_id = userID`)
  - `POST /v1/orgs/{org}/crm/email/inbox/{id}/assign` body `{entity_type, entity_id}` — creates `Message{type: "email"}` on target Thread, deletes from `crm_unassigned_emails`
  (traces: FR-29)
  - Acceptance: Inbox scoped to rep; assign moves message to entity timeline; idempotent

#### Subphase 4.3: Gmail API + Pub/Sub Push Handler

- **Task 4.3.1:** Gmail watch lifecycle for `sales_crm` EmailInbox records. When rep stores OAuth token (from Phase 4.4): call Gmail API `users.watch` with Pub/Sub topic, store subscription name and initial `historyId` in `EmailInbox.PubSubSubscription` / `LastHistoryID`. Background goroutine (in channel package startup) renews watches before 7-day expiry for all `sales_crm` inboxes with a token. On disconnect: call `users.stop`, clear fields. (traces: FR-27, FR-29)
  - Acceptance: Watch created when token first stored; renewal goroutine tested with mocked Gmail API; watch stopped on disconnect

- **Task 4.3.2:** `POST /v1/crm/email/webhook` — unified Gmail Pub/Sub push endpoint. Lookup `EmailInbox` by the notification's email address. Decrypt `OAuthRefreshToken`, fetch new messages since `LastHistoryID` via Gmail API. For each message: construct `InboundEvent{ChannelType: email, RoutingAction: inbox.RoutingAction, OrgID: inbox.OrgID, AssignedUserID: inbox.AssignedUserID, ...}`. Pass to `Gateway.Process`. Update `LastHistoryID`. Idempotent on `gmail_message_id`. Publish `email.received` async event for LLM summary (Phase 5.3). (traces: FR-29, FR-46, NFR-8)
  - Acceptance: Messages routed via Gateway for correct routing action; idempotent on replay; LLM summary event published async; fuzz ≥50 on webhook payload; mocked in tests

#### Subphase 4.4: CRM Profile OAuth2 Flow

- **Task 4.4.1:** Gmail OAuth2 endpoints (in `channel` package, separate from admin inbox CRUD):
  - `GET /v1/crm/email/auth` — redirects to Google OAuth2 consent. Scopes: `https://mail.google.com/`, `gmail.send`, `pubsub`. State encodes `{user_id, org_id}`.
  - `GET /v1/crm/email/callback` — exchanges code for tokens. Looks up `EmailInbox` by `assigned_user_id = userID` and `routing_action = "sales_crm"`. Stores encrypted refresh token in `EmailInbox.OAuthRefreshToken`. Triggers Gmail watch setup (Task 4.3.1). Returns 400 if no matching EmailInbox exists (admin must create the inbox record first).
  - `DELETE /v1/crm/email/disconnect` — revokes Google token, clears `OAuthRefreshToken` + watch fields.
  - `GET /v1/crm/email/status` — returns `{connected: bool, email_address: string}` for the requesting user's inbox.
  (traces: FR-27, NFR-1)
  - Acceptance: OAuth round-trip works (mocked in tests); token encrypted; not in logs; 400 if admin hasn't pre-registered inbox; disconnect revokes Google token and stops watch

#### Subphase 4.5: Admin Channel Settings UI Extension

- **Task 4.5.1:** Extend existing `email-inbox-form` frontend component: add `sales_crm` → "Sales CRM" to routing action dropdown. When `sales_crm` selected: show `AssignedUserID` user-picker (searches DEFT sales org members). Hide IMAP credential fields (`imap_host`, `imap_port`, `password`) for `sales_crm` inboxes — rep provides credentials separately. Add info note: *"The assigned rep connects their Gmail account via CRM Settings → Email."* (traces: FR-27)
  - Acceptance: Form shows/hides correct fields per routing action; `AssignedUserID` required for `sales_crm`; existing form behaviour unchanged; Vitest component test

#### Subphase 4.6: `sales@deft.co` Lead Generation Inbox

- **Task 4.6.1:** Admin configures `sales@deft.co` as an `EmailInbox` in system channel settings with `routing_action = "sales_lead"`. The existing `Gateway.createLeadThread` already handles this correctly. Verify the `sales_lead` path also creates/updates a `models.Lead` record (not just a Thread): on `sales_lead` event, upsert `Lead{email: sender, name: from_header, source: "email", status: "anonymous"}` and send admin notification. This is an additive fix to the existing Gateway. (traces: FR-32a)
  - Acceptance: Inbound to `sales@deft.co` creates Lead record; deduplicates by sender email; admin notification fires; mocked in tests

- **Task 4.7:** Phase 4 tests — `sales_crm` routing action validation, Gateway routing for all four actions (`support_ticket`, `sales_lead`, `sales_crm`, `general`), OAuth callback token storage and encryption (mocked Google API), Gmail watch lifecycle, Pub/Sub webhook handler (matched + unmatched + duplicate), unassigned inbox assign flow, `sales@deft.co` Lead creation and deduplication. Fuzz ≥50 on webhook payload. `task test:coverage` ≥85%. (traces: NFR-3)

---

### Phase 5: LLM Features
*Depends on: Phases 2, 3, 4.*

#### Subphase 5.1: LLMProvider Interface Extension

- **Task 5.1.1:** Extend `api/internal/llm/provider.go` with 5 new interface methods (additive — existing `Summarize` and `SuggestNextAction` methods and the `POST .../threads/{thread}/enrich` handler are preserved unchanged):
  ```go
  Briefing(ctx context.Context, userID string, opps []models.Thread, tasks []crmtask.CRMTask, recentActivity []models.Message) (string, error)
  EmailSummary(ctx context.Context, email models.Message, entityThread models.Thread) (string, error)
  PipelineStrategy(ctx context.Context, opps []models.Thread) (string, error)
  DealStrategy(ctx context.Context, opp models.Thread, messages []models.Message, tasks []crmtask.CRMTask) (string, error)
  QualityMessage(ctx context.Context, violations []QualityViolation, record models.Thread) (string, error)
  ```
  Implement all 5 in `GrokProvider`. Add `MockLLMProvider` returning deterministic responses for all methods. (traces: FR-45–FR-49, NFR-8)
  - Acceptance: GrokProvider implements interface; existing `Summarize`/`SuggestNextAction` tests still pass; mock returns stable test output; no direct Grok API calls outside this package

#### Subphase 5.2: AI Briefing & On-Demand Strategy Endpoints

- **Task 5.2.1:** `POST /v1/orgs/{org}/crm/ai/brief` (authenticated, DEFT sales org member). Context assembly: load requesting user's open Opportunities (all ACL-visible, sorted by stage/close date), open Tasks (assigned to user, sorted by due date/priority), recent activity (Messages on user's entities in last 7 days). Call `LLMProvider.Briefing`. Respond with Server-Sent Events stream. (traces: FR-45, NFR-8)
  - Acceptance: Context scoped to user's ACL; SSE stream delivers incrementally; mock LLM in tests; returns within 15s against real provider

- **Task 5.2.2:** Daily briefing job — on startup, register background goroutine. At 08:00 UTC (configurable), for each DEFT sales org member with `notification_preferences.daily_crm_brief = true`: assemble context same as 5.2.1, call `LLMProvider.Briefing`, deliver as in-app notification via `NotificationProvider`. Skip users with zero open records. (traces: FR-45)
  - Acceptance: Only fires for opted-in users; skips empty pipelines; preference flag respected; mock LLM in tests

- **Task 5.2.3:** `POST /v1/orgs/{org}/crm/opportunities/{id}/strategy` — "Close This Deal Now". Authorization: Opportunity owner or admin. Context: Thread + last 20 Messages (all types) + linked Company Thread + linked Contact Threads + open Tasks. Call `LLMProvider.DealStrategy`. SSE streaming response. (traces: FR-48, NFR-8)
  - Acceptance: Non-owner/non-admin gets 403; context assembled correctly (verified in tests by inspecting mock LLM call args); SSE stream works; 15s budget

- **Task 5.2.4:** `POST /v1/orgs/{org}/crm/ai/pipeline-strategy` — CEO strategic analysis. Authorization: `DEFT_CEO_USER_ID` or admin only (403 otherwise). Context: all open Opportunities across org (unfiltered by ACL). Call `LLMProvider.PipelineStrategy`. SSE streaming response. (traces: FR-47, NFR-8)
  - Acceptance: Non-CEO/non-admin gets 403; unfiltered context (all org Opps); SSE streams

#### Subphase 5.3: Inbound Email AI Summary

- **Task 5.3.1:** Extend Phase 4.3.2 webhook handler: after persisting matched inbound Message, publish async `email.received` event `{messageID, entityThreadID, recipientUserID}`. Background subscriber: call `LLMProvider.EmailSummary(email, entityThread)`. Deliver result as `Message{type: "system", body: "📧 " + summary}` notification to the entity owner / assigned rep via in-app `NotificationProvider`. (traces: FR-46, NFR-8)
  - Acceptance: Webhook handler returns immediately; summary notification fires asynchronously; mock LLM in tests; unmatched emails (no entity thread) skipped

- **Task 5.4:** Phase 5 tests — all LLM endpoints with mock provider, context assembly correctness (inspect mock call arguments), CEO access gate, async email summary non-blocking (webhook returns before LLM call), daily briefing skips opted-out users. `task test:coverage` ≥85%. (traces: NFR-3)

---

### Phase 6: Reports Backend — Extend Existing `reporting/` Package
*Depends on: Phases 2, 3. Can be developed in parallel with Phases 4, 7, 8.*

> **Existing assets:** `api/internal/reporting/` already implements `GetPipelineFunnel`, `GetLeadVelocity`, `GetWinLossCounts`, `GetAvgDealValue`, `GetLeadsByAssignee`, `GetScoreDistribution`, `GetStageTransitions`, and platform-wide admin variants of all of these. The frontend at `/reports/sales/` has working charts for pipeline funnel, lead velocity, leads by assignee, score distribution, stage conversion, and time in stage. All new queries are **added to the existing `reporting/` package** — no new package is created.

#### Subphase 6.1: Pipeline & Forecasting Reports

- **Task 6.1.1:** Add to `api/internal/reporting/repository-pipeline-crm.go` (new file in existing package):
  - `PipelineByStage` — extends existing `GetPipelineFunnel` to add USD value, weighted forecast (`deal_amount × effective_probability / 100`), and avg deal size per stage. Adds `FilterByACL` scope for non-admin users. Accept common filter struct: `DateRange`, `OwnerID`, `Stage`, `OpportunityType`, `LeadSource`.
  - `PipelineByOwner` — total value + weighted forecast + count per owner (extends existing `GetLeadsByAssignee` to include USD amounts)
  - `PipelineByCloseDate` — Opportunities grouped by month/quarter of expected_close_date
  - `OverdueOpportunities` — open Opportunities past expected_close_date
  (traces: FR-33, FR-32)
  - Acceptance: Existing reporting tests unaffected; new queries return correct aggregations; ACL scoping verified; benchmark <5s for 12 months data

- **Task 6.1.2:** Add to `api/internal/reporting/repository-forecast.go` (new file in existing package):
  - `ForecastByOwner` — weighted forecast for current month/quarter/year per owner
  - `ForecastByStage` — weighted forecast contribution per stage
  - `ForecastAccuracy` — historical weighted forecast snapshot vs. actual closed-won

  Weighted forecast = `SUM(deal_amount × effective_probability / 100)` where effective_probability = `probability_override` if set, else stage default probability from `StageInfo`. (traces: FR-34)
  - Acceptance: Weighted forecast math unit-tested with exact inputs; distinct from existing `lead_score` queries

#### Subphase 6.2: Activity, Performance & Entity Reports

- **Task 6.2.1:** Add to `api/internal/reporting/repository-activity.go` (new file in existing package):
  - `ActivityByUser` — joins Messages + `crm_tasks` to count emails sent, tasks created/completed, notes added, calls logged per user over date range
  - `TasksOverdue` — `crm_tasks` past due_date, grouped by assignee
  - `TasksCompleted` — completion rate per assignee + period
  (traces: FR-35)
  - Acceptance: Counts match fixture data; completion rate = completed / (completed + open + in-progress + cancelled)

- **Task 6.2.2:** Add to `api/internal/reporting/repository-performance.go` (new file in existing package):
  - `WinRate` — `GetWinLossCounts` already exists; extend to segment by owner/source/type/period
  - `AvgSalesCycle` — avg days from Opportunity `created_at` to `closed_at` (from audit log; `GetStageTransitions` already queries audit log — extend this pattern)
  - `ConversionByStage` — `GetStageTransitions` already exists; compute stage-to-stage conversion rates from it
  - `SourcePerformance` — Opportunity count + closed-won value by `metadata.lead_source`
  (traces: FR-36)
  - Acceptance: Win rate extends existing `GetWinLossCounts`; sales cycle uses existing audit log query pattern; no query duplication

- **Task 6.2.3:** Add to `api/internal/reporting/repository-entities.go` (new file in existing package):
  - `CompaniesByStatus` — count + list grouped by `metadata.status` on `crm_type=company` Threads, ACL-scoped
  - `ContactsByCompany` — Contact count per Company via `crm_links`
  - `RecentlyModified` — Companies/Contacts/Opportunities sorted by `updated_at`
  (traces: FR-37)
  - Acceptance: Counts match fixture data; ACL-scoped for non-admin

#### Subphase 6.3: Show Me the Money Report

- **Task 6.3.1:** Add to `api/internal/reporting/repository-smtm.go` (new file in existing package) — two sections:

  *Section 1:*
  - Summary metrics: weighted forecast closing this month, this quarter, unweighted pipeline closing this quarter, deals closing in 30/60/90 days, overdue open deal count
  - Deals closing soon: all open Opps with expected_close_date within selector period, columns: name, company, primary contact, stage, effective probability, weighted value, deal amount, expected close date, owner name, days until close (negative = overdue), sorted by close_date ASC + weighted_value DESC

  *Section 2:*
  - All open Opportunities: same columns + opportunity age (days since created_at) + days in current stage (from stage history)
  - Pipeline stats: total open count, total pipeline value, total weighted pipeline, breakdown by stage (count + value), breakdown by owner (count + value + weighted), top 10 by deal_amount, stalled Opportunities (no Message activity in last `StaleOpportunityDays` days)

  **No ACL filtering** — unfiltered across entire org. CEO/admin access gate enforced in handler. (traces: FR-39, FR-40)
  - Acceptance: All metric calculations verified with exact fixture data; stalled uses `CRMConfig.StaleOpportunityDays`; performance <5s

- **Task 6.3.2:** Extend existing report handler in `api/internal/reporting/handler.go` to add new CRM report routes. Add new route group `GET /v1/orgs/{org}/reports/crm/{type}` alongside existing `/reports/sales` and `/reports/support` routes. Common filter middleware (date range, owner, stage, type, source). `?format=csv` returns RFC 4180 CSV. CEO report endpoint (`/reports/crm/show-me-the-money`): 403 for non-CEO/non-admin. (traces: FR-32, FR-38, FR-40)
  - Acceptance: All new report types accessible; existing `/reports/sales` endpoints unaffected; CSV export correct; ACL applied; CEO gate tested

- **Task 6.4:** Phase 6 tests — all new report queries with deterministic fixture data, ACL scoping verified, weighted forecast math, CSV export correctness, CEO gate enforced, benchmarks for slow queries. Confirm existing `reporting/` tests still pass. `task test:coverage` ≥85%. (traces: NFR-3)

---

### Phase 7: CSV Import
*Depends on: Phase 2. Can be developed in parallel with Phases 4, 6, 8.*

- **Task 7.1:** `api/internal/crmimport/parser.go` — `POST /v1/orgs/{org}/crm/import/preview` (multipart form, `file` field + `entity_type` field). Parse CSV rows, validate each row:
  - Required fields present (name for Company; name+email for Contact)
  - Email format (Contact)
  - Field length limits (name ≤255, description ≤10000, etc.)
  - Duplicate detection: query DB for existing Company by name / Contact by email

  Return: `{valid_rows: [{row_num, data}], error_rows: [{row_num, field, message}], duplicate_warnings: [{row_num, existing_id, message}]}`. **No DB writes.** (traces: FR-18, FR-19)
  - Acceptance: All validation types caught; duplicate detection queries DB; fuzz ≥50 CSV inputs (malformed, encoding edge cases, injection attempts)

- **Task 7.2:** `POST /v1/orgs/{org}/crm/import/apply` body: `{entity_type, rows: [{...validated row data}]}`. Executes within single `db.Transaction`. For each row: create Thread + ACL grant (owner = `owner_email` field resolved to user ID, else importing user). On any row failure: rollback entire batch. Write `crm_import_history` entry on success or failure. Return: `{imported_count, errors: [...]}`. (traces: FR-18, FR-19, FR-26, NFR-5)
  - Acceptance: All rows created or none; history entry written on both success and failure; ACL grants created for each row; owner_email resolution falls back to importer

- **Task 7.3:** Phase 7 tests — validation catches all error types, full rollback on single-row failure, history written in both cases, ACL grants verified, fuzz ≥50 on CSV parser. `task test:coverage` ≥85%. (traces: NFR-3)

---

### Phase 8: Closed Won Handoff
*Depends on: Phase 2. Can be developed in parallel with Phases 4, 6, 7.*

- **Task 8.1:** `api/internal/crmhandoff/service.go` — `HandoffService` registered on server startup. Subscribes to `event.PipelineStageChanged` from the existing event bus.

  > **Existing Closed Won subscribers:** The existing `api/internal/provision/` package and `api/internal/conversion/` package also subscribe to `PipelineStageChanged` on `closed_won` — `provision` auto-provisions a customer Org+Spaces, and `conversion` promotes the lead user to Tier 3. The `HandoffService` is a third independent subscriber. All three are designed to run in parallel and independently — each is non-blocking on failure. No orchestration dependency between them is required. Tests must verify all three fire on a single `closed_won` event without interfering.

  On `new_stage == "closed_won"`:
  1. Load Opportunity Thread + metadata
  2. Load linked Company Thread (via crm_links `opportunity_company`)
  3. Load primary Contact Thread (via crm_links `opportunity_contact` where `is_primary = true`)
  4. Load last 5 Messages from Opportunity Thread
  5. Compose support ticket: title = "New Customer Onboarding: {company_name}", body = structured summary (see PRD FR-44a), `entity_ref` = Opportunity thread ID
  6. Call `support.Service.CreateTicket(ctx, ticket)`
  7. Notify `support_team_user_ids` from org metadata as ticket assignees
  8. Send copy notification to `finance_team_user_ids`
  9. Send confirmation notification to Opportunity owner: "Handoff ticket #{id} created for {company_name}"
  10. On any failure in steps 5–9: write audit log entry `{action: "handoff_failed", error: ...}` + send admin alert notification. **Do NOT return error — stage transition already committed.** (traces: FR-44a, NFR-4)
  - Acceptance: Ticket created with all required fields; support+finance notified; owner notified; failure is non-blocking; integration test verifies all three `closed_won` subscribers (handoff, provision, conversion) fire independently without interference

- **Task 8.2:** Phase 8 tests — success path (ticket created, all notifications fire), failure path (ticket creation fails → stage transition still succeeds → audit entry + admin alert), Closed Lost does NOT trigger handoff, mock support service. `task test:coverage` ≥85%. (traces: NFR-3)

---

### Phase 9: Frontend — Core Entity Views
*Depends on: Phase 2 complete. Frontend work begins here.*

- **Task 9.1:** Update sidebar navigation to include CRM section: Companies, Contacts, Pipeline, Inbox, Import. CRM nav items visible only to DEFT sales org members. Add breadcrumb support for `/crm/...` routes. (traces: FR-42)
  - Acceptance: Nav hidden for non-DEFT-sales-org users; breadcrumbs render correctly; Vitest component test

- **Task 9.2:** Company list page (`/crm/companies`): sortable/filterable table (name, industry, status, owner, created date), cursor pagination, search bar. Company detail page (`/crm/companies/[id]`): attributes card, linked Contacts panel (ACL-filtered, shows "0 visible contacts" if none), Opportunity summary panel (count + open/closed breakdown), activity timeline component (Messages of all types, sorted descending). (traces: FR-3, FR-4, FR-5)
  - Acceptance: Non-visible Contacts/Opps hidden; activity timeline shows all message types; Playwright E2E

- **Task 9.3:** Company create/edit form: all attribute fields, real-time duplicate check (debounced call to 2.3.2 on name change), warning dialog on duplicate requiring confirmation. (traces: FR-3, FR-16, FR-17)
  - Acceptance: Duplicate warning appears; form validation; successful create navigates to detail

- **Task 9.4:** Contact list + detail pages mirroring Company. Detail: parent Company link (one-click nav), Opportunity list panel (ACL-filtered), activity timeline. List page: Contact not visible for non-owner/non-linked-rep (API returns 404 → redirect to list with message). (traces: FR-6, FR-7, FR-8)
  - Acceptance: Playwright E2E for ACL: rep A creates Contact → rep B (not linked) cannot see it

- **Task 9.5:** Contact create/edit form with Company association picker (searchable dropdown of visible Companies), email duplicate check. (traces: FR-6, FR-16, FR-17)
  - Acceptance: Company picker works; email duplicate check fires; form validation

- **Task 9.6:** Opportunity Kanban view (`/crm/pipeline`) — **extends existing `web/src/app/crm/page.tsx` and `crm-pipeline-view.tsx`**. The existing Kanban aggregates CRM threads across orgs and already renders stage columns. Extend to: add deal amount, weighted forecast, and close date to cards; wire drag-and-drop to the new transition endpoint (with reason modal for backward moves, close modal for won/lost); add overdue badge; add filter bar (owner, stage, type, source); add list view toggle. (traces: FR-9, FR-11, FR-13)
  - Acceptance: Existing Kanban preserved as starting point; drag fires transition API; invalid transition shows error toast; overdue badge appears; Playwright E2E for drag-and-drop

- **Task 9.7:** Opportunity detail page (`/crm/pipeline/[id]`) — **extends existing `web/src/app/crm/leads/[org]/[space]/[board]/[thread]/page.tsx`**. The existing lead detail page renders Thread content and messages. Extend to: add two-column layout with metadata sidebar (probability override field, weighted forecast auto-display, stage progress bar with days-in-stage + age + overdue indicator); add full action bar (Advance Stage, Close Won/Lost with required reason modal, Create Task, Add Note, Attach File, Reassign, Edit, "Close This Deal Now" button); add Related Tasks panel. (traces: FR-11, FR-13)
  - Acceptance: Existing thread display preserved; all new action buttons trigger correct API calls; required modals enforce reasons; Playwright E2E for full opportunity lifecycle

- **Task 9.8:** Opportunity create/edit form: Company picker (required, shows validation error if empty), Contact picker (primary + secondary multi-select), deal amount (USD formatted), expected close date picker, opportunity type select, lead source select, pipeline stage select, probability override (optional numeric input 0–100). Weighted forecast auto-calculated and displayed as user types amount or probability. (traces: FR-9, FR-16)
  - Acceptance: Company required; weighted forecast updates live; contact picker works

- **Task 9.9:** Phase 9 tests — Vitest component tests for all new components, Playwright E2E: (sign-in → create Company → create Contact linked to Company → create Opportunity → advance through pipeline → close won → verify handoff notification). ACL E2E: rep A creates Contact, rep B cannot see it in list or via direct URL. `task test:coverage` ≥85%. (traces: NFR-3)

---

### Phase 10: Frontend — Email & Tasks
*Depends on: Phase 4 (email backend), Phase 9 (entity views).*

- **Task 10.1:** Gmail OAuth configuration UI in CRM profile/settings page — "Email" section. Checks `/v1/crm/email/status` on load. If **connected**: displays connected address + "Disconnect" button. If **disconnected**: shows "Connect your @deft.co Gmail account" button → triggers `/v1/crm/email/auth` OAuth redirect. If **no inbox registered** (admin hasn't created an EmailInbox for this rep yet): shows informational message "Contact your admin to enable email integration for your account." (traces: FR-27)
  - Acceptance: All three states render correctly; OAuth flow completes (mocked in E2E); connected state persists on reload; disconnect clears token and shows disconnected state; Vitest component test for all states

- **Task 10.2:** Tiptap email composer component — reusable `<EmailComposer>` embedded within Company, Contact, and Opportunity detail pages (behind "Compose Email" button → slide-over panel). Fields: to, cc, bcc, subject, rich body (bold/italic/lists/tables/images/links), file attachment picker (CRM uploads + local), signature selector (stored in user profile), email template picker. "Send" button calls `/v1/orgs/{org}/crm/email/send`. If Gmail not connected: shows "Connect Gmail in Settings first" inline prompt. Sent email appears in activity timeline without page reload. (traces: FR-28, FR-30, FR-31)
  - Acceptance: Email appears in timeline after send; attachment upload works; template picker populated; "not connected" state shown; Vitest component tests

- **Task 10.3:** Unassigned inbox view (`/crm/inbox`): table of unmatched inbound emails (sender, subject, snippet, date). Row action: "Assign to..." opens entity picker modal (search Companies/Contacts/Opportunities). Nav badge shows unread count. (traces: FR-29)
  - Acceptance: Assign moves email to entity timeline; badge updates; Vitest test

- **Task 10.4:** Task slide-over panel `<TaskPanel>` — create/edit task: title, description (rich text), assignee picker (DEFT sales org members, searchable), due date, priority select, status select. Cancel/Save buttons. Task list panel (`<TaskList>`) on each Company/Contact/Opportunity detail page: shows open tasks sorted by due date, with complete/cancel inline actions. Global "My Tasks" view at `/crm/tasks`: all tasks assigned to current user, grouped by due date with overdue section. (traces: FR-20, FR-21, FR-22)
  - Acceptance: Task appears on entity detail after create; assignee receives in-app notification; overdue tasks highlighted in red; Vitest component tests

- **Task 10.5:** Extend existing notification preferences page: add "CRM Tasks" section with toggles for `task_assigned`, `task_due_tomorrow`, `task_overdue` per channel (in-app / email). Add "Daily AI Pipeline Briefing" toggle (opt-in, default off). (traces: FR-22, FR-45)
  - Acceptance: Toggles persist; notifications respect preferences on next firing; Vitest test

- **Task 10.6:** Phase 10 tests — Vitest component tests for EmailComposer, TaskPanel, TaskList, inbox view, preferences toggles. Playwright E2E: (connect Gmail mock → open Opportunity → compose email → email in timeline; create task on Contact → assignee receives notification). `task test:coverage` ≥85%. (traces: NFR-3)

---

### Phase 11: Frontend — LLM Features
*Depends on: Phase 5 (LLM backend), Phase 9 (entity views).*

- **Task 11.1:** "Brief Me" panel — floating action button in CRM dashboard sidebar/header. Click: calls `POST /v1/orgs/{org}/crm/ai/brief`, renders streamed response in right-side slide-over panel with loading skeleton. Response rendered as formatted markdown. Each Opportunity in the briefing has a "Close This Deal Now →" shortcut button that opens the strategy panel (Task 11.2) for that Opportunity. Panel dismissible. (traces: FR-45)
  - Acceptance: Panel opens; streaming renders incrementally; CTDN shortcut works; loading state shown; Vitest component test; mock API in tests

- **Task 11.2:** "Close This Deal Now" slide-over — triggered by the "Close This Deal Now" button in Opportunity detail (Task 9.7) and from Brief Me panel (Task 11.1). Calls `POST /v1/orgs/{org}/crm/opportunities/{id}/strategy`, streams response. Footer actions: "Save as Note" button (calls Messages create API with type=note + LLM output as body → appears in activity timeline) + "Dismiss". (traces: FR-48)
  - Acceptance: Strategy streams and renders; Save as Note creates Message visible in timeline; Dismiss closes panel; Vitest component test

- **Task 11.3:** CEO strategic analysis in Show Me the Money report (integrated with Phase 12.4). "Generate Strategic Analysis" button visible only to CEO / admin (hidden via `DEFT_CEO_USER_ID` check client-side + server enforces). Click: calls `POST /v1/orgs/{org}/crm/ai/pipeline-strategy`, renders streaming response in a collapsible section above Section 1. "Regenerate" button re-runs analysis. (traces: FR-47)
  - Acceptance: Button hidden for non-CEO; response renders; regenerate clears and re-streams; Vitest test

- **Task 11.4:** Phase 11 tests — Vitest component tests for all three LLM panels (mock SSE stream), CEO button visibility gate, Save as Note creates correct API call. `task test:coverage` ≥85%. (traces: NFR-3)

---

### Phase 12: Frontend — Reports (Extend Existing `/reports/sales/`)
*Depends on: Phase 6 (reports backend), Phase 9 (entity views). Phase 11.3 integrated here.*

> **Existing assets:** `/reports/sales/page.tsx` is a fully implemented React page with working charts for: pipeline funnel, lead velocity, leads by assignee, score distribution, stage conversion, and time in stage — plus a `DateRangePicker`, `AssigneeFilter`, and `ExportButton`. All new report views are added to this existing page as additional tabs/sections. The existing chart components (`PipelineFunnelChart`, `LeadVelocityChart`, etc.) are reused where applicable. **No new `/reports/crm/` route is created.**

- **Task 12.1:** Extend existing `/reports/sales/page.tsx` — add tab navigation: Pipeline, Forecasting, Activity, Performance, Company/Contact (existing pipeline funnel/velocity/assignee/score views become sub-sections of the Pipeline tab). Extend common filter bar with: stage multi-select, opportunity type, lead source (date range picker and assignee filter already exist). Role-based tab visibility: members see data filtered to own records; admins/CEO see unfiltered data. Summary metric cards row at top of each new section. (traces: FR-32, FR-38)
  - Acceptance: Existing pipeline charts still render under new tab structure; new tabs render correctly; filter bar updates all data; member vs. admin scoping verified in Playwright; Vitest tests

- **Task 12.2:** Pipeline & Forecasting views:
  - Pipeline by Stage: grouped bar chart (value + weighted forecast per stage) + summary table (count, value, weighted, avg deal size)
  - Pipeline by Owner: table with total/weighted/count per rep
  - Pipeline by Close Date: stacked bar chart grouped by month/quarter + table
  - Overdue list: table with days overdue column, sortable
  - Forecast by Owner: summary cards (this month / quarter / year) per rep
  - Forecast by Stage: horizontal bar chart + table
  - Forecast Accuracy: line chart comparing historical forecast vs. actuals
  (traces: FR-33, FR-34)
  - Acceptance: Charts render with correct data; Vitest tests for chart data transformation

- **Task 12.3:** Activity, Performance & Entity views:
  - Activity Summary: table by user (emails, tasks created, tasks completed, notes, calls)
  - Tasks Overdue: grouped table by assignee with priority badges
  - Tasks Completed: completion rate bars per user
  - Win Rate: stat cards (by owner, by source, by type) + bar chart
  - Avg Sales Cycle: bar chart by owner/stage
  - Conversion by Stage: funnel chart
  - Source Performance: bar chart + table
  - Companies by Status: donut chart + table
  - Contacts by Company: table
  - Recently Modified: tabbed list (Companies / Contacts / Opportunities)
  (traces: FR-35, FR-36, FR-37)
  - Acceptance: All views render; charts use correct chart library (existing stack); Vitest tests

- **Task 12.4:** "Show Me the Money" report (`/reports/sales/show-me-the-money`) — new route added alongside the existing `/reports/sales/` page. Access: CEO (`DEFT_CEO_USER_ID`) and admins only; others redirected to 403 page. Layout:
  - Header: date range selector (default: current month + next month), quick filter row (owner, stage, company, close date range), CSV Export button
  - LLM strategic analysis section: "Generate Strategic Analysis" button (Phase 11.3) → collapsible narrative block
  - Section 1 — Closing Deals: 5 summary metric cards, deals-closing-soon sortable table, pipeline funnel/bar chart (reuses existing `PipelineFunnelChart` component), "**Total Closing: $X,XXX,XXX**" prominently at bottom
  - Section 2 — Pipeline Overview: all open Opps paginated table (+ age + days-in-stage columns), pipeline stats summary cards, top 10 largest Opps list, stalled Opps table
  (traces: FR-39, FR-40)
  - Acceptance: Non-CEO redirected; all sections render; existing chart components reused; CSV export downloads correct data; Playwright E2E for CEO access, filter updates, CSV download

- **Task 12.5:** Phase 12 tests — Vitest component tests for all new report views and extended filter bar; confirm existing chart component tests still pass. Playwright E2E (reports page access control: member sees own data, admin sees all, CEO sees Show Me the Money at `/reports/sales/show-me-the-money`, non-CEO gets 403, filter bar updates data, CSV export). `task test:coverage` ≥85%. (traces: NFR-3)

---

### Phase 13: Frontend — CSV Import
*Depends on: Phase 7 (import backend), Phase 9 (entity views).*

- **Task 13.1:** Import UI page (`/crm/import`):
  1. **Step 1 — Upload**: entity type selector (Company / Contact), drag-drop file zone or file picker, "Preview" button → calls preview endpoint
  2. **Step 2 — Preview**: three-tab table view — Valid (green rows), Errors (red rows with per-row error detail inline), Duplicates (amber rows with existing record link). Row count summary per tab. "Confirm Import" button (disabled if zero valid rows). "Start Over" button.
  3. **Step 3 — Processing**: progress indicator
  4. **Step 4 — Result**: success count, error count, link to view imported records
  - Import history table at bottom of page: past imports with timestamp, user, entity type, success/error counts, expandable error detail row
  (traces: FR-18, FR-19)
  - Acceptance: Preview shows correct color coding; confirm disabled with zero valid rows; error detail shown inline; history table updates; Playwright E2E for full import flow (upload → preview → confirm → records in Company list)

- **Task 13.2:** Phase 13 tests — Vitest component tests for all import steps, Playwright E2E (full import lifecycle). `task test:coverage` ≥85%. (traces: NFR-3)

---

## Testing Strategy

### Backend (Go)
- Unit tests for all service and repository functions; ≥85% coverage enforced per package via `task test:coverage`
- **ACL engine**: explicit unit tests for every grant type (owner, task_assignee, linked_opportunity_owner), every revoke path, admin bypass, and concurrent recalculation (parallel goroutines)
- **Weighted forecast**: unit tests with exact known inputs for all combinations of stage default vs. per-opportunity override, including edge cases (0%, 100%, nil override, zero amount)
- **Pipeline transitions**: unit tests for every valid and invalid transition in the stage graph; backward movement reason enforcement; close reason enforcement
- **CSV import**: fuzz tests ≥50 inputs on CSV parser (malformed CSV, BOM, CRLF/LF, injection attempts, multi-byte encoding edge cases)
- **Gmail / channel integration**: all Google API calls mocked via interface; OAuth token lifecycle (connect, refresh, expire, disconnect); Pub/Sub webhook handler for all four routing actions; email matching for `sales_crm` (CRM Contact thread lookup); `sales@deft.co` Lead creation; duplicate message idempotency; existing IMAP-based inboxes unaffected
- **LLM features**: `MockLLMProvider` returns deterministic responses; context assembly correctness verified by inspecting mock call arguments; async features (email summary, quality notifications) verified to be non-blocking
- **Concurrency**: ACL recalculation and crm_links mutations tested with parallel goroutines; no visibility gaps permitted
- **Reporting extension**: existing `reporting/` tests must pass unchanged after new queries are added; new queries unit-tested independently with fixture data
- **Closed Won orchestration**: integration test verifies `HandoffService`, `provision.AutoProvision`, and `conversion.SalesConvert` all fire independently on a single `closed_won` event without interfering
- **Existing test suite**: `task check` must pass fully (all pre-existing tests green) before any PR merges

### Frontend (TypeScript)
- Vitest + React Testing Library for all new components (≥85% coverage threshold in `vitest.config.ts`)
- Playwright E2E covering all critical user paths:
  1. Company → Contact → Opportunity → Pipeline → Close Won → Handoff ticket visible in support
  2. ACL enforcement: rep A's Contact/Opportunity not visible to rep B
  3. Gmail OAuth connect → compose email → email appears in activity timeline
  4. Create task on Opportunity → assignee receives in-app notification
  5. CSV import: upload → preview → confirm → records in list
  6. CEO "Show Me the Money": CEO sees report, non-CEO gets 403
  7. Reports access: member sees own data, admin sees all, filter bar updates data
  8. "Brief Me" → response streams → CTDN shortcut → strategy panel → Save as Note
  9. Pipeline drag-and-drop stage transition

---

## Deployment

No new infrastructure required. Existing targets apply.

### Backend (Fly.io)
- New DB tables auto-migrate on startup via GORM `AutoMigrate` (existing pattern)
- **New required secrets** (add to Fly.io secrets + `secrets/secrets.example.env`):
  ```
  DEFT_CEO_USER_ID=<clerk_user_id>
  GMAIL_CLIENT_ID=<google_oauth2_client_id>
  GMAIL_CLIENT_SECRET=<google_oauth2_client_secret>
  GMAIL_PUBSUB_PROJECT_ID=<gcp_project_id>
  ```
- Google Cloud project must have Gmail API and Cloud Pub/Sub API enabled; push subscription endpoint `https://<api-domain>/v1/crm/email/webhook` whitelisted
- Admin configures `sales@deft.co` as an `EmailInbox` record (`routing_action = sales_lead`) in system channel settings post-deploy
- Admin pre-creates `EmailInbox` records for each sales rep (`routing_action = sales_crm`, `assigned_user_id` set) before reps can connect Gmail
- Each rep connects Gmail via CRM Settings → Email after their inbox is pre-registered
- `support_team_user_ids` and `finance_team_user_ids` configured in DEFT org metadata post-deploy

### Frontend (Vercel)
- No new build config required
- New env var: `NEXT_PUBLIC_CRM_ENABLED=true` (feature flag for CRM nav; default false until backend phases complete)

### Pre-deploy Checklist
1. `task check` passes fully on both `api/` and `web/` (lint + typecheck + test:coverage ≥85%)
2. All existing tests green (no regressions)
3. New secrets configured in target environment
4. GORM AutoMigrate run successfully on staging DB before production deploy
5. Gmail API + Pub/Sub enabled on Google Cloud project
6. Playwright smoke test suite run against staging environment

---

*Source: `vbrief/specification-sales-crm.vbrief.json` | PRD: `PRD.md` | Status: **draft — awaiting approval***
*To approve: update `vbrief/specification-sales-crm.vbrief.json` status field from `"draft"` to `"approved"`*
