# Metadata Consolidation: Three Plans

This document analyses every current REST endpoint and identifies what could be
re-implemented as metadata get/set operations, presented at three levels of aggression.

---

## Reference: Current API Inventory by Suitability

Before the plans, here is a full disposition table.

### Already IS metadata (just informal or undocumented)

| Feature | Current location | Notes |
|---|---|---|
| Pipeline stage | `POST .../threads/{thread}/stage` | Writes directly to `thread.metadata.stage` via `metadata.DeepMerge`. The endpoint is a validated wrapper. |
| Lead score | `scoring/service.go` (internal) | Writes to `thread.metadata.lead_score`. No HTTP exposure. |
| Scoring rules | `scoring/service.go` reads `org.Metadata` | Config lives in `org.metadata.scoring_rules`. |
| Pipeline stages config | `GET /orgs/{org}/pipeline/stages` | Reads from `org.Metadata`; falls back to defaults. |
| Thread status/priority/assigned_to | `globalspace` `UpdateInput` | Written via metadata patch; exposed as generated columns. |

### Strong candidates (clearly config-shaped, no relational requirements)

| Feature | Current location | Notes |
|---|---|---|
| Channel config | `GET/PUT /orgs/{org}/channels/{type}` | `ChannelConfig` is `(org_id, channel_type, settings JSON, enabled bool)`. Pure per-org JSON config keyed by type. |
| Vote weight config | `GET /vote/weights` | A static `WeightConfig` struct (role → int, tier → int). Platform-wide config. |
| Feature flags (global) | `GET/PATCH /admin/feature-flags` | Simple `key → enabled bool`. Already moving toward system metadata. |
| Org-scoped feature flag overrides | `FeatureFlag.OrgScope` | Per-org boolean overrides; currently mixed into system table. |

### Possible but lossy (relational needs exist)

| Feature | Current location | Why it's lossy as metadata |
|---|---|---|
| Votes | `POST .../threads/{thread}/vote` | Per-user uniqueness and weighted sum require a relation. The aggregate score (`thread.vote_score`) and the weight config are metadata-appropriate; the vote records themselves are not. |
| Thread booleans (`is_pinned`, `is_locked`, `is_hidden`) | First-class columns | Used as indexed WHERE predicates; moving to JSON breaks efficient filtering. |
| Board `is_locked` | First-class column | Same: used as a query predicate and gate in middleware. |
| Notification preferences | `PUT /notifications/preferences` | Multiple rows per user (event_type × channel); needs per-row upsert. Becomes a fat JSON blob with no index support. |
| Home preferences | `PUT /me/home-preferences` | Single record per user; more tractable as user-level metadata, but no user entity exists in the hierarchy yet. |

### Not appropriate (transactional / lifecycle records)

| Feature | Reason |
|---|---|
| Votes (individual records) | Relation, not config |
| Moderation flags | Lifecycle state machine (pending → resolved/dismissed) |
| Moderation thread ops (move, merge, hide) | Structural operations on the entity graph |
| Webhooks | Operational records with delivery history |
| Billing (invoices, customers) | Financial records |
| Channel DLQ | Transactional event log |
| Notifications (inbox) | Transactional delivery records |
| Audit log | Append-only event log |
| Revisions | Append-only history |
| Support entries | Complex lifecycle (draft → published → immutable) |
| Uploads | File management records |
| Voice call logs | Transactional records |
| Conversion operations | State-machine transitions |

---

## Plan 1: Aggressive

**Goal:** Fit as much as possible into the metadata model. Accept some ergonomic costs
for the sake of surface-area reduction.

### Changes

**1. Channel config → org metadata**

Remove `channel_configs` table and `GET/PUT /orgs/{org}/channels/{type}` endpoints.
Channel configuration moves to `org.metadata.channels`:

```json
{
  "channels": {
    "email": { "enabled": true, "settings": { "from": "support@acme.com" } },
    "sms":   { "enabled": false, "settings": {} }
  }
}
```

