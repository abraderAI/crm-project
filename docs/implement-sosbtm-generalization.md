# SOSBTM Feature Generalization: Three Plans

The **SOSBTM** hierarchy is: **S**ystem / **O**rg / **S**pace / **B**oard / **T**hread / **M**essage.

This document identifies every feature currently locked to a specific hierarchy level
and proposes three plans for generalizing them to work at any SOSBTM level.

---

## Preliminary: What Is Already Entity-Agnostic

Two key models already use `entity_type` + `entity_id` and require only routing/calling
changes to generalize — no model changes needed:

| Model | Already has | Currently called from |
|---|---|---|
| `Revision` | `entity_type`, `entity_id` | Thread body edits, Message body edits |
| `Upload` | `entity_type`, `entity_id` (form fields) | Thread attachments (globalspace only) |
| `Notification` | `entity_type`, `entity_id` | Thread and message events |

Two models are **hard-coded to Thread** and require structural changes to generalize:

| Model | Hard-coded field | Required change |
|---|---|---|
| `Flag` | `thread_id TEXT NOT NULL` | Replace with `entity_type` + `entity_id` |
| `Vote` | `thread_id TEXT NOT NULL` (unique with user_id) | Replace with `entity_type` + `entity_id` |

Three boolean states exist only on specific models and have no shared mechanism:

| State | Currently on | Missing from |
|---|---|---|
| `is_pinned` | Thread | Message, Board, Space, Org |
| `is_locked` | Thread, Board | Space, Org |
| `is_hidden` | Thread | Message, Board, Space |

---

## Plan 1: Aggressive

**Goal:** Generalize every scope-specific feature to work at any SOSBTM level.
Introduce a shared entity-state mechanism.

### 1A. Generalize Votes to any SOSBTM entity

Replace `Vote.ThreadID` with `entity_type` + `entity_id`:

```go
type Vote struct {
    BaseModel
    EntityType string `gorm:"type:text;not null;uniqueIndex:idx_vote_unique"`
    EntityID   string `gorm:"type:text;not null;uniqueIndex:idx_vote_unique"`
    UserID     string `gorm:"type:text;not null;uniqueIndex:idx_vote_unique"`
    Weight     int    `gorm:"default:1"`
}
```

New routes at every level:
```
POST /orgs/{org}/vote
POST /orgs/{org}/spaces/{space}/vote
POST /orgs/{org}/spaces/{space}/boards/{board}/vote
POST /orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/vote          ← existing
POST /orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/messages/{message}/vote
```

The vote weight table config moves to system metadata (see consolidation plan).

**Files:** `models/vote.go`, `vote/repository.go`, `vote/service.go`, `server/server.go`

### 1B. Generalize Flags to any SOSBTM entity

Replace `Flag.ThreadID` with `entity_type` + `entity_id`:

```go
type Flag struct {
    BaseModel
    EntityType string     `gorm:"type:text;not null;index"`
    EntityID   string     `gorm:"type:text;not null;index"`
    UserID     string     `gorm:"type:text;not null;index"`
    Reason     string     `gorm:"type:text;not null"`
    Status     FlagStatus `gorm:"type:text;not null;default:'open'"`
    ResolvedBy string     `gorm:"type:text"`
}
```

`FlagInput` accepts `entity_type` + `entity_id` instead of `thread_id`.
The org-level flag queue (`ListOrgFlags`) queries by all entities within the org's scope.

```
POST /orgs/{org}/flags  { "entity_type": "message", "entity_id": "..." }
```

**Files:** `models/flag.go`, `moderation/service.go`, `moderation/handler.go`

### 1C. Shared entity-state table for pin/lock/hide

Replace per-model boolean columns with a shared `entity_states` table:

```go
type EntityState struct {
    EntityType string `gorm:"type:text;not null;uniqueIndex:idx_estate"`
    EntityID   string `gorm:"type:text;not null;uniqueIndex:idx_estate"`
    StateKey   string `gorm:"type:text;not null;uniqueIndex:idx_estate"` // "pinned","locked","hidden"
    StateValue bool   `gorm:"not null;default:false"`
    SetBy      string `gorm:"type:text"`
}
```