Access via `GET/PATCH /orgs/{org}/metadata`. Health and DLQ endpoints remain unchanged
(they are transactional, not config).

**2. Feature flags (global + per-org) → metadata**

Global flags move to system metadata under `feature_flags`:

```json
{ "feature_flags": { "voice_module": false, "community_voting": true } }
```

Per-org overrides move to `org.metadata.feature_overrides`:

```json
{ "feature_overrides": { "voice_module": true } }
```

Remove the `feature_flags` table, `SeedFeatureFlags`, `ToggleFeatureFlag`,
`GET/PATCH /admin/feature-flags`. Evaluation logic merges system metadata +
org metadata override at read time.

**3. Vote weight config → system metadata**

`GET /vote/weights` is replaced by `GET /admin/metadata` — the weight config lives
under `vote_weights` in system metadata. The service reads it from there at startup
or per-request. Remove the dedicated weights endpoint.

**4. Pipeline stage transition → pure metadata PATCH**

`POST .../threads/{thread}/stage` is removed. Callers write directly to thread metadata:

```
PATCH .../threads/{thread}/metadata
{ "stage": "qualified" }
```

Stage validation (valid stage name, valid transition) moves to a metadata write hook
or middleware. The pipeline service becomes a pure domain library (no HTTP handler).

**5. Thread booleans → metadata**

`is_pinned`, `is_locked`, `is_hidden` on `Thread`, and `is_locked` on `Board`, move to
their respective metadata objects. Remove first-class columns and the dedicated
`pin/unpin`, `lock/unlock`, `hide/unhide` endpoints. All operations become:

```
PATCH .../threads/{thread}/metadata  { "is_pinned": true }
PATCH .../boards/{board}/metadata    { "is_locked": true }
```

Filtering/listing by these values requires JSON extraction queries (or retained
generated columns).

**6. Notification preferences → user metadata (requires new user scope)**

Add a `users` entity to the hierarchy at the system level with a `metadata` column.
Notification preferences and home preferences both move to `user.metadata`:

```json
{
  "notification_preferences": { "thread.created": { "email": true, "push": false } },
  "home_preferences": { "tier": 2, "layout": [...] },
  "digest": { "frequency": "daily", "enabled": true }
}
```

Remove `notification_preferences`, `digest_schedules`, and `user_home_preferences`
tables and their endpoints.

### Net change
- **Removes:** `channel_configs`, `feature_flags`, `user_home_preferences`,
  `notification_preferences`, `digest_schedules` tables
- **Removes:** ~12 dedicated endpoints
- **Adds:** user scope entity + metadata
- **Risk:** Loss of indexed filtering for thread booleans; feature flag evaluation
  becomes a two-level metadata merge on every request; notification preference
  queries require JSON extraction

---

## Plan 2: Practical

**Goal:** Consolidate things that are genuinely config-shaped and map cleanly to a
single entity's metadata. Retain dedicated surfaces where relational properties,
lifecycle, or query performance matter.

### Changes

**1. Channel config → org metadata**

Same as Plan 1. This is a clean fit: channel config is per-org, JSON-structured,
keyed by channel type, and has no relational requirements.

Remove `channel_configs` table. Migrate existing rows into `org.metadata.channels`
on deploy. Remove `GET/PUT /orgs/{org}/channels/{type}` endpoints.

Keep: DLQ and health endpoints (transactional, unrelated to config).

**2. Global feature flags → system metadata**

Global flags (no `OrgScope`) move to system metadata under `feature_flags`. The
`SeedFeatureFlags` logic becomes part of `SeedSettings`. The flags endpoint
`GET/PATCH /admin/feature-flags` is folded into `GET/PATCH /admin/metadata`.

**Per-org overrides stay as a separate table** (or move to org metadata, see Plan 3
for the minimal take). They need to be queryable to determine which orgs have flag
overrides, and the current `OrgScope` column supports that.

**3. Vote weight config → system metadata**

The `WeightConfig` (role weights + tier bonuses) is static platform config. It moves
to system metadata under `vote_weights`. The vote service reads it from there.
`GET /vote/weights` becomes a read from `GET /admin/metadata`.