Remove `is_pinned`, `is_locked`, `is_hidden` from `Thread`; `is_locked` from `Board`.
All pin/lock/hide operations route through a single service.

Generic endpoints at every SOSBTM level:
```
POST /{entity-path}/pin      POST /{entity-path}/unpin
POST /{entity-path}/lock     POST /{entity-path}/unlock
POST /{entity-path}/hide     POST /{entity-path}/unhide
```

Note: filtering by these states (e.g. listing only non-hidden threads) requires a JOIN
on `entity_states`. Retained generated columns on Thread are an acceptable denormalization
for query performance.

**Files:** new `models/entity-state.go`, all packages that currently set booleans

### 1D. Revisions at any SOSBTM level

The model is already entity-agnostic. Extend revision creation to any entity update:
- Board name/description changes
- Space name/description changes
- Org name/description changes
- Thread metadata changes (not just body)
- Message metadata changes

No model change. Add revision snapshot calls in `org/service.go`, `space/service.go`,
`board/service.go`.

### 1E. Uploads at any SOSBTM level

The upload service already accepts `entity_type` + `entity_id`. Add attachment listing
routes at every level:

```
POST   /orgs/{org}/spaces/{space}/attachments
GET    /orgs/{org}/spaces/{space}/attachments
POST   /orgs/{org}/spaces/{space}/boards/{board}/attachments
GET    /orgs/{org}/spaces/{space}/boards/{board}/attachments
POST   /orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/attachments
GET    /orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/attachments
POST   /orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/messages/{message}/attachments
GET    /orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/messages/{message}/attachments
```

**Files:** `server/server.go`, `upload/handler.go` (add scoped list endpoint)

### 1F. LLM enrichment at any SOSBTM level

`POST .../threads/{thread}/enrich` currently enriches only threads. Extend to any entity:

```
POST /orgs/{org}/enrich
POST /orgs/{org}/spaces/{space}/enrich
POST /orgs/{org}/spaces/{space}/boards/{board}/enrich
POST /orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/enrich          ← existing
POST /orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/messages/{message}/enrich
```

**Files:** `llm/handler.go`, `llm/provider.go`, `server/server.go`

### 1G. Entity-level notification subscriptions

Currently notification preferences are user-level by event type only. Add entity-level
watches so users can subscribe to events at any SOSBTM node:

```
POST /orgs/{org}/watch
POST /orgs/{org}/spaces/{space}/watch
POST /orgs/{org}/spaces/{space}/boards/{board}/watch
POST /orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/watch
```

New `EntityWatch` model: `(user_id, entity_type, entity_id, event_types[])`.
Notification routing checks watches in addition to global preferences.

### Net change (Aggressive)
- **Removes:** `is_pinned`/`is_locked`/`is_hidden` columns from Thread and Board
- **Adds:** `entity_states` table; generalized vote/flag routes at all levels;
  attachment routes at all levels; enrichment at all levels; entity watches
- **Risk:** High. Entity-state table requires JOIN for all filtered list queries.

---

## Plan 2: Practical

**Goal:** Generalize features where the gap is genuinely user-facing and the cost
is contained. No shared entity-state table.

### 2A. Generalize Flags to include Messages

The most impactful single change: users can only currently flag a thread, not the
specific message within it that's abusive. Extend `Flag` to support messages:

```go
type Flag struct {
    BaseModel
    EntityType string     // "thread" or "message" (for now)
    EntityID   string
    UserID     string
    Reason     string
    Status     FlagStatus
    ResolvedBy string
}
```

`FlagInput` adds optional `message_id`; if provided, `entity_type = "message"`,
otherwise `entity_type = "thread"`. The org flag queue lists both.

No route change — `POST /orgs/{org}/flags` accepts both.

**Files:** `models/flag.go`, `moderation/service.go`

### 2B. Generalize Votes to include Messages

Adding upvotes/reactions to individual messages is standard UX (Slack, Discord,
Reddit). Extend Vote to support messages alongside threads:

Same model change as Plan 1A but limited to thread + message only (no org/board/space votes).