The `votes` table and the `POST .../threads/{thread}/vote` toggle endpoint remain
unchanged — individual vote records need the relational uniqueness constraint.

**4. Pipeline stage transition → keep endpoint, make it explicit it writes metadata**

Do NOT remove the `/stage` endpoint — it provides validated transitions (checking
valid stage names and transition rules against org config). It is too useful to drop.

However, document clearly that the endpoint is a metadata write shortcut: the same
effect is achievable via `PATCH .../threads/{thread}/metadata`. Callers who want to
bypass validation can use the raw metadata PATCH.

**5. Scoring rules and pipeline stages config → document as org metadata**

These are already stored in `org.metadata`. No code change; add documentation and
OpenAPI descriptions making the metadata keys explicit and their schemas stable.

### Net change
- **Removes:** `channel_configs` table, `feature_flags` table (global rows only)
- **Removes:** `GET/PUT /orgs/{org}/channels/{type}`,
  `GET/PATCH /admin/feature-flags` (replaced by system metadata)
- **Retains:** Votes, thread booleans, notification preferences, home preferences
  as dedicated surfaces
- **Risk:** Low; changes are mostly config consolidation with no query performance impact

---

## Plan 3: Minimal

**Goal:** Only move things that are already effectively metadata or where the dedicated
endpoint is pure overhead with no added value.

### Changes

**1. Document pipeline stage / scoring as already-metadata operations**

No code changes. Add OpenAPI documentation and code comments stating:
- `POST .../threads/{thread}/stage` is a validated alias for writing `thread.metadata.stage`
- Thread `metadata.lead_score`, `metadata.stage`, `metadata.status`, `metadata.priority`,
  `metadata.assigned_to` are first-class metadata fields with defined schemas
- `GET /orgs/{org}/pipeline/stages` reads from `org.metadata.pipeline_stages` (or defaults)
- Scoring rules live in `org.metadata.scoring_rules`

**2. Vote weight config → system metadata**

The only endpoint in the vote package that is pure config is `GET /vote/weights`.
This moves to system metadata (`system.metadata.vote_weights`). The toggle endpoint
and votes table stay as-is.

This is the only endpoint removed in the minimal plan.

### Net change
- **Removes:** `GET /vote/weights` endpoint (config moves to system metadata)
- **Adds:** documentation of existing metadata keys with stable schemas
- **Risk:** Essentially zero

---

## Specific answer to the vote API question

The vote API **cannot** simply become a metadata operation because:

1. `votes` table has a `UNIQUE(thread_id, user_id)` constraint — one vote per user.
   Storing voters in `thread.metadata.voters` loses this enforcement at the DB level.
2. The weighted sum must be recalculated transactionally whenever a vote is added or
   removed — this needs to read and write across multiple rows atomically.
3. Querying "has user X voted on thread Y" requires an indexed lookup, not a JSON
   blob scan.

What **can** become metadata:
- `Thread.VoteScore` (the aggregate) is already a first-class column and
  could additionally be mirrored in thread metadata for clients that read
  the metadata blob. It is already denormalized — no fundamental change needed.
- `WeightConfig` (role → weight, tier → bonus) is static platform config with no
  relational requirements. This is appropriate for system metadata (Plan 2 + Plan 3).

---

## Recommendation

**Implement Plan 2.** It removes two complete tables and their endpoints
(`channel_configs`, global `feature_flags`) with minimal risk, makes the vote weight
config a proper system-level setting, and leaves the transactional and query-sensitive
surfaces (votes, thread booleans, notification preferences) where they are.

Plan 1 is interesting but the loss of indexed boolean columns for `is_pinned`,
`is_locked`, `is_hidden` on Thread is a real query performance regression, and the
feature flag evaluation change (two-level metadata merge per request) adds latency
without meaningful simplification.

Plan 3 is safe but too conservative — channel config and global feature flags are
genuinely config-shaped and belong in metadata.