New route:
```
POST /orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/messages/{message}/vote
```

**Files:** `models/vote.go`, `vote/repository.go`, `vote/service.go`, `server/server.go`

### 2C. Add Lock to Space

Space is the only SOSBTM level currently missing a lock mechanism:
- System: maintenance mode (effectively a lock)
- Org: suspension
- Space: **missing**
- Board: `is_locked`
- Thread: `is_locked`
- Message: `is_immutable`

Add `is_locked bool` to `Space`. Add `POST/DELETE /orgs/{org}/spaces/{space}/lock`
routes. Lock semantics: locked space rejects new board and thread creation.

**Files:** `models/space.go`, `space/service.go`, `space/handler.go`, `server/server.go`

### 2D. Extend Revisions to Board, Space, and Org

The model is already entity-agnostic. When `board.name`/`board.description`,
`space.name`/`space.description`, or `org.name`/`org.description` is updated,
snapshot a revision exactly as thread/message updates do today. Zero model changes.

**Files:** `org/service.go`, `space/service.go`, `board/service.go`

### 2E. Extend Uploads to Messages

Attach files directly to messages (not just threads). The upload service already
handles arbitrary `entity_type`/`entity_id`. Add a message-level attachment listing
route alongside the existing thread-level one:

```
POST   /orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/messages/{message}/attachments
GET    /orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/messages/{message}/attachments
```

**Files:** `server/server.go`, `globalspace/handler.go` (mirror pattern)

### Net change (Practical)
- **Removes:** Nothing structural
- **Adds:** `entity_type`/`entity_id` on Flag and Vote; `is_locked` on Space;
  message-level vote and attachment routes; revisions on org/space/board edits
- **Risk:** Low-medium. Flag/vote model changes require migrations. No query
  performance impact (all new queries remain indexed).

---

## Plan 3: Minimal

**Goal:** Only fix genuine functional gaps — things that are missing or broken
today and have clear user impact. No structural model changes.

### 3A. Generalize Flags to include Messages

Same as Plan 2B. This is the clearest functional gap: a user can report a thread
but not a specific harmful message within it. The `Flag` model change is small
(`thread_id` → `entity_type` + `entity_id`) and the migration is straightforward
(backfill `entity_type = 'thread'`, `entity_id = thread_id`).

### 3B. Extend Revisions to Space and Org

Zero model changes required — `Revision` already has `entity_type` + `entity_id`.
Two service call sites to add. This is a free correctness fix.

### 3C. Document SOSBTM as the canonical hierarchy

Update `AGENTS.md`, OpenAPI descriptions, and code comments to formally establish
SOSBTM (System / Org / Space / Board / Thread / Message) as the canonical hierarchy
name. No code changes.

### Net change (Minimal)
- **Changes:** `Flag` model migration (thread_id → entity_type + entity_id)
- **Adds:** Two revision call sites
- **Risk:** Essentially zero

---

## Summary Comparison

| Feature | Aggressive | Practical | Minimal |
|---|---|---|---|
| Votes | All SOSBTM levels | Thread + Message | No change |
| Flags | All SOSBTM levels | Thread + Message | Thread + Message |
| Pin/unpin | All SOSBTM levels (entity_states) | No change | No change |
| Lock/unlock | All SOSBTM levels (entity_states) | Add to Space | No change |
| Hide/unhide | All SOSBTM levels (entity_states) | No change | No change |
| Revisions | All SOSBTM levels | Org + Space + Board | Org + Space + Board |
| Uploads | All SOSBTM levels | Add to Message | No change |
| LLM enrichment | All SOSBTM levels | No change | No change |
| Entity watches | New subscription model | No change | No change |

## Recommendation

**Start with Plan 3 (Minimal)** immediately — flags extending to messages is a
genuine product gap with near-zero risk, and revision call sites are a one-line
addition per service.

**Then Plan 2 (Practical)** in the next phase — message-level votes, space lock,
and message attachments are all high-value and low-risk.

**Plan 1 (Aggressive)** is an architectural aspiration. The entity-state table
is the right long-term shape but carries meaningful migration risk. Schedule separately.
